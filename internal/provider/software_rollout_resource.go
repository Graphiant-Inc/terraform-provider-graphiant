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
	_ resource.Resource                = &softwareRolloutResource{}
	_ resource.ResourceWithConfigure   = &softwareRolloutResource{}
	_ resource.ResourceWithImportState = &softwareRolloutResource{}
)

func NewSoftwareRolloutResource() resource.Resource {
	return &softwareRolloutResource{}
}

type softwareRolloutResource struct {
	pd *providerData
}

type softwareRolloutResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Action      types.String `tfsdk:"action"`
	Name        types.String `tfsdk:"name"`
	Release     types.String `tfsdk:"release"`
	Description types.String `tfsdk:"description"`
	DeviceIDs   types.List   `tfsdk:"device_ids"`
	TriggerNow  types.Bool   `tfsdk:"trigger_now"`
	Status      types.String `tfsdk:"status"`
	HasFailed   types.Bool   `tfsdk:"has_failed"`
	NumDevices  types.Int64  `tfsdk:"num_devices"`
}

func (r *softwareRolloutResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_software_rollout"
}

func (r *softwareRolloutResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A software upgrade rollout campaign for a set of devices. Recurring schedule configuration " +
			"(RolloutConfig.Schedule) is not yet exposed here; use trigger_now for an immediate schedule instead.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"action": schema.StringAttribute{
				Required: true,
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"release": schema.StringAttribute{
				Required: true,
			},
			"description": schema.StringAttribute{
				Optional: true,
			},
			"device_ids": schema.ListAttribute{
				Optional:    true,
				ElementType: types.Int64Type,
			},
			"trigger_now": schema.BoolAttribute{
				Optional:    true,
				Description: "When true, triggers an immediate schedule run (V1SoftwareRolloutsSchedulePost) on every apply.",
			},
			"status": schema.StringAttribute{
				Computed: true,
			},
			"has_failed": schema.BoolAttribute{
				Computed: true,
			},
			"num_devices": schema.Int64Attribute{
				Computed: true,
			},
		},
	}
}

func (r *softwareRolloutResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.pd = configurePD(req.ProviderData, &resp.Diagnostics)
}

func (m *softwareRolloutResourceModel) buildConfig(ctx context.Context) (sdk.UpgradeRolloutConfig, diag.Diagnostics) {
	cfg := sdk.UpgradeRolloutConfig{
		Action:      m.Action.ValueString(),
		Name:        m.Name.ValueString(),
		Release:     m.Release.ValueString(),
		Description: m.Description.ValueStringPointer(),
	}
	if !m.DeviceIDs.IsNull() && !m.DeviceIDs.IsUnknown() {
		diags := m.DeviceIDs.ElementsAs(ctx, &cfg.DeviceIds, false)
		if diags.HasError() {
			return cfg, diags
		}
	}
	return cfg, nil
}

func (m *softwareRolloutResourceModel) applyRollout(ctx context.Context, rollout *sdk.UpgradeRollout) diag.Diagnostics {
	if rollout.Id != nil {
		m.ID = types.StringValue(int64ID(*rollout.Id))
	} else {
		m.ID = types.StringNull()
	}
	m.Status = types.StringPointerValue(rollout.Status)
	m.HasFailed = types.BoolPointerValue(rollout.HasFailed)
	if rollout.NumDevices != nil {
		m.NumDevices = types.Int64Value(int64(*rollout.NumDevices))
	} else {
		m.NumDevices = types.Int64Value(0)
	}
	if rollout.RolloutConfig != nil {
		m.Action = types.StringValue(rollout.RolloutConfig.Action)
		m.Name = types.StringValue(rollout.RolloutConfig.Name)
		m.Release = types.StringValue(rollout.RolloutConfig.Release)
		m.Description = types.StringPointerValue(rollout.RolloutConfig.Description)
		deviceIDs, diags := types.ListValueFrom(ctx, types.Int64Type, rollout.RolloutConfig.DeviceIds)
		if diags.HasError() {
			return diags
		}
		m.DeviceIDs = deviceIDs
	}
	return nil
}

func (r *softwareRolloutResource) readByID(ctx context.Context, id int64) (*sdk.UpgradeRollout, bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	out, httpResp, err := r.pd.api.DefaultAPI.V1SoftwareRolloutsIdGet(ctx, id).Authorization(r.pd.token).Execute()
	if err != nil {
		if isNotFound(httpResp) {
			closeBody(httpResp)
			return nil, false, diags
		}
		closeBody(httpResp)
		diags.AddError("Unable to read software rollout", apiErrorDetail(err))
		return nil, false, diags
	}
	closeBody(httpResp)
	if out == nil || out.Rollout == nil {
		return nil, false, diags
	}
	return out.Rollout, true, diags
}

func (r *softwareRolloutResource) triggerSchedule(ctx context.Context, id int64) diag.Diagnostics {
	var diags diag.Diagnostics
	body := sdk.V1SoftwareRolloutsSchedulePostRequest{Id: id}
	_, httpResp, err := r.pd.api.DefaultAPI.V1SoftwareRolloutsSchedulePost(ctx).
		Authorization(r.pd.token).
		V1SoftwareRolloutsSchedulePostRequest(body).
		Execute()
	closeBody(httpResp)
	if err != nil {
		diags.AddError("Unable to trigger software rollout schedule", apiErrorDetail(err))
	}
	return diags
}

func (r *softwareRolloutResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan softwareRolloutResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg, diags := plan.buildConfig(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := sdk.V1SoftwareRolloutsPostRequest{RolloutConfig: cfg}
	out, httpResp, err := r.pd.api.DefaultAPI.V1SoftwareRolloutsPost(ctx).
		Authorization(r.pd.token).
		V1SoftwareRolloutsPostRequest(body).
		Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create software rollout", apiErrorDetail(err))
		return
	}
	if out == nil || out.Id == nil {
		resp.Diagnostics.AddError("Unable to create software rollout", "API returned an empty response")
		return
	}

	rollout, found, diags2 := r.readByID(ctx, *out.Id)
	resp.Diagnostics.Append(diags2...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.Diagnostics.AddError("Unable to read created software rollout", "rollout was created but could not be read back")
		return
	}

	resp.Diagnostics.Append(plan.applyRollout(ctx, rollout)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.TriggerNow.ValueBool() {
		resp.Diagnostics.Append(r.triggerSchedule(ctx, *out.Id)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *softwareRolloutResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state softwareRolloutResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := parseInt64ID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid software rollout id", err.Error())
		return
	}

	rollout, found, diags := r.readByID(ctx, id)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(state.applyRollout(ctx, rollout)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *softwareRolloutResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan softwareRolloutResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := parseInt64ID(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid software rollout id", err.Error())
		return
	}

	cfg, diags := plan.buildConfig(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := sdk.V1SoftwareRolloutsPutRequest{Id: id, RolloutConfig: cfg}
	_, httpResp, err := r.pd.api.DefaultAPI.V1SoftwareRolloutsPut(ctx).
		Authorization(r.pd.token).
		V1SoftwareRolloutsPutRequest(body).
		Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update software rollout", apiErrorDetail(err))
		return
	}

	rollout, found, diags2 := r.readByID(ctx, id)
	resp.Diagnostics.Append(diags2...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.Diagnostics.AddError("Unable to update software rollout", "rollout no longer exists")
		return
	}

	resp.Diagnostics.Append(plan.applyRollout(ctx, rollout)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if plan.TriggerNow.ValueBool() {
		resp.Diagnostics.Append(r.triggerSchedule(ctx, id)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *softwareRolloutResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state softwareRolloutResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := parseInt64ID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid software rollout id", err.Error())
		return
	}

	_, httpResp, err := r.pd.api.DefaultAPI.V1SoftwareRolloutsIdDelete(ctx, id).Authorization(r.pd.token).Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to delete software rollout", apiErrorDetail(err))
	}
}

func (r *softwareRolloutResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
