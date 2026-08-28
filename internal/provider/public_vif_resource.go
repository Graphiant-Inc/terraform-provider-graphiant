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
	_ resource.Resource                = &publicVifResource{}
	_ resource.ResourceWithConfigure   = &publicVifResource{}
	_ resource.ResourceWithImportState = &publicVifResource{}
)

func NewPublicVifResource() resource.Resource {
	return &publicVifResource{}
}

type publicVifResource struct {
	pd *providerData
}

type pvifConsumerLanModel struct {
	ConsumerPrefixes types.List `tfsdk:"consumer_prefixes"`
}

var pvifConsumerLanAttrTypes = map[string]attrType{
	"consumer_prefixes": types.ListType{ElemType: types.StringType},
}

type pvifBgpNeighborModel struct {
	AsOverride       types.Bool   `tfsdk:"as_override"`
	DefaultOriginate types.String `tfsdk:"default_originate"`
	Enabled          types.Bool   `tfsdk:"enabled"`
	HoldTimer        types.Int64  `tfsdk:"hold_timer"`
	KeepaliveTimer   types.Int64  `tfsdk:"keepalive_timer"`
	LocalAddress     types.String `tfsdk:"local_address"`
	PeerAsn          types.Int64  `tfsdk:"peer_asn"`
	RemoteAddress    types.String `tfsdk:"remote_address"`
	RemovePrivateAs  types.Bool   `tfsdk:"remove_private_as"`
	SendCommunity    types.Bool   `tfsdk:"send_community"`
}

var pvifBgpNeighborAttrTypes = map[string]attrType{
	"as_override":       types.BoolType,
	"default_originate": types.StringType,
	"enabled":           types.BoolType,
	"hold_timer":        types.Int64Type,
	"keepalive_timer":   types.Int64Type,
	"local_address":     types.StringType,
	"peer_asn":          types.Int64Type,
	"remote_address":    types.StringType,
	"remove_private_as": types.BoolType,
	"send_community":    types.BoolType,
}

var pvifSiteInfoAttrTypes = map[string]attrType{
	"site_lists": types.ListType{ElemType: types.Int64Type},
	"sites":      types.ListType{ElemType: types.Int64Type},
}

type pvifSiteInfoModel struct {
	SiteLists types.List `tfsdk:"site_lists"`
	Sites     types.List `tfsdk:"sites"`
}

var pvifNatStrategyAttrTypes = map[string]attrType{
	"centralized":   types.ObjectType{AttrTypes: map[string]attrType{"consumer_prefix": types.MapType{ElemType: types.StringType}}},
	"decentralized": types.ObjectType{AttrTypes: map[string]attrType{"prefixes": types.MapType{ElemType: types.StringType}}},
}

type pvifNatStrategyModel struct {
	Centralized   types.Object `tfsdk:"centralized"`
	Decentralized types.Object `tfsdk:"decentralized"`
}

type pvifCentralizedModel struct {
	ConsumerPrefix types.Map `tfsdk:"consumer_prefix"`
}

type pvifDecentralizedModel struct {
	Prefixes types.Map `tfsdk:"prefixes"`
}

type publicVifResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	ServiceName         types.String `tfsdk:"service_name"`
	LanSegmentID        types.Int64  `tfsdk:"lan_segment_id"`
	RegionID            types.Int64  `tfsdk:"region_id"`
	StorageProvider     types.String `tfsdk:"storage_provider"`
	ConsumerLanSegments types.Map    `tfsdk:"consumer_lan_segments"`
	CoveringPrefixes    types.List   `tfsdk:"covering_prefixes"`
	Advertisement       types.Object `tfsdk:"advertisement"`
	GatewayBgpNeighbors types.Map    `tfsdk:"gateway_bgp_neighbors"`
	NatPrefixStrategy   types.Object `tfsdk:"nat_prefix_strategy"`
}

func (r *publicVifResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_public_vif"
}

func (r *publicVifResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A gateway Public VIF data exchange service. gateway_bgp_neighbors only exposes scalar BGP " +
			"neighbor fields — address_families, allow_as_in, bfd, and other nested sub-configs are not yet exposed. " +
			"Exactly one of nat_prefix_strategy.centralized / .decentralized should be set (not enforced by a validator).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"service_name": schema.StringAttribute{
				Required: true,
			},
			"lan_segment_id": schema.Int64Attribute{
				Required:    true,
				Description: "Producer LAN segment (VRF) on gateway appliances.",
			},
			"region_id": schema.Int64Attribute{
				Required: true,
			},
			"storage_provider": schema.StringAttribute{
				Required: true,
			},
			"covering_prefixes": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
			},
			"consumer_lan_segments": schema.MapNestedAttribute{
				Required:    true,
				Description: "Keyed by consumer LAN segment id.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"consumer_prefixes": schema.ListAttribute{
							Optional:    true,
							ElementType: types.StringType,
						},
					},
				},
			},
			"gateway_bgp_neighbors": schema.MapNestedAttribute{
				Required:    true,
				Description: "Keyed by neighbor id.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"as_override":       schema.BoolAttribute{Optional: true},
						"default_originate": schema.StringAttribute{Optional: true},
						"enabled":           schema.BoolAttribute{Optional: true},
						"hold_timer":        schema.Int64Attribute{Optional: true},
						"keepalive_timer":   schema.Int64Attribute{Optional: true},
						"local_address":     schema.StringAttribute{Optional: true},
						"peer_asn":          schema.Int64Attribute{Optional: true},
						"remote_address":    schema.StringAttribute{Optional: true},
						"remove_private_as": schema.BoolAttribute{Optional: true},
						"send_community":    schema.BoolAttribute{Optional: true},
					},
				},
			},
			"advertisement": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"site_lists": schema.ListAttribute{Optional: true, ElementType: types.Int64Type},
					"sites":      schema.ListAttribute{Optional: true, ElementType: types.Int64Type},
				},
			},
			"nat_prefix_strategy": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"centralized": schema.SingleNestedAttribute{
						Optional: true,
						Attributes: map[string]schema.Attribute{
							"consumer_prefix": schema.MapAttribute{Optional: true, ElementType: types.StringType},
						},
					},
					"decentralized": schema.SingleNestedAttribute{
						Optional: true,
						Attributes: map[string]schema.Attribute{
							"prefixes": schema.MapAttribute{Optional: true, ElementType: types.StringType},
						},
					},
				},
			},
		},
	}
}

func (r *publicVifResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.pd = configurePD(req.ProviderData, &resp.Diagnostics)
}

func buildPvifConsumerLanSegments(ctx context.Context, m types.Map) (map[string]sdk.ManaV2PublicVifGatewayConsumerLanDevices, diag.Diagnostics) {
	result := map[string]sdk.ManaV2PublicVifGatewayConsumerLanDevices{}
	var diags diag.Diagnostics
	if m.IsNull() || m.IsUnknown() {
		return result, diags
	}
	var models map[string]pvifConsumerLanModel
	diags.Append(m.ElementsAs(ctx, &models, false)...)
	if diags.HasError() {
		return nil, diags
	}
	for k, v := range models {
		var prefixes []string
		if !v.ConsumerPrefixes.IsNull() && !v.ConsumerPrefixes.IsUnknown() {
			diags.Append(v.ConsumerPrefixes.ElementsAs(ctx, &prefixes, false)...)
			if diags.HasError() {
				return nil, diags
			}
		}
		result[k] = sdk.ManaV2PublicVifGatewayConsumerLanDevices{ConsumerPrefixes: prefixes}
	}
	return result, diags
}

func applyPvifConsumerLanSegments(ctx context.Context, in map[string]sdk.ManaV2PublicVifGatewayConsumerLanDevices) (types.Map, diag.Diagnostics) {
	models := make(map[string]pvifConsumerLanModel, len(in))
	for k, v := range in {
		list, diags := types.ListValueFrom(ctx, types.StringType, v.ConsumerPrefixes)
		if diags.HasError() {
			return types.MapNull(types.ObjectType{AttrTypes: pvifConsumerLanAttrTypes}), diags
		}
		models[k] = pvifConsumerLanModel{ConsumerPrefixes: list}
	}
	return types.MapValueFrom(ctx, types.ObjectType{AttrTypes: pvifConsumerLanAttrTypes}, models)
}

func buildPvifBgpNeighbors(ctx context.Context, m types.Map) (map[string]sdk.ManaV2BgpNeighborConfig, diag.Diagnostics) {
	result := map[string]sdk.ManaV2BgpNeighborConfig{}
	var diags diag.Diagnostics
	if m.IsNull() || m.IsUnknown() {
		return result, diags
	}
	var models map[string]pvifBgpNeighborModel
	diags.Append(m.ElementsAs(ctx, &models, false)...)
	if diags.HasError() {
		return nil, diags
	}
	for k, v := range models {
		n := sdk.ManaV2BgpNeighborConfig{
			AsOverride:       v.AsOverride.ValueBoolPointer(),
			DefaultOriginate: v.DefaultOriginate.ValueStringPointer(),
			Enabled:          v.Enabled.ValueBoolPointer(),
			LocalAddress:     v.LocalAddress.ValueStringPointer(),
			RemoteAddress:    v.RemoteAddress.ValueStringPointer(),
			RemovePrivateAs:  v.RemovePrivateAs.ValueBoolPointer(),
			SendCommunity:    v.SendCommunity.ValueBoolPointer(),
			HoldTimer:        int32PtrFromInt64(v.HoldTimer),
			KeepaliveTimer:   int32PtrFromInt64(v.KeepaliveTimer),
			PeerAsn:          int32PtrFromInt64(v.PeerAsn),
		}
		result[k] = n
	}
	return result, diags
}

func applyPvifBgpNeighbors(ctx context.Context, in map[string]sdk.ManaV2BgpNeighbor) (types.Map, diag.Diagnostics) {
	models := make(map[string]pvifBgpNeighborModel, len(in))
	for k, v := range in {
		m := pvifBgpNeighborModel{
			AsOverride:       types.BoolPointerValue(v.AsOverride),
			DefaultOriginate: types.StringPointerValue(v.DefaultOriginate),
			Enabled:          types.BoolPointerValue(v.Enabled),
			LocalAddress:     types.StringPointerValue(v.LocalAddress),
			RemoteAddress:    types.StringPointerValue(v.RemoteAddress),
			RemovePrivateAs:  types.BoolPointerValue(v.RemovePrivateAs),
			SendCommunity:    types.BoolPointerValue(v.SendCommunity),
		}
		if v.HoldTimer != nil {
			m.HoldTimer = types.Int64Value(int64(*v.HoldTimer))
		}
		if v.KeepaliveTimer != nil {
			m.KeepaliveTimer = types.Int64Value(int64(*v.KeepaliveTimer))
		}
		if v.PeerAsn != nil {
			m.PeerAsn = types.Int64Value(int64(*v.PeerAsn))
		}
		models[k] = m
	}
	return types.MapValueFrom(ctx, types.ObjectType{AttrTypes: pvifBgpNeighborAttrTypes}, models)
}

func buildPvifSiteInfo(ctx context.Context, obj types.Object) (*sdk.ManaV2SiteInformation, diag.Diagnostics) {
	if obj.IsNull() || obj.IsUnknown() {
		return nil, nil
	}
	var m pvifSiteInfoModel
	diags := obj.As(ctx, &m, objectAsOptions)
	if diags.HasError() {
		return nil, diags
	}
	info := &sdk.ManaV2SiteInformation{}
	if !m.SiteLists.IsNull() && !m.SiteLists.IsUnknown() {
		diags.Append(m.SiteLists.ElementsAs(ctx, &info.SiteLists, false)...)
	}
	if !m.Sites.IsNull() && !m.Sites.IsUnknown() {
		diags.Append(m.Sites.ElementsAs(ctx, &info.Sites, false)...)
	}
	if diags.HasError() {
		return nil, diags
	}
	return info, diags
}

func applyPvifSiteInfo(ctx context.Context, info *sdk.ManaV2SiteInformation) (types.Object, diag.Diagnostics) {
	if info == nil {
		return types.ObjectNull(pvifSiteInfoAttrTypes), nil
	}
	siteLists, diags := types.ListValueFrom(ctx, types.Int64Type, info.SiteLists)
	if diags.HasError() {
		return types.ObjectNull(pvifSiteInfoAttrTypes), diags
	}
	sites, diags2 := types.ListValueFrom(ctx, types.Int64Type, info.Sites)
	if diags2.HasError() {
		return types.ObjectNull(pvifSiteInfoAttrTypes), diags2
	}
	return types.ObjectValueFrom(ctx, pvifSiteInfoAttrTypes, pvifSiteInfoModel{SiteLists: siteLists, Sites: sites})
}

func buildPvifNatStrategy(ctx context.Context, obj types.Object) (sdk.ManaV2PublicVifGatewayNatPrefixStrategy, diag.Diagnostics) {
	var strategy sdk.ManaV2PublicVifGatewayNatPrefixStrategy
	if obj.IsNull() || obj.IsUnknown() {
		return strategy, nil
	}
	var m pvifNatStrategyModel
	diags := obj.As(ctx, &m, objectAsOptions)
	if diags.HasError() {
		return strategy, diags
	}
	if !m.Centralized.IsNull() && !m.Centralized.IsUnknown() {
		var c pvifCentralizedModel
		diags.Append(m.Centralized.As(ctx, &c, objectAsOptions)...)
		if diags.HasError() {
			return strategy, diags
		}
		var prefixes map[string]string
		if !c.ConsumerPrefix.IsNull() && !c.ConsumerPrefix.IsUnknown() {
			diags.Append(c.ConsumerPrefix.ElementsAs(ctx, &prefixes, false)...)
		}
		strategy.Centralized = &sdk.ManaV2PublicVifGatewayCentralizedNat{ConsumerPrefix: prefixes}
	}
	if !m.Decentralized.IsNull() && !m.Decentralized.IsUnknown() {
		var d pvifDecentralizedModel
		diags.Append(m.Decentralized.As(ctx, &d, objectAsOptions)...)
		if diags.HasError() {
			return strategy, diags
		}
		var prefixes map[string]string
		if !d.Prefixes.IsNull() && !d.Prefixes.IsUnknown() {
			diags.Append(d.Prefixes.ElementsAs(ctx, &prefixes, false)...)
		}
		strategy.Decentralized = &sdk.ManaV2PublicVifGatewayDecentralizedPrefixes{Prefixes: prefixes}
	}
	return strategy, diags
}

func applyPvifNatStrategy(ctx context.Context, strategy *sdk.ManaV2PublicVifGatewayNatPrefixStrategy) (types.Object, diag.Diagnostics) {
	if strategy == nil {
		return types.ObjectNull(pvifNatStrategyAttrTypes), nil
	}
	m := pvifNatStrategyModel{
		Centralized:   types.ObjectNull(map[string]attrType{"consumer_prefix": types.MapType{ElemType: types.StringType}}),
		Decentralized: types.ObjectNull(map[string]attrType{"prefixes": types.MapType{ElemType: types.StringType}}),
	}
	if strategy.Centralized != nil {
		prefixMap, diags := types.MapValueFrom(ctx, types.StringType, strategy.Centralized.ConsumerPrefix)
		if diags.HasError() {
			return types.ObjectNull(pvifNatStrategyAttrTypes), diags
		}
		obj, diags2 := types.ObjectValueFrom(ctx, map[string]attrType{"consumer_prefix": types.MapType{ElemType: types.StringType}}, pvifCentralizedModel{ConsumerPrefix: prefixMap})
		if diags2.HasError() {
			return types.ObjectNull(pvifNatStrategyAttrTypes), diags2
		}
		m.Centralized = obj
	}
	if strategy.Decentralized != nil {
		prefixMap, diags := types.MapValueFrom(ctx, types.StringType, strategy.Decentralized.Prefixes)
		if diags.HasError() {
			return types.ObjectNull(pvifNatStrategyAttrTypes), diags
		}
		obj, diags2 := types.ObjectValueFrom(ctx, map[string]attrType{"prefixes": types.MapType{ElemType: types.StringType}}, pvifDecentralizedModel{Prefixes: prefixMap})
		if diags2.HasError() {
			return types.ObjectNull(pvifNatStrategyAttrTypes), diags2
		}
		m.Decentralized = obj
	}
	return types.ObjectValueFrom(ctx, pvifNatStrategyAttrTypes, m)
}

func (m *publicVifResourceModel) buildWriteRequest(ctx context.Context) (sdk.ManaV2PublicVifGatewayWriteRequest, diag.Diagnostics) {
	var out sdk.ManaV2PublicVifGatewayWriteRequest
	var diags diag.Diagnostics

	consumerLans, d := buildPvifConsumerLanSegments(ctx, m.ConsumerLanSegments)
	diags.Append(d...)
	bgpNeighbors, d2 := buildPvifBgpNeighbors(ctx, m.GatewayBgpNeighbors)
	diags.Append(d2...)
	advertisement, d3 := buildPvifSiteInfo(ctx, m.Advertisement)
	diags.Append(d3...)
	natStrategy, d4 := buildPvifNatStrategy(ctx, m.NatPrefixStrategy)
	diags.Append(d4...)
	if diags.HasError() {
		return out, diags
	}

	out.ServiceName = m.ServiceName.ValueString()
	out.LanSegmentId = m.LanSegmentID.ValueInt64()
	if p := int32PtrFromInt64(m.RegionID); p != nil {
		out.RegionId = *p
	}
	out.StorageProvider = m.StorageProvider.ValueString()
	out.ConsumerLanSegments = consumerLans
	out.GatewayBgpNeighbors = bgpNeighbors
	out.Advertisement = advertisement
	out.NatPrefixStrategy = natStrategy
	if !m.CoveringPrefixes.IsNull() && !m.CoveringPrefixes.IsUnknown() {
		diags.Append(m.CoveringPrefixes.ElementsAs(ctx, &out.CoveringPrefixes, false)...)
	}

	return out, diags
}

func (m *publicVifResourceModel) applyResponse(ctx context.Context, id *int64, serviceName, lanSegmentID *string, regionID *int32, storageProvider *string,
	consumerLans map[string]sdk.ManaV2PublicVifGatewayConsumerLanDevices, coveringPrefixes []string,
	advertisement *sdk.ManaV2SiteInformation, bgpNeighbors map[string]sdk.ManaV2BgpNeighbor, natStrategy *sdk.ManaV2PublicVifGatewayNatPrefixStrategy) diag.Diagnostics {
	var diags diag.Diagnostics

	if id != nil {
		m.ID = types.StringValue(int64ID(*id))
	}
	m.ServiceName = types.StringPointerValue(serviceName)
	if regionID != nil {
		m.RegionID = types.Int64Value(int64(*regionID))
	}
	m.StorageProvider = types.StringPointerValue(storageProvider)

	coveringList, d := types.ListValueFrom(ctx, types.StringType, coveringPrefixes)
	diags.Append(d...)
	m.CoveringPrefixes = coveringList

	consumerLansMap, d2 := applyPvifConsumerLanSegments(ctx, consumerLans)
	diags.Append(d2...)
	m.ConsumerLanSegments = consumerLansMap

	bgpMap, d3 := applyPvifBgpNeighbors(ctx, bgpNeighbors)
	diags.Append(d3...)
	m.GatewayBgpNeighbors = bgpMap

	advObj, d4 := applyPvifSiteInfo(ctx, advertisement)
	diags.Append(d4...)
	m.Advertisement = advObj

	natObj, d5 := applyPvifNatStrategy(ctx, natStrategy)
	diags.Append(d5...)
	m.NatPrefixStrategy = natObj

	return diags
}

func (r *publicVifResource) readByID(ctx context.Context, id int64) (*sdk.V1PvifIdDetailsGetResponse, bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	out, httpResp, err := r.pd.api.DefaultAPI.V1PvifIdDetailsGet(ctx, id).Authorization(r.pd.token).Execute()
	if err != nil {
		if isNotFound(httpResp) {
			closeBody(httpResp)
			return nil, false, diags
		}
		closeBody(httpResp)
		diags.AddError("Unable to read public VIF", apiErrorDetail(err))
		return nil, false, diags
	}
	closeBody(httpResp)
	if out == nil {
		return nil, false, diags
	}
	return out, true, diags
}

func (r *publicVifResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan publicVifResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	write, diags := plan.buildWriteRequest(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Field-by-field rather than a type conversion (ManaV2PublicVifGatewayWriteRequest
	// and V1PvifPostRequest happen to share a field layout today): the two types are
	// independently generated for different endpoints, so relying on that coincidence
	// would silently break if either one's fields are ever reordered/added upstream.
	body := sdk.V1PvifPostRequest{ //nolint:staticcheck // see comment above
		ServiceName:         write.ServiceName,
		LanSegmentId:        write.LanSegmentId,
		RegionId:            write.RegionId,
		StorageProvider:     write.StorageProvider,
		ConsumerLanSegments: write.ConsumerLanSegments,
		GatewayBgpNeighbors: write.GatewayBgpNeighbors,
		NatPrefixStrategy:   write.NatPrefixStrategy,
		Advertisement:       write.Advertisement,
		CoveringPrefixes:    write.CoveringPrefixes,
	}

	out, httpResp, err := r.pd.api.DefaultAPI.V1PvifPost(ctx).
		Authorization(r.pd.token).
		V1PvifPostRequest(body).
		Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create public VIF", apiErrorDetail(err))
		return
	}
	if out == nil || out.Id == nil {
		resp.Diagnostics.AddError("Unable to create public VIF", "API returned an empty response")
		return
	}

	var bgpOut map[string]sdk.ManaV2BgpNeighbor
	if out.GatewayBgpNeighbors != nil {
		bgpOut = *out.GatewayBgpNeighbors
	}
	var consumerOut map[string]sdk.ManaV2PublicVifGatewayConsumerLanDevices
	if out.ConsumerLanSegments != nil {
		consumerOut = *out.ConsumerLanSegments
	}
	resp.Diagnostics.Append(plan.applyResponse(ctx, out.Id, out.ServiceName, nil, out.RegionId, out.StorageProvider,
		consumerOut, out.CoveringPrefixes, out.Advertisement, bgpOut, out.NatPrefixStrategy)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *publicVifResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state publicVifResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := parseInt64ID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid public VIF id", err.Error())
		return
	}

	out, found, diags := r.readByID(ctx, id)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	var bgpOut map[string]sdk.ManaV2BgpNeighbor
	if out.GatewayBgpNeighbors != nil {
		bgpOut = *out.GatewayBgpNeighbors
	}
	var consumerOut map[string]sdk.ManaV2PublicVifGatewayConsumerLanDevices
	if out.ConsumerLanSegments != nil {
		consumerOut = *out.ConsumerLanSegments
	}
	resp.Diagnostics.Append(state.applyResponse(ctx, out.Id, out.ServiceName, nil, out.RegionId, out.StorageProvider,
		consumerOut, out.CoveringPrefixes, out.Advertisement, bgpOut, out.NatPrefixStrategy)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *publicVifResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan publicVifResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := parseInt64ID(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid public VIF id", err.Error())
		return
	}

	write, diags := plan.buildWriteRequest(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := sdk.V1PvifIdPutRequest{Configuration: write}
	out, httpResp, err := r.pd.api.DefaultAPI.V1PvifIdPut(ctx, id).
		Authorization(r.pd.token).
		V1PvifIdPutRequest(body).
		Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update public VIF", apiErrorDetail(err))
		return
	}
	if out == nil {
		resp.Diagnostics.AddError("Unable to update public VIF", "API returned an empty response")
		return
	}

	var bgpOut map[string]sdk.ManaV2BgpNeighbor
	if out.GatewayBgpNeighbors != nil {
		bgpOut = *out.GatewayBgpNeighbors
	}
	var consumerOut map[string]sdk.ManaV2PublicVifGatewayConsumerLanDevices
	if out.ConsumerLanSegments != nil {
		consumerOut = *out.ConsumerLanSegments
	}
	resp.Diagnostics.Append(plan.applyResponse(ctx, out.Id, out.ServiceName, nil, out.RegionId, out.StorageProvider,
		consumerOut, out.CoveringPrefixes, out.Advertisement, bgpOut, out.NatPrefixStrategy)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *publicVifResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state publicVifResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := parseInt64ID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid public VIF id", err.Error())
		return
	}

	_, httpResp, err := r.pd.api.DefaultAPI.V1PvifIdDelete(ctx, id).Authorization(r.pd.token).Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to delete public VIF", apiErrorDetail(err))
	}
}

func (r *publicVifResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
