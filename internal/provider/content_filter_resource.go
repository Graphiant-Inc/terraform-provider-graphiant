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
	"github.com/Graphiant-Inc/terraform-provider-graphiant/internal/provider/generated/resource_content_filter"
)

var (
	_ resource.Resource                = &contentFilterResource{}
	_ resource.ResourceWithConfigure   = &contentFilterResource{}
	_ resource.ResourceWithImportState = &contentFilterResource{}
)

func NewContentFilterResource() resource.Resource {
	return &contentFilterResource{}
}

type contentFilterResource struct {
	client *gClient
}

// contentFilterRuleModel mirrors manaV2GlobalContentFilterRule.
type contentFilterRuleModel struct {
	DomainCategoryId   types.Int64 `tfsdk:"domain_category_id"`
	ExceptionWildcards types.List  `tfsdk:"exception_wildcards"`
}

// contentFilterResourceModel mirrors manaV2GlobalContentFilterConfig
// together with the id/timestamps surfaced by the list endpoint (the only
// one that returns them; see readInto).
type contentFilterResourceModel struct {
	Id          types.Int64              `tfsdk:"id"`
	Name        types.String             `tfsdk:"name"`
	LanNames    types.List               `tfsdk:"lan_names"`
	Rules       []contentFilterRuleModel `tfsdk:"rules"`
	SiteListId  types.Int64              `tfsdk:"site_list_id"`
	UseAllSites types.Bool               `tfsdk:"use_all_sites"`
	CreatedAt   types.String             `tfsdk:"created_at"`
	UpdatedAt   types.String             `tfsdk:"updated_at"`
}

func (r *contentFilterResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_content_filter"
}

// Schema is generated from the OpenAPI spec (see api/generator_config.yml,
// resources.content_filter) via `make generate-schemas`; created_at/updated_at
// are appended by hand since they're server-derived and only available from
// the list endpoint (see readInto), not the content-filter-by-id response
// the generated schema is derived from.
func (r *contentFilterResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_content_filter.ContentFilterResourceSchema(ctx)
	resp.Schema.Description = "Manages a Graphiant global content filter: a set of domain-category blocking rules applied to a scope of LAN segments and/or sites."
	resp.Schema.Attributes["id"] = schema.Int64Attribute{
		Computed:    true,
		Description: "Content filter identifier assigned by the Graphiant controller.",
		PlanModifiers: []planmodifier.Int64{
			int64planmodifier.UseStateForUnknown(),
		},
	}
	resp.Schema.Attributes["name"] = schema.StringAttribute{
		Required:    true,
		Description: "Content filter name.",
		Validators: []validator.String{
			stringvalidator.LengthAtLeast(1),
		},
	}
	resp.Schema.Attributes["site_list_id"] = schema.Int64Attribute{
		Optional:    true,
		Description: "Site list whose members this filter applies to. Mutually exclusive with use_all_sites.",
	}
	resp.Schema.Attributes["use_all_sites"] = schema.BoolAttribute{
		Optional:    true,
		Description: "Apply this filter to all sites in the tenant. Mutually exclusive with site_list_id; the API requires this to be true when set.",
	}
	// created_at/updated_at are server-derived. created_at never changes
	// after creation (UseStateForUnknown keeps it out of the plan diff);
	// updated_at genuinely changes on every update, so it intentionally has
	// no plan modifier and shows as unknown then.
	resp.Schema.Attributes["created_at"] = schema.StringAttribute{
		Computed:    true,
		Description: "Creation timestamp (RFC3339, UTC).",
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
		},
	}
	resp.Schema.Attributes["updated_at"] = schema.StringAttribute{Computed: true, Description: "Last update timestamp (RFC3339, UTC)."}
}

func (r *contentFilterResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func expandContentFilterRules(ctx context.Context, rules []contentFilterRuleModel) ([]graphiant.ManaV2GlobalContentFilterRule, diag.Diagnostics) {
	var diags diag.Diagnostics
	out := make([]graphiant.ManaV2GlobalContentFilterRule, 0, len(rules))
	for _, rule := range rules {
		r := graphiant.NewManaV2GlobalContentFilterRuleWithDefaults()
		if v := int64Ptr(rule.DomainCategoryId); v != nil {
			r.SetDomainCategoryId(*v)
		}
		if !rule.ExceptionWildcards.IsNull() && !rule.ExceptionWildcards.IsUnknown() {
			var wildcards []string
			diags.Append(rule.ExceptionWildcards.ElementsAs(ctx, &wildcards, false)...)
			r.SetExceptionWildcards(wildcards)
		}
		out = append(out, *r)
	}
	return out, diags
}

func flattenContentFilterRules(ctx context.Context, rules []graphiant.ManaV2GlobalContentFilterRule) ([]contentFilterRuleModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	out := make([]contentFilterRuleModel, 0, len(rules))
	for _, rule := range rules {
		wildcards, d := stringListValue(ctx, rule.ExceptionWildcards)
		diags.Append(d...)
		out = append(out, contentFilterRuleModel{
			DomainCategoryId:   int64Value(rule.DomainCategoryId),
			ExceptionWildcards: wildcards,
		})
	}
	return out, diags
}

func (r *contentFilterResource) expandConfig(ctx context.Context, m *contentFilterResourceModel) (*graphiant.ManaV2GlobalContentFilterConfig, diag.Diagnostics) {
	cfg := graphiant.NewManaV2GlobalContentFilterConfigWithDefaults()
	var diags diag.Diagnostics

	if v := strPtr(m.Name); v != nil {
		cfg.SetName(*v)
	}
	if !m.LanNames.IsNull() && !m.LanNames.IsUnknown() {
		var lanNames []string
		diags.Append(m.LanNames.ElementsAs(ctx, &lanNames, false)...)
		cfg.SetLanNames(lanNames)
	}
	rules, d := expandContentFilterRules(ctx, m.Rules)
	diags.Append(d...)
	cfg.SetRules(rules)
	if v := int64Ptr(m.SiteListId); v != nil {
		cfg.SetSiteListId(*v)
	}
	if v := boolPtr(m.UseAllSites); v != nil {
		cfg.SetUseAllSites(*v)
	}

	return cfg, diags
}

// flatten fills m from the content filter's Config (from the id-specific
// endpoint, the only one that returns the full config) and its
// created/updated timestamps (from the list endpoint, the only one that
// returns them) — see findContentFilterRow/getContentFilterConfig.
func (r *contentFilterResource) flatten(ctx context.Context, row *graphiant.V1GlobalContentFiltersGetResponseRow, cfg *graphiant.ManaV2GlobalContentFilterConfig, m *contentFilterResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	m.Id = int64Value(row.GlobalContentFilterId)
	m.CreatedAt = timestampValue(row.CreatedAt)
	m.UpdatedAt = timestampValue(row.UpdatedAt)

	m.Name = strValue(cfg.Name)
	lanNames, d := stringListValue(ctx, cfg.LanNames)
	diags.Append(d...)
	m.LanNames = lanNames

	rules, d := flattenContentFilterRules(ctx, cfg.Rules)
	diags.Append(d...)
	m.Rules = rules

	m.SiteListId = int64Value(cfg.SiteListId)
	m.UseAllSites = boolValue(cfg.UseAllSites)

	return diags
}

// findContentFilterRow looks up a content filter's list-endpoint row by ID,
// which is the only place created/updated timestamps are exposed. There is
// no dedicated "list one" filter, so this lists every content filter and
// filters client-side, mirroring findSite/findGroup.
func (r *contentFilterResource) findContentFilterRow(ctx context.Context, id int64) (*graphiant.V1GlobalContentFiltersGetResponseRow, error) {
	out, httpRes, err := r.client.api.DefaultAPI.V1GlobalContentFiltersGet(ctx).Authorization(r.client.authHeader()).Execute()
	defer closeBody(httpRes)
	if err != nil {
		return nil, err
	}
	if out == nil {
		return nil, nil
	}
	for _, row := range out.GetRows() {
		if row.GetGlobalContentFilterId() == id {
			return &row, nil
		}
	}
	return nil, nil
}

// getContentFilterConfig fetches a content filter's full config, which is
// only available from the id-specific endpoint (the list endpoint returns a
// display-oriented summary instead, e.g. rule domain category names rather
// than IDs).
func (r *contentFilterResource) getContentFilterConfig(ctx context.Context, id int64) (*graphiant.ManaV2GlobalContentFilterConfig, error) {
	out, httpRes, err := r.client.api.DefaultAPI.V1GlobalContentFiltersGlobalContentFilterIdGet(ctx, id).Authorization(r.client.authHeader()).Execute()
	defer closeBody(httpRes)
	if err != nil {
		return nil, err
	}
	if out == nil {
		return nil, nil
	}
	return out.Config, nil
}

func (r *contentFilterResource) readInto(ctx context.Context, id int64, m *contentFilterResourceModel) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	row, err := r.findContentFilterRow(ctx, id)
	if err != nil {
		diags.AddError("Error reading content filter", apiErrorDetail(err))
		return false, diags
	}
	if row == nil {
		return false, diags
	}

	cfg, err := r.getContentFilterConfig(ctx, id)
	if err != nil {
		diags.AddError("Error reading content filter config", apiErrorDetail(err))
		return false, diags
	}
	if cfg == nil {
		cfg = graphiant.NewManaV2GlobalContentFilterConfigWithDefaults()
	}

	diags.Append(r.flatten(ctx, row, cfg, m)...)
	return true, diags
}

func (r *contentFilterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan contentFilterResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "creating content filter", map[string]any{"name": plan.Name.ValueString()})

	cfg, diags := r.expandConfig(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := graphiant.NewV1GlobalContentFiltersPostRequestWithDefaults()
	body.SetConfig(*cfg)

	out, httpRes, err := r.client.api.DefaultAPI.V1GlobalContentFiltersPost(ctx).Authorization(r.client.authHeader()).V1GlobalContentFiltersPostRequest(*body).Execute()
	defer closeBody(httpRes)
	if err != nil {
		resp.Diagnostics.AddError("Error creating content filter", apiErrorDetail(err))
		return
	}
	if out == nil || out.GlobalContentFilterId == nil {
		resp.Diagnostics.AddError("Error creating content filter", "the API did not return the created content filter's id")
		return
	}

	found, diags := r.readInto(ctx, *out.GlobalContentFilterId, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.Diagnostics.AddError("Error creating content filter", "the content filter was created but could not be found afterwards")
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	tflog.Trace(ctx, "created content filter", map[string]any{"id": plan.Id.ValueInt64()})
}

func (r *contentFilterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state contentFilterResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "reading content filter", map[string]any{"id": state.Id.ValueInt64()})

	found, diags := r.readInto(ctx, state.Id.ValueInt64(), &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		tflog.Debug(ctx, "content filter no longer exists, removing from state", map[string]any{"id": state.Id.ValueInt64()})
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *contentFilterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state contentFilterResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "updating content filter", map[string]any{"id": state.Id.ValueInt64()})

	cfg, diags := r.expandConfig(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := graphiant.NewV1GlobalContentFiltersGlobalContentFilterIdPutRequestWithDefaults()
	body.SetConfig(*cfg)

	_, httpRes, err := r.client.api.DefaultAPI.V1GlobalContentFiltersGlobalContentFilterIdPut(ctx, state.Id.ValueInt64()).Authorization(r.client.authHeader()).V1GlobalContentFiltersGlobalContentFilterIdPutRequest(*body).Execute()
	defer closeBody(httpRes)
	if err != nil {
		resp.Diagnostics.AddError("Error updating content filter", apiErrorDetail(err))
		return
	}

	found, diags := r.readInto(ctx, state.Id.ValueInt64(), &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.Diagnostics.AddError("Error updating content filter", "the content filter was updated but could not be found afterwards")
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	tflog.Trace(ctx, "updated content filter", map[string]any{"id": plan.Id.ValueInt64()})
}

func (r *contentFilterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state contentFilterResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "deleting content filter", map[string]any{"id": state.Id.ValueInt64()})

	_, httpRes, err := r.client.api.DefaultAPI.V1GlobalContentFiltersGlobalContentFilterIdDelete(ctx, state.Id.ValueInt64()).Authorization(r.client.authHeader()).Execute()
	defer closeBody(httpRes)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting content filter", apiErrorDetail(err))
	}
}

func (r *contentFilterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
