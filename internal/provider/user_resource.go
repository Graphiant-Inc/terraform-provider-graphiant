package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	graphiant "github.com/Graphiant-Inc/graphiant-sdk-go"
)

var (
	_ resource.Resource                = &userResource{}
	_ resource.ResourceWithConfigure   = &userResource{}
	_ resource.ResourceWithImportState = &userResource{}
)

func NewUserResource() resource.Resource {
	return &userResource{}
}

type userResource struct {
	client *gClient
}

// userResourceModel mirrors commonUser together with the subset of fields
// that can be set via v1UsersPutRequest. The user's email doubles as its
// identifier (userId) in this API, so it is used as the resource ID.
type userResourceModel struct {
	Id           types.String `tfsdk:"id"`
	Email        types.String `tfsdk:"email"`
	FirstName    types.String `tfsdk:"first_name"`
	LastName     types.String `tfsdk:"last_name"`
	GroupId      types.String `tfsdk:"group_id"`
	TimeZone     types.String `tfsdk:"time_zone"`
	EnterpriseId types.Int64  `tfsdk:"enterprise_id"`
	Verified     types.Bool   `tfsdk:"verified"`
	MfaFactor    types.String `tfsdk:"mfa_factor"`
	PhoneNumber  types.String `tfsdk:"phone_number"`
	LastActiveAt types.String `tfsdk:"last_active_at"`
}

func (r *userResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (r *userResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Graphiant IAM user.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "User identifier. Equal to the user's email address.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"email": schema.StringAttribute{
				Required:    true,
				Description: "User email address. Identifies the user and cannot be changed after creation.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"first_name": schema.StringAttribute{
				Required:    true,
				Description: "User's first name.",
			},
			"last_name": schema.StringAttribute{
				Required:    true,
				Description: "User's last name.",
			},
			"group_id": schema.StringAttribute{
				Optional:    true,
				Description: "ID of the IAM group this user is assigned to.",
			},
			"time_zone": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "User's time zone (e.g. \"America/Los_Angeles\").",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"enterprise_id": schema.Int64Attribute{
				Computed:    true,
				Description: "Enterprise the user belongs to.",
			},
			"verified": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the user has verified their email address.",
			},
			"mfa_factor": schema.StringAttribute{
				Computed:    true,
				Description: "The user's configured MFA factor, if any.",
			},
			"phone_number": schema.StringAttribute{
				Computed:    true,
				Description: "User's phone number.",
			},
			"last_active_at": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp of the user's last activity (RFC3339, UTC).",
			},
		},
	}
}

func (r *userResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*gClient)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("expected *gClient, got: %T", req.ProviderData))
		return
	}
	r.client = client
}

func (r *userResource) flatten(u *graphiant.CommonUser, m *userResourceModel) {
	m.Id = strValue(u.UserId)
	if u.UserId == nil {
		// The API does not always populate UserId on the user object; the
		// email is the natural identifier callers key requests on.
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

func (r *userResource) findUser(ctx context.Context, id string) (*graphiant.CommonUser, error) {
	out, httpRes, err := r.client.api.DefaultAPI.V1UsersGet(ctx).Authorization(r.client.authHeader()).Id(id).Execute()
	if httpRes != nil {
		defer httpRes.Body.Close()
	}
	if err != nil {
		return nil, err
	}
	if out == nil {
		return nil, nil
	}
	users := out.GetUsers()
	if len(users) == 0 {
		return nil, nil
	}
	return &users[0], nil
}

func (r *userResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan userResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := graphiant.NewV1UsersPutRequest(plan.Email.ValueString(), plan.FirstName.ValueString(), plan.LastName.ValueString())
	if v := strPtr(plan.GroupId); v != nil {
		body.SetGroupId(*v)
	}
	if v := strPtr(plan.TimeZone); v != nil {
		body.SetTimeZone(*v)
	}

	httpRes, err := r.client.api.DefaultAPI.V1UsersPut(ctx).Authorization(r.client.authHeader()).V1UsersPutRequest(*body).Execute()
	if httpRes != nil {
		defer httpRes.Body.Close()
	}
	if err != nil {
		resp.Diagnostics.AddError("Error creating user", err.Error())
		return
	}

	user, err := r.findUser(ctx, plan.Email.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading back created user", err.Error())
		return
	}
	if user == nil {
		resp.Diagnostics.AddError("Error creating user", "the user was created but could not be found afterwards")
		return
	}

	r.flatten(user, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *userResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state userResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	user, err := r.findUser(ctx, state.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading user", err.Error())
		return
	}
	if user == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	r.flatten(user, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update re-issues the same PUT used for creation: v1UsersPut is keyed by
// email, so calling it again with the same email replaces the mutable
// fields (first/last name, group, time zone) on the existing user.
func (r *userResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan userResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := graphiant.NewV1UsersPutRequest(plan.Email.ValueString(), plan.FirstName.ValueString(), plan.LastName.ValueString())
	if v := strPtr(plan.GroupId); v != nil {
		body.SetGroupId(*v)
	}
	if v := strPtr(plan.TimeZone); v != nil {
		body.SetTimeZone(*v)
	}

	httpRes, err := r.client.api.DefaultAPI.V1UsersPut(ctx).Authorization(r.client.authHeader()).V1UsersPutRequest(*body).Execute()
	if httpRes != nil {
		defer httpRes.Body.Close()
	}
	if err != nil {
		resp.Diagnostics.AddError("Error updating user", err.Error())
		return
	}

	user, err := r.findUser(ctx, plan.Email.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading back updated user", err.Error())
		return
	}
	if user == nil {
		resp.Diagnostics.AddError("Error updating user", "the user was updated but could not be found afterwards")
		return
	}

	r.flatten(user, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *userResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state userResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	httpRes, err := r.client.api.DefaultAPI.V1UsersIdDelete(ctx, state.Id.ValueString()).Authorization(r.client.authHeader()).Execute()
	if httpRes != nil {
		defer httpRes.Body.Close()
	}
	if err != nil {
		resp.Diagnostics.AddError("Error deleting user", err.Error())
	}
}

func (r *userResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
