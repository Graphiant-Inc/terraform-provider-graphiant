package provider

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	graphiant "github.com/Graphiant-Inc/graphiant-sdk-go"
)

var (
	_ resource.Resource                = &userResource{}
	_ resource.ResourceWithConfigure   = &userResource{}
	_ resource.ResourceWithImportState = &userResource{}
)

// emailRegexp is a deliberately permissive "does this look like an email"
// check — it's meant to catch obvious typos client-side, not to be a
// complete RFC 5322 validator (the API is the source of truth for that).
var emailRegexp = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

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
				Validators: []validator.String{
					stringvalidator.RegexMatches(emailRegexp, "must look like an email address"),
				},
			},
			"first_name": schema.StringAttribute{
				Required:    true,
				Description: "User's first name.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"last_name": schema.StringAttribute{
				Required:    true,
				Description: "User's last name.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
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
			// enterprise_id/verified/mfa_factor/phone_number/last_active_at
			// are all server-derived and aren't touched by this resource's
			// Update (which only sends first/last name, group, and time
			// zone), so UseStateForUnknown keeps them out of the plan diff
			// when nothing relevant changed.
			"enterprise_id": schema.Int64Attribute{
				Computed:    true,
				Description: "Enterprise the user belongs to.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"verified": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the user has verified their email address.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"mfa_factor": schema.StringAttribute{
				Computed:    true,
				Description: "The user's configured MFA factor, if any.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"phone_number": schema.StringAttribute{
				Computed:    true,
				Description: "User's phone number.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"last_active_at": schema.StringAttribute{
				Computed:    true,
				Description: "Timestamp of the user's last activity (RFC3339, UTC).",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
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
	defer closeBody(httpRes)
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

	tflog.Debug(ctx, "creating user", map[string]any{"email": plan.Email.ValueString()})

	body := graphiant.NewV1UsersPutRequest(plan.Email.ValueString(), plan.FirstName.ValueString(), plan.LastName.ValueString())
	if v := strPtr(plan.GroupId); v != nil {
		body.SetGroupId(*v)
	}
	if v := strPtr(plan.TimeZone); v != nil {
		body.SetTimeZone(*v)
	}

	httpRes, err := r.client.api.DefaultAPI.V1UsersPut(ctx).Authorization(r.client.authHeader()).V1UsersPutRequest(*body).Execute()
	defer closeBody(httpRes)
	if err != nil {
		resp.Diagnostics.AddError("Error creating user", apiErrorDetail(err))
		return
	}

	user, err := r.findUser(ctx, plan.Email.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading back created user", apiErrorDetail(err))
		return
	}
	if user == nil {
		resp.Diagnostics.AddError("Error creating user", "the user was created but could not be found afterwards")
		return
	}

	r.flatten(user, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	tflog.Trace(ctx, "created user", map[string]any{"id": plan.Id.ValueString()})
}

func (r *userResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state userResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "reading user", map[string]any{"id": state.Id.ValueString()})

	user, err := r.findUser(ctx, state.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading user", apiErrorDetail(err))
		return
	}
	if user == nil {
		tflog.Debug(ctx, "user no longer exists, removing from state", map[string]any{"id": state.Id.ValueString()})
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

	tflog.Debug(ctx, "updating user", map[string]any{"email": plan.Email.ValueString()})

	body := graphiant.NewV1UsersPutRequest(plan.Email.ValueString(), plan.FirstName.ValueString(), plan.LastName.ValueString())
	if v := strPtr(plan.GroupId); v != nil {
		body.SetGroupId(*v)
	}
	if v := strPtr(plan.TimeZone); v != nil {
		body.SetTimeZone(*v)
	}

	httpRes, err := r.client.api.DefaultAPI.V1UsersPut(ctx).Authorization(r.client.authHeader()).V1UsersPutRequest(*body).Execute()
	defer closeBody(httpRes)
	if err != nil {
		resp.Diagnostics.AddError("Error updating user", apiErrorDetail(err))
		return
	}

	user, err := r.findUser(ctx, plan.Email.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading back updated user", apiErrorDetail(err))
		return
	}
	if user == nil {
		resp.Diagnostics.AddError("Error updating user", "the user was updated but could not be found afterwards")
		return
	}

	r.flatten(user, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	tflog.Trace(ctx, "updated user", map[string]any{"id": plan.Id.ValueString()})
}

func (r *userResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state userResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "deleting user", map[string]any{"id": state.Id.ValueString()})

	httpRes, err := r.client.api.DefaultAPI.V1UsersIdDelete(ctx, state.Id.ValueString()).Authorization(r.client.authHeader()).Execute()
	defer closeBody(httpRes)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting user", apiErrorDetail(err))
	}
}

func (r *userResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
