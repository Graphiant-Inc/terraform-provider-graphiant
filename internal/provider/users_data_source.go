package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/Graphiant-Inc/terraform-provider-graphiant/internal/provider/generated/datasource_users"
)

var (
	_ datasource.DataSource              = &usersDataSource{}
	_ datasource.DataSourceWithConfigure = &usersDataSource{}
)

func NewUsersDataSource() datasource.DataSource {
	return &usersDataSource{}
}

type usersDataSource struct {
	client *gClient
}

type usersDataSourceModel struct {
	Users []userDataSourceModel `tfsdk:"users"`
}

func (d *usersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_users"
}

// Schema is generated from the OpenAPI spec (see api/generator_config.yml,
// data_sources.users) via `make generate-schemas`; each item's "id" is
// appended by hand as a plain mirror of "email" for the same reason as the
// resource (see user_resource.go).
func (d *usersDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = datasource_users.UsersDataSourceSchema(ctx)
	resp.Schema.Description = "Lists all Graphiant users in the enterprise."

	users := resp.Schema.Attributes["users"].(schema.ListNestedAttribute)
	users.NestedObject.Attributes["id"] = schema.StringAttribute{Computed: true, Description: "User identifier. Equal to the user's email address."}
	resp.Schema.Attributes["users"] = users
}

func (d *usersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *usersDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	tflog.Debug(ctx, "reading users data source")

	out, httpRes, err := d.client.api.DefaultAPI.V1UsersGet(ctx).Authorization(d.client.authHeader()).Execute()
	defer closeBody(httpRes)
	if err != nil {
		resp.Diagnostics.AddError("Error reading users", apiErrorDetail(err))
		return
	}

	var state usersDataSourceModel
	if out != nil {
		for _, u := range out.GetUsers() {
			var m userDataSourceModel
			flattenUserDataSource(&u, &m)
			state.Users = append(state.Users, m)
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
