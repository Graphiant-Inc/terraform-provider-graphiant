package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	sdk "github.com/Graphiant-Inc/graphiant-sdk-go"
)

var (
	_ resource.Resource                   = &playbookConfigResource{}
	_ resource.ResourceWithConfigure      = &playbookConfigResource{}
	_ resource.ResourceWithImportState    = &playbookConfigResource{}
	_ resource.ResourceWithValidateConfig = &playbookConfigResource{}
)

func NewPlaybookConfigResource() resource.Resource {
	return &playbookConfigResource{}
}

type playbookConfigResource struct {
	pd *providerData
}

type playbookConfigResourceModel struct {
	ID              types.String `tfsdk:"id"`
	BundleKey       types.String `tfsdk:"bundle_key"`
	Name            types.String `tfsdk:"name"`
	Description     types.String `tfsdk:"description"`
	Files           types.List   `tfsdk:"files"`
	Status          types.String `tfsdk:"status"`
	CreatedByUserID types.String `tfsdk:"created_by_user_id"`
}

func (r *playbookConfigResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_playbook_config"
}

func (r *playbookConfigResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "An Automation Workflows playbook config: a catalog bundle plus the module YAML files that " +
			"configure it. This is the recommended way to manage network configuration with this provider: the " +
			"bundle's graphiant-playbooks modules handle the underlying device, policy and routing APIs, so one " +
			"config can replace many lower-level resources. Create and update validate the files server-side (failing the apply with the validation " +
			"errors) and then stage the config with name/description, matching the Portal's Edit & Validate → Save & " +
			"Stage steps. Staging does not change the network: run it with graphiant_playbook_job, which performs " +
			"the mandatory dry-run and only deploys once approved. Use the graphiant_playbook_bundles, " +
			"graphiant_playbook_module_slots and graphiant_playbook_templates data sources to discover bundle keys, " +
			"module keys and starter YAML.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Config id.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"bundle_key": schema.StringAttribute{
				Required:      true,
				Description:   "Catalog bundle key (e.g. system_bundle), from graphiant_playbook_bundles. Changing it forces a new config.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Display name shown in the Portal's Pending Executions table.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Description: "Notes saved with the config when it is staged.",
			},
			"files": schema.ListNestedAttribute{
				Required:    true,
				Description: "Module YAML files for the bundle's module slots (see graphiant_playbook_module_slots).",
				Validators:  []validator.List{listvalidator.SizeAtLeast(1)},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"module_key": schema.StringAttribute{
							Required:    true,
							Description: "Catalog module key of the slot this file fills.",
						},
						"filename": schema.StringAttribute{
							Required:    true,
							Description: "File name, e.g. ntp.yaml.",
						},
						"content": schema.StringAttribute{
							Required:    true,
							Description: "YAML content of the file, typically file(\"${path.module}/configs/ntp.yaml\").",
						},
					},
				},
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Description: "Config status (e.g. STAGED). Changes as jobs run against the config.",
			},
			"created_by_user_id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *playbookConfigResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.pd = configurePD(req.ProviderData, &resp.Diagnostics)
}

func (r *playbookConfigResource) stage(ctx context.Context, id string, m *playbookConfigResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	body := sdk.V1SdkAutomationPlaybookConfigsConfigIdStagePutRequest{
		Name:        m.Name.ValueStringPointer(),
		Description: m.descriptionForAPI(),
	}
	_, httpResp, err := r.pd.api.DefaultAPI.V1SdkAutomationPlaybookConfigsConfigIdStagePut(ctx, id).
		Authorization(r.pd.token).
		V1SdkAutomationPlaybookConfigsConfigIdStagePutRequest(body).
		Execute()
	closeBody(httpResp)
	if err != nil {
		diags.AddError("Unable to stage playbook config", apiErrorDetail(err))
	}
	return diags
}

// descriptionForAPI sends an unset description as "" rather than omitting it:
// the field is omitempty on the wire, so omitting it would leave a previously
// set description in place on the server.
func (m *playbookConfigResourceModel) descriptionForAPI() *string {
	return sdk.PtrString(m.Description.ValueString())
}

// ValidateConfig rejects two files entries with the same module_key and filename,
// which the API would otherwise accept but which can't round-trip through state.
func (r *playbookConfigResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var files types.List
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("files"), &files)...)
	if resp.Diagnostics.HasError() || files.IsNull() || files.IsUnknown() {
		return
	}
	var models []playbookModuleFileModel
	resp.Diagnostics.Append(files.ElementsAs(ctx, &models, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	seen := make(map[string]bool, len(models))
	for i, m := range models {
		if m.ModuleKey.IsUnknown() || m.Filename.IsUnknown() {
			continue
		}
		k := m.ModuleKey.ValueString() + "\x00" + m.Filename.ValueString()
		if seen[k] {
			resp.Diagnostics.AddAttributeError(path.Root("files").AtListIndex(i), "Duplicate playbook file",
				fmt.Sprintf("module_key %q with filename %q appears more than once in files.", m.ModuleKey.ValueString(), m.Filename.ValueString()))
		}
		seen[k] = true
	}
}

func (r *playbookConfigResource) readByID(ctx context.Context, id string) (*sdk.SdkAutomationPlaybookConfig, bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	out, httpResp, err := r.pd.api.DefaultAPI.V1SdkAutomationPlaybookConfigsConfigIdGet(ctx, id).Authorization(r.pd.token).Execute()
	closeBody(httpResp)
	if err != nil {
		if isNotFound(httpResp) {
			return nil, false, diags
		}
		diags.AddError("Unable to read playbook config", apiErrorDetail(err))
		return nil, false, diags
	}
	if out == nil || out.Config == nil {
		return nil, false, diags
	}
	return out.Config, true, diags
}

func (r *playbookConfigResource) deleteByID(ctx context.Context, id string) diag.Diagnostics {
	var diags diag.Diagnostics
	httpResp, err := r.pd.api.DefaultAPI.V1SdkAutomationPlaybookConfigsConfigIdDelete(ctx, id).Authorization(r.pd.token).Execute()
	closeBody(httpResp)
	if err != nil && !isNotFound(httpResp) {
		diags.AddError("Unable to delete playbook config", apiErrorDetail(err))
	}
	return diags
}

// applyConfig copies server-side fields into m. Files are only refreshed when
// withFiles is set (Read); Create/Update keep the planned files so a server-side
// reformat of the YAML can't trip Terraform's post-apply consistency check, and
// any such difference surfaces as drift on the next plan instead.
func (m *playbookConfigResourceModel) applyConfig(ctx context.Context, c *sdk.SdkAutomationPlaybookConfig, withFiles bool) diag.Diagnostics {
	if c.GetConfigId() != "" {
		m.ID = types.StringValue(c.GetConfigId())
	}
	if c.GetBundleKey() != "" {
		m.BundleKey = types.StringValue(c.GetBundleKey())
	}
	if c.GetName() != "" {
		m.Name = types.StringValue(c.GetName())
	}
	m.Description = optionalStringValue(c.Description, m.Description)
	m.Status = types.StringPointerValue(c.Status)
	m.CreatedByUserID = types.StringPointerValue(c.CreatedByUserId)
	if withFiles {
		files, diags := flattenPlaybookFiles(ctx, c.Files, m.Files)
		if diags.HasError() {
			return diags
		}
		m.Files = files
	}
	return nil
}

func (r *playbookConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan playbookConfigResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	files, diags := expandPlaybookFiles(ctx, plan.Files)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := sdk.V1SdkAutomationPlaybookConfigsPostRequest{
		BundleKey:    plan.BundleKey.ValueStringPointer(),
		Files:        files,
		ValidateOnly: sdk.PtrBool(false),
	}
	out, httpResp, err := r.pd.api.DefaultAPI.V1SdkAutomationPlaybookConfigsPost(ctx).
		Authorization(r.pd.token).
		V1SdkAutomationPlaybookConfigsPostRequest(body).
		Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create playbook config", apiErrorDetail(err))
		return
	}
	if out == nil {
		resp.Diagnostics.AddError("Unable to create playbook config", "API returned an empty response")
		return
	}
	id := out.GetConfigId()
	if len(out.Errors) > 0 {
		if id != "" {
			// Don't leak a config Terraform will never track.
			resp.Diagnostics.Append(r.deleteByID(ctx, id)...)
		}
		resp.Diagnostics.AddError("Playbook config failed validation", playbookValidationErrorsDetail(out.Errors))
		return
	}
	if id == "" {
		resp.Diagnostics.AddError("Unable to create playbook config", "API response did not include a config id")
		return
	}

	if diags := r.stage(ctx, id, &plan); diags.HasError() {
		resp.Diagnostics.Append(diags...)
		resp.Diagnostics.Append(r.deleteByID(ctx, id)...)
		return
	}

	cfg, found, diags := r.readByID(ctx, id)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.Diagnostics.AddError("Unable to read created playbook config", "config was created but could not be read back")
		return
	}
	resp.Diagnostics.Append(plan.applyConfig(ctx, cfg, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *playbookConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state playbookConfigResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg, found, diags := r.readByID(ctx, state.ID.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(state.applyConfig(ctx, cfg, true)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *playbookConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan playbookConfigResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	id := plan.ID.ValueString()

	files, diags := expandPlaybookFiles(ctx, plan.Files)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := sdk.V1SdkAutomationPlaybookConfigsConfigIdPutRequest{
		Name:        plan.Name.ValueStringPointer(),
		Description: plan.descriptionForAPI(),
		Files:       files,
	}
	out, httpResp, err := r.pd.api.DefaultAPI.V1SdkAutomationPlaybookConfigsConfigIdPut(ctx, id).
		Authorization(r.pd.token).
		V1SdkAutomationPlaybookConfigsConfigIdPutRequest(body).
		Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update playbook config", apiErrorDetail(err))
		return
	}
	if out != nil && (len(out.Errors) > 0 || (out.IsValid != nil && !*out.IsValid)) {
		resp.Diagnostics.AddError("Playbook config failed validation", playbookValidationErrorsDetail(out.Errors))
		return
	}

	// Re-stage so the updated config is runnable again, as the Portal's Save & Stage does.
	resp.Diagnostics.Append(r.stage(ctx, id, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg, found, diags := r.readByID(ctx, id)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.Diagnostics.AddError("Unable to update playbook config", "config no longer exists")
		return
	}
	resp.Diagnostics.Append(plan.applyConfig(ctx, cfg, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *playbookConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state playbookConfigResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(r.deleteByID(ctx, state.ID.ValueString())...)
}

func (r *playbookConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
