package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	sdk "github.com/Graphiant-Inc/graphiant-sdk-go"
)

var (
	_ resource.Resource              = &b2bConsumerResource{}
	_ resource.ResourceWithConfigure = &b2bConsumerResource{}
)

func NewB2bConsumerResource() resource.Resource {
	return &b2bConsumerResource{}
}

type b2bConsumerResource struct {
	pd *providerData
}

var b2bConsumerLanPrefixesAttrTypes = map[string]attrType{
	"consumer_prefixes":   types.ListType{ElemType: types.StringType},
	"service_prefix_dnat": types.MapType{ElemType: types.StringType},
}

type b2bConsumerLanPrefixesModel struct {
	ConsumerPrefixes  types.List `tfsdk:"consumer_prefixes"`
	ServicePrefixDnat types.Map  `tfsdk:"service_prefix_dnat"`
}

var b2bConsumerPolicyAttrTypes = map[string]attrType{
	"consumer_lan_segments": types.MapType{ElemType: types.ObjectType{AttrTypes: b2bConsumerLanPrefixesAttrTypes}},
	"nat_translation_mode":  types.ObjectType{AttrTypes: b2bNatModeAttrTypes},
	"sites":                 types.ListType{ElemType: b2bSiteInfoListType},
}

type b2bConsumerPolicyModel struct {
	ConsumerLanSegments types.Map    `tfsdk:"consumer_lan_segments"`
	NatTranslationMode  types.Object `tfsdk:"nat_translation_mode"`
	Sites               types.List   `tfsdk:"sites"`
}

type b2bConsumerResourceModel struct {
	ID         types.String `tfsdk:"id"`
	CustomerID types.Int64  `tfsdk:"customer_id"`
	MatchID    types.Int64  `tfsdk:"match_id"`
	ServiceID  types.Int64  `tfsdk:"service_id"`
	Status     types.String `tfsdk:"status"`
	Policy     types.Object `tfsdk:"policy"`
}

func (r *b2bConsumerResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_b2b_consumer"
}

func (r *b2bConsumerResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The consumer side of a B2B ('partner') data exchange match (graphiant_b2b_match): the " +
			"customer's acceptance/configuration for a match. All attributes are force-new — the API's read " +
			"endpoint (V1ExtranetB2bConsumersCustomerIdGet) is scoped by customer_id and returns at most one " +
			"consumer config per customer, an assumption this resource relies on since there is no get-by-consumer-id " +
			"endpoint. service_id is not accepted by the update endpoint, so it is also force-new.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"customer_id": schema.Int64Attribute{
				Required: true,
			},
			"match_id": schema.Int64Attribute{
				Required: true,
			},
			"service_id": schema.Int64Attribute{
				Required:    true,
				Description: "Producer service id being consumed.",
			},
			"status": schema.StringAttribute{
				Computed: true,
			},
			"policy": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"consumer_lan_segments": schema.MapNestedAttribute{
						Required:    true,
						Description: "Keyed by consumer LAN segment id.",
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"consumer_prefixes":   schema.ListAttribute{Optional: true, ElementType: types.StringType},
								"service_prefix_dnat": schema.MapAttribute{Optional: true, ElementType: types.StringType},
							},
						},
					},
					"nat_translation_mode": b2bNatModeAttribute(),
					"sites":                b2bSitesAttribute(),
				},
			},
		},
	}
}

func (r *b2bConsumerResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.pd = configurePD(req.ProviderData, &resp.Diagnostics)
}

func buildB2bConsumerPolicy(ctx context.Context, obj types.Object) (*sdk.ManaV2ExtranetServiceConsumerPolicy, diag.Diagnostics) {
	var diags diag.Diagnostics
	if obj.IsNull() || obj.IsUnknown() {
		return nil, diags
	}
	var m b2bConsumerPolicyModel
	diags.Append(obj.As(ctx, &m, objectAsOptions)...)
	if diags.HasError() {
		return nil, diags
	}

	policy := &sdk.ManaV2ExtranetServiceConsumerPolicy{ConsumerLanSegments: map[string]sdk.ManaV2ExtranetConsumerLanPrefixes{}}
	if !m.ConsumerLanSegments.IsNull() && !m.ConsumerLanSegments.IsUnknown() {
		var byLan map[string]b2bConsumerLanPrefixesModel
		diags.Append(m.ConsumerLanSegments.ElementsAs(ctx, &byLan, false)...)
		if diags.HasError() {
			return nil, diags
		}
		for k, v := range byLan {
			var lp sdk.ManaV2ExtranetConsumerLanPrefixes
			if !v.ConsumerPrefixes.IsNull() && !v.ConsumerPrefixes.IsUnknown() {
				diags.Append(v.ConsumerPrefixes.ElementsAs(ctx, &lp.ConsumerPrefixes, false)...)
			}
			if !v.ServicePrefixDnat.IsNull() && !v.ServicePrefixDnat.IsUnknown() {
				var dnat map[string]string
				diags.Append(v.ServicePrefixDnat.ElementsAs(ctx, &dnat, false)...)
				lp.ServicePrefixDnat = &dnat
			}
			policy.ConsumerLanSegments[k] = lp
		}
	}

	natMode, d := buildB2bNatMode(ctx, m.NatTranslationMode)
	diags.Append(d...)
	policy.NatTranslationMode = natMode

	sites, d2 := buildB2bSiteInfoList(ctx, m.Sites)
	diags.Append(d2...)
	policy.Sites = sites

	if diags.HasError() {
		return nil, diags
	}
	return policy, diags
}

func applyB2bConsumerPolicy(ctx context.Context, policy *sdk.ManaV2ExtranetServiceConsumerPolicy) (types.Object, diag.Diagnostics) {
	if policy == nil {
		return types.ObjectNull(b2bConsumerPolicyAttrTypes), nil
	}
	var diags diag.Diagnostics

	byLan := make(map[string]b2bConsumerLanPrefixesModel, len(policy.ConsumerLanSegments))
	for k, v := range policy.ConsumerLanSegments {
		consumerPrefixes, d := types.ListValueFrom(ctx, types.StringType, v.ConsumerPrefixes)
		diags.Append(d...)
		var dnatMap map[string]string
		if v.ServicePrefixDnat != nil {
			dnatMap = *v.ServicePrefixDnat
		}
		dnat, d2 := types.MapValueFrom(ctx, types.StringType, dnatMap)
		diags.Append(d2...)
		byLan[k] = b2bConsumerLanPrefixesModel{ConsumerPrefixes: consumerPrefixes, ServicePrefixDnat: dnat}
	}
	lanSegments, d3 := types.MapValueFrom(ctx, types.ObjectType{AttrTypes: b2bConsumerLanPrefixesAttrTypes}, byLan)
	diags.Append(d3...)

	natMode, d4 := applyB2bNatMode(ctx, policy.NatTranslationMode)
	diags.Append(d4...)

	sites, d5 := applyB2bSiteInfoList(ctx, policy.Sites)
	diags.Append(d5...)

	if diags.HasError() {
		return types.ObjectNull(b2bConsumerPolicyAttrTypes), diags
	}

	obj, diags2 := types.ObjectValueFrom(ctx, b2bConsumerPolicyAttrTypes, b2bConsumerPolicyModel{
		ConsumerLanSegments: lanSegments,
		NatTranslationMode:  natMode,
		Sites:               sites,
	})
	diags.Append(diags2...)
	return obj, diags
}

// readByCustomer looks up the (at most one, per this resource's documented assumption)
// consumer config for a customer, optionally scoped further by service id.
func (r *b2bConsumerResource) readByCustomer(ctx context.Context, customerID, serviceID int64) (*sdk.V1ExtranetB2bConsumersCustomerIdGetResponse, bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	out, httpResp, err := r.pd.api.DefaultAPI.V1ExtranetB2bConsumersCustomerIdGet(ctx, customerID).
		ServiceId(serviceID).
		Authorization(r.pd.token).
		Execute()
	if err != nil {
		if isNotFound(httpResp) {
			closeBody(httpResp)
			return nil, false, diags
		}
		closeBody(httpResp)
		diags.AddError("Unable to read B2B consumer", apiErrorDetail(err))
		return nil, false, diags
	}
	closeBody(httpResp)
	if out == nil {
		return nil, false, diags
	}
	return out, true, diags
}

func (m *b2bConsumerResourceModel) applyFromGet(ctx context.Context, out *sdk.V1ExtranetB2bConsumersCustomerIdGetResponse) diag.Diagnostics {
	if out.Id != nil {
		m.ID = types.StringValue(int64ID(*out.Id))
	}
	if out.MatchId != nil {
		m.MatchID = types.Int64Value(*out.MatchId)
	}
	m.Status = types.StringPointerValue(out.Status)

	policyObj, diags := applyB2bConsumerPolicy(ctx, out.Policy)
	if diags.HasError() {
		return diags
	}
	m.Policy = policyObj
	return diags
}

func (r *b2bConsumerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan b2bConsumerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	policy, diags := buildB2bConsumerPolicy(ctx, plan.Policy)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	matchID := plan.MatchID.ValueInt64()
	body := sdk.V1ExtranetB2bMatchesMatchIdConsumerPostRequest{Policy: policy, ServiceId: plan.ServiceID.ValueInt64Pointer()}
	out, httpResp, err := r.pd.api.DefaultAPI.V1ExtranetB2bMatchesMatchIdConsumerPost(ctx, matchID).
		Authorization(r.pd.token).
		V1ExtranetB2bMatchesMatchIdConsumerPostRequest(body).
		Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create B2B consumer", apiErrorDetail(err))
		return
	}
	if out == nil || out.Id == nil {
		resp.Diagnostics.AddError("Unable to create B2B consumer", "API returned an empty response")
		return
	}
	plan.ID = types.StringValue(int64ID(*out.Id))

	got, found, diags2 := r.readByCustomer(ctx, plan.CustomerID.ValueInt64(), plan.ServiceID.ValueInt64())
	resp.Diagnostics.Append(diags2...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.Diagnostics.AddError("Unable to read created B2B consumer", "consumer was created but could not be read back")
		return
	}

	resp.Diagnostics.Append(plan.applyFromGet(ctx, got)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *b2bConsumerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state b2bConsumerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	got, found, diags := r.readByCustomer(ctx, state.CustomerID.ValueInt64(), state.ServiceID.ValueInt64())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found || got.Id == nil || int64ID(*got.Id) != state.ID.ValueString() {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(state.applyFromGet(ctx, got)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update only runs when a plan changes computed-only output: every configurable
// attribute is effectively force-new since V1ExtranetB2bConsumersIdPut's Policy-only
// body still requires re-deriving the id, and this resource has no reliable id-based
// read path to confirm it applied to the right object — see the schema description.
func (r *b2bConsumerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan b2bConsumerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := parseInt64ID(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid B2B consumer id", err.Error())
		return
	}

	policy, diags := buildB2bConsumerPolicy(ctx, plan.Policy)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := sdk.V1ExtranetB2bConsumersIdPutRequest{Policy: policy}
	_, httpResp, err := r.pd.api.DefaultAPI.V1ExtranetB2bConsumersIdPut(ctx, id).
		Authorization(r.pd.token).
		V1ExtranetB2bConsumersIdPutRequest(body).
		Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update B2B consumer", apiErrorDetail(err))
		return
	}

	got, found, diags2 := r.readByCustomer(ctx, plan.CustomerID.ValueInt64(), plan.ServiceID.ValueInt64())
	resp.Diagnostics.Append(diags2...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.Diagnostics.AddError("Unable to update B2B consumer", "consumer no longer exists")
		return
	}

	resp.Diagnostics.Append(plan.applyFromGet(ctx, got)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *b2bConsumerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state b2bConsumerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := parseInt64ID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid B2B consumer id", err.Error())
		return
	}

	_, httpResp, err := r.pd.api.DefaultAPI.V1ExtranetB2bConsumersIdDelete(ctx, id).Authorization(r.pd.token).Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to delete B2B consumer", apiErrorDetail(err))
	}
}
