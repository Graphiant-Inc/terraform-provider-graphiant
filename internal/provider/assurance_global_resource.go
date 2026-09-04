package provider

import (
	"context"
	"errors"
	"net/http"
	"strings"

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
	_ resource.Resource                = &assuranceGlobalResource{}
	_ resource.ResourceWithConfigure   = &assuranceGlobalResource{}
	_ resource.ResourceWithImportState = &assuranceGlobalResource{}
)

func NewAssuranceGlobalResource() resource.Resource {
	return &assuranceGlobalResource{}
}

type assuranceGlobalResource struct {
	pd *providerData
}

type assuranceBucketAppModel struct {
	BucketID      types.Int64  `tfsdk:"bucket_id"`
	CustomAppID   types.Int64  `tfsdk:"custom_app_id"`
	BuiltinAppID  types.Int64  `tfsdk:"builtin_app_id"`
	Name          types.String `tfsdk:"name"`
	IsDomain      types.Bool   `tfsdk:"is_domain"`
	UseAllServers types.Bool   `tfsdk:"use_all_servers"`
}

var assuranceBucketAppAttrTypes = map[string]attrType{
	"bucket_id":       types.Int64Type,
	"custom_app_id":   types.Int64Type,
	"builtin_app_id":  types.Int64Type,
	"name":            types.StringType,
	"is_domain":       types.BoolType,
	"use_all_servers": types.BoolType,
}

var assuranceBucketAppListType = types.ObjectType{AttrTypes: assuranceBucketAppAttrTypes}

type assuranceGlobalResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	FlexAlgo    types.String `tfsdk:"flex_algo"`
	SiteListID  types.Int64  `tfsdk:"site_list_id"`
	UseAllSites types.Bool   `tfsdk:"use_all_sites"`
	LanNames    types.List   `tfsdk:"lan_names"`
	Apps        types.List   `tfsdk:"apps"`
}

func (r *assuranceGlobalResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_assurance_global"
}

func (r *assuranceGlobalResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A global SLA assurance configuration (data-assurance domain): which apps/LANs/sites to " +
			"monitor and which flex algo to score against.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Optional: true,
			},
			"flex_algo": schema.StringAttribute{
				Optional: true,
			},
			"site_list_id": schema.Int64Attribute{
				Optional:    true,
				Description: "Mutually exclusive with use_all_sites (not enforced by a validator).",
			},
			"use_all_sites": schema.BoolAttribute{
				Optional: true,
			},
			"lan_names": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
			},
			"apps": schema.ListNestedAttribute{
				Optional: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"bucket_id":       schema.Int64Attribute{Computed: true},
						"custom_app_id":   schema.Int64Attribute{Optional: true},
						"builtin_app_id":  schema.Int64Attribute{Optional: true},
						"name":            schema.StringAttribute{Optional: true},
						"is_domain":       schema.BoolAttribute{Optional: true},
						"use_all_servers": schema.BoolAttribute{Optional: true},
					},
				},
			},
		},
	}
}

func (r *assuranceGlobalResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.pd = configurePD(req.ProviderData, &resp.Diagnostics)
}

func (m *assuranceGlobalResourceModel) buildConfig(ctx context.Context) (*sdk.ManaV2AssuranceConfig, diag.Diagnostics) {
	cfg := &sdk.ManaV2AssuranceConfig{
		Name:       m.Name.ValueStringPointer(),
		FlexAlgo:   m.FlexAlgo.ValueStringPointer(),
		SiteListId: m.SiteListID.ValueInt64Pointer(),
	}
	// UseAllSites is only ever sent as true, never as an explicit false: the
	// update endpoint's validation rejects the field when present and false
	// ("invalid AssuranceConfig.UseAllSites: value must equal true"), so
	// "not using all sites" must be expressed by omitting the field (nil)
	// rather than sending false, and site_list_id carries the real signal.
	if m.UseAllSites.ValueBool() {
		cfg.UseAllSites = m.UseAllSites.ValueBoolPointer()
	}
	var diags diag.Diagnostics
	if !m.LanNames.IsNull() && !m.LanNames.IsUnknown() {
		diags.Append(m.LanNames.ElementsAs(ctx, &cfg.LanNames, false)...)
	}
	if !m.Apps.IsNull() && !m.Apps.IsUnknown() {
		var apps []assuranceBucketAppModel
		diags.Append(m.Apps.ElementsAs(ctx, &apps, false)...)
		for _, a := range apps {
			cfg.Apps = append(cfg.Apps, sdk.ManaV2BucketApp{
				CustomAppId:   a.CustomAppID.ValueInt64Pointer(),
				BuiltinAppId:  a.BuiltinAppID.ValueInt64Pointer(),
				Name:          a.Name.ValueStringPointer(),
				IsDomain:      a.IsDomain.ValueBoolPointer(),
				UseAllServers: a.UseAllServers.ValueBoolPointer(),
			})
		}
	}
	if diags.HasError() {
		return nil, diags
	}
	return cfg, diags
}

func (m *assuranceGlobalResourceModel) applyConfig(ctx context.Context, cfg *sdk.ManaV2AssuranceConfig) diag.Diagnostics {
	var diags diag.Diagnostics
	m.Name = types.StringPointerValue(cfg.Name)
	m.FlexAlgo = types.StringPointerValue(cfg.FlexAlgo)
	m.SiteListID = types.Int64PointerValue(cfg.SiteListId)
	// use_all_sites is Optional but not Computed, so an explicit false in
	// config/state must round-trip as false, not null: since buildConfig
	// only ever sends this field as true (never false, see there), the API
	// omitting it from a read response means false, not "unset".
	useAllSites := cfg.UseAllSites
	if useAllSites == nil {
		f := false
		useAllSites = &f
	}
	m.UseAllSites = types.BoolPointerValue(useAllSites)

	lanNames, d := types.ListValueFrom(ctx, types.StringType, cfg.LanNames)
	diags.Append(d...)
	m.LanNames = lanNames

	// A nil (not merely empty) slice here matters: apps is Optional but not
	// Computed, so when the config never sets it, Terraform's plan value is
	// null, and ListValueFrom must be handed a nil slice to round-trip that
	// back to null too — an empty-but-non-nil slice instead produces an
	// empty list, which fails Terraform's post-apply consistency check.
	var apps []assuranceBucketAppModel
	for _, a := range cfg.Apps {
		apps = append(apps, assuranceBucketAppModel{
			BucketID:      types.Int64PointerValue(intPtr32To64(a.BucketId)),
			CustomAppID:   types.Int64PointerValue(a.CustomAppId),
			BuiltinAppID:  types.Int64PointerValue(a.BuiltinAppId),
			Name:          types.StringPointerValue(a.Name),
			IsDomain:      types.BoolPointerValue(a.IsDomain),
			UseAllServers: types.BoolPointerValue(a.UseAllServers),
		})
	}
	appsList, d2 := types.ListValueFrom(ctx, assuranceBucketAppListType, apps)
	diags.Append(d2...)
	m.Apps = appsList

	return diags
}

func (r *assuranceGlobalResource) readByID(ctx context.Context, id int64) (*sdk.ManaV2AssuranceConfig, bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	out, httpResp, err := r.pd.api.DefaultAPI.V1DataAssuranceAssurancesGlobalIdGet(ctx, id).Authorization(r.pd.token).Execute()
	if err != nil {
		if isNotFound(httpResp) {
			closeBody(httpResp)
			return nil, false, diags
		}
		closeBody(httpResp)
		diags.AddError("Unable to read assurance config", apiErrorDetail(err))
		return nil, false, diags
	}
	closeBody(httpResp)
	if out == nil || out.Config == nil {
		return nil, false, diags
	}
	return out.Config, true, diags
}

func (r *assuranceGlobalResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan assuranceGlobalResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg, diags := plan.buildConfig(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := sdk.V1DataAssuranceAssurancesGlobalPostRequest{Config: cfg}
	out, httpResp, err := r.pd.api.DefaultAPI.V1DataAssuranceAssurancesGlobalPost(ctx).
		Authorization(r.pd.token).
		V1DataAssuranceAssurancesGlobalPostRequest(body).
		Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create assurance config", apiErrorDetail(err))
		return
	}
	if out == nil || out.AssuranceId == nil {
		resp.Diagnostics.AddError("Unable to create assurance config", "API returned an empty response")
		return
	}

	got, found, diags2 := r.readByID(ctx, *out.AssuranceId)
	resp.Diagnostics.Append(diags2...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.Diagnostics.AddError("Unable to read created assurance config", "config was created but could not be read back")
		return
	}

	plan.ID = types.StringValue(int64ID(*out.AssuranceId))
	resp.Diagnostics.Append(plan.applyConfig(ctx, got)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *assuranceGlobalResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state assuranceGlobalResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := parseInt64ID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid assurance config id", err.Error())
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

func (r *assuranceGlobalResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan assuranceGlobalResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := parseInt64ID(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid assurance config id", err.Error())
		return
	}

	cfg, diags := plan.buildConfig(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := sdk.V1DataAssuranceAssurancesGlobalIdPutRequest{Config: cfg}
	_, httpResp, err := r.pd.api.DefaultAPI.V1DataAssuranceAssurancesGlobalIdPut(ctx, id).
		Authorization(r.pd.token).
		V1DataAssuranceAssurancesGlobalIdPutRequest(body).
		Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update assurance config", apiErrorDetail(err))
		return
	}

	got, found, diags2 := r.readByID(ctx, id)
	resp.Diagnostics.Append(diags2...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.Diagnostics.AddError("Unable to update assurance config", "config no longer exists")
		return
	}

	resp.Diagnostics.Append(plan.applyConfig(ctx, got)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *assuranceGlobalResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state assuranceGlobalResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := parseInt64ID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid assurance config id", err.Error())
		return
	}

	// A config still scoped to use_all_sites=true (from the API's perspective — this can lag
	// what Terraform's state says, e.g. after an update that pointed site_list_id at a site
	// list that was, or later became, empty) can't be deleted directly: "must detach ... from
	// all sites prior to deletion". React to that specific error rather than trusting state,
	// by repointing at an existing site list and retrying once.
	_, httpResp, err := r.pd.api.DefaultAPI.V1DataAssuranceAssurancesGlobalIdDelete(ctx, id).Authorization(r.pd.token).Execute()
	closeBody(httpResp)
	if err != nil && isAssuranceMustDetachFromAllSitesError(err) {
		diags := r.detachFromAllSites(ctx, id)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		var httpResp2 *http.Response
		_, httpResp2, err = r.pd.api.DefaultAPI.V1DataAssuranceAssurancesGlobalIdDelete(ctx, id).Authorization(r.pd.token).Execute()
		closeBody(httpResp2)
	}
	if err != nil {
		resp.Diagnostics.AddError("Unable to delete assurance config", apiErrorDetail(err))
	}
}

// isAssuranceMustDetachFromAllSitesError reports whether err is the API's "must detach ... from all
// sites prior to deletion" rejection, as opposed to some other delete failure that
// detachFromAllSites's site-list dance wouldn't help with.
func isAssuranceMustDetachFromAllSitesError(err error) bool {
	var apiErr *sdk.GenericOpenAPIError
	if !errors.As(err, &apiErr) {
		return false
	}
	return strings.Contains(string(apiErr.Body()), "from all sites prior to deletion")
}

// detachFromAllSites repoints a use_all_sites=true config at an existing, non-empty site
// list so it can be deleted. There's no dedicated detach call — the API only clears
// use_all_sites once the config's site_list_id points at a site list that actually has
// sites in it. This reuses any such site list already in the tenant rather than creating
// a throwaway one: a scratch site + site list created here would themselves need cleanup,
// and a failure partway through would leave orphaned objects behind in the tenant.
func (r *assuranceGlobalResource) detachFromAllSites(ctx context.Context, id int64) diag.Diagnostics {
	var diags diag.Diagnostics

	// The update endpoint is a full replace, not a patch (it separately rejects a request
	// missing Name: "attempting to modify an assurance without providing its name"), so the
	// current config must be read back and modified in place rather than PUT with only
	// SiteListId set.
	current, found, diags2 := r.readByID(ctx, id)
	diags.Append(diags2...)
	if diags.HasError() {
		return diags
	}
	if !found {
		return diags
	}

	lists, httpResp, err := r.pd.api.DefaultAPI.V1GlobalSiteListsGet(ctx).Authorization(r.pd.token).Execute()
	closeBody(httpResp)
	if err != nil {
		diags.AddError("Unable to detach assurance config from all sites", "listing site lists: "+apiErrorDetail(err))
		return diags
	}

	var siteListID *int64
	if lists != nil {
		for _, e := range lists.Entries {
			if e.Id == nil {
				continue
			}
			details, detailsResp, err := r.pd.api.DefaultAPI.V1GlobalSiteListsIdGet(ctx, *e.Id).Authorization(r.pd.token).Execute()
			closeBody(detailsResp)
			if err != nil {
				diags.AddError("Unable to detach assurance config from all sites", "reading site list: "+apiErrorDetail(err))
				return diags
			}
			if details != nil && len(details.Entries) > 0 {
				siteListID = e.Id
				break
			}
		}
	}
	if siteListID == nil {
		diags.AddError(
			"Unable to delete assurance config",
			"this config is still scoped to use_all_sites=true, and the API refuses to delete it in that state "+
				"(\"must detach ... from all sites prior to deletion\"). No existing site list with at least one "+
				"site was found in this tenant to repoint it at automatically. Create a graphiant_site_list with "+
				"a real site entry, set this resource's site_list_id to it and use_all_sites to false, apply, then "+
				"destroy again.",
		)
		return diags
	}

	current.SiteListId = siteListID
	current.UseAllSites = nil

	body := sdk.V1DataAssuranceAssurancesGlobalIdPutRequest{Config: current}
	_, putResp, err := r.pd.api.DefaultAPI.V1DataAssuranceAssurancesGlobalIdPut(ctx, id).
		Authorization(r.pd.token).
		V1DataAssuranceAssurancesGlobalIdPutRequest(body).
		Execute()
	closeBody(putResp)
	if err != nil {
		diags.AddError("Unable to detach assurance config from all sites", apiErrorDetail(err))
	}
	return diags
}

func (r *assuranceGlobalResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
