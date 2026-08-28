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
	_ resource.Resource                = &b2bProducerServiceResource{}
	_ resource.ResourceWithConfigure   = &b2bProducerServiceResource{}
	_ resource.ResourceWithImportState = &b2bProducerServiceResource{}
)

func NewB2bProducerServiceResource() resource.Resource {
	return &b2bProducerServiceResource{}
}

type b2bProducerServiceResource struct {
	pd *providerData
}

var b2bProducerPolicyAttrTypes = map[string]attrType{
	"description":          types.StringType,
	"service_lan_segment":  types.Int64Type,
	"sites":                types.ListType{ElemType: b2bSiteInfoListType},
	"prefix_tags":          types.ListType{ElemType: b2bPrefixTagListType},
	"nat_translation_mode": types.ObjectType{AttrTypes: b2bNatModeAttrTypes},
}

type b2bProducerPolicyModel struct {
	Description        types.String `tfsdk:"description"`
	ServiceLanSegment  types.Int64  `tfsdk:"service_lan_segment"`
	Sites              types.List   `tfsdk:"sites"`
	PrefixTags         types.List   `tfsdk:"prefix_tags"`
	NatTranslationMode types.Object `tfsdk:"nat_translation_mode"`
}

type b2bProducerServiceResourceModel struct {
	ID          types.String `tfsdk:"id"`
	ServiceName types.String `tfsdk:"service_name"`
	ServiceType types.String `tfsdk:"service_type"`
	Status      types.String `tfsdk:"status"`
	Policy      types.Object `tfsdk:"policy"`
}

func (r *b2bProducerServiceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_b2b_producer_service"
}

func (r *b2bProducerServiceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A B2B ('partner') data exchange producer service. Part of a 4-object workflow: producer " +
			"service -> customer invite (graphiant_b2b_customer) -> match (graphiant_b2b_match) -> consumer accept " +
			"(graphiant_b2b_consumer). service_name/service_type are force-new: the update endpoint only accepts " +
			"policy changes. service_type has no closed enum in the SDK; known values are peering_service and " +
			"client_to_server.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"service_name": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"service_type": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"status": schema.StringAttribute{
				Computed: true,
			},
			"policy": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"description": schema.StringAttribute{Optional: true},
					"service_lan_segment": schema.Int64Attribute{
						Optional:    true,
						Description: "LAN segment ID for the service.",
					},
					"sites":                b2bSitesAttribute(),
					"prefix_tags":          b2bPrefixTagsAttribute(),
					"nat_translation_mode": b2bNatModeAttribute(),
				},
			},
		},
	}
}

func (r *b2bProducerServiceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.pd = configurePD(req.ProviderData, &resp.Diagnostics)
}

func buildB2bProducerPolicy(ctx context.Context, obj types.Object) (sdk.ManaV2ExtranetServiceProducerPolicy, diag.Diagnostics) {
	var out sdk.ManaV2ExtranetServiceProducerPolicy
	var diags diag.Diagnostics
	if obj.IsNull() || obj.IsUnknown() {
		return out, diags
	}
	var m b2bProducerPolicyModel
	diags.Append(obj.As(ctx, &m, objectAsOptions)...)
	if diags.HasError() {
		return out, diags
	}

	out.Description = m.Description.ValueStringPointer()
	out.ServiceLanSegment = m.ServiceLanSegment.ValueInt64Pointer()

	sites, d := buildB2bSiteInfoList(ctx, m.Sites)
	diags.Append(d...)
	out.Sites = sites

	prefixTags, d2 := buildB2bPrefixTags(ctx, m.PrefixTags)
	diags.Append(d2...)
	out.PrefixTags = prefixTags

	natMode, d3 := buildB2bNatMode(ctx, m.NatTranslationMode)
	diags.Append(d3...)
	out.NatTranslationMode = natMode

	return out, diags
}

func applyB2bProducerPolicy(ctx context.Context, policy *sdk.ManaV2ExtranetServiceProducerPolicy) (types.Object, diag.Diagnostics) {
	if policy == nil {
		return types.ObjectNull(b2bProducerPolicyAttrTypes), nil
	}
	var diags diag.Diagnostics

	sites, d := applyB2bSiteInfoList(ctx, policy.Sites)
	diags.Append(d...)
	prefixTags, d2 := applyB2bPrefixTags(ctx, policy.PrefixTags)
	diags.Append(d2...)
	natMode, d3 := applyB2bNatMode(ctx, policy.NatTranslationMode)
	diags.Append(d3...)

	if diags.HasError() {
		return types.ObjectNull(b2bProducerPolicyAttrTypes), diags
	}

	m := b2bProducerPolicyModel{
		Description:        types.StringPointerValue(policy.Description),
		ServiceLanSegment:  types.Int64PointerValue(policy.ServiceLanSegment),
		Sites:              sites,
		PrefixTags:         prefixTags,
		NatTranslationMode: natMode,
	}
	obj, diags2 := types.ObjectValueFrom(ctx, b2bProducerPolicyAttrTypes, m)
	diags.Append(diags2...)
	return obj, diags
}

func (r *b2bProducerServiceResource) readByID(ctx context.Context, id int64) (*sdk.V1ExtranetB2bProducerIdGetResponse, bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	out, httpResp, err := r.pd.api.DefaultAPI.V1ExtranetB2bProducerIdGet(ctx, id).Authorization(r.pd.token).Execute()
	if err != nil {
		if isNotFound(httpResp) {
			closeBody(httpResp)
			return nil, false, diags
		}
		closeBody(httpResp)
		diags.AddError("Unable to read B2B producer service", apiErrorDetail(err))
		return nil, false, diags
	}
	closeBody(httpResp)
	if out == nil {
		return nil, false, diags
	}
	return out, true, diags
}

func (m *b2bProducerServiceResourceModel) applyFromGet(ctx context.Context, out *sdk.V1ExtranetB2bProducerIdGetResponse) diag.Diagnostics {
	var diags diag.Diagnostics
	if out.Id != nil {
		m.ID = types.StringValue(int64ID(*out.Id))
	}
	m.Status = types.StringPointerValue(out.Status)
	if out.Policy != nil {
		m.ServiceName = types.StringPointerValue(out.Policy.ServiceName)
		m.ServiceType = types.StringPointerValue(out.Policy.ServiceType)
		policyObj, d := applyB2bProducerPolicy(ctx, out.Policy.Policy)
		diags.Append(d...)
		m.Policy = policyObj
	}
	return diags
}

func (r *b2bProducerServiceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan b2bProducerServiceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	policy, diags := buildB2bProducerPolicy(ctx, plan.Policy)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := sdk.V1ExtranetB2bProducerPostRequest{
		Policy:      policy,
		ServiceName: plan.ServiceName.ValueString(),
		ServiceType: plan.ServiceType.ValueString(),
	}
	out, httpResp, err := r.pd.api.DefaultAPI.V1ExtranetB2bProducerPost(ctx).
		Authorization(r.pd.token).
		V1ExtranetB2bProducerPostRequest(body).
		Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create B2B producer service", apiErrorDetail(err))
		return
	}
	if out == nil || out.Id == nil {
		resp.Diagnostics.AddError("Unable to create B2B producer service", "API returned an empty response")
		return
	}

	got, found, diags2 := r.readByID(ctx, *out.Id)
	resp.Diagnostics.Append(diags2...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.Diagnostics.AddError("Unable to read created B2B producer service", "service was created but could not be read back")
		return
	}

	resp.Diagnostics.Append(plan.applyFromGet(ctx, got)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *b2bProducerServiceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state b2bProducerServiceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := parseInt64ID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid B2B producer service id", err.Error())
		return
	}

	got, found, diags := r.readByID(ctx, id)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(state.applyFromGet(ctx, got)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *b2bProducerServiceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan b2bProducerServiceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := parseInt64ID(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid B2B producer service id", err.Error())
		return
	}

	policy, diags := buildB2bProducerPolicy(ctx, plan.Policy)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := sdk.V1ExtranetB2bProducerIdPutRequest{Policy: &policy}
	_, httpResp, err := r.pd.api.DefaultAPI.V1ExtranetB2bProducerIdPut(ctx, id).
		Authorization(r.pd.token).
		V1ExtranetB2bProducerIdPutRequest(body).
		Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update B2B producer service", apiErrorDetail(err))
		return
	}

	got, found, diags2 := r.readByID(ctx, id)
	resp.Diagnostics.Append(diags2...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.Diagnostics.AddError("Unable to update B2B producer service", "service no longer exists")
		return
	}

	resp.Diagnostics.Append(plan.applyFromGet(ctx, got)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *b2bProducerServiceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state b2bProducerServiceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := parseInt64ID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid B2B producer service id", err.Error())
		return
	}

	_, httpResp, err := r.pd.api.DefaultAPI.V1ExtranetB2bProducerIdDelete(ctx, id).Authorization(r.pd.token).Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to delete B2B producer service", apiErrorDetail(err))
	}
}

func (r *b2bProducerServiceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
