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
	_ resource.Resource                = &b2bMatchResource{}
	_ resource.ResourceWithConfigure   = &b2bMatchResource{}
	_ resource.ResourceWithImportState = &b2bMatchResource{}
)

func NewB2bMatchResource() resource.Resource {
	return &b2bMatchResource{}
}

type b2bMatchResource struct {
	pd *providerData
}

var b2bMatchAttrTypes = map[string]attrType{
	"service_id":           types.Int64Type,
	"lan_segment":          types.Int64Type,
	"consumer_prefixes":    types.ListType{ElemType: types.StringType},
	"num_customers":        types.Int64Type,
	"service_prefixes":     types.ListType{ElemType: b2bPrefixTagListType},
	"nat_translation_mode": types.ObjectType{AttrTypes: b2bNatModeAttrTypes},
}

type b2bMatchModel struct {
	ServiceID          types.Int64  `tfsdk:"service_id"`
	LanSegment         types.Int64  `tfsdk:"lan_segment"`
	ConsumerPrefixes   types.List   `tfsdk:"consumer_prefixes"`
	NumCustomers       types.Int64  `tfsdk:"num_customers"`
	ServicePrefixes    types.List   `tfsdk:"service_prefixes"`
	NatTranslationMode types.Object `tfsdk:"nat_translation_mode"`
}

type b2bMatchResourceModel struct {
	ID           types.String `tfsdk:"id"`
	CustomerID   types.Int64  `tfsdk:"customer_id"`
	CustomerName types.String `tfsdk:"customer_name"`
	ServiceName  types.String `tfsdk:"service_name"`
	Status       types.String `tfsdk:"status"`
	Match        types.Object `tfsdk:"match"`
}

func (r *b2bMatchResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_b2b_match"
}

func (r *b2bMatchResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A B2B ('partner') data exchange match, linking a customer (graphiant_b2b_customer) to a " +
			"producer service (graphiant_b2b_producer_service). customer_id is force-new and not echoed back by " +
			"the read endpoint, so it's preserved from configuration rather than refreshed from the API.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"customer_id": schema.Int64Attribute{
				Required: true,
			},
			"customer_name": schema.StringAttribute{
				Computed: true,
			},
			"service_name": schema.StringAttribute{
				Computed: true,
			},
			"status": schema.StringAttribute{
				Computed: true,
			},
			"match": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"service_id": schema.Int64Attribute{
						Required:    true,
						Description: "Producer service id.",
					},
					"lan_segment": schema.Int64Attribute{
						Optional: true,
					},
					"consumer_prefixes": schema.ListAttribute{
						Optional:    true,
						ElementType: types.StringType,
					},
					"num_customers": schema.Int64Attribute{
						Computed: true,
					},
					"service_prefixes":     b2bPrefixTagsAttribute(),
					"nat_translation_mode": b2bNatModeAttribute(),
				},
			},
		},
	}
}

func (r *b2bMatchResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.pd = configurePD(req.ProviderData, &resp.Diagnostics)
}

func buildB2bMatch(ctx context.Context, obj types.Object) (*sdk.ManaV2B2bExtranetMatch, diag.Diagnostics) {
	var diags diag.Diagnostics
	if obj.IsNull() || obj.IsUnknown() {
		return nil, diags
	}
	var m b2bMatchModel
	diags.Append(obj.As(ctx, &m, objectAsOptions)...)
	if diags.HasError() {
		return nil, diags
	}

	match := &sdk.ManaV2B2bExtranetMatch{
		ServiceId:  m.ServiceID.ValueInt64Pointer(),
		LanSegment: m.LanSegment.ValueInt64Pointer(),
	}
	if !m.ConsumerPrefixes.IsNull() && !m.ConsumerPrefixes.IsUnknown() {
		diags.Append(m.ConsumerPrefixes.ElementsAs(ctx, &match.ConsumerPrefixes, false)...)
	}
	servicePrefixes, d := buildB2bPrefixTags(ctx, m.ServicePrefixes)
	diags.Append(d...)
	match.ServicePrefixes = servicePrefixes

	natMode, d2 := buildB2bNatMode(ctx, m.NatTranslationMode)
	diags.Append(d2...)
	match.NatTranslationMode = natMode

	if diags.HasError() {
		return nil, diags
	}
	return match, diags
}

func applyB2bMatch(ctx context.Context, match *sdk.ManaV2B2bExtranetMatch) (types.Object, diag.Diagnostics) {
	if match == nil {
		return types.ObjectNull(b2bMatchAttrTypes), nil
	}
	var diags diag.Diagnostics

	consumerPrefixes, d := types.ListValueFrom(ctx, types.StringType, match.ConsumerPrefixes)
	diags.Append(d...)
	servicePrefixes, d2 := applyB2bPrefixTags(ctx, match.ServicePrefixes)
	diags.Append(d2...)
	natMode, d3 := applyB2bNatMode(ctx, match.NatTranslationMode)
	diags.Append(d3...)
	if diags.HasError() {
		return types.ObjectNull(b2bMatchAttrTypes), diags
	}

	m := b2bMatchModel{
		ServiceID:          types.Int64PointerValue(match.ServiceId),
		LanSegment:         types.Int64PointerValue(match.LanSegment),
		ConsumerPrefixes:   consumerPrefixes,
		ServicePrefixes:    servicePrefixes,
		NatTranslationMode: natMode,
	}
	if match.NumCustomers != nil {
		m.NumCustomers = types.Int64Value(int64(*match.NumCustomers))
	}
	obj, diags2 := types.ObjectValueFrom(ctx, b2bMatchAttrTypes, m)
	diags.Append(diags2...)
	return obj, diags
}

func (r *b2bMatchResource) readByID(ctx context.Context, id int64) (*sdk.V1ExtranetB2bMatchesMatchIdGetResponse, bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	out, httpResp, err := r.pd.api.DefaultAPI.V1ExtranetB2bMatchesMatchIdGet(ctx, id).Authorization(r.pd.token).Execute()
	if err != nil {
		if isNotFound(httpResp) {
			closeBody(httpResp)
			return nil, false, diags
		}
		closeBody(httpResp)
		diags.AddError("Unable to read B2B match", apiErrorDetail(err))
		return nil, false, diags
	}
	closeBody(httpResp)
	if out == nil {
		return nil, false, diags
	}
	return out, true, diags
}

func (m *b2bMatchResourceModel) applyFromGet(ctx context.Context, out *sdk.V1ExtranetB2bMatchesMatchIdGetResponse) diag.Diagnostics {
	if out.MatchId != nil {
		m.ID = types.StringValue(int64ID(*out.MatchId))
	}
	m.CustomerName = types.StringPointerValue(out.CustomerName)
	m.ServiceName = types.StringPointerValue(out.ServiceName)
	m.Status = types.StringPointerValue(out.Status)

	// The read endpoint does not echo lan_segment back on the match object, so
	// capture the value already known from plan/prior state before it gets
	// overwritten below, and restore it if the API response omits it.
	var priorLanSegment types.Int64
	if !m.Match.IsNull() && !m.Match.IsUnknown() {
		var prior b2bMatchModel
		if d := m.Match.As(ctx, &prior, objectAsOptions); !d.HasError() {
			priorLanSegment = prior.LanSegment
		}
	}

	matchObj, diags := applyB2bMatch(ctx, out.Match)
	if diags.HasError() {
		return diags
	}

	if out.Match != nil && out.Match.LanSegment == nil && !priorLanSegment.IsNull() && !priorLanSegment.IsUnknown() {
		var matchModel b2bMatchModel
		diags.Append(matchObj.As(ctx, &matchModel, objectAsOptions)...)
		if diags.HasError() {
			return diags
		}
		matchModel.LanSegment = priorLanSegment
		patched, d := types.ObjectValueFrom(ctx, b2bMatchAttrTypes, matchModel)
		diags.Append(d...)
		if diags.HasError() {
			return diags
		}
		matchObj = patched
	}

	m.Match = matchObj
	return diags
}

func (r *b2bMatchResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan b2bMatchResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	match, diags := buildB2bMatch(ctx, plan.Match)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := sdk.V1ExtranetB2bMatchesPostRequest{CustomerId: plan.CustomerID.ValueInt64Pointer(), Match: match}
	out, httpResp, err := r.pd.api.DefaultAPI.V1ExtranetB2bMatchesPost(ctx).
		Authorization(r.pd.token).
		V1ExtranetB2bMatchesPostRequest(body).
		Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create B2B match", apiErrorDetail(err))
		return
	}
	if out == nil || out.MatchId == nil {
		resp.Diagnostics.AddError("Unable to create B2B match", "API returned an empty response")
		return
	}

	got, found, diags2 := r.readByID(ctx, *out.MatchId)
	resp.Diagnostics.Append(diags2...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.Diagnostics.AddError("Unable to read created B2B match", "match was created but could not be read back")
		return
	}

	resp.Diagnostics.Append(plan.applyFromGet(ctx, got)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *b2bMatchResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state b2bMatchResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := parseInt64ID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid B2B match id", err.Error())
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

func (r *b2bMatchResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan b2bMatchResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := parseInt64ID(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid B2B match id", err.Error())
		return
	}

	match, diags := buildB2bMatch(ctx, plan.Match)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := sdk.V1ExtranetB2bMatchesMatchIdPutRequest{Match: match}
	_, httpResp, err := r.pd.api.DefaultAPI.V1ExtranetB2bMatchesMatchIdPut(ctx, id).
		Authorization(r.pd.token).
		V1ExtranetB2bMatchesMatchIdPutRequest(body).
		Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update B2B match", apiErrorDetail(err))
		return
	}

	got, found, diags2 := r.readByID(ctx, id)
	resp.Diagnostics.Append(diags2...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.Diagnostics.AddError("Unable to update B2B match", "match no longer exists")
		return
	}

	resp.Diagnostics.Append(plan.applyFromGet(ctx, got)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *b2bMatchResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state b2bMatchResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := parseInt64ID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid B2B match id", err.Error())
		return
	}

	_, httpResp, err := r.pd.api.DefaultAPI.V1ExtranetB2bMatchesMatchIdDelete(ctx, id).Authorization(r.pd.token).Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to delete B2B match", apiErrorDetail(err))
	}
}

func (r *b2bMatchResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
