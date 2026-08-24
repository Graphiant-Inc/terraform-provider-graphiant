package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	graphiant "github.com/Graphiant-Inc/graphiant-sdk-go"
	"github.com/Graphiant-Inc/terraform-provider-graphiant/internal/provider/generated/resource_site_list"
)

var (
	_ resource.Resource                = &siteListResource{}
	_ resource.ResourceWithConfigure   = &siteListResource{}
	_ resource.ResourceWithImportState = &siteListResource{}
)

func NewSiteListResource() resource.Resource {
	return &siteListResource{}
}

type siteListResource struct {
	client *gClient
}

// routeTagIdModel mirrors manaV2RouteTagId, a hierarchical route-tag
// reference (as opposed to a plain site ID) that a site list entry can point
// at instead of a site.
type routeTagIdModel struct {
	LevelZero types.Int64 `tfsdk:"level_zero"`
	LevelOne  types.Int64 `tfsdk:"level_one"`
	LevelTwo  types.Int64 `tfsdk:"level_two"`
}

// siteListEntryModel mirrors manaV2SiteListEntry. Exactly one of SiteId
// ("regular" in the API) or Tag should be set per entry; the API is the
// source of truth for that constraint and returns a clear validation error
// if it's violated, so it isn't re-implemented as a schema-level validator.
type siteListEntryModel struct {
	SiteId types.Int64      `tfsdk:"site_id"`
	Tag    *routeTagIdModel `tfsdk:"tag"`
}

// siteListResourceModel mirrors the site list object exposed by
// v1/global/site-lists together with the subset of fields settable via
// v1GlobalSiteListsPostRequest / v1GlobalSiteListsIdPutRequest.
type siteListResourceModel struct {
	Id                 types.Int64          `tfsdk:"id"`
	Name               types.String         `tfsdk:"name"`
	Description        types.String         `tfsdk:"description"`
	Entries            []siteListEntryModel `tfsdk:"entries"`
	EdgeReferences     types.Int64          `tfsdk:"edge_references"`
	PolicyReferences   types.Int64          `tfsdk:"policy_references"`
	SiteListReferences types.Int64          `tfsdk:"site_list_references"`
	CreatedAt          types.String         `tfsdk:"created_at"`
}

func (r *siteListResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_site_list"
}

// Schema is generated from the OpenAPI spec (see api/generator_config.yml,
// resources.site_list) via `make generate-schemas`; edge_references/
// policy_references/site_list_references/created_at are appended by hand
// since they're server-derived counters/timestamps only available from the
// list endpoint (see readInto), not the site-list-by-id response (entries
// only) the generated schema is derived from.
func (r *siteListResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_site_list.SiteListResourceSchema(ctx)
	resp.Schema.Description = "Manages a Graphiant global site list: a named, reusable group of sites (or route tags) referenced by other global config such as content filters."
	resp.Schema.Attributes["id"] = schema.Int64Attribute{
		Computed:    true,
		Description: "Site list identifier assigned by the Graphiant controller.",
		PlanModifiers: []planmodifier.Int64{
			int64planmodifier.UseStateForUnknown(),
		},
	}
	resp.Schema.Attributes["name"] = schema.StringAttribute{
		Required:    true,
		Description: "Site list name. The API has no rename endpoint, so changing this replaces the site list.",
		Validators: []validator.String{
			stringvalidator.LengthAtLeast(1),
		},
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.RequiresReplace(),
		},
	}
	// edge_references/policy_references/site_list_references/created_at are
	// all server-derived and don't change as a side effect of updating the
	// fields above, so UseStateForUnknown keeps them out of the plan diff
	// when nothing relevant changed.
	resp.Schema.Attributes["edge_references"] = schema.Int64Attribute{
		Computed:    true,
		Description: "Number of edge devices referencing this site list.",
		PlanModifiers: []planmodifier.Int64{
			int64planmodifier.UseStateForUnknown(),
		},
	}
	resp.Schema.Attributes["policy_references"] = schema.Int64Attribute{
		Computed:    true,
		Description: "Number of policies referencing this site list.",
		PlanModifiers: []planmodifier.Int64{
			int64planmodifier.UseStateForUnknown(),
		},
	}
	resp.Schema.Attributes["site_list_references"] = schema.Int64Attribute{
		Computed:    true,
		Description: "Number of other site lists referencing this site list.",
		PlanModifiers: []planmodifier.Int64{
			int64planmodifier.UseStateForUnknown(),
		},
	}
	resp.Schema.Attributes["created_at"] = schema.StringAttribute{
		Computed:    true,
		Description: "Creation timestamp (RFC3339, UTC).",
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
		},
	}
}

func (r *siteListResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*gClient)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("expected *gClient, got: %T", req.ProviderData))
		return
	}
	r.client = client
}

func expandRouteTagId(m *routeTagIdModel) *graphiant.ManaV2RouteTagId {
	if m == nil {
		return nil
	}
	tag := graphiant.NewManaV2RouteTagIdWithDefaults()
	if v := int64Ptr(m.LevelZero); v != nil {
		tag.SetLevelZero(*v)
	}
	if v := int64Ptr(m.LevelOne); v != nil {
		tag.SetLevelOne(*v)
	}
	if v := int64Ptr(m.LevelTwo); v != nil {
		tag.SetLevelTwo(*v)
	}
	return tag
}

func flattenRouteTagId(tag *graphiant.ManaV2RouteTagId) *routeTagIdModel {
	if tag == nil {
		return nil
	}
	return &routeTagIdModel{
		LevelZero: int64Value(tag.LevelZero),
		LevelOne:  int64Value(tag.LevelOne),
		LevelTwo:  int64Value(tag.LevelTwo),
	}
}

func expandSiteListEntries(entries []siteListEntryModel) []graphiant.ManaV2SiteListEntry {
	out := make([]graphiant.ManaV2SiteListEntry, 0, len(entries))
	for _, e := range entries {
		entry := graphiant.NewManaV2SiteListEntryWithDefaults()
		if v := int64Ptr(e.SiteId); v != nil {
			entry.SetRegular(*v)
		}
		if e.Tag != nil {
			entry.SetTag(*expandRouteTagId(e.Tag))
		}
		out = append(out, *entry)
	}
	return out
}

func flattenSiteListEntries(entries []graphiant.ManaV2SiteListEntry) []siteListEntryModel {
	out := make([]siteListEntryModel, 0, len(entries))
	for _, e := range entries {
		out = append(out, siteListEntryModel{
			SiteId: int64Value(e.Regular),
			Tag:    flattenRouteTagId(e.Tag),
		})
	}
	return out
}

// flatten fills m from the site list's summary (from the list endpoint,
// which is the only one that returns name/description/reference counts) and
// its entries (from the id-specific endpoint, which is the only one that
// returns members) — see findSiteList/getSiteListEntries.
func (r *siteListResource) flatten(summary *graphiant.V1GlobalSiteListsGetResponseEntry, entries []graphiant.ManaV2SiteListEntry, m *siteListResourceModel) {
	m.Id = int64Value(summary.Id)
	m.Name = strValue(summary.Name)
	m.Description = strValue(summary.Description)
	m.EdgeReferences = int32Value(summary.EdgeReferences)
	m.PolicyReferences = int32Value(summary.PolicyReferences)
	m.SiteListReferences = int32Value(summary.SiteListReferences)
	m.CreatedAt = timestampValue(summary.CreatedAt)
	m.Entries = flattenSiteListEntries(entries)
}

// findSiteList looks up a site list's summary by ID. There is no get-by-id
// endpoint that returns name/description/reference counts (only entries),
// so this lists every site list and filters client-side, mirroring
// findSite/findGroup.
func (r *siteListResource) findSiteList(ctx context.Context, id int64) (*graphiant.V1GlobalSiteListsGetResponseEntry, error) {
	out, httpRes, err := r.client.api.DefaultAPI.V1GlobalSiteListsGet(ctx).Authorization(r.client.authHeader()).Execute()
	defer closeBody(httpRes)
	if err != nil {
		return nil, err
	}
	if out == nil {
		return nil, nil
	}
	for _, s := range out.GetEntries() {
		if s.GetId() == id {
			return &s, nil
		}
	}
	return nil, nil
}

// getSiteListEntries fetches a site list's member entries, which are only
// available from the id-specific endpoint (the list endpoint returns
// reference counts instead).
func (r *siteListResource) getSiteListEntries(ctx context.Context, id int64) ([]graphiant.ManaV2SiteListEntry, error) {
	out, httpRes, err := r.client.api.DefaultAPI.V1GlobalSiteListsIdGet(ctx, id).Authorization(r.client.authHeader()).Execute()
	defer closeBody(httpRes)
	if err != nil {
		return nil, err
	}
	if out == nil {
		return nil, nil
	}
	return out.GetEntries(), nil
}

func (r *siteListResource) readInto(ctx context.Context, id int64, m *siteListResourceModel) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	summary, err := r.findSiteList(ctx, id)
	if err != nil {
		diags.AddError("Error reading site list", apiErrorDetail(err))
		return false, diags
	}
	if summary == nil {
		return false, diags
	}

	entries, err := r.getSiteListEntries(ctx, id)
	if err != nil {
		diags.AddError("Error reading site list entries", apiErrorDetail(err))
		return false, diags
	}

	r.flatten(summary, entries, m)
	return true, diags
}

func (r *siteListResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan siteListResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "creating site list", map[string]any{"name": plan.Name.ValueString()})

	body := graphiant.NewV1GlobalSiteListsPostRequestWithDefaults()
	if v := strPtr(plan.Name); v != nil {
		body.SetName(*v)
	}
	if v := strPtr(plan.Description); v != nil {
		body.SetDescription(*v)
	}
	body.SetEntries(expandSiteListEntries(plan.Entries))

	out, httpRes, err := r.client.api.DefaultAPI.V1GlobalSiteListsPost(ctx).Authorization(r.client.authHeader()).V1GlobalSiteListsPostRequest(*body).Execute()
	defer closeBody(httpRes)
	if err != nil {
		resp.Diagnostics.AddError("Error creating site list", apiErrorDetail(err))
		return
	}
	if out == nil || out.Id == nil {
		resp.Diagnostics.AddError("Error creating site list", "the API did not return the created site list's id")
		return
	}

	found, diags := r.readInto(ctx, *out.Id, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.Diagnostics.AddError("Error creating site list", "the site list was created but could not be found afterwards")
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	tflog.Trace(ctx, "created site list", map[string]any{"id": plan.Id.ValueInt64()})
}

func (r *siteListResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state siteListResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "reading site list", map[string]any{"id": state.Id.ValueInt64()})

	found, diags := r.readInto(ctx, state.Id.ValueInt64(), &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		tflog.Debug(ctx, "site list no longer exists, removing from state", map[string]any{"id": state.Id.ValueInt64()})
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *siteListResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state siteListResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "updating site list", map[string]any{"id": state.Id.ValueInt64()})

	// v1GlobalSiteListsIdPutRequest has no name field: the API doesn't
	// support renaming, which is why name has RequiresReplace in the schema.
	body := graphiant.NewV1GlobalSiteListsIdPutRequestWithDefaults()
	if v := strPtr(plan.Description); v != nil {
		body.SetDescription(*v)
	}
	body.SetEntries(expandSiteListEntries(plan.Entries))

	_, httpRes, err := r.client.api.DefaultAPI.V1GlobalSiteListsIdPut(ctx, state.Id.ValueInt64()).Authorization(r.client.authHeader()).V1GlobalSiteListsIdPutRequest(*body).Execute()
	defer closeBody(httpRes)
	if err != nil {
		resp.Diagnostics.AddError("Error updating site list", apiErrorDetail(err))
		return
	}

	found, diags := r.readInto(ctx, state.Id.ValueInt64(), &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.Diagnostics.AddError("Error updating site list", "the site list was updated but could not be found afterwards")
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	tflog.Trace(ctx, "updated site list", map[string]any{"id": plan.Id.ValueInt64()})
}

func (r *siteListResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state siteListResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "deleting site list", map[string]any{"id": state.Id.ValueInt64()})

	httpRes, err := r.client.api.DefaultAPI.V1GlobalSiteListsIdDelete(ctx, state.Id.ValueInt64()).Authorization(r.client.authHeader()).Execute()
	defer closeBody(httpRes)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting site list", apiErrorDetail(err))
	}
}

func (r *siteListResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
