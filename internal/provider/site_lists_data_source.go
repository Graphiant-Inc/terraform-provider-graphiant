package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Graphiant-Inc/terraform-provider-graphiant/internal/provider/generated/datasource_site_lists"
)

var (
	_ datasource.DataSource              = &siteListsDataSource{}
	_ datasource.DataSourceWithConfigure = &siteListsDataSource{}
)

func NewSiteListsDataSource() datasource.DataSource {
	return &siteListsDataSource{}
}

type siteListsDataSource struct {
	client *gClient
}

// siteListSummaryModel mirrors v1GlobalSiteListsGetResponseEntry: summary
// fields only. Unlike graphiant_site_list, this data source's underlying
// endpoint doesn't return a list's member entries (that would require an
// id-specific request per site list) — use the singular graphiant_site_list
// data source for a list's entries.
type siteListSummaryModel struct {
	Id                 types.Int64  `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	Description        types.String `tfsdk:"description"`
	EdgeReferences     types.Int64  `tfsdk:"edge_references"`
	PolicyReferences   types.Int64  `tfsdk:"policy_references"`
	SiteListReferences types.Int64  `tfsdk:"site_list_references"`
	CreatedAt          types.String `tfsdk:"created_at"`
}

type siteListsDataSourceModel struct {
	SiteLists []siteListSummaryModel `tfsdk:"site_lists"`
}

func (d *siteListsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_site_lists"
}

// Schema is generated from the OpenAPI spec (see api/generator_config.yml,
// data_sources.site_lists) via `make generate-schemas`.
func (d *siteListsDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = datasource_site_lists.SiteListsDataSourceSchema(ctx)
	resp.Schema.Description = "Lists all Graphiant global site lists. Unlike graphiant_site_list, this does not resolve each list's entries — only summary fields (see graphiant_site_list to read entries for a specific list)."
}

func (d *siteListsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

// Read intentionally does not call V1GlobalSiteListsIdGet per list (that
// would be N+1 requests for what is meant to be a lightweight overview); use
// the singular graphiant_site_list data source to read one list's entries.
func (d *siteListsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Debug(ctx, "reading site lists data source")

	out, httpRes, err := d.client.api.DefaultAPI.V1GlobalSiteListsGet(ctx).Authorization(d.client.authHeader()).Execute()
	closeBody(httpRes)
	if err != nil {
		resp.Diagnostics.AddError("Error reading site lists", apiErrorDetail(err))
		return
	}

	var state siteListsDataSourceModel
	if out != nil {
		for _, s := range out.GetEntries() {
			state.SiteLists = append(state.SiteLists, siteListSummaryModel{
				Id:                 int64Value(s.Id),
				Name:               strValue(s.Name),
				Description:        strValue(s.Description),
				EdgeReferences:     int32Value(s.EdgeReferences),
				PolicyReferences:   int32Value(s.PolicyReferences),
				SiteListReferences: int32Value(s.SiteListReferences),
				CreatedAt:          timestampValue(s.CreatedAt),
			})
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
