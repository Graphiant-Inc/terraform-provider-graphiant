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
	"github.com/Graphiant-Inc/terraform-provider-graphiant/internal/provider/generated/datasource_custom_app"
)

var (
	_ datasource.DataSource              = &customAppDataSource{}
	_ datasource.DataSourceWithConfigure = &customAppDataSource{}
)

func NewCustomAppDataSource() datasource.DataSource {
	return &customAppDataSource{}
}

type customAppDataSource struct {
	client *gClient
}

type customAppDataSourceModel struct {
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

// flattenCustomAppDataSource mirrors customAppResource.flatten: everything
// comes from a single list-endpoint entry (App/AppConfig plus reference
// counters). See custom_app_resource.go for why no second call is needed.
func flattenCustomAppDataSource(ctx context.Context, entry *graphiant.V1GlobalAppsCustomGetResponseEntry, m *customAppDataSourceModel) diag.Diagnostics {
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

func (d *customAppDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_app"
}

// Schema is generated from the OpenAPI spec (see api/generator_config.yml,
// data_sources.custom_app) via `make generate-schemas`.
func (d *customAppDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = datasource_custom_app.CustomAppDataSourceSchema(ctx)
	resp.Schema.Description = "Looks up a single Graphiant custom app by ID."
	resp.Schema.Attributes["id"] = schema.Int64Attribute{Required: true, Description: "Custom app identifier."}
}

func (d *customAppDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *customAppDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config customAppDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	wantId := config.Id.ValueInt64()
	tflog.Debug(ctx, "reading custom app data source", map[string]any{"id": wantId})

	out, httpRes, err := d.client.api.DefaultAPI.V1GlobalAppsCustomGet(ctx).Authorization(d.client.authHeader()).Execute()
	closeBody(httpRes)
	if err != nil {
		resp.Diagnostics.AddError("Error reading custom app", apiErrorDetail(err))
		return
	}

	var found *graphiant.V1GlobalAppsCustomGetResponseEntry
	if out != nil {
		for _, entry := range out.GetEntries() {
			if entry.App != nil && entry.App.Identifier != nil && entry.App.Identifier.Id != nil && *entry.App.Identifier.Id == wantId {
				found = &entry
				break
			}
		}
	}
	if found == nil {
		resp.Diagnostics.AddError("Custom app not found", fmt.Sprintf("no custom app with id %d was found", wantId))
		return
	}

	resp.Diagnostics.Append(flattenCustomAppDataSource(ctx, found, &config)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
