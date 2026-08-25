package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	graphiant "github.com/Graphiant-Inc/graphiant-sdk-go"
	"github.com/Graphiant-Inc/terraform-provider-graphiant/internal/provider/generated/datasource_site_list"
)

var (
	_ datasource.DataSource              = &siteListDataSource{}
	_ datasource.DataSourceWithConfigure = &siteListDataSource{}
)

func NewSiteListDataSource() datasource.DataSource {
	return &siteListDataSource{}
}

type siteListDataSource struct {
	client *gClient
}

type siteListDataSourceModel struct {
	Id                 types.Int64          `tfsdk:"id"`
	Name               types.String         `tfsdk:"name"`
	Description        types.String         `tfsdk:"description"`
	Entries            []siteListEntryModel `tfsdk:"entries"`
	EdgeReferences     types.Int64          `tfsdk:"edge_references"`
	PolicyReferences   types.Int64          `tfsdk:"policy_references"`
	SiteListReferences types.Int64          `tfsdk:"site_list_references"`
	CreatedAt          types.String         `tfsdk:"created_at"`
}

// flattenSiteListDataSource fills m the same way siteListResource.flatten
// does: summary (name/description/counts) from the list endpoint, entries
// from the id-specific endpoint. See site_list_resource.go for why both
// calls are needed.
func flattenSiteListDataSource(summary *graphiant.V1GlobalSiteListsGetResponseEntry, entries []graphiant.ManaV2SiteListEntry, m *siteListDataSourceModel) {
	m.Id = int64Value(summary.Id)
	m.Name = strValue(summary.Name)
	m.Description = strValue(summary.Description)
	m.EdgeReferences = int32Value(summary.EdgeReferences)
	m.PolicyReferences = int32Value(summary.PolicyReferences)
	m.SiteListReferences = int32Value(summary.SiteListReferences)
	m.CreatedAt = timestampValue(summary.CreatedAt)
	m.Entries = flattenSiteListEntries(entries)
}

func (d *siteListDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_site_list"
}

// Schema is generated from the OpenAPI spec (see api/generator_config.yml,
// data_sources.site_list) via `make generate-schemas`; name/edge_references/
// policy_references/site_list_references/created_at are appended by hand
// since they're only available from the list endpoint (see Read below), not
// the site-list-by-id response (entries only) the generated schema is
// derived from.
func (d *siteListDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = datasource_site_list.SiteListDataSourceSchema(ctx)
	resp.Schema.Description = "Looks up a single Graphiant global site list by ID."
	resp.Schema.Attributes["id"] = schema.Int64Attribute{Required: true, Description: "Site list identifier."}
	resp.Schema.Attributes["name"] = schema.StringAttribute{Computed: true, Description: "Site list name."}
	resp.Schema.Attributes["edge_references"] = schema.Int64Attribute{Computed: true, Description: "Number of edge devices referencing this site list."}
	resp.Schema.Attributes["policy_references"] = schema.Int64Attribute{Computed: true, Description: "Number of policies referencing this site list."}
	resp.Schema.Attributes["site_list_references"] = schema.Int64Attribute{Computed: true, Description: "Number of other site lists referencing this site list."}
	resp.Schema.Attributes["created_at"] = schema.StringAttribute{Computed: true, Description: "Creation timestamp (RFC3339, UTC)."}
}

func (d *siteListDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*gClient)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Data Source Configure Type", fmt.Sprintf("expected *gClient, got: %T", req.ProviderData))
		return
	}
	d.client = client
}

func (d *siteListDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config siteListDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	wantId := config.Id.ValueInt64()
	tflog.Debug(ctx, "reading site list data source", map[string]any{"id": wantId})

	listOut, httpRes, err := d.client.api.DefaultAPI.V1GlobalSiteListsGet(ctx).Authorization(d.client.authHeader()).Execute()
	closeBody(httpRes)
	if err != nil {
		resp.Diagnostics.AddError("Error reading site list", apiErrorDetail(err))
		return
	}

	var summary *graphiant.V1GlobalSiteListsGetResponseEntry
	if listOut != nil {
		for _, s := range listOut.GetEntries() {
			if s.GetId() == wantId {
				summary = &s
				break
			}
		}
	}
	if summary == nil {
		resp.Diagnostics.AddError("Site list not found", fmt.Sprintf("no site list with id %d was found", wantId))
		return
	}

	idOut, httpRes, err := d.client.api.DefaultAPI.V1GlobalSiteListsIdGet(ctx, wantId).Authorization(d.client.authHeader()).Execute()
	closeBody(httpRes)
	if err != nil {
		resp.Diagnostics.AddError("Error reading site list entries", apiErrorDetail(err))
		return
	}
	var entries []graphiant.ManaV2SiteListEntry
	if idOut != nil {
		entries = idOut.GetEntries()
	}

	flattenSiteListDataSource(summary, entries, &config)
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
