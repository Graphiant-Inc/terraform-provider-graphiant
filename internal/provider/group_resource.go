package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	graphiant "github.com/Graphiant-Inc/graphiant-sdk-go"
	"github.com/Graphiant-Inc/terraform-provider-graphiant/internal/provider/generated/resource_group"
)

var (
	_ resource.Resource                     = &groupResource{}
	_ resource.ResourceWithConfigure        = &groupResource{}
	_ resource.ResourceWithConfigValidators = &groupResource{}
	_ resource.ResourceWithImportState      = &groupResource{}
)

func NewGroupResource() resource.Resource {
	return &groupResource{}
}

type groupResource struct {
	client *gClient
}

// groupResourceModel mirrors iamGroup together with the subset of fields
// that can be set via v1GroupsPutRequest / v1GroupsIdPatchRequest.
type groupResourceModel struct {
	Id                 types.String      `tfsdk:"id"`
	Name               types.String      `tfsdk:"name"`
	Description        types.String      `tfsdk:"description"`
	GroupType          types.String      `tfsdk:"group_type"`
	GroupId            types.String      `tfsdk:"group_id"`
	ManagesEnterprises types.Bool        `tfsdk:"manages_enterprises"`
	TimeWindowStart    types.Int64       `tfsdk:"time_window_start"`
	TimeWindowEnd      types.Int64       `tfsdk:"time_window_end"`
	Permissions        *permissionsModel `tfsdk:"permissions"`
	EnterpriseIds      types.List        `tfsdk:"enterprise_ids"`
}

func (r *groupResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group"
}

// Schema is generated from the OpenAPI spec (see api/generator_config.yml,
// resources.group) via `make generate-schemas`. Note: the API's raw field
// for what this provider calls "idp_group_id" is actually named "groupId"
// (see manaV2's v1GroupsPutRequest); the generated attribute is named
// group_id accordingly rather than idp_group_id.
func (r *groupResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_group.GroupResourceSchema(ctx)
	resp.Schema.Description = "Manages a Graphiant IAM group (a named collection of permissions that users can be assigned to)."
	resp.Schema.Attributes["id"] = schema.StringAttribute{
		Computed:    true,
		Description: "Group identifier assigned by the Graphiant controller.",
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
		},
	}
	resp.Schema.Attributes["name"] = schema.StringAttribute{
		Required:    true,
		Description: "Group name.",
		Validators: []validator.String{
			stringvalidator.LengthAtLeast(1),
		},
	}
	resp.Schema.Attributes["description"] = schema.StringAttribute{
		Required:    true,
		Description: "Group description.",
		Validators: []validator.String{
			stringvalidator.LengthAtLeast(1),
		},
	}
	resp.Schema.Attributes["group_type"] = schema.StringAttribute{
		Optional:    true,
		Computed:    true,
		Description: "Group type (e.g. \"custom\").",
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
		},
	}
	resp.Schema.Attributes["group_id"] = schema.StringAttribute{
		Optional:    true,
		Computed:    true,
		Description: "External group ID. Only supply this if the enterprise uses an identity provider (IdP) for group management.",
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.RequiresReplace(),
		},
	}
	resp.Schema.Attributes["manages_enterprises"] = schema.BoolAttribute{
		Optional:    true,
		Computed:    true,
		Description: "Whether members of this group can manage sub-enterprises. Can only be set at creation.",
		PlanModifiers: []planmodifier.Bool{
			boolplanmodifier.RequiresReplace(),
		},
	}
	resp.Schema.Attributes["time_window_start"] = schema.Int64Attribute{
		Optional:    true,
		Description: "Unix timestamp for the start of the access time window. Must be set together with time_window_end.",
	}
	resp.Schema.Attributes["time_window_end"] = schema.Int64Attribute{
		Optional:    true,
		Description: "Unix timestamp for the end of the access time window. Must be set together with time_window_start.",
	}
	// enterprise_ids is purely server-derived (from group membership
	// elsewhere) and never changes as a side effect of updating the fields
	// above.
	resp.Schema.Attributes["enterprise_ids"] = schema.ListAttribute{
		Computed:    true,
		ElementType: types.Int64Type,
		Description: "Enterprises this group has access to.",
		PlanModifiers: []planmodifier.List{
			listplanmodifier.UseStateForUnknown(),
		},
	}
}

func (r *groupResource) ConfigValidators(_ context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		resourcevalidator.RequiredTogether(
			path.MatchRoot("time_window_start"),
			path.MatchRoot("time_window_end"),
		),
	}
}

func (r *groupResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *groupResource) flatten(ctx context.Context, g *graphiant.IamGroup, m *groupResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	m.Id = strValue(g.Id)
	m.Name = strValue(g.Name)
	m.Description = strValue(g.Description)
	m.GroupType = strValue(g.GroupType)
	m.TimeWindowStart = int64Value(g.TimeWindowStart)
	m.TimeWindowEnd = int64Value(g.TimeWindowEnd)
	m.Permissions = flattenPermissions(g.Permissions)

	ids := g.EnterpriseIds
	if ids == nil {
		ids = []int64{}
	}
	list, d := types.ListValueFrom(ctx, types.Int64Type, ids)
	diags.Append(d...)
	m.EnterpriseIds = list
	return diags
}

// findGroup looks up a group by ID. There is no get-by-id endpoint for
// groups, so this lists every group and filters client-side.
func (r *groupResource) findGroup(ctx context.Context, id string) (*graphiant.IamGroup, error) {
	out, httpRes, err := r.client.api.DefaultAPI.V1GroupsGet(ctx).Authorization(r.client.authHeader()).Execute()
	defer closeBody(httpRes)
	if err != nil {
		return nil, err
	}
	if out == nil {
		return nil, nil
	}
	for _, g := range out.GetGroups() {
		if g.GetId() == id {
			return &g, nil
		}
	}
	return nil, nil
}

// findGroupByName looks up a group by name, used right after creation since
// v1GroupsPut does not return the created group (and therefore not its
// server-assigned ID).
func (r *groupResource) findGroupByName(ctx context.Context, name string) (*graphiant.IamGroup, error) {
	out, httpRes, err := r.client.api.DefaultAPI.V1GroupsGet(ctx).Authorization(r.client.authHeader()).Execute()
	defer closeBody(httpRes)
	if err != nil {
		return nil, err
	}
	if out == nil {
		return nil, nil
	}
	for _, g := range out.GetGroups() {
		if g.GetName() == name {
			return &g, nil
		}
	}
	return nil, nil
}

func (r *groupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan groupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "creating group", map[string]any{"name": plan.Name.ValueString()})

	body := graphiant.NewV1GroupsPutRequest(plan.Description.ValueString(), plan.Name.ValueString())
	if v := strPtr(plan.GroupType); v != nil {
		body.SetGroupType(*v)
	}
	if v := strPtr(plan.GroupId); v != nil {
		body.SetGroupId(*v)
	}
	if v := boolPtr(plan.ManagesEnterprises); v != nil {
		body.SetManagesEnterprises(*v)
	}
	if v := int64Ptr(plan.TimeWindowStart); v != nil {
		body.SetTimeWindowStart(*v)
	}
	if v := int64Ptr(plan.TimeWindowEnd); v != nil {
		body.SetTimeWindowEnd(*v)
	}
	if plan.Permissions != nil {
		body.SetPermissions(*expandPermissions(plan.Permissions))
	}

	httpRes, err := r.client.api.DefaultAPI.V1GroupsPut(ctx).Authorization(r.client.authHeader()).V1GroupsPutRequest(*body).Execute()
	defer closeBody(httpRes)
	if err != nil {
		resp.Diagnostics.AddError("Error creating group", apiErrorDetail(err))
		return
	}

	group, err := r.findGroupByName(ctx, plan.Name.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading back created group", apiErrorDetail(err))
		return
	}
	if group == nil {
		resp.Diagnostics.AddError("Error creating group", "the group was created but could not be found afterwards by name")
		return
	}

	resp.Diagnostics.Append(r.flatten(ctx, group, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	tflog.Trace(ctx, "created group", map[string]any{"id": plan.Id.ValueString()})
}

func (r *groupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state groupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "reading group", map[string]any{"id": state.Id.ValueString()})

	group, err := r.findGroup(ctx, state.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading group", apiErrorDetail(err))
		return
	}
	if group == nil {
		tflog.Debug(ctx, "group no longer exists, removing from state", map[string]any{"id": state.Id.ValueString()})
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(r.flatten(ctx, group, &state)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *groupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state groupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "updating group", map[string]any{"id": state.Id.ValueString()})

	body := graphiant.NewV1GroupsIdPatchRequestWithDefaults()
	if v := strPtr(plan.Description); v != nil {
		body.SetDescription(*v)
	}
	if v := strPtr(plan.Name); v != nil {
		body.SetDisplayName(*v)
	}
	if v := strPtr(plan.GroupType); v != nil {
		body.SetGroupType(*v)
	}
	if v := int64Ptr(plan.TimeWindowStart); v != nil {
		body.SetTimeWindowStart(*v)
	}
	if v := int64Ptr(plan.TimeWindowEnd); v != nil {
		body.SetTimeWindowEnd(*v)
	}
	if plan.Permissions != nil {
		body.SetPermissions(*expandPermissions(plan.Permissions))
	}

	httpRes, err := r.client.api.DefaultAPI.V1GroupsIdPatch(ctx, state.Id.ValueString()).Authorization(r.client.authHeader()).V1GroupsIdPatchRequest(*body).Execute()
	defer closeBody(httpRes)
	if err != nil {
		resp.Diagnostics.AddError("Error updating group", apiErrorDetail(err))
		return
	}

	group, err := r.findGroup(ctx, state.Id.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading back updated group", apiErrorDetail(err))
		return
	}
	if group == nil {
		resp.Diagnostics.AddError("Error updating group", "the group was updated but could not be found afterwards")
		return
	}

	resp.Diagnostics.Append(r.flatten(ctx, group, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	tflog.Trace(ctx, "updated group", map[string]any{"id": plan.Id.ValueString()})
}

func (r *groupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state groupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "deleting group", map[string]any{"id": state.Id.ValueString()})

	httpRes, err := r.client.api.DefaultAPI.V1GroupsIdDelete(ctx, state.Id.ValueString()).Authorization(r.client.authHeader()).Execute()
	defer closeBody(httpRes)
	if err != nil {
		resp.Diagnostics.AddError("Error deleting group", apiErrorDetail(err))
	}
}

func (r *groupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
