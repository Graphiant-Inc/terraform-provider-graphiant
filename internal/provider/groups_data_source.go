package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Graphiant-Inc/terraform-provider-graphiant/internal/provider/generated/datasource_groups"
)

var (
	_ datasource.DataSource              = &groupsDataSource{}
	_ datasource.DataSourceWithConfigure = &groupsDataSource{}
)

func NewGroupsDataSource() datasource.DataSource {
	return &groupsDataSource{}
}

type groupsDataSource struct {
	client *gClient
}

type groupsDataSourceModel struct {
	Groups []groupDataSourceModel `tfsdk:"groups"`
}

func (d *groupsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_groups"
}

// Schema is generated from the OpenAPI spec (see api/generator_config.yml,
// data_sources.groups) via `make generate-schemas`.
func (d *groupsDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = datasource_groups.GroupsDataSourceSchema(ctx)
	resp.Schema.Description = "Lists all Graphiant IAM groups in the enterprise."
}

func (d *groupsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *groupsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Debug(ctx, "reading groups data source")

	out, httpRes, err := d.client.api.DefaultAPI.V1GroupsGet(ctx).Authorization(d.client.authHeader()).Execute()
	defer closeBody(httpRes)
	if err != nil {
		resp.Diagnostics.AddError("Error reading groups", apiErrorDetail(err))
		return
	}

	var state groupsDataSourceModel
	if out != nil {
		for _, g := range out.GetGroups() {
			var m groupDataSourceModel
			resp.Diagnostics.Append(flattenGroupDataSource(&g, &m, ctx)...)
			state.Groups = append(state.Groups, m)
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
