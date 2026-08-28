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
	_ resource.Resource                = &siteResource{}
	_ resource.ResourceWithConfigure   = &siteResource{}
	_ resource.ResourceWithImportState = &siteResource{}
)

func NewSiteResource() resource.Resource {
	return &siteResource{}
}

type siteResource struct {
	pd *providerData
}

type siteLocationModel struct {
	AddressLine1 types.String  `tfsdk:"address_line1"`
	AddressLine2 types.String  `tfsdk:"address_line2"`
	City         types.String  `tfsdk:"city"`
	State        types.String  `tfsdk:"state"`
	StateCode    types.String  `tfsdk:"state_code"`
	ProvinceCode types.String  `tfsdk:"province_code"`
	Country      types.String  `tfsdk:"country"`
	CountryCode  types.String  `tfsdk:"country_code"`
	Latitude     types.Float64 `tfsdk:"latitude"`
	Longitude    types.Float64 `tfsdk:"longitude"`
	Notes        types.String  `tfsdk:"notes"`
}

var siteLocationAttrTypes = map[string]attrType{
	"address_line1": types.StringType,
	"address_line2": types.StringType,
	"city":          types.StringType,
	"state":         types.StringType,
	"state_code":    types.StringType,
	"province_code": types.StringType,
	"country":       types.StringType,
	"country_code":  types.StringType,
	"latitude":      types.Float64Type,
	"longitude":     types.Float64Type,
	"notes":         types.StringType,
}

type siteRouteTagModel struct {
	LevelZero types.String `tfsdk:"level_zero"`
	LevelOne  types.String `tfsdk:"level_one"`
	LevelTwo  types.String `tfsdk:"level_two"`
}

type siteResourceModel struct {
	ID           types.String `tfsdk:"id"`
	EnterpriseID types.Int64  `tfsdk:"enterprise_id"`
	Name         types.String `tfsdk:"name"`
	Notes        types.String `tfsdk:"notes"`
	Location     types.Object `tfsdk:"location"`
	RouteTag     types.Object `tfsdk:"route_tag"`
	Tags         types.List   `tfsdk:"tags"`
}

func (r *siteResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_site"
}

func (r *siteResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "An enterprise site. Advanced policy-attachment fields on the underlying API " +
			"(prefix sets, routing/traffic policy, NTP/SNMP/syslog/IPFIX operations) are not yet exposed here.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Server-assigned site ID.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"enterprise_id": schema.Int64Attribute{
				Optional:    true,
				Description: "Enterprise to create the site under (MSP use). Cannot be changed after creation.",
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Site name.",
			},
			"notes": schema.StringAttribute{
				Optional:    true,
				Description: "Free-form notes.",
			},
			"tags": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Server-assigned tags.",
			},
		},
		Blocks: map[string]schema.Block{
			"location": schema.SingleNestedBlock{
				Description: "Site physical location.",
				Attributes: map[string]schema.Attribute{
					"address_line1": schema.StringAttribute{Optional: true},
					"address_line2": schema.StringAttribute{Optional: true},
					"city":          schema.StringAttribute{Optional: true},
					"state":         schema.StringAttribute{Optional: true},
					"state_code":    schema.StringAttribute{Optional: true},
					"province_code": schema.StringAttribute{Optional: true},
					"country":       schema.StringAttribute{Optional: true},
					"country_code":  schema.StringAttribute{Optional: true},
					"latitude":      schema.Float64Attribute{Optional: true},
					"longitude":     schema.Float64Attribute{Optional: true},
					"notes":         schema.StringAttribute{Optional: true},
				},
			},
			"route_tag": schema.SingleNestedBlock{
				Description: "Route tag levels applied to the site.",
				Attributes: map[string]schema.Attribute{
					"level_zero": schema.StringAttribute{Optional: true},
					"level_one":  schema.StringAttribute{Optional: true},
					"level_two":  schema.StringAttribute{Optional: true},
				},
			},
		},
	}
}

func (r *siteResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.pd = configurePD(req.ProviderData, &resp.Diagnostics)
}

// buildSite converts the plan's location/route_tag blocks into an SDK ManaV2NewSite.
func (m *siteResourceModel) buildSite(ctx context.Context) (*sdk.ManaV2NewSite, diag.Diagnostics) {
	site := &sdk.ManaV2NewSite{
		Name:  m.Name.ValueStringPointer(),
		Notes: m.Notes.ValueStringPointer(),
	}

	if !m.Location.IsNull() && !m.Location.IsUnknown() {
		var loc siteLocationModel
		diags := m.Location.As(ctx, &loc, objectAsOptions)
		if diags.HasError() {
			return nil, diags
		}
		site.Location = &sdk.ManaV2Location{
			AddressLine1: loc.AddressLine1.ValueStringPointer(),
			AddressLine2: loc.AddressLine2.ValueStringPointer(),
			City:         loc.City.ValueStringPointer(),
			State:        loc.State.ValueStringPointer(),
			StateCode:    loc.StateCode.ValueStringPointer(),
			ProvinceCode: loc.ProvinceCode.ValueStringPointer(),
			Country:      loc.Country.ValueStringPointer(),
			CountryCode:  loc.CountryCode.ValueStringPointer(),
			Latitude:     loc.Latitude.ValueFloat64Pointer(),
			Longitude:    loc.Longitude.ValueFloat64Pointer(),
			Notes:        loc.Notes.ValueStringPointer(),
		}
	}

	if !m.RouteTag.IsNull() && !m.RouteTag.IsUnknown() {
		var rt siteRouteTagModel
		diags := m.RouteTag.As(ctx, &rt, objectAsOptions)
		if diags.HasError() {
			return nil, diags
		}
		site.RouteTag = &sdk.ManaV2RouteTag{
			LevelZero: rt.LevelZero.ValueStringPointer(),
			LevelOne:  rt.LevelOne.ValueStringPointer(),
			LevelTwo:  rt.LevelTwo.ValueStringPointer(),
		}
	}

	return site, nil
}

// applySite copies an API-returned ManaV2Site back into the resource model. It
// deliberately leaves m.RouteTag untouched: the API doesn't echo route_tag back on
// ManaV2Site (only a resolved PolicyTag summary), so whatever the caller already
// had in the model (from plan or prior state) is preserved as-is.
func (m *siteResourceModel) applySite(ctx context.Context, site *sdk.ManaV2Site) diag.Diagnostics {
	m.ID = types.StringValue(int64PtrID(site.Id))
	m.Name = types.StringPointerValue(site.Name)
	m.Notes = types.StringPointerValue(site.Notes)

	tags, diags := types.ListValueFrom(ctx, types.StringType, site.Tags)
	if diags.HasError() {
		return diags
	}
	m.Tags = tags

	if site.Location != nil {
		loc := siteLocationModel{
			AddressLine1: types.StringPointerValue(site.Location.AddressLine1),
			AddressLine2: types.StringPointerValue(site.Location.AddressLine2),
			City:         types.StringPointerValue(site.Location.City),
			State:        types.StringPointerValue(site.Location.State),
			StateCode:    types.StringPointerValue(site.Location.StateCode),
			ProvinceCode: types.StringPointerValue(site.Location.ProvinceCode),
			Country:      types.StringPointerValue(site.Location.Country),
			CountryCode:  types.StringPointerValue(site.Location.CountryCode),
			Latitude:     types.Float64PointerValue(site.Location.Latitude),
			Longitude:    types.Float64PointerValue(site.Location.Longitude),
			Notes:        types.StringPointerValue(site.Location.Notes),
		}
		obj, d := types.ObjectValueFrom(ctx, siteLocationAttrTypes, loc)
		if d.HasError() {
			return d
		}
		m.Location = obj
	} else {
		m.Location = types.ObjectNull(siteLocationAttrTypes)
	}

	return nil
}

func (r *siteResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan siteResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site, diags := plan.buildSite(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := sdk.V1SitesPostRequest{
		EnterpriseId: plan.EnterpriseID.ValueInt64Pointer(),
		Site:         site,
	}

	out, httpResp, err := r.pd.api.DefaultAPI.V1SitesPost(ctx).
		Authorization(r.pd.token).
		V1SitesPostRequest(body).
		Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create site", apiErrorDetail(err))
		return
	}
	if out == nil || out.Site == nil {
		resp.Diagnostics.AddError("Unable to create site", "API returned an empty response")
		return
	}

	resp.Diagnostics.Append(plan.applySite(ctx, out.Site)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *siteResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state siteResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site, found, diags := r.findSite(ctx, state.ID.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(state.applySite(ctx, site)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *siteResource) findSite(ctx context.Context, id string) (*sdk.ManaV2Site, bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	out, httpResp, err := r.pd.api.DefaultAPI.V1SitesGet(ctx).Authorization(r.pd.token).Execute()
	closeBody(httpResp)
	if err != nil {
		diags.AddError("Unable to list sites", apiErrorDetail(err))
		return nil, false, diags
	}
	if out == nil {
		return nil, false, diags
	}
	for i := range out.Sites {
		if int64PtrID(out.Sites[i].Id) == id {
			return &out.Sites[i], true, diags
		}
	}
	return nil, false, diags
}

func (r *siteResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan siteResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site, diags := plan.buildSite(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	siteID, err := parseInt64ID(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid site id", err.Error())
		return
	}

	body := sdk.V1SitesSiteIdPostRequest{Site: site}
	out, httpResp, err := r.pd.api.DefaultAPI.V1SitesSiteIdPost(ctx, siteID).
		Authorization(r.pd.token).
		V1SitesSiteIdPostRequest(body).
		Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update site", apiErrorDetail(err))
		return
	}
	if out == nil || out.Site == nil {
		resp.Diagnostics.AddError("Unable to update site", "API returned an empty response")
		return
	}

	resp.Diagnostics.Append(plan.applySite(ctx, out.Site)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *siteResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state siteResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	siteID, err := parseInt64ID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid site id", err.Error())
		return
	}

	httpResp, err := r.pd.api.DefaultAPI.V1SitesSiteIdDelete(ctx, siteID).Authorization(r.pd.token).Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to delete site", apiErrorDetail(err))
	}
}

func (r *siteResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
