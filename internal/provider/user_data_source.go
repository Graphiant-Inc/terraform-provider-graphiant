package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	graphiant "github.com/Graphiant-Inc/graphiant-sdk-go"
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

func userDataSourceAttributes(idRequired bool) map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Required:    idRequired,
			Computed:    !idRequired,
			Description: "User identifier. Equal to the user's email address.",
		},
		"email":          schema.StringAttribute{Computed: true, Description: "User email address."},
		"first_name":     schema.StringAttribute{Computed: true, Description: "User's first name."},
		"last_name":      schema.StringAttribute{Computed: true, Description: "User's last name."},
		"time_zone":      schema.StringAttribute{Computed: true, Description: "User's time zone."},
		"enterprise_id":  schema.Int64Attribute{Computed: true, Description: "Enterprise the user belongs to."},
		"verified":       schema.BoolAttribute{Computed: true, Description: "Whether the user has verified their email address."},
		"mfa_factor":     schema.StringAttribute{Computed: true, Description: "The user's configured MFA factor, if any."},
		"phone_number":   schema.StringAttribute{Computed: true, Description: "User's phone number."},
		"last_active_at": schema.StringAttribute{Computed: true, Description: "Timestamp of the user's last activity (RFC3339, UTC)."},
	}
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

func (d *userDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up a single Graphiant user by ID (email).",
		Attributes:  userDataSourceAttributes(true),
	}
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
