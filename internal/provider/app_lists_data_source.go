package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Graphiant-Inc/terraform-provider-graphiant/internal/provider/generated/datasource_app_lists"
)

var (
	_ datasource.DataSource              = &appListsDataSource{}
	_ datasource.DataSourceWithConfigure = &appListsDataSource{}
)

func NewAppListsDataSource() datasource.DataSource {
	return &appListsDataSource{}
}

type appListsDataSource struct {
	client *gClient
}

// appListsEntryModel mirrors v1GlobalAppsAppListsGetResponseEntry. Unlike
// graphiant_app_list, the list endpoint doesn't resolve member apps (that
// would require an id-specific request per app list) — use the singular
// graphiant_app_list data source for a list's members.
type appListsEntryModel struct {
	Id                   types.Int64  `tfsdk:"id"`
	Name                 types.String `tfsdk:"name"`
	Description          types.String `tfsdk:"description"`
	AppCount             types.Int64  `tfsdk:"app_count"`
	PolicyReferenceCount types.Int64  `tfsdk:"policy_reference_count"`
}

type appListsDataSourceModel struct {
	AppLists []appListsEntryModel `tfsdk:"app_lists"`
}

func (d *appListsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app_lists"
}

// Schema is generated from the OpenAPI spec (see api/generator_config.yml,
// data_sources.app_lists) via `make generate-schemas`.
func (d *appListsDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = datasource_app_lists.AppListsDataSourceSchema(ctx)
	resp.Schema.Description = "Lists all Graphiant global app lists. Member apps aren't included — use graphiant_app_list for a specific list's members."
}

func (d *appListsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *appListsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Debug(ctx, "reading app lists data source")

	out, httpRes, err := d.client.api.DefaultAPI.V1GlobalAppsAppListsGet(ctx).Authorization(d.client.authHeader()).Execute()
	closeBody(httpRes)
	if err != nil {
		resp.Diagnostics.AddError("Error reading app lists", apiErrorDetail(err))
		return
	}

	var state appListsDataSourceModel
	if out != nil {
		for _, entry := range out.GetEntries() {
			m := appListsEntryModel{
				AppCount:             int32Value(entry.AppCount),
				PolicyReferenceCount: int32Value(entry.PolicyReferenceCount),
			}
			if entry.AppList != nil {
				m.Name = strValue(entry.AppList.Name)
				m.Description = strValue(entry.AppList.Description)
				if entry.AppList.Identifier != nil {
					m.Id = int64Value(entry.AppList.Identifier.Id)
				}
			}
			state.AppLists = append(state.AppLists, m)
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
