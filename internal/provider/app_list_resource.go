package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	graphiant "github.com/Graphiant-Inc/graphiant-sdk-go"
	"github.com/Graphiant-Inc/terraform-provider-graphiant/internal/provider/generated/resource_app_list"
)

var (
	_ resource.Resource                = &appListResource{}
	_ resource.ResourceWithConfigure   = &appListResource{}
	_ resource.ResourceWithImportState = &appListResource{}
)

func NewAppListResource() resource.Resource {
	return &appListResource{}
}

type appListResource struct {
	client *gClient
}

// appIdentifierModel mirrors manaV2AppIdentifier: a reference to any app-like
// object (a custom app, a Graphiant catalog app, or another app list),
// discriminated by Type.
type appIdentifierModel struct {
	Id   types.Int64  `tfsdk:"id"`
	Type types.String `tfsdk:"type"`
}

// appListResourceModel mirrors manaV2AppListConfig together with the
// server-derived counters only available from the list endpoint (see
// readInto).
type appListResourceModel struct {
	Id                   types.Int64          `tfsdk:"id"`
	Name                 types.String         `tfsdk:"name"`
	Description          types.String         `tfsdk:"description"`
	Apps                 []appIdentifierModel `tfsdk:"apps"`
	AppCount             types.Int64          `tfsdk:"app_count"`
	PolicyReferenceCount types.Int64          `tfsdk:"policy_reference_count"`
}

func (r *appListResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app_list"
}

// Schema is generated from the OpenAPI spec (see api/generator_config.yml,
// resources.app_list) via `make generate-schemas`; app_count/policy_reference_count
// are appended by hand since they're server-derived counters only available
// from the list endpoint (see readInto), not the app-list-by-id response the
// generated schema is derived from.
func (r *appListResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_app_list.AppListResourceSchema(ctx)
	resp.Schema.Description = "Manages a Graphiant global app list: a named, reusable group of apps referenced by policies."
	resp.Schema.Attributes["id"] = schema.Int64Attribute{
		Computed:    true,
		Description: "App list identifier assigned by the Graphiant controller.",
		PlanModifiers: []planmodifier.Int64{
			int64planmodifier.UseStateForUnknown(),
		},
	}
	resp.Schema.Attributes["name"] = schema.StringAttribute{
		Required:    true,
		Description: "App list name.",
		Validators: []validator.String{
			stringvalidator.LengthAtLeast(1),
		},
	}
	resp.Schema.Attributes["app_count"] = schema.Int64Attribute{
		Computed:    true,
		Description: "Number of apps in this app list.",
		PlanModifiers: []planmodifier.Int64{
			int64planmodifier.UseStateForUnknown(),
		},
	}
	resp.Schema.Attributes["policy_reference_count"] = schema.Int64Attribute{
		Computed:    true,
		Description: "Number of policies referencing this app list.",
		PlanModifiers: []planmodifier.Int64{
			int64planmodifier.UseStateForUnknown(),
		},
	}
}

func (r *appListResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func expandAppIdentifiers(apps []appIdentifierModel) []graphiant.ManaV2AppIdentifier {
	out := make([]graphiant.ManaV2AppIdentifier, 0, len(apps))
	for _, a := range apps {
		id := graphiant.NewManaV2AppIdentifierWithDefaults()
		if v := int64Ptr(a.Id); v != nil {
			id.SetId(*v)
		}
		if v := strPtr(a.Type); v != nil {
			id.SetType(*v)
		}
		out = append(out, *id)
	}
	return out
}

func flattenAppIdentifiers(apps []graphiant.ManaV2AppIdentifier) []appIdentifierModel {
	out := make([]appIdentifierModel, 0, len(apps))
	for _, a := range apps {
		out = append(out, appIdentifierModel{
			Id:   int64Value(a.Id),
			Type: strValue(a.Type),
		})
	}
	return out
}

func (r *appListResource) expandConfig(m *appListResourceModel) *graphiant.ManaV2AppListConfig {
	cfg := graphiant.NewManaV2AppListConfigWithDefaults()
	if v := strPtr(m.Name); v != nil {
		cfg.SetName(*v)
	}
	if v := strPtr(m.Description); v != nil {
		cfg.SetDescription(*v)
	}
	cfg.SetApps(expandAppIdentifiers(m.Apps))
	return cfg
}

// flatten fills m from the app list's Config (from the id-specific endpoint,
// the source of truth for name/description/apps) and its reference counters
// (from the list endpoint, the only place they're exposed) — see
// getAppListConfig/findAppListCounters.
func (r *appListResource) flatten(id int64, cfg *graphiant.ManaV2AppListConfig, entry *graphiant.V1GlobalAppsAppListsGetResponseEntry, m *appListResourceModel) {
	m.Id = int64Value(&id)
	m.Name = strValue(cfg.Name)
	m.Description = strValue(cfg.Description)
	m.Apps = flattenAppIdentifiers(cfg.Apps)

	if entry != nil {
		m.AppCount = int32Value(entry.AppCount)
		m.PolicyReferenceCount = int32Value(entry.PolicyReferenceCount)
	} else {
		m.AppCount = types.Int64Null()
		m.PolicyReferenceCount = types.Int64Null()
	}
}

// getAppListConfig fetches an app list's full config (name, description,
// member apps) from the id-specific endpoint. Unlike site/group/user, this
// is a genuine get-by-id call, so a deleted app list surfaces as an HTTP 404
// here rather than as a missing entry in a list — handled by returning
// (nil, nil), the same "not found" signal a list-and-filter lookup gives.
func (r *appListResource) getAppListConfig(ctx context.Context, id int64) (*graphiant.ManaV2AppListConfig, error) {
	out, httpRes, err := r.client.api.DefaultAPI.V1GlobalAppsAppListsAppListIdGet(ctx, id).Authorization(r.client.authHeader()).Execute()
	defer closeBody(httpRes)
	if httpRes != nil && httpRes.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if out == nil {
		return nil, nil
	}
	return out.AppListConfig, nil
}

// findAppListCounters looks up an app list's list-endpoint entry by ID,
// which is the only place app_count/policy_reference_count are exposed.
// There is no dedicated "list one" filter, so this lists every app list and
// filters client-side, mirroring findSite/findGroup.
func (r *appListResource) findAppListCounters(ctx context.Context, id int64) (*graphiant.V1GlobalAppsAppListsGetResponseEntry, error) {
	out, httpRes, err := r.client.api.DefaultAPI.V1GlobalAppsAppListsGet(ctx).Authorization(r.client.authHeader()).Execute()
	defer closeBody(httpRes)
	if err != nil {
		return nil, err
	}
	if out == nil {
		return nil, nil
	}
	for _, entry := range out.GetEntries() {
		if entry.AppList != nil && entry.AppList.Identifier != nil && entry.AppList.Identifier.Id != nil && *entry.AppList.Identifier.Id == id {
			return &entry, nil
		}
	}
	return nil, nil
}

func (r *appListResource) readInto(ctx context.Context, id int64, m *appListResourceModel) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	cfg, err := r.getAppListConfig(ctx, id)
	if err != nil {
		diags.AddError("Error reading app list", apiErrorDetail(err))
		return false, diags
	}
	if cfg == nil {
		return false, diags
	}

	entry, err := r.findAppListCounters(ctx, id)
	if err != nil {
		diags.AddError("Error reading app list counters", apiErrorDetail(err))
		return false, diags
	}

	r.flatten(id, cfg, entry, m)
	return true, diags
}

func (r *appListResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan appListResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "creating app list", map[string]any{"name": plan.Name.ValueString()})

	body := graphiant.NewV1GlobalAppsAppListsPostRequestWithDefaults()
	body.SetAppListConfig(*r.expandConfig(&plan))

	out, httpRes, err := r.client.api.DefaultAPI.V1GlobalAppsAppListsPost(ctx).Authorization(r.client.authHeader()).V1GlobalAppsAppListsPostRequest(*body).Execute()
	defer closeBody(httpRes)
	if err != nil {
		resp.Diagnostics.AddError("Error creating app list", apiErrorDetail(err))
		return
	}
	if out == nil || out.AppIdentifier == nil || out.AppIdentifier.Id == nil {
		resp.Diagnostics.AddError("Error creating app list", "the API did not return the created app list's id")
		return
	}

	found, diags := r.readInto(ctx, *out.AppIdentifier.Id, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.Diagnostics.AddError("Error creating app list", "the app list was created but could not be found afterwards")
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	tflog.Trace(ctx, "created app list", map[string]any{"id": plan.Id.ValueInt64()})
}

func (r *appListResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state appListResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "reading app list", map[string]any{"id": state.Id.ValueInt64()})

	found, diags := r.readInto(ctx, state.Id.ValueInt64(), &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		tflog.Debug(ctx, "app list no longer exists, removing from state", map[string]any{"id": state.Id.ValueInt64()})
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *appListResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state appListResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "updating app list", map[string]any{"id": state.Id.ValueInt64()})

	body := graphiant.NewV1GlobalAppsAppListsAppListIdPutRequestWithDefaults()
	body.SetAppListConfig(*r.expandConfig(&plan))

	_, httpRes, err := r.client.api.DefaultAPI.V1GlobalAppsAppListsAppListIdPut(ctx, state.Id.ValueInt64()).Authorization(r.client.authHeader()).V1GlobalAppsAppListsAppListIdPutRequest(*body).Execute()
	defer closeBody(httpRes)
	if err != nil {
		resp.Diagnostics.AddError("Error updating app list", apiErrorDetail(err))
		return
	}

	found, diags := r.readInto(ctx, state.Id.ValueInt64(), &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.Diagnostics.AddError("Error updating app list", "the app list was updated but could not be found afterwards")
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	tflog.Trace(ctx, "updated app list", map[string]any{"id": plan.Id.ValueInt64()})
}

func (r *appListResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state appListResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "deleting app list", map[string]any{"id": state.Id.ValueInt64()})

	_, httpRes, err := r.client.api.DefaultAPI.V1GlobalAppsAppListsAppListIdDelete(ctx, state.Id.ValueInt64()).Authorization(r.client.authHeader()).Execute()
	defer closeBody(httpRes)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting app list", apiErrorDetail(err))
	}
}

func (r *appListResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
