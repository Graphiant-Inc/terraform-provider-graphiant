package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	graphiant "github.com/Graphiant-Inc/graphiant-sdk-go"
	"github.com/Graphiant-Inc/terraform-provider-graphiant/internal/provider/generated/datasource_user"
)

var (
	_ datasource.DataSource              = &userDataSource{}
	_ datasource.DataSourceWithConfigure = &userDataSource{}
)

func NewUserDataSource() datasource.DataSource {
	return &userDataSource{}
}

type userDataSource struct {
	client *gClient
}

type userDataSourceModel struct {
	Id           types.String `tfsdk:"id"`
	Email        types.String `tfsdk:"email"`
	FirstName    types.String `tfsdk:"first_name"`
	LastName     types.String `tfsdk:"last_name"`
	TimeZone     types.String `tfsdk:"time_zone"`
	EnterpriseId types.Int64  `tfsdk:"enterprise_id"`
	Verified     types.Bool   `tfsdk:"verified"`
	MfaFactor    types.String `tfsdk:"mfa_factor"`
	PhoneNumber  types.String `tfsdk:"phone_number"`
	LastActiveAt types.String `tfsdk:"last_active_at"`
}

func flattenUserDataSource(u *graphiant.CommonUser, m *userDataSourceModel) {
	m.Id = strValue(u.UserId)
	if u.UserId == nil {
		m.Id = strValue(u.Email)
	}
	m.Email = strValue(u.Email)
	m.FirstName = strValue(u.FirstName)
	m.LastName = strValue(u.LastName)
	m.TimeZone = strValue(u.TimeZone)
	m.EnterpriseId = int64Value(u.EnterpriseId)
	m.Verified = boolValue(u.Verified)
	m.MfaFactor = strValue(u.MfaFactor)
	m.PhoneNumber = strValue(u.PhoneNumber)
	m.LastActiveAt = timestampValue(u.LastActiveAt)
}

func (d *userDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

// Schema is generated from the OpenAPI spec (see api/generator_config.yml,
// data_sources.user) via `make generate-schemas`; "id" is appended by hand
// as a plain mirror of "email" for the same reason as the resource (see
// user_resource.go).
func (d *userDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = datasource_user.UserDataSourceSchema(ctx)
	resp.Schema.Description = "Looks up a single Graphiant user by ID (email)."
	resp.Schema.Attributes["id"] = schema.StringAttribute{Required: true, Description: "User identifier. Equal to the user's email address."}
}

func (d *userDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *userDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config userDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "reading user data source", map[string]any{"id": config.Id.ValueString()})

	out, httpRes, err := d.client.api.DefaultAPI.V1UsersGet(ctx).Authorization(d.client.authHeader()).Id(config.Id.ValueString()).Execute()
	defer closeBody(httpRes)
	if err != nil {
		resp.Diagnostics.AddError("Error reading user", apiErrorDetail(err))
		return
	}

	var users []graphiant.CommonUser
	if out != nil {
		users = out.GetUsers()
	}
	if len(users) == 0 {
		resp.Diagnostics.AddError("User not found", fmt.Sprintf("no user with id %q was found", config.Id.ValueString()))
		return
	}

	flattenUserDataSource(&users[0], &config)
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
