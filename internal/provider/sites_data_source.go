package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &sitesDataSource{}
	_ datasource.DataSourceWithConfigure = &sitesDataSource{}
)

func NewSitesDataSource() datasource.DataSource {
	return &sitesDataSource{}
}

type sitesDataSource struct {
	client *gClient
}

type sitesDataSourceModel struct {
	Sites []siteDataSourceModel `tfsdk:"sites"`
}

func (d *sitesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sites"
}

func (d *sitesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all Graphiant sites in the enterprise.",
		Attributes: map[string]schema.Attribute{
			"sites": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: siteDataSourceAttributes(false),
				},
			},
		},
	}
}

func (d *sitesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *sitesDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Debug(ctx, "reading sites data source")

	out, httpRes, err := d.client.api.DefaultAPI.V1SitesGet(ctx).Authorization(d.client.authHeader()).Execute()
	defer closeBody(httpRes)
	if err != nil {
		resp.Diagnostics.AddError("Error reading sites", apiErrorDetail(err))
		return
	}

	var state sitesDataSourceModel
	if out != nil {
		for _, s := range out.GetSites() {
			var m siteDataSourceModel
			resp.Diagnostics.Append(flattenSiteDataSource(ctx, &s, &m)...)
			state.Sites = append(state.Sites, m)
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
