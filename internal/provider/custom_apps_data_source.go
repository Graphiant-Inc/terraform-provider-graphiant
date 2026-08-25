package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Graphiant-Inc/terraform-provider-graphiant/internal/provider/generated/datasource_custom_apps"
)

var (
	_ datasource.DataSource              = &customAppsDataSource{}
	_ datasource.DataSourceWithConfigure = &customAppsDataSource{}
)

func NewCustomAppsDataSource() datasource.DataSource {
	return &customAppsDataSource{}
}

type customAppsDataSource struct {
	client *gClient
}

type customAppsDataSourceModel struct {
	CustomApps []customAppDataSourceModel `tfsdk:"custom_apps"`
}

func (d *customAppsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_apps"
}

// Schema is generated from the OpenAPI spec (see api/generator_config.yml,
// data_sources.custom_apps) via `make generate-schemas`.
func (d *customAppsDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = datasource_custom_apps.CustomAppsDataSourceSchema(ctx)
	resp.Schema.Description = "Lists all Graphiant custom apps."
}

func (d *customAppsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *customAppsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Debug(ctx, "reading custom apps data source")

	out, httpRes, err := d.client.api.DefaultAPI.V1GlobalAppsCustomGet(ctx).Authorization(d.client.authHeader()).Execute()
	closeBody(httpRes)
	if err != nil {
		resp.Diagnostics.AddError("Error reading custom apps", apiErrorDetail(err))
		return
	}

	var state customAppsDataSourceModel
	if out != nil {
		for _, entry := range out.GetEntries() {
			var m customAppDataSourceModel
			resp.Diagnostics.Append(flattenCustomAppDataSource(ctx, &entry, &m)...)
			state.CustomApps = append(state.CustomApps, m)
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
