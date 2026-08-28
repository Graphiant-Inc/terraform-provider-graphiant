package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	sdk "github.com/Graphiant-Inc/graphiant-sdk-go"
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
	pd *providerData
}

type userResourceModel struct {
	ID           types.String `tfsdk:"id"`
	Email        types.String `tfsdk:"email"`
	FirstName    types.String `tfsdk:"first_name"`
	LastName     types.String `tfsdk:"last_name"`
	GroupID      types.String `tfsdk:"group_id"`
	TimeZone     types.String `tfsdk:"time_zone"`
	Verified     types.Bool   `tfsdk:"verified"`
	EnterpriseID types.Int64  `tfsdk:"enterprise_id"`
	PhoneNumber  types.String `tfsdk:"phone_number"`
	MfaFactor    types.String `tfsdk:"mfa_factor"`
}

func (r *userResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (r *userResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A Graphiant user. The API has no general update endpoint for user profile fields, " +
			"so every configurable attribute forces recreation of the resource on change.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Server-assigned user ID.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"email": schema.StringAttribute{
				Required:      true,
				Description:   "User email address, used as the create/lookup key.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"first_name": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"last_name": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"group_id": schema.StringAttribute{
				Optional:      true,
				Description:   "Group to add the user to at creation time.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"time_zone": schema.StringAttribute{
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"verified": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the user has verified their account.",
			},
			"enterprise_id": schema.Int64Attribute{
				Computed: true,
			},
			"phone_number": schema.StringAttribute{
				Computed: true,
			},
			"mfa_factor": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (r *userResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.pd = configurePD(req.ProviderData, &resp.Diagnostics)
}

// applyUser copies an API-returned CommonUser back into the resource model. It
// deliberately leaves m.GroupID untouched: CommonUser doesn't report group
// membership, so whatever the model already has (from plan or prior state) stands.
func (m *userResourceModel) applyUser(u *sdk.CommonUser) {
	m.ID = types.StringPointerValue(u.UserId)
	m.Email = types.StringPointerValue(u.Email)
	m.FirstName = types.StringPointerValue(u.FirstName)
	m.LastName = types.StringPointerValue(u.LastName)
	m.TimeZone = types.StringPointerValue(u.TimeZone)
	m.Verified = types.BoolPointerValue(u.Verified)
	m.EnterpriseID = types.Int64PointerValue(u.EnterpriseId)
	m.PhoneNumber = types.StringPointerValue(u.PhoneNumber)
	m.MfaFactor = types.StringPointerValue(u.MfaFactor)
}

func (r *userResource) findUserByEmail(ctx context.Context, email string) (*sdk.CommonUser, bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	out, httpResp, err := r.pd.api.DefaultAPI.V1UsersGet(ctx).Authorization(r.pd.token).Execute()
	closeBody(httpResp)
	if err != nil {
		diags.AddError("Unable to list users", apiErrorDetail(err))
		return nil, false, diags
	}
	if out == nil {
		return nil, false, diags
	}
	for i := range out.Users {
		if out.Users[i].Email != nil && *out.Users[i].Email == email {
			return &out.Users[i], true, diags
		}
	}
	return nil, false, diags
}

func (r *userResource) findUserByID(ctx context.Context, id string) (*sdk.CommonUser, bool, diag.Diagnostics) {
	var diags diag.Diagnostics

	out, httpResp, err := r.pd.api.DefaultAPI.V1UsersGet(ctx).Id(id).Authorization(r.pd.token).Execute()
	closeBody(httpResp)
	if err != nil {
		diags.AddError("Unable to read user", apiErrorDetail(err))
		return nil, false, diags
	}
	if out == nil || len(out.Users) == 0 {
		return nil, false, diags
	}
	return &out.Users[0], true, diags
}

func (r *userResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan userResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := sdk.V1UsersPutRequest{
		Email:     plan.Email.ValueString(),
		FirstName: plan.FirstName.ValueString(),
		LastName:  plan.LastName.ValueString(),
		GroupId:   plan.GroupID.ValueStringPointer(),
		TimeZone:  plan.TimeZone.ValueStringPointer(),
	}

	httpResp, err := r.pd.api.DefaultAPI.V1UsersPut(ctx).
		Authorization(r.pd.token).
		V1UsersPutRequest(body).
		Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create user", apiErrorDetail(err))
		return
	}

	// V1UsersPut returns no response body, so the new user is located by the email just submitted.
	user, found, diags := r.findUserByEmail(ctx, plan.Email.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.Diagnostics.AddError("Unable to find created user", "the user was created but could not be located afterward by email")
		return
	}

	plan.applyUser(user)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *userResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state userResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	user, found, diags := r.findUserByID(ctx, state.ID.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	state.applyUser(user)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update only runs when a plan changes computed-only output, since every
// configurable attribute is RequiresReplace above (the API has no endpoint to
// update user profile fields) — it simply refreshes computed attributes from the API.
func (r *userResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan userResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	user, found, diags := r.findUserByID(ctx, plan.ID.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.Diagnostics.AddError("Unable to update user", "user no longer exists")
		return
	}

	plan.applyUser(user)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *userResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state userResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := r.pd.api.DefaultAPI.V1UsersIdDelete(ctx, state.ID.ValueString()).Authorization(r.pd.token).Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to delete user", apiErrorDetail(err))
	}
}

func (r *userResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
