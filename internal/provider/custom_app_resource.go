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
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	graphiant "github.com/Graphiant-Inc/graphiant-sdk-go"
	"github.com/Graphiant-Inc/terraform-provider-graphiant/internal/provider/generated/resource_custom_app"
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
	client *gClient
}

// portRangeModel mirrors manaV2GlobalAppPortRange.
type portRangeModel struct {
	Lower types.Int64 `tfsdk:"lower"`
	Upper types.Int64 `tfsdk:"upper"`
}

// customAppResourceModel mirrors manaV2GlobalAppConfig together with the
// reference counters only available from the list endpoint (see readInto).
// Unlike site lists/content filters/app lists, the list endpoint here
// returns the full config alongside those counters, so no second id-specific
// call is needed to read this resource.
type customAppResourceModel struct {
	Id                    types.Int64      `tfsdk:"id"`
	Name                  types.String     `tfsdk:"name"`
	Description           types.String     `tfsdk:"description"`
	Url                   types.String     `tfsdk:"url"`
	IpProtocol            types.String     `tfsdk:"ip_protocol"`
	IpLists               types.List       `tfsdk:"ip_lists"`
	IpPrefixes            types.List       `tfsdk:"ip_prefixes"`
	PortRanges            []portRangeModel `tfsdk:"port_ranges"`
	AppListReferenceCount types.Int64      `tfsdk:"app_list_reference_count"`
	PolicyReferenceCount  types.Int64      `tfsdk:"policy_reference_count"`
}

func (r *customAppResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_app"
}

// Schema is generated from the OpenAPI spec (see api/generator_config.yml,
// resources.custom_app) via `make generate-schemas`; app_list_reference_count/
// policy_reference_count are appended by hand since they're server-derived
// counters, not part of the underlying app config the generated schema is
// derived from.
func (r *customAppResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_custom_app.CustomAppResourceSchema(ctx)
	resp.Schema.Description = "Manages a Graphiant custom app: a user-defined app match (by URL, IP, and/or port) referenced by app lists and policies."
	resp.Schema.Attributes["id"] = schema.Int64Attribute{
		Computed:    true,
		Description: "Custom app identifier assigned by the Graphiant controller.",
		PlanModifiers: []planmodifier.Int64{
			int64planmodifier.UseStateForUnknown(),
		},
	}
	resp.Schema.Attributes["name"] = schema.StringAttribute{
		Required:    true,
		Description: "Custom app name.",
		Validators: []validator.String{
			stringvalidator.LengthAtLeast(1),
		},
	}
	resp.Schema.Attributes["app_list_reference_count"] = schema.Int64Attribute{
		Computed:    true,
		Description: "Number of app lists referencing this custom app.",
		PlanModifiers: []planmodifier.Int64{
			int64planmodifier.UseStateForUnknown(),
		},
	}
	resp.Schema.Attributes["policy_reference_count"] = schema.Int64Attribute{
		Computed:    true,
		Description: "Number of policies referencing this custom app.",
		PlanModifiers: []planmodifier.Int64{
			int64planmodifier.UseStateForUnknown(),
		},
	}
}

func (r *customAppResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func expandPortRanges(ranges []portRangeModel) []graphiant.ManaV2GlobalAppPortRange {
	out := make([]graphiant.ManaV2GlobalAppPortRange, 0, len(ranges))
	for _, pr := range ranges {
		p := graphiant.NewManaV2GlobalAppPortRangeWithDefaults()
		if v := int64Ptr(pr.Lower); v != nil {
			lower := int32(*v)
			p.SetLower(lower)
		}
		if v := int64Ptr(pr.Upper); v != nil {
			upper := int32(*v)
			p.SetUpper(upper)
		}
		out = append(out, *p)
	}
	return out
}

func flattenPortRanges(ranges []graphiant.ManaV2GlobalAppPortRange) []portRangeModel {
	out := make([]portRangeModel, 0, len(ranges))
	for _, pr := range ranges {
		out = append(out, portRangeModel{
			Lower: int32Value(pr.Lower),
			Upper: int32Value(pr.Upper),
		})
	}
	return out
}

func (r *customAppResource) expandConfig(ctx context.Context, m *customAppResourceModel) (*graphiant.ManaV2GlobalAppConfig, diag.Diagnostics) {
	var diags diag.Diagnostics
	cfg := graphiant.NewManaV2GlobalAppConfigWithDefaults()

	if v := strPtr(m.Name); v != nil {
		cfg.SetName(*v)
	}
	if v := strPtr(m.Description); v != nil {
		cfg.SetDescription(*v)
	}
	if v := strPtr(m.Url); v != nil {
		cfg.SetUrl(*v)
	}
	if v := strPtr(m.IpProtocol); v != nil {
		cfg.SetIpProtocol(*v)
	}
	if !m.IpLists.IsNull() && !m.IpLists.IsUnknown() {
		var ipLists []string
		diags.Append(m.IpLists.ElementsAs(ctx, &ipLists, false)...)
		cfg.SetIpLists(ipLists)
	}
	if !m.IpPrefixes.IsNull() && !m.IpPrefixes.IsUnknown() {
		var ipPrefixes []string
		diags.Append(m.IpPrefixes.ElementsAs(ctx, &ipPrefixes, false)...)
		cfg.SetIpPrefixes(ipPrefixes)
	}
	cfg.SetPortRanges(expandPortRanges(m.PortRanges))

	return cfg, diags
}

// flatten fills m from a single list-endpoint entry, which uniquely among
// this provider's split-read resources already carries the full config
// (App/AppConfig) alongside the reference counters — no second id-specific
// call is needed. See findCustomAppEntry.
func (r *customAppResource) flatten(ctx context.Context, entry *graphiant.V1GlobalAppsCustomGetResponseEntry, m *customAppResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	if entry.App != nil && entry.App.Identifier != nil {
		m.Id = int64Value(entry.App.Identifier.Id)
	}
	m.AppListReferenceCount = int32Value(entry.AppListReferenceCount)
	m.PolicyReferenceCount = int32Value(entry.PolicyReferenceCount)

	cfg := entry.AppConfig
	if cfg == nil {
		cfg = graphiant.NewManaV2GlobalAppConfigWithDefaults()
	}
	m.Name = strValue(cfg.Name)
	m.Description = strValue(cfg.Description)
	m.Url = strValue(cfg.Url)
	m.IpProtocol = strValue(cfg.IpProtocol)

	ipLists, d := stringListValue(ctx, cfg.IpLists)
	diags.Append(d...)
	m.IpLists = ipLists

	ipPrefixes, d := stringListValue(ctx, cfg.IpPrefixes)
	diags.Append(d...)
	m.IpPrefixes = ipPrefixes

	m.PortRanges = flattenPortRanges(cfg.PortRanges)

	return diags
}

// findCustomAppEntry looks up a custom app's list-endpoint entry by ID.
// There is no dedicated "list one" filter, so this lists every custom app
// and filters client-side, mirroring findSite/findGroup.
func (r *customAppResource) findCustomAppEntry(ctx context.Context, id int64) (*graphiant.V1GlobalAppsCustomGetResponseEntry, error) {
	out, httpRes, err := r.client.api.DefaultAPI.V1GlobalAppsCustomGet(ctx).Authorization(r.client.authHeader()).Execute()
	defer closeBody(httpRes)
	if err != nil {
		return nil, err
	}
	if out == nil {
		return nil, nil
	}
	for _, entry := range out.GetEntries() {
		if entry.App != nil && entry.App.Identifier != nil && entry.App.Identifier.Id != nil && *entry.App.Identifier.Id == id {
			return &entry, nil
		}
	}
	return nil, nil
}

func (r *customAppResource) readInto(ctx context.Context, id int64, m *customAppResourceModel) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	entry, err := r.findCustomAppEntry(ctx, id)
	if err != nil {
		diags.AddError("Error reading custom app", apiErrorDetail(err))
		return false, diags
	}
	if entry == nil {
		return false, diags
	}

	diags.Append(r.flatten(ctx, entry, m)...)
	return true, diags
}

func (r *customAppResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan customAppResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "creating custom app", map[string]any{"name": plan.Name.ValueString()})

	cfg, diags := r.expandConfig(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := graphiant.NewV1GlobalAppsCustomPostRequestWithDefaults()
	body.SetAppConfig(*cfg)

	out, httpRes, err := r.client.api.DefaultAPI.V1GlobalAppsCustomPost(ctx).Authorization(r.client.authHeader()).V1GlobalAppsCustomPostRequest(*body).Execute()
	defer closeBody(httpRes)
	if err != nil {
		resp.Diagnostics.AddError("Error creating custom app", apiErrorDetail(err))
		return
	}
	if out == nil || out.AppIdentifier == nil || out.AppIdentifier.Id == nil {
		resp.Diagnostics.AddError("Error creating custom app", "the API did not return the created custom app's id")
		return
	}

	found, diags := r.readInto(ctx, *out.AppIdentifier.Id, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.Diagnostics.AddError("Error creating custom app", "the custom app was created but could not be found afterwards")
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	tflog.Trace(ctx, "created custom app", map[string]any{"id": plan.Id.ValueInt64()})
}

func (r *customAppResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state customAppResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "reading custom app", map[string]any{"id": state.Id.ValueInt64()})

	found, diags := r.readInto(ctx, state.Id.ValueInt64(), &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		tflog.Debug(ctx, "custom app no longer exists, removing from state", map[string]any{"id": state.Id.ValueInt64()})
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *customAppResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state customAppResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "updating custom app", map[string]any{"id": state.Id.ValueInt64()})

	cfg, diags := r.expandConfig(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := graphiant.NewV1GlobalAppsCustomAppIdPutRequestWithDefaults()
	body.SetAppConfig(*cfg)

	_, httpRes, err := r.client.api.DefaultAPI.V1GlobalAppsCustomAppIdPut(ctx, state.Id.ValueInt64()).Authorization(r.client.authHeader()).V1GlobalAppsCustomAppIdPutRequest(*body).Execute()
	defer closeBody(httpRes)
	if err != nil {
		resp.Diagnostics.AddError("Error updating custom app", apiErrorDetail(err))
		return
	}

	found, diags := r.readInto(ctx, state.Id.ValueInt64(), &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.Diagnostics.AddError("Error updating custom app", "the custom app was updated but could not be found afterwards")
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	tflog.Trace(ctx, "updated custom app", map[string]any{"id": plan.Id.ValueInt64()})
}

func (r *customAppResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state customAppResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "deleting custom app", map[string]any{"id": state.Id.ValueInt64()})

	_, httpRes, err := r.client.api.DefaultAPI.V1GlobalAppsCustomAppIdDelete(ctx, state.Id.ValueInt64()).Authorization(r.client.authHeader()).Execute()
	defer closeBody(httpRes)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting custom app", apiErrorDetail(err))
	}
}

func (r *customAppResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
