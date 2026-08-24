package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	graphiant "github.com/Graphiant-Inc/graphiant-sdk-go"
	"github.com/Graphiant-Inc/terraform-provider-graphiant/internal/provider/generated/datasource_content_filter"
)

var (
	_ datasource.DataSource              = &contentFilterDataSource{}
	_ datasource.DataSourceWithConfigure = &contentFilterDataSource{}
)

func NewContentFilterDataSource() datasource.DataSource {
	return &contentFilterDataSource{}
}

type contentFilterDataSource struct {
	client *gClient
}

type contentFilterDataSourceModel struct {
	Id          types.Int64              `tfsdk:"id"`
	Name        types.String             `tfsdk:"name"`
	LanNames    types.List               `tfsdk:"lan_names"`
	Rules       []contentFilterRuleModel `tfsdk:"rules"`
	SiteListId  types.Int64              `tfsdk:"site_list_id"`
	UseAllSites types.Bool               `tfsdk:"use_all_sites"`
	CreatedAt   types.String             `tfsdk:"created_at"`
	UpdatedAt   types.String             `tfsdk:"updated_at"`
}

// flattenContentFilterDataSource mirrors contentFilterResource.flatten: row
// (from the list endpoint) supplies id/timestamps, cfg (from the id-specific
// endpoint) supplies everything else. See content_filter_resource.go for why
// both calls are needed.
func flattenContentFilterDataSource(ctx context.Context, row *graphiant.V1GlobalContentFiltersGetResponseRow, cfg *graphiant.ManaV2GlobalContentFilterConfig, m *contentFilterDataSourceModel) diag.Diagnostics {
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

func (d *contentFilterDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_content_filter"
}

// Schema is generated from the OpenAPI spec (see api/generator_config.yml,
// data_sources.content_filter) via `make generate-schemas`.
func (d *contentFilterDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = datasource_content_filter.ContentFilterDataSourceSchema(ctx)
	resp.Schema.Description = "Looks up a single Graphiant global content filter by ID."
	resp.Schema.Attributes["id"] = schema.Int64Attribute{Required: true, Description: "Content filter identifier."}
}

func (d *contentFilterDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *contentFilterDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config contentFilterDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	wantId := config.Id.ValueInt64()
	tflog.Debug(ctx, "reading content filter data source", map[string]any{"id": wantId})

	listOut, httpRes, err := d.client.api.DefaultAPI.V1GlobalContentFiltersGet(ctx).Authorization(d.client.authHeader()).Execute()
	closeBody(httpRes)
	if err != nil {
		resp.Diagnostics.AddError("Error reading content filter", apiErrorDetail(err))
		return
	}

	var row *graphiant.V1GlobalContentFiltersGetResponseRow
	if listOut != nil {
		for _, rowEntry := range listOut.GetRows() {
			if rowEntry.GetGlobalContentFilterId() == wantId {
				row = &rowEntry
				break
			}
		}
	}
	if row == nil {
		resp.Diagnostics.AddError("Content filter not found", fmt.Sprintf("no content filter with id %d was found", wantId))
		return
	}

	idOut, httpRes, err := d.client.api.DefaultAPI.V1GlobalContentFiltersGlobalContentFilterIdGet(ctx, wantId).Authorization(d.client.authHeader()).Execute()
	closeBody(httpRes)
	if err != nil {
		resp.Diagnostics.AddError("Error reading content filter config", apiErrorDetail(err))
		return
	}
	cfg := graphiant.NewManaV2GlobalContentFilterConfigWithDefaults()
	if idOut != nil && idOut.Config != nil {
		cfg = idOut.Config
	}

	resp.Diagnostics.Append(flattenContentFilterDataSource(ctx, row, cfg, &config)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
