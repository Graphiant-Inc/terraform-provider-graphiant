package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	sdk "github.com/Graphiant-Inc/graphiant-sdk-go"
)

var (
	_ resource.Resource                = &appListResource{}
	_ resource.ResourceWithConfigure   = &appListResource{}
	_ resource.ResourceWithImportState = &appListResource{}
)

func NewAppListResource() resource.Resource {
	return &appListResource{}
}

type appListResource struct {
	pd *providerData
}

type appListEntryModel struct {
	ID   types.Int64  `tfsdk:"id"`
	Type types.String `tfsdk:"type"`
}

var appListEntryAttrTypes = map[string]attrType{
	"id":   types.Int64Type,
	"type": types.StringType,
}

type appListResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Apps        types.List   `tfsdk:"apps"`
}

func (r *appListResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app_list"
}

func (r *appListResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A named list of applications (custom or built-in), used as a match target in policies.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"description": schema.StringAttribute{
				Optional: true,
			},
			"apps": schema.ListNestedAttribute{
				Optional: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							Required:    true,
							Description: "ID of a custom app or built-in app/category.",
						},
						"type": schema.StringAttribute{
							Required:    true,
							Description: "App identifier type, as defined by the API (values are not enumerated in the SDK).",
						},
					},
				},
			},
		},
	}
}

func (r *appListResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.pd = configurePD(req.ProviderData, &resp.Diagnostics)
}

func (m *appListResourceModel) buildConfig(ctx context.Context) (*sdk.ManaV2AppListConfig, diag.Diagnostics) {
	cfg := &sdk.ManaV2AppListConfig{
		Name:        m.Name.ValueStringPointer(),
		Description: m.Description.ValueStringPointer(),
	}

	if !m.Apps.IsNull() && !m.Apps.IsUnknown() {
		var apps []appListEntryModel
		diags := m.Apps.ElementsAs(ctx, &apps, false)
		if diags.HasError() {
			return nil, diags
		}
		for _, a := range apps {
			cfg.Apps = append(cfg.Apps, sdk.ManaV2AppIdentifier{
				Id:   a.ID.ValueInt64Pointer(),
				Type: a.Type.ValueStringPointer(),
			})
		}
	}

	return cfg, nil
}

func (m *appListResourceModel) applyConfig(ctx context.Context, cfg *sdk.ManaV2AppListConfig) diag.Diagnostics {
	m.Name = types.StringPointerValue(cfg.Name)
	m.Description = types.StringPointerValue(cfg.Description)

	entries := make([]appListEntryModel, 0, len(cfg.Apps))
	for _, a := range cfg.Apps {
		entries = append(entries, appListEntryModel{
			ID:   types.Int64PointerValue(a.Id),
			Type: types.StringPointerValue(a.Type),
		})
	}
	list, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: appListEntryAttrTypes}, entries)
	if diags.HasError() {
		return diags
	}
	m.Apps = list
	return nil
}

func (r *appListResource) readByID(ctx context.Context, id int64) (*sdk.ManaV2AppListConfig, bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	out, httpResp, err := r.pd.api.DefaultAPI.V1GlobalAppsAppListsAppListIdGet(ctx, id).Authorization(r.pd.token).Execute()
	if err != nil {
		if isNotFound(httpResp) {
			closeBody(httpResp)
			return nil, false, diags
		}
		closeBody(httpResp)
		diags.AddError("Unable to read app list", apiErrorDetail(err))
		return nil, false, diags
	}
	closeBody(httpResp)
	if out == nil || out.AppListConfig == nil {
		return nil, false, diags
	}
	return out.AppListConfig, true, diags
}

func (r *appListResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan appListResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg, diags := plan.buildConfig(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := sdk.V1GlobalAppsAppListsPostRequest{AppListConfig: cfg}
	out, httpResp, err := r.pd.api.DefaultAPI.V1GlobalAppsAppListsPost(ctx).
		Authorization(r.pd.token).
		V1GlobalAppsAppListsPostRequest(body).
		Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create app list", apiErrorDetail(err))
		return
	}
	if out == nil || out.AppIdentifier == nil || out.AppIdentifier.Id == nil {
		resp.Diagnostics.AddError("Unable to create app list", "API returned an empty response")
		return
	}

	created, found, diags2 := r.readByID(ctx, *out.AppIdentifier.Id)
	resp.Diagnostics.Append(diags2...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.Diagnostics.AddError("Unable to read created app list", "app list was created but could not be read back")
		return
	}

	plan.ID = types.StringValue(int64ID(*out.AppIdentifier.Id))
	resp.Diagnostics.Append(plan.applyConfig(ctx, created)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *appListResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state appListResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := parseInt64ID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid app list id", err.Error())
		return
	}

	cfg, found, diags := r.readByID(ctx, id)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(state.applyConfig(ctx, cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *appListResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan appListResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg, diags := plan.buildConfig(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := parseInt64ID(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid app list id", err.Error())
		return
	}

	body := sdk.V1GlobalAppsAppListsAppListIdPutRequest{AppListConfig: cfg}
	_, httpResp, err := r.pd.api.DefaultAPI.V1GlobalAppsAppListsAppListIdPut(ctx, id).
		Authorization(r.pd.token).
		V1GlobalAppsAppListsAppListIdPutRequest(body).
		Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update app list", apiErrorDetail(err))
		return
	}

	updated, found, diags2 := r.readByID(ctx, id)
	resp.Diagnostics.Append(diags2...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.Diagnostics.AddError("Unable to update app list", "app list no longer exists")
		return
	}

	resp.Diagnostics.Append(plan.applyConfig(ctx, updated)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *appListResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state appListResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := parseInt64ID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid app list id", err.Error())
		return
	}

	_, httpResp, err := r.pd.api.DefaultAPI.V1GlobalAppsAppListsAppListIdDelete(ctx, id).Authorization(r.pd.token).Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to delete app list", apiErrorDetail(err))
	}
}

func (r *appListResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
