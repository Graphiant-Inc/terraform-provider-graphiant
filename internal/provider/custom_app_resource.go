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
	_ resource.Resource                = &customAppResource{}
	_ resource.ResourceWithConfigure   = &customAppResource{}
	_ resource.ResourceWithImportState = &customAppResource{}
)

func NewCustomAppResource() resource.Resource {
	return &customAppResource{}
}

type customAppResource struct {
	pd *providerData
}

type customAppPortRangeModel struct {
	Lower types.Int64 `tfsdk:"lower"`
	Upper types.Int64 `tfsdk:"upper"`
}

var customAppPortRangeAttrTypes = map[string]attrType{
	"lower": types.Int64Type,
	"upper": types.Int64Type,
}

type customAppResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	URL         types.String `tfsdk:"url"`
	IPProtocol  types.String `tfsdk:"ip_protocol"`
	IPLists     types.List   `tfsdk:"ip_lists"`
	IPPrefixes  types.List   `tfsdk:"ip_prefixes"`
	PortRanges  types.List   `tfsdk:"port_ranges"`
}

func (r *customAppResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_app"
}

func (r *customAppResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A custom application definition, matched by URL, IP lists/prefixes, and/or port ranges.",
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
			"url": schema.StringAttribute{
				Optional: true,
			},
			"ip_protocol": schema.StringAttribute{
				Optional:    true,
				Description: "Protocol enum value, as defined by the API (values are not enumerated in the SDK).",
			},
			"ip_lists": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
			},
			"ip_prefixes": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
			},
			"port_ranges": schema.ListNestedAttribute{
				Optional: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"lower": schema.Int64Attribute{Required: true},
						"upper": schema.Int64Attribute{Required: true},
					},
				},
			},
		},
	}
}

func (r *customAppResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.pd = configurePD(req.ProviderData, &resp.Diagnostics)
}

func (m *customAppResourceModel) buildConfig(ctx context.Context) (*sdk.ManaV2GlobalAppConfig, diag.Diagnostics) {
	cfg := &sdk.ManaV2GlobalAppConfig{
		Name:        m.Name.ValueStringPointer(),
		Description: m.Description.ValueStringPointer(),
		Url:         m.URL.ValueStringPointer(),
		IpProtocol:  m.IPProtocol.ValueStringPointer(),
	}

	if !m.IPLists.IsNull() && !m.IPLists.IsUnknown() {
		diags := m.IPLists.ElementsAs(ctx, &cfg.IpLists, false)
		if diags.HasError() {
			return nil, diags
		}
	}
	if !m.IPPrefixes.IsNull() && !m.IPPrefixes.IsUnknown() {
		diags := m.IPPrefixes.ElementsAs(ctx, &cfg.IpPrefixes, false)
		if diags.HasError() {
			return nil, diags
		}
	}
	if !m.PortRanges.IsNull() && !m.PortRanges.IsUnknown() {
		var ranges []customAppPortRangeModel
		diags := m.PortRanges.ElementsAs(ctx, &ranges, false)
		if diags.HasError() {
			return nil, diags
		}
		for _, pr := range ranges {
			var lower, upper *int32
			if v := pr.Lower.ValueInt64Pointer(); v != nil {
				l := int32(*v)
				lower = &l
			}
			if v := pr.Upper.ValueInt64Pointer(); v != nil {
				u := int32(*v)
				upper = &u
			}
			cfg.PortRanges = append(cfg.PortRanges, sdk.ManaV2GlobalAppPortRange{Lower: lower, Upper: upper})
		}
	}

	return cfg, nil
}

func (m *customAppResourceModel) applyConfig(ctx context.Context, cfg *sdk.ManaV2GlobalAppConfig) diag.Diagnostics {
	m.Name = types.StringPointerValue(cfg.Name)
	m.Description = types.StringPointerValue(cfg.Description)
	m.URL = types.StringPointerValue(cfg.Url)

	// ip_protocol is Optional but not Computed, so an unset config must round-trip as
	// null: the API doesn't omit this field when it was never set, it returns the
	// literal sentinel "UnknownIPProtocol" (that enum's zero/unspecified value), which
	// would otherwise fail Terraform's post-apply consistency check against the null
	// plan value.
	ipProtocol := cfg.IpProtocol
	if ipProtocol != nil && *ipProtocol == "UnknownIPProtocol" {
		ipProtocol = nil
	}
	m.IPProtocol = types.StringPointerValue(ipProtocol)

	ipLists, diags := types.ListValueFrom(ctx, types.StringType, cfg.IpLists)
	if diags.HasError() {
		return diags
	}
	m.IPLists = ipLists

	ipPrefixes, diags2 := types.ListValueFrom(ctx, types.StringType, cfg.IpPrefixes)
	if diags2.HasError() {
		return diags2
	}
	m.IPPrefixes = ipPrefixes

	if len(cfg.PortRanges) == 0 {
		m.PortRanges = types.ListNull(types.ObjectType{AttrTypes: customAppPortRangeAttrTypes})
	} else {
		ranges := make([]customAppPortRangeModel, 0, len(cfg.PortRanges))
		for _, pr := range cfg.PortRanges {
			var lower, upper *int64
			if pr.Lower != nil {
				l := int64(*pr.Lower)
				lower = &l
			}
			if pr.Upper != nil {
				u := int64(*pr.Upper)
				upper = &u
			}
			ranges = append(ranges, customAppPortRangeModel{
				Lower: types.Int64PointerValue(lower),
				Upper: types.Int64PointerValue(upper),
			})
		}
		portRanges, diags3 := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: customAppPortRangeAttrTypes}, ranges)
		if diags3.HasError() {
			return diags3
		}
		m.PortRanges = portRanges
	}

	return nil
}

func (r *customAppResource) readByID(ctx context.Context, id int64) (*sdk.ManaV2GlobalAppConfig, bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	out, httpResp, err := r.pd.api.DefaultAPI.V1GlobalAppsCustomAppIdGet(ctx, id).Authorization(r.pd.token).Execute()
	if err != nil {
		if isNotFound(httpResp) {
			closeBody(httpResp)
			return nil, false, diags
		}
		closeBody(httpResp)
		diags.AddError("Unable to read custom app", apiErrorDetail(err))
		return nil, false, diags
	}
	closeBody(httpResp)
	if out == nil || out.AppConfig == nil {
		return nil, false, diags
	}
	return out.AppConfig, true, diags
}

func (r *customAppResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan customAppResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg, diags := plan.buildConfig(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := sdk.V1GlobalAppsCustomPostRequest{AppConfig: cfg}
	out, httpResp, err := r.pd.api.DefaultAPI.V1GlobalAppsCustomPost(ctx).
		Authorization(r.pd.token).
		V1GlobalAppsCustomPostRequest(body).
		Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create custom app", apiErrorDetail(err))
		return
	}
	if out == nil || out.AppIdentifier == nil || out.AppIdentifier.Id == nil {
		resp.Diagnostics.AddError("Unable to create custom app", "API returned an empty response")
		return
	}

	created, found, diags2 := r.readByID(ctx, *out.AppIdentifier.Id)
	resp.Diagnostics.Append(diags2...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.Diagnostics.AddError("Unable to read created custom app", "custom app was created but could not be read back")
		return
	}

	plan.ID = types.StringValue(int64ID(*out.AppIdentifier.Id))
	resp.Diagnostics.Append(plan.applyConfig(ctx, created)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *customAppResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state customAppResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := parseInt64ID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid custom app id", err.Error())
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

func (r *customAppResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan customAppResourceModel
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
		resp.Diagnostics.AddError("Invalid custom app id", err.Error())
		return
	}

	body := sdk.V1GlobalAppsCustomAppIdPutRequest{AppConfig: cfg}
	_, httpResp, err := r.pd.api.DefaultAPI.V1GlobalAppsCustomAppIdPut(ctx, id).
		Authorization(r.pd.token).
		V1GlobalAppsCustomAppIdPutRequest(body).
		Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update custom app", apiErrorDetail(err))
		return
	}

	updated, found, diags2 := r.readByID(ctx, id)
	resp.Diagnostics.Append(diags2...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.Diagnostics.AddError("Unable to update custom app", "custom app no longer exists")
		return
	}

	resp.Diagnostics.Append(plan.applyConfig(ctx, updated)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *customAppResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state customAppResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := parseInt64ID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid custom app id", err.Error())
		return
	}

	_, httpResp, err := r.pd.api.DefaultAPI.V1GlobalAppsCustomAppIdDelete(ctx, id).Authorization(r.pd.token).Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to delete custom app", apiErrorDetail(err))
	}
}

func (r *customAppResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
