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
	_ resource.Resource                = &gatewayResource{}
	_ resource.ResourceWithConfigure   = &gatewayResource{}
	_ resource.ResourceWithImportState = &gatewayResource{}
)

func NewGatewayResource() resource.Resource {
	return &gatewayResource{}
}

type gatewayResource struct {
	pd *providerData
}

type ipsecTunnelModel struct {
	InsideIpv4Cidr       types.String `tfsdk:"inside_ipv4_cidr"`
	InsideIpv6Cidr       types.String `tfsdk:"inside_ipv6_cidr"`
	LocalIkePeerIdentity types.String `tfsdk:"local_ike_peer_identity"`
	Psk                  types.String `tfsdk:"psk"`
}

var ipsecTunnelAttrTypes = map[string]attrType{
	"inside_ipv4_cidr":        types.StringType,
	"inside_ipv6_cidr":        types.StringType,
	"local_ike_peer_identity": types.StringType,
	"psk":                     types.StringType,
}

type ipsecGatewayModel struct {
	DestinationAddress    types.String `tfsdk:"destination_address"`
	IkeInitiator          types.Bool   `tfsdk:"ike_initiator"`
	Mtu                   types.Int64  `tfsdk:"mtu"`
	Name                  types.String `tfsdk:"name"`
	RemoteIkePeerIdentity types.String `tfsdk:"remote_ike_peer_identity"`
	TcpMss                types.Int64  `tfsdk:"tcp_mss"`
	VpnProfile            types.String `tfsdk:"vpn_profile"`
	Tunnel1               types.Object `tfsdk:"tunnel1"`
	Tunnel2               types.Object `tfsdk:"tunnel2"`
}

var ipsecGatewayAttrTypes = map[string]attrType{
	"destination_address":      types.StringType,
	"ike_initiator":            types.BoolType,
	"mtu":                      types.Int64Type,
	"name":                     types.StringType,
	"remote_ike_peer_identity": types.StringType,
	"tcp_mss":                  types.Int64Type,
	"vpn_profile":              types.StringType,
	"tunnel1":                  types.ObjectType{AttrTypes: ipsecTunnelAttrTypes},
	"tunnel2":                  types.ObjectType{AttrTypes: ipsecTunnelAttrTypes},
}

type gatewayResourceModel struct {
	ID            types.String `tfsdk:"id"`
	RegionID      types.Int64  `tfsdk:"region_id"`
	VrfID         types.Int64  `tfsdk:"vrf_id"`
	Speed         types.String `tfsdk:"speed"`
	Description   types.String `tfsdk:"description"`
	IpsecGateway  types.Object `tfsdk:"ipsec_gateway"`
	Name          types.String `tfsdk:"name"`
	Status        types.String `tfsdk:"status"`
	SupportStatus types.String `tfsdk:"support_status"`
	Type          types.String `tfsdk:"type"`
}

func (r *gatewayResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_gateway"
}

func (r *gatewayResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A Graphiant gateway service. Cloud-provider gateway types (AWS/Azure/GCP/OCI) and multi-peer " +
			"IPsec (ipsec_gateway_peers) are not yet exposed — only region/VRF core fields and single-peer IPsec.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"region_id": schema.Int64Attribute{
				Required: true,
			},
			"vrf_id": schema.Int64Attribute{
				Required:    true,
				Description: "Segment (VRF) this gateway is associated with.",
			},
			"speed": schema.StringAttribute{
				Optional: true,
			},
			"description": schema.StringAttribute{
				Optional: true,
			},
			"name": schema.StringAttribute{
				Computed: true,
			},
			"status": schema.StringAttribute{
				Computed: true,
			},
			"support_status": schema.StringAttribute{
				Computed: true,
			},
			"type": schema.StringAttribute{
				Computed: true,
			},
			"ipsec_gateway": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"destination_address":      schema.StringAttribute{Optional: true},
					"ike_initiator":            schema.BoolAttribute{Optional: true},
					"mtu":                      schema.Int64Attribute{Optional: true},
					"name":                     schema.StringAttribute{Optional: true},
					"remote_ike_peer_identity": schema.StringAttribute{Optional: true},
					"tcp_mss":                  schema.Int64Attribute{Optional: true},
					"vpn_profile":              schema.StringAttribute{Optional: true},
					"tunnel1":                  gatewayTunnelAttribute(),
					"tunnel2":                  gatewayTunnelAttribute(),
				},
			},
		},
	}
}

func gatewayTunnelAttribute() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Optional: true,
		Attributes: map[string]schema.Attribute{
			"inside_ipv4_cidr":        schema.StringAttribute{Optional: true},
			"inside_ipv6_cidr":        schema.StringAttribute{Optional: true},
			"local_ike_peer_identity": schema.StringAttribute{Optional: true},
			"psk":                     schema.StringAttribute{Optional: true, Sensitive: true},
		},
	}
}

func (r *gatewayResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.pd = configurePD(req.ProviderData, &resp.Diagnostics)
}

func buildIpsecTunnel(ctx context.Context, obj types.Object) (*sdk.ManaV2IPsecGatewayTunnelDetails, diag.Diagnostics) {
	if obj.IsNull() || obj.IsUnknown() {
		return nil, nil
	}
	var m ipsecTunnelModel
	diags := obj.As(ctx, &m, objectAsOptions)
	if diags.HasError() {
		return nil, diags
	}
	return &sdk.ManaV2IPsecGatewayTunnelDetails{
		InsideIpv4Cidr:       m.InsideIpv4Cidr.ValueStringPointer(),
		InsideIpv6Cidr:       m.InsideIpv6Cidr.ValueStringPointer(),
		LocalIkePeerIdentity: m.LocalIkePeerIdentity.ValueStringPointer(),
		Psk:                  m.Psk.ValueStringPointer(),
	}, nil
}

func applyIpsecTunnel(ctx context.Context, t *sdk.ManaV2IPsecGatewayTunnelDetails) (types.Object, diag.Diagnostics) {
	if t == nil {
		return types.ObjectNull(ipsecTunnelAttrTypes), nil
	}
	m := ipsecTunnelModel{
		InsideIpv4Cidr:       types.StringPointerValue(t.InsideIpv4Cidr),
		InsideIpv6Cidr:       types.StringPointerValue(t.InsideIpv6Cidr),
		LocalIkePeerIdentity: types.StringPointerValue(t.LocalIkePeerIdentity),
		Psk:                  types.StringPointerValue(t.Psk),
	}
	return types.ObjectValueFrom(ctx, ipsecTunnelAttrTypes, m)
}

func (m *gatewayResourceModel) buildDetails(ctx context.Context) (*sdk.ManaV2GatewayDetails, diag.Diagnostics) {
	details := &sdk.ManaV2GatewayDetails{
		RegionId:    int32PtrFromInt64(m.RegionID),
		VrfId:       m.VrfID.ValueInt64Pointer(),
		Speed:       m.Speed.ValueStringPointer(),
		Description: m.Description.ValueStringPointer(),
	}

	if !m.IpsecGateway.IsNull() && !m.IpsecGateway.IsUnknown() {
		var ig ipsecGatewayModel
		diags := m.IpsecGateway.As(ctx, &ig, objectAsOptions)
		if diags.HasError() {
			return nil, diags
		}
		tunnel1, diags := buildIpsecTunnel(ctx, ig.Tunnel1)
		if diags.HasError() {
			return nil, diags
		}
		tunnel2, diags := buildIpsecTunnel(ctx, ig.Tunnel2)
		if diags.HasError() {
			return nil, diags
		}
		details.IpsecGateway = &sdk.ManaV2IPsecGatewayDetails{
			DestinationAddress:    ig.DestinationAddress.ValueStringPointer(),
			IkeInitiator:          ig.IkeInitiator.ValueBoolPointer(),
			Mtu:                   int32PtrFromInt64(ig.Mtu),
			Name:                  ig.Name.ValueStringPointer(),
			RemoteIkePeerIdentity: ig.RemoteIkePeerIdentity.ValueStringPointer(),
			TcpMss:                int32PtrFromInt64(ig.TcpMss),
			VpnProfile:            ig.VpnProfile.ValueStringPointer(),
			Tunnel1:               tunnel1,
			Tunnel2:               tunnel2,
		}
	}

	return details, nil
}

func (m *gatewayResourceModel) applyDetails(ctx context.Context, details *sdk.ManaV2GatewayDetails) diag.Diagnostics {
	if details.RegionId != nil {
		m.RegionID = types.Int64Value(int64(*details.RegionId))
	}
	m.VrfID = types.Int64PointerValue(details.VrfId)
	m.Speed = types.StringPointerValue(details.Speed)
	m.Description = types.StringPointerValue(details.Description)

	if details.IpsecGateway == nil {
		m.IpsecGateway = types.ObjectNull(ipsecGatewayAttrTypes)
		return nil
	}
	ig := details.IpsecGateway
	tunnel1, diags := applyIpsecTunnel(ctx, ig.Tunnel1)
	if diags.HasError() {
		return diags
	}
	tunnel2, diags := applyIpsecTunnel(ctx, ig.Tunnel2)
	if diags.HasError() {
		return diags
	}
	igModel := ipsecGatewayModel{
		DestinationAddress:    types.StringPointerValue(ig.DestinationAddress),
		IkeInitiator:          types.BoolPointerValue(ig.IkeInitiator),
		Name:                  types.StringPointerValue(ig.Name),
		RemoteIkePeerIdentity: types.StringPointerValue(ig.RemoteIkePeerIdentity),
		VpnProfile:            types.StringPointerValue(ig.VpnProfile),
		Tunnel1:               tunnel1,
		Tunnel2:               tunnel2,
	}
	if ig.Mtu != nil {
		igModel.Mtu = types.Int64Value(int64(*ig.Mtu))
	}
	if ig.TcpMss != nil {
		igModel.TcpMss = types.Int64Value(int64(*ig.TcpMss))
	}
	obj, diags := types.ObjectValueFrom(ctx, ipsecGatewayAttrTypes, igModel)
	if diags.HasError() {
		return diags
	}
	m.IpsecGateway = obj
	return nil
}

func int32PtrFromInt64(v types.Int64) *int32 {
	p := v.ValueInt64Pointer()
	if p == nil {
		return nil
	}
	i := int32(*p)
	return &i
}

func (r *gatewayResource) findSummary(ctx context.Context, id string) (*sdk.ManaV2GatewaySummary, bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	out, httpResp, err := r.pd.api.DefaultAPI.V1GatewaysSummaryGet(ctx).Authorization(r.pd.token).Execute()
	closeBody(httpResp)
	if err != nil {
		diags.AddError("Unable to list gateways", apiErrorDetail(err))
		return nil, false, diags
	}
	if out == nil {
		return nil, false, diags
	}
	for i := range out.Summaries {
		if int64PtrID(out.Summaries[i].Id) == id {
			return &out.Summaries[i], true, diags
		}
	}
	return nil, false, diags
}

func (m *gatewayResourceModel) applySummary(s *sdk.ManaV2GatewaySummary) {
	m.Name = types.StringPointerValue(s.Name)
	m.Status = types.StringPointerValue(s.Status)
	m.SupportStatus = types.StringPointerValue(s.SupportStatus)
	m.Type = types.StringPointerValue(s.Type)
}

func (r *gatewayResource) readByID(ctx context.Context, id int64) (*sdk.ManaV2GatewayDetails, bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	out, httpResp, err := r.pd.api.DefaultAPI.V1GatewaysIdDetailsGet(ctx, id).Authorization(r.pd.token).Execute()
	if err != nil {
		if isNotFound(httpResp) {
			closeBody(httpResp)
			return nil, false, diags
		}
		closeBody(httpResp)
		diags.AddError("Unable to read gateway", apiErrorDetail(err))
		return nil, false, diags
	}
	closeBody(httpResp)
	if out == nil || out.Details == nil {
		return nil, false, diags
	}
	return out.Details, true, diags
}

func (r *gatewayResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan gatewayResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	details, diags := plan.buildDetails(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := sdk.V1GatewaysPostRequest{Details: details}
	out, httpResp, err := r.pd.api.DefaultAPI.V1GatewaysPost(ctx).
		Authorization(r.pd.token).
		V1GatewaysPostRequest(body).
		Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create gateway", apiErrorDetail(err))
		return
	}
	if out == nil || out.Id == nil {
		resp.Diagnostics.AddError("Unable to create gateway", "API returned an empty response")
		return
	}

	plan.ID = types.StringValue(int64ID(*out.Id))

	summary, found, diags2 := r.findSummary(ctx, plan.ID.ValueString())
	resp.Diagnostics.Append(diags2...)
	if resp.Diagnostics.HasError() {
		return
	}
	if found {
		plan.applySummary(summary)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *gatewayResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state gatewayResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := parseInt64ID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid gateway id", err.Error())
		return
	}

	details, found, diags := r.readByID(ctx, id)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(state.applyDetails(ctx, details)...)
	if resp.Diagnostics.HasError() {
		return
	}

	summary, found2, diags2 := r.findSummary(ctx, state.ID.ValueString())
	resp.Diagnostics.Append(diags2...)
	if resp.Diagnostics.HasError() {
		return
	}
	if found2 {
		state.applySummary(summary)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *gatewayResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan gatewayResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	details, diags := plan.buildDetails(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := parseInt64ID(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid gateway id", err.Error())
		return
	}

	body := sdk.V1GatewaysPutRequest{Id: &id, Details: details}
	_, httpResp, err := r.pd.api.DefaultAPI.V1GatewaysPut(ctx).
		Authorization(r.pd.token).
		V1GatewaysPutRequest(body).
		Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update gateway", apiErrorDetail(err))
		return
	}

	updated, found, diags2 := r.readByID(ctx, id)
	resp.Diagnostics.Append(diags2...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.Diagnostics.AddError("Unable to update gateway", "gateway no longer exists")
		return
	}

	resp.Diagnostics.Append(plan.applyDetails(ctx, updated)...)
	if resp.Diagnostics.HasError() {
		return
	}

	summary, found2, diags3 := r.findSummary(ctx, plan.ID.ValueString())
	resp.Diagnostics.Append(diags3...)
	if resp.Diagnostics.HasError() {
		return
	}
	if found2 {
		plan.applySummary(summary)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *gatewayResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state gatewayResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := parseInt64ID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid gateway id", err.Error())
		return
	}

	_, httpResp, err := r.pd.api.DefaultAPI.V1GatewaysDelete(ctx).Id(id).Authorization(r.pd.token).Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to delete gateway", apiErrorDetail(err))
	}
}

func (r *gatewayResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
