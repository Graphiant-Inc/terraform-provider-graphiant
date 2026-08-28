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
	_ resource.Resource                = &extranetResource{}
	_ resource.ResourceWithConfigure   = &extranetResource{}
	_ resource.ResourceWithImportState = &extranetResource{}
)

func NewExtranetResource() resource.Resource {
	return &extranetResource{}
}

type extranetResource struct {
	pd *providerData
}

var extranetAutoAttrTypes = map[string]attrType{
	"auto_propagate":    types.BoolType,
	"excluded_prefixes": types.ListType{ElemType: types.StringType},
}

type extranetAutoModel struct {
	AutoPropagate    types.Bool `tfsdk:"auto_propagate"`
	ExcludedPrefixes types.List `tfsdk:"excluded_prefixes"`
}

var extranetManualAttrTypes = map[string]attrType{
	"prefixes": types.ListType{ElemType: types.StringType},
}

type extranetManualModel struct {
	Prefixes types.List `tfsdk:"prefixes"`
}

var extranetTargetAttrTypes = map[string]attrType{
	"excluded_devices": types.ListType{ElemType: types.Int64Type},
	"sites":            types.ListType{ElemType: types.Int64Type},
}

type extranetTargetModel struct {
	ExcludedDevices types.List `tfsdk:"excluded_devices"`
	Sites           types.List `tfsdk:"sites"`
}

type extranetResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Description    types.String `tfsdk:"description"`
	Type           types.String `tfsdk:"type"`
	Auto           types.Object `tfsdk:"auto"`
	Manual         types.Object `tfsdk:"manual"`
	Source         types.Object `tfsdk:"source"`
	Branches       types.Object `tfsdk:"branches"`
	SharedPrefixes types.List   `tfsdk:"shared_prefixes"`
	SharedSegment  types.Int64  `tfsdk:"shared_segment"`
	TargetSegments types.List   `tfsdk:"target_segments"`
}

func (r *extranetResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_extranet"
}

func (r *extranetResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A local (intra-tenant) data exchange policy, sharing routes between segments/sites. " +
			"host_prefix_set / source.prefix_set / branches.prefix_set (enterprise prefix set references) are not " +
			"yet exposed. Exactly one of auto / manual should be set (not enforced by a validator).",
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
			"type": schema.StringAttribute{
				Optional:    true,
				Description: "No enum documented in the SDK; passed through verbatim.",
			},
			"shared_prefixes": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
			},
			"shared_segment": schema.Int64Attribute{
				Optional: true,
			},
			"target_segments": schema.ListAttribute{
				Optional:    true,
				ElementType: types.Int64Type,
			},
			"auto": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"auto_propagate":    schema.BoolAttribute{Optional: true},
					"excluded_prefixes": schema.ListAttribute{Optional: true, ElementType: types.StringType},
				},
			},
			"manual": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"prefixes": schema.ListAttribute{Optional: true, ElementType: types.StringType},
				},
			},
			"source":   extranetTargetAttribute(),
			"branches": extranetTargetAttribute(),
		},
	}
}

func extranetTargetAttribute() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Optional: true,
		Attributes: map[string]schema.Attribute{
			"excluded_devices": schema.ListAttribute{Optional: true, ElementType: types.Int64Type},
			"sites":            schema.ListAttribute{Optional: true, ElementType: types.Int64Type},
		},
	}
}

func (r *extranetResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.pd = configurePD(req.ProviderData, &resp.Diagnostics)
}

func buildExtranetTarget(ctx context.Context, obj types.Object) (*sdk.ManaV2PolicyTargetInput, diag.Diagnostics) {
	if obj.IsNull() || obj.IsUnknown() {
		return nil, nil
	}
	var m extranetTargetModel
	diags := obj.As(ctx, &m, objectAsOptions)
	if diags.HasError() {
		return nil, diags
	}
	target := &sdk.ManaV2PolicyTargetInput{}
	if !m.ExcludedDevices.IsNull() && !m.ExcludedDevices.IsUnknown() {
		diags.Append(m.ExcludedDevices.ElementsAs(ctx, &target.ExcludedDevices, false)...)
	}
	if !m.Sites.IsNull() && !m.Sites.IsUnknown() {
		diags.Append(m.Sites.ElementsAs(ctx, &target.Sites, false)...)
	}
	if diags.HasError() {
		return nil, diags
	}
	return target, diags
}

func applyExtranetTarget(ctx context.Context, target *sdk.ManaV2PolicyTarget) (types.Object, diag.Diagnostics) {
	if target == nil {
		return types.ObjectNull(extranetTargetAttrTypes), nil
	}
	deviceIDs := make([]int64, 0, len(target.ExcludedDevices))
	for _, d := range target.ExcludedDevices {
		if d.Id != nil {
			deviceIDs = append(deviceIDs, *d.Id)
		}
	}
	siteIDs := make([]int64, 0, len(target.Sites))
	for _, s := range target.Sites {
		if s.Id != nil {
			siteIDs = append(siteIDs, *s.Id)
		}
	}
	devicesList, diags := types.ListValueFrom(ctx, types.Int64Type, deviceIDs)
	if diags.HasError() {
		return types.ObjectNull(extranetTargetAttrTypes), diags
	}
	sitesList, diags2 := types.ListValueFrom(ctx, types.Int64Type, siteIDs)
	if diags2.HasError() {
		return types.ObjectNull(extranetTargetAttrTypes), diags2
	}
	return types.ObjectValueFrom(ctx, extranetTargetAttrTypes, extranetTargetModel{ExcludedDevices: devicesList, Sites: sitesList})
}

func (m *extranetResourceModel) buildPolicy(ctx context.Context) (*sdk.ManaV2ExtranetPolicyInput, diag.Diagnostics) {
	policy := &sdk.ManaV2ExtranetPolicyInput{
		Name:          m.Name.ValueStringPointer(),
		Description:   m.Description.ValueStringPointer(),
		Type:          m.Type.ValueStringPointer(),
		SharedSegment: m.SharedSegment.ValueInt64Pointer(),
	}

	var diags diag.Diagnostics
	if !m.SharedPrefixes.IsNull() && !m.SharedPrefixes.IsUnknown() {
		diags.Append(m.SharedPrefixes.ElementsAs(ctx, &policy.SharedPrefixes, false)...)
	}
	if !m.TargetSegments.IsNull() && !m.TargetSegments.IsUnknown() {
		diags.Append(m.TargetSegments.ElementsAs(ctx, &policy.TargetSegments, false)...)
	}

	if !m.Auto.IsNull() && !m.Auto.IsUnknown() {
		var a extranetAutoModel
		diags.Append(m.Auto.As(ctx, &a, objectAsOptions)...)
		auto := &sdk.ManaV2ExtranetAutoReverseRoutes{AutoPropagate: a.AutoPropagate.ValueBoolPointer()}
		if !a.ExcludedPrefixes.IsNull() && !a.ExcludedPrefixes.IsUnknown() {
			diags.Append(a.ExcludedPrefixes.ElementsAs(ctx, &auto.ExcludedPrefixes, false)...)
		}
		policy.Auto = auto
	}
	if !m.Manual.IsNull() && !m.Manual.IsUnknown() {
		var man extranetManualModel
		diags.Append(m.Manual.As(ctx, &man, objectAsOptions)...)
		manual := &sdk.ManaV2ExtranetManualReverseRoutes{}
		if !man.Prefixes.IsNull() && !man.Prefixes.IsUnknown() {
			diags.Append(man.Prefixes.ElementsAs(ctx, &manual.Prefixes, false)...)
		}
		policy.Manual = manual
	}

	source, d := buildExtranetTarget(ctx, m.Source)
	diags.Append(d...)
	policy.Source = source

	branches, d2 := buildExtranetTarget(ctx, m.Branches)
	diags.Append(d2...)
	policy.Branches = branches

	if diags.HasError() {
		return nil, diags
	}
	return policy, diags
}

func (m *extranetResourceModel) applyPolicy(ctx context.Context, policy *sdk.ManaV2ExtranetPolicy) diag.Diagnostics {
	var diags diag.Diagnostics

	if policy.Id != nil {
		m.ID = types.StringValue(int64ID(*policy.Id))
	}
	m.Name = types.StringPointerValue(policy.Name)
	m.Description = types.StringPointerValue(policy.Description)
	m.Type = types.StringPointerValue(policy.Type)

	sharedPrefixes, d := types.ListValueFrom(ctx, types.StringType, policy.SharedPrefixes)
	diags.Append(d...)
	m.SharedPrefixes = sharedPrefixes

	if policy.SharedSegment != nil {
		m.SharedSegment = types.Int64PointerValue(policy.SharedSegment.Id)
	} else {
		m.SharedSegment = types.Int64Null()
	}

	targetIDs := make([]int64, 0, len(policy.TargetSegments))
	for _, vrf := range policy.TargetSegments {
		if vrf.Id != nil {
			targetIDs = append(targetIDs, *vrf.Id)
		}
	}
	targetSegments, d2 := types.ListValueFrom(ctx, types.Int64Type, targetIDs)
	diags.Append(d2...)
	m.TargetSegments = targetSegments

	if policy.Auto != nil {
		excluded, d3 := types.ListValueFrom(ctx, types.StringType, policy.Auto.ExcludedPrefixes)
		diags.Append(d3...)
		obj, d4 := types.ObjectValueFrom(ctx, extranetAutoAttrTypes, extranetAutoModel{
			AutoPropagate:    types.BoolPointerValue(policy.Auto.AutoPropagate),
			ExcludedPrefixes: excluded,
		})
		diags.Append(d4...)
		m.Auto = obj
	} else {
		m.Auto = types.ObjectNull(extranetAutoAttrTypes)
	}

	if policy.Manual != nil {
		prefixes, d5 := types.ListValueFrom(ctx, types.StringType, policy.Manual.Prefixes)
		diags.Append(d5...)
		obj, d6 := types.ObjectValueFrom(ctx, extranetManualAttrTypes, extranetManualModel{Prefixes: prefixes})
		diags.Append(d6...)
		m.Manual = obj
	} else {
		m.Manual = types.ObjectNull(extranetManualAttrTypes)
	}

	source, d7 := applyExtranetTarget(ctx, policy.Source)
	diags.Append(d7...)
	m.Source = source

	branches, d8 := applyExtranetTarget(ctx, policy.Branches)
	diags.Append(d8...)
	m.Branches = branches

	return diags
}

func (r *extranetResource) readByID(ctx context.Context, id int64) (*sdk.ManaV2ExtranetPolicy, bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	out, httpResp, err := r.pd.api.DefaultAPI.V1ExtranetsIdGet(ctx, id).Authorization(r.pd.token).Execute()
	if err != nil {
		if isNotFound(httpResp) {
			closeBody(httpResp)
			return nil, false, diags
		}
		closeBody(httpResp)
		diags.AddError("Unable to read extranet", apiErrorDetail(err))
		return nil, false, diags
	}
	closeBody(httpResp)
	if out == nil || out.Policy == nil {
		return nil, false, diags
	}
	return out.Policy, true, diags
}

func (r *extranetResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan extranetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	policyInput, diags := plan.buildPolicy(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := sdk.V1ExtranetsPostRequest{Policy: policyInput}
	out, httpResp, err := r.pd.api.DefaultAPI.V1ExtranetsPost(ctx).
		Authorization(r.pd.token).
		V1ExtranetsPostRequest(body).
		Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create extranet", apiErrorDetail(err))
		return
	}
	if out == nil || out.Policy == nil {
		resp.Diagnostics.AddError("Unable to create extranet", "API returned an empty response")
		return
	}

	resp.Diagnostics.Append(plan.applyPolicy(ctx, out.Policy)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *extranetResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state extranetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := parseInt64ID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid extranet id", err.Error())
		return
	}

	policy, found, diags := r.readByID(ctx, id)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(state.applyPolicy(ctx, policy)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *extranetResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan extranetResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := parseInt64ID(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid extranet id", err.Error())
		return
	}

	policyInput, diags := plan.buildPolicy(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := sdk.V1ExtranetsIdPutRequest{Policy: policyInput}
	out, httpResp, err := r.pd.api.DefaultAPI.V1ExtranetsIdPut(ctx, id).
		Authorization(r.pd.token).
		V1ExtranetsIdPutRequest(body).
		Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update extranet", apiErrorDetail(err))
		return
	}
	if out == nil || out.Policy == nil {
		resp.Diagnostics.AddError("Unable to update extranet", "API returned an empty response")
		return
	}

	resp.Diagnostics.Append(plan.applyPolicy(ctx, out.Policy)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *extranetResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state extranetResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := parseInt64ID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid extranet id", err.Error())
		return
	}

	_, httpResp, err := r.pd.api.DefaultAPI.V1ExtranetsIdDelete(ctx, id).Authorization(r.pd.token).Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to delete extranet", apiErrorDetail(err))
	}
}

func (r *extranetResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
