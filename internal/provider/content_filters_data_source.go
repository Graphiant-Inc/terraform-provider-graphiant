package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Graphiant-Inc/terraform-provider-graphiant/internal/provider/generated/datasource_content_filters"
)

var (
	_ datasource.DataSource              = &contentFiltersDataSource{}
	_ datasource.DataSourceWithConfigure = &contentFiltersDataSource{}
)

func NewContentFiltersDataSource() datasource.DataSource {
	return &contentFiltersDataSource{}
}

type contentFiltersDataSource struct {
	client *gClient
}

// contentFiltersEntryModel mirrors v1GlobalContentFiltersGetResponseRow.
// Unlike graphiant_content_filter, the list endpoint resolves rules/scope to
// display names (domain category name, LAN/site names) rather than the IDs
// used by the create/update config — use the singular graphiant_content_filter
// data source if you need the ID-based config for a specific filter.
type contentFiltersEntryModel struct {
	Id        types.Int64                    `tfsdk:"id"`
	Name      types.String                   `tfsdk:"name"`
	LanNames  types.List                     `tfsdk:"lan_names"`
	SiteNames types.List                     `tfsdk:"site_names"`
	Rules     []contentFiltersEntryRuleModel `tfsdk:"rules"`
	CreatedAt types.String                   `tfsdk:"created_at"`
	UpdatedAt types.String                   `tfsdk:"updated_at"`
}

type contentFiltersEntryRuleModel struct {
	DomainCategory     types.String `tfsdk:"domain_category"`
	ExceptionWildcards types.List   `tfsdk:"exception_wildcards"`
}

type contentFiltersDataSourceModel struct {
	ContentFilters []contentFiltersEntryModel `tfsdk:"content_filters"`
}

func (d *contentFiltersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_content_filters"
}

// Schema is generated from the OpenAPI spec (see api/generator_config.yml,
// data_sources.content_filters) via `make generate-schemas`. The raw list
// endpoint nests lan/site names one level deeper than this data source
// exposes them (lans: [{lan_name}], sites: [{site_name}]) and uses
// global_content_filter_id/_name instead of id/name; api/patch_ir.py
// reshapes the generated IR to keep this data source's existing attribute
// names rather than force a breaking rename on every user of it.
func (d *contentFiltersDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = datasource_content_filters.ContentFiltersDataSourceSchema(ctx)
	resp.Schema.Description = "Lists all Graphiant global content filters, with rules/scope resolved to display names. Use graphiant_content_filter for a specific filter's ID-based config."
}

func (d *contentFiltersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

// Read intentionally does not call V1GlobalContentFiltersGlobalContentFilterIdGet
// per filter (that would be N+1 requests for what is meant to be a
// lightweight overview); use the singular graphiant_content_filter data
// source for a filter's full ID-based config.
func (d *contentFiltersDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Debug(ctx, "reading content filters data source")

	out, httpRes, err := d.client.api.DefaultAPI.V1GlobalContentFiltersGet(ctx).Authorization(d.client.authHeader()).Execute()
	closeBody(httpRes)
	if err != nil {
		resp.Diagnostics.AddError("Error reading content filters", apiErrorDetail(err))
		return
	}

	var state contentFiltersDataSourceModel
	if out != nil {
		for _, row := range out.GetRows() {
			m := contentFiltersEntryModel{
				Id:        int64Value(row.GlobalContentFilterId),
				Name:      strValue(row.GlobalContentFilterName),
				CreatedAt: timestampValue(row.CreatedAt),
				UpdatedAt: timestampValue(row.UpdatedAt),
			}

			lanNames := make([]string, 0, len(row.Lans))
			for _, lan := range row.GetLans() {
				if lan.LanName != nil {
					lanNames = append(lanNames, *lan.LanName)
				}
			}
			list, d := stringListValue(ctx, lanNames)
			resp.Diagnostics.Append(d...)
			m.LanNames = list

			siteNames := make([]string, 0, len(row.Sites))
			for _, site := range row.GetSites() {
				if site.SiteName != nil {
					siteNames = append(siteNames, *site.SiteName)
				}
			}
			list, d = stringListValue(ctx, siteNames)
			resp.Diagnostics.Append(d...)
			m.SiteNames = list

			for _, rule := range row.GetRules() {
				wildcards, d := stringListValue(ctx, rule.ExceptionWildcards)
				resp.Diagnostics.Append(d...)
				m.Rules = append(m.Rules, contentFiltersEntryRuleModel{
					DomainCategory:     strValue(rule.DomainCategory),
					ExceptionWildcards: wildcards,
				})
			}

			state.ContentFilters = append(state.ContentFilters, m)
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
