package provider

import (
	"context"
	"fmt"

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
	_ resource.Resource                = &groupResource{}
	_ resource.ResourceWithConfigure   = &groupResource{}
	_ resource.ResourceWithImportState = &groupResource{}
)

func NewGroupResource() resource.Resource {
	return &groupResource{}
}

type groupResource struct {
	pd *providerData
}

type groupPermissionsModel struct {
	AssetManager                 types.String `tfsdk:"asset_manager"`
	B2b                          types.String `tfsdk:"b2b"`
	B2bSecurityProfileExternal   types.String `tfsdk:"b2b_security_profile_external"`
	BillingAndInvoicing          types.String `tfsdk:"billing_and_invoicing"`
	Compliance                   types.String `tfsdk:"compliance"`
	DeveloperTools               types.String `tfsdk:"developer_tools"`
	Gateway                      types.String `tfsdk:"gateway"`
	GlobalServices               types.String `tfsdk:"global_services"`
	Insights                     types.String `tfsdk:"insights"`
	Licensing                    types.String `tfsdk:"licensing"`
	Logs                         types.String `tfsdk:"logs"`
	MonitoringAndTroubleshooting types.String `tfsdk:"monitoring_and_troubleshooting"`
	NetworkConfiguration         types.String `tfsdk:"network_configuration"`
	OrderStatus                  types.String `tfsdk:"order_status"`
	Reports                      types.String `tfsdk:"reports"`
	SafetyAndSecurity            types.String `tfsdk:"safety_and_security"`
	ServicePolicies              types.String `tfsdk:"service_policies"`
	Support                      types.String `tfsdk:"support"`
	UserAndTenantManagement      types.String `tfsdk:"user_and_tenant_management"`
}

var groupPermissionsAttrTypes = map[string]attrType{
	"asset_manager":                  types.StringType,
	"b2b":                            types.StringType,
	"b2b_security_profile_external":  types.StringType,
	"billing_and_invoicing":          types.StringType,
	"compliance":                     types.StringType,
	"developer_tools":                types.StringType,
	"gateway":                        types.StringType,
	"global_services":                types.StringType,
	"insights":                       types.StringType,
	"licensing":                      types.StringType,
	"logs":                           types.StringType,
	"monitoring_and_troubleshooting": types.StringType,
	"network_configuration":          types.StringType,
	"order_status":                   types.StringType,
	"reports":                        types.StringType,
	"safety_and_security":            types.StringType,
	"service_policies":               types.StringType,
	"support":                        types.StringType,
	"user_and_tenant_management":     types.StringType,
}

func permissionsToSDK(ctx context.Context, obj types.Object) (*sdk.CommonPermissions, diag.Diagnostics) {
	if obj.IsNull() || obj.IsUnknown() {
		return nil, nil
	}
	var m groupPermissionsModel
	diags := obj.As(ctx, &m, objectAsOptions)
	if diags.HasError() {
		return nil, diags
	}
	return &sdk.CommonPermissions{
		AssetManager:                 m.AssetManager.ValueStringPointer(),
		B2b:                          m.B2b.ValueStringPointer(),
		B2bSecurityProfileExternal:   m.B2bSecurityProfileExternal.ValueStringPointer(),
		BillingAndInvoicing:          m.BillingAndInvoicing.ValueStringPointer(),
		Compliance:                   m.Compliance.ValueStringPointer(),
		DeveloperTools:               m.DeveloperTools.ValueStringPointer(),
		Gateway:                      m.Gateway.ValueStringPointer(),
		GlobalServices:               m.GlobalServices.ValueStringPointer(),
		Insights:                     m.Insights.ValueStringPointer(),
		Licensing:                    m.Licensing.ValueStringPointer(),
		Logs:                         m.Logs.ValueStringPointer(),
		MonitoringAndTroubleshooting: m.MonitoringAndTroubleshooting.ValueStringPointer(),
		NetworkConfiguration:         m.NetworkConfiguration.ValueStringPointer(),
		OrderStatus:                  m.OrderStatus.ValueStringPointer(),
		Reports:                      m.Reports.ValueStringPointer(),
		SafetyAndSecurity:            m.SafetyAndSecurity.ValueStringPointer(),
		ServicePolicies:              m.ServicePolicies.ValueStringPointer(),
		Support:                      m.Support.ValueStringPointer(),
		UserAndTenantManagement:      m.UserAndTenantManagement.ValueStringPointer(),
	}, nil
}

func permissionsFromSDK(ctx context.Context, p *sdk.CommonPermissions) (types.Object, diag.Diagnostics) {
	if p == nil {
		return types.ObjectNull(groupPermissionsAttrTypes), nil
	}
	m := groupPermissionsModel{
		AssetManager:                 types.StringPointerValue(p.AssetManager),
		B2b:                          types.StringPointerValue(p.B2b),
		B2bSecurityProfileExternal:   types.StringPointerValue(p.B2bSecurityProfileExternal),
		BillingAndInvoicing:          types.StringPointerValue(p.BillingAndInvoicing),
		Compliance:                   types.StringPointerValue(p.Compliance),
		DeveloperTools:               types.StringPointerValue(p.DeveloperTools),
		Gateway:                      types.StringPointerValue(p.Gateway),
		GlobalServices:               types.StringPointerValue(p.GlobalServices),
		Insights:                     types.StringPointerValue(p.Insights),
		Licensing:                    types.StringPointerValue(p.Licensing),
		Logs:                         types.StringPointerValue(p.Logs),
		MonitoringAndTroubleshooting: types.StringPointerValue(p.MonitoringAndTroubleshooting),
		NetworkConfiguration:         types.StringPointerValue(p.NetworkConfiguration),
		OrderStatus:                  types.StringPointerValue(p.OrderStatus),
		Reports:                      types.StringPointerValue(p.Reports),
		SafetyAndSecurity:            types.StringPointerValue(p.SafetyAndSecurity),
		ServicePolicies:              types.StringPointerValue(p.ServicePolicies),
		Support:                      types.StringPointerValue(p.Support),
		UserAndTenantManagement:      types.StringPointerValue(p.UserAndTenantManagement),
	}
	return types.ObjectValueFrom(ctx, groupPermissionsAttrTypes, m)
}

type groupResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	Description        types.String `tfsdk:"description"`
	GroupID            types.String `tfsdk:"group_id"`
	GroupType          types.String `tfsdk:"group_type"`
	ManagesEnterprises types.Bool   `tfsdk:"manages_enterprises"`
	Permissions        types.Object `tfsdk:"permissions"`
	TimeWindowStart    types.Int64  `tfsdk:"time_window_start"`
	TimeWindowEnd      types.Int64  `tfsdk:"time_window_end"`
	Members            types.Set    `tfsdk:"members"`
}

func (r *groupResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group"
}

func (r *groupResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A Graphiant permissions group. Create has no response body, so the new group is located " +
			"afterward by matching name+description in the group list — this will fail ambiguously if another " +
			"group with the same name and description already exists.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"description": schema.StringAttribute{
				Required: true,
			},
			"group_id": schema.StringAttribute{
				Optional:      true,
				Description:   "Only supply if the enterprise uses an IdP for group provisioning.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"group_type": schema.StringAttribute{
				Optional: true,
			},
			"manages_enterprises": schema.BoolAttribute{
				Optional:    true,
				Description: "MSP use: whether this group manages child enterprises.",
			},
			"time_window_start": schema.Int64Attribute{
				Optional: true,
			},
			"time_window_end": schema.Int64Attribute{
				Optional: true,
			},
			"members": schema.SetAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "User IDs belonging to this group. Managed as a full replace on every change.",
			},
		},
		Blocks: map[string]schema.Block{
			"permissions": schema.SingleNestedBlock{
				Description: "Per-module permission levels (values are opaque strings defined by the API).",
				Attributes: map[string]schema.Attribute{
					"asset_manager":                  schema.StringAttribute{Optional: true},
					"b2b":                            schema.StringAttribute{Optional: true},
					"b2b_security_profile_external":  schema.StringAttribute{Optional: true},
					"billing_and_invoicing":          schema.StringAttribute{Optional: true},
					"compliance":                     schema.StringAttribute{Optional: true},
					"developer_tools":                schema.StringAttribute{Optional: true},
					"gateway":                        schema.StringAttribute{Optional: true},
					"global_services":                schema.StringAttribute{Optional: true},
					"insights":                       schema.StringAttribute{Optional: true},
					"licensing":                      schema.StringAttribute{Optional: true},
					"logs":                           schema.StringAttribute{Optional: true},
					"monitoring_and_troubleshooting": schema.StringAttribute{Optional: true},
					"network_configuration":          schema.StringAttribute{Optional: true},
					"order_status":                   schema.StringAttribute{Optional: true},
					"reports":                        schema.StringAttribute{Optional: true},
					"safety_and_security":            schema.StringAttribute{Optional: true},
					"service_policies":               schema.StringAttribute{Optional: true},
					"support":                        schema.StringAttribute{Optional: true},
					"user_and_tenant_management":     schema.StringAttribute{Optional: true},
				},
			},
		},
	}
}

func (r *groupResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.pd = configurePD(req.ProviderData, &resp.Diagnostics)
}

func (r *groupResource) findGroupByID(ctx context.Context, id string) (*sdk.IamGroup, bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	out, httpResp, err := r.pd.api.DefaultAPI.V1GroupsGet(ctx).Authorization(r.pd.token).Execute()
	closeBody(httpResp)
	if err != nil {
		diags.AddError("Unable to list groups", apiErrorDetail(err))
		return nil, false, diags
	}
	if out == nil {
		return nil, false, diags
	}
	for i := range out.Groups {
		if out.Groups[i].Id != nil && *out.Groups[i].Id == id {
			return &out.Groups[i], true, diags
		}
	}
	return nil, false, diags
}

func (r *groupResource) findGroupByNameAndDescription(ctx context.Context, name, description string) (*sdk.IamGroup, diag.Diagnostics) {
	var diags diag.Diagnostics
	out, httpResp, err := r.pd.api.DefaultAPI.V1GroupsGet(ctx).Authorization(r.pd.token).Execute()
	closeBody(httpResp)
	if err != nil {
		diags.AddError("Unable to list groups", apiErrorDetail(err))
		return nil, diags
	}
	if out == nil {
		diags.AddError("Unable to find created group", "group list came back empty")
		return nil, diags
	}
	var matches []*sdk.IamGroup
	for i := range out.Groups {
		g := &out.Groups[i]
		if g.Name != nil && *g.Name == name && g.Description != nil && *g.Description == description {
			matches = append(matches, g)
		}
	}
	switch len(matches) {
	case 1:
		return matches[0], diags
	case 0:
		diags.AddError("Unable to find created group", "no group in the list matched the submitted name and description")
		return nil, diags
	default:
		diags.AddError(
			"Ambiguous group lookup",
			fmt.Sprintf("%d groups matched name %q and description %q; the API's create endpoint returns no ID, "+
				"so this provider cannot disambiguate. Use a unique name+description, or set group_id explicitly.", len(matches), name, description),
		)
		return nil, diags
	}
}

func (m *groupResourceModel) applyGroup(ctx context.Context, g *sdk.IamGroup) diag.Diagnostics {
	m.ID = types.StringPointerValue(g.Id)
	m.Name = types.StringPointerValue(g.Name)
	m.Description = types.StringPointerValue(g.Description)
	m.GroupType = types.StringPointerValue(g.GroupType)
	m.TimeWindowStart = types.Int64PointerValue(g.TimeWindowStart)
	m.TimeWindowEnd = types.Int64PointerValue(g.TimeWindowEnd)

	perms, diags := permissionsFromSDK(ctx, g.Permissions)
	if diags.HasError() {
		return diags
	}
	m.Permissions = perms
	return nil
}

func (r *groupResource) syncMembers(ctx context.Context, id string, members types.Set) diag.Diagnostics {
	var diags diag.Diagnostics
	if members.IsNull() || members.IsUnknown() {
		return diags
	}

	var ids []string
	d := members.ElementsAs(ctx, &ids, false)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}

	replace := true
	body := sdk.V1GroupsIdMembersPostRequest{MemberIds: ids, ReplaceExisting: &replace}
	httpResp, err := r.pd.api.DefaultAPI.V1GroupsIdMembersPost(ctx, id).
		Authorization(r.pd.token).
		V1GroupsIdMembersPostRequest(body).
		Execute()
	closeBody(httpResp)
	if err != nil {
		diags.AddError("Unable to set group members", apiErrorDetail(err))
	}
	return diags
}

func (r *groupResource) readMembers(ctx context.Context, id string) (types.Set, diag.Diagnostics) {
	out, httpResp, err := r.pd.api.DefaultAPI.V1GroupsIdMembersGet(ctx, id).Authorization(r.pd.token).Execute()
	closeBody(httpResp)
	if err != nil {
		var diags diag.Diagnostics
		diags.AddError("Unable to read group members", apiErrorDetail(err))
		return types.SetNull(types.StringType), diags
	}
	var ids []string
	if out != nil {
		for _, u := range out.Users {
			if u.UserId != nil {
				ids = append(ids, *u.UserId)
			}
		}
	}
	return types.SetValueFrom(ctx, types.StringType, ids)
}

func (r *groupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan groupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	perms, diags := permissionsToSDK(ctx, plan.Permissions)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := sdk.V1GroupsPutRequest{
		Description:        plan.Description.ValueString(),
		Name:               plan.Name.ValueString(),
		GroupId:            plan.GroupID.ValueStringPointer(),
		GroupType:          plan.GroupType.ValueStringPointer(),
		ManagesEnterprises: plan.ManagesEnterprises.ValueBoolPointer(),
		Permissions:        perms,
		TimeWindowStart:    plan.TimeWindowStart.ValueInt64Pointer(),
		TimeWindowEnd:      plan.TimeWindowEnd.ValueInt64Pointer(),
	}

	httpResp, err := r.pd.api.DefaultAPI.V1GroupsPut(ctx).
		Authorization(r.pd.token).
		V1GroupsPutRequest(body).
		Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create group", apiErrorDetail(err))
		return
	}

	group, diags2 := r.findGroupByNameAndDescription(ctx, plan.Name.ValueString(), plan.Description.ValueString())
	resp.Diagnostics.Append(diags2...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(plan.applyGroup(ctx, group)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.syncMembers(ctx, plan.ID.ValueString(), plan.Members)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *groupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state groupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	group, found, diags := r.findGroupByID(ctx, state.ID.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(state.applyGroup(ctx, group)...)
	if resp.Diagnostics.HasError() {
		return
	}

	members, diags3 := r.readMembers(ctx, state.ID.ValueString())
	resp.Diagnostics.Append(diags3...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Members = members

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *groupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan groupResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	perms, diags := permissionsToSDK(ctx, plan.Permissions)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := sdk.V1GroupsIdPatchRequest{
		Description:     plan.Description.ValueStringPointer(),
		DisplayName:     plan.Name.ValueStringPointer(),
		GroupType:       plan.GroupType.ValueStringPointer(),
		Permissions:     perms,
		TimeWindowStart: plan.TimeWindowStart.ValueInt64Pointer(),
		TimeWindowEnd:   plan.TimeWindowEnd.ValueInt64Pointer(),
	}

	httpResp, err := r.pd.api.DefaultAPI.V1GroupsIdPatch(ctx, plan.ID.ValueString()).
		Authorization(r.pd.token).
		V1GroupsIdPatchRequest(body).
		Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update group", apiErrorDetail(err))
		return
	}

	group, found, diags2 := r.findGroupByID(ctx, plan.ID.ValueString())
	resp.Diagnostics.Append(diags2...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.Diagnostics.AddError("Unable to update group", "group no longer exists")
		return
	}

	resp.Diagnostics.Append(plan.applyGroup(ctx, group)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.syncMembers(ctx, plan.ID.ValueString(), plan.Members)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *groupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state groupResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	httpResp, err := r.pd.api.DefaultAPI.V1GroupsIdDelete(ctx, state.ID.ValueString()).Authorization(r.pd.token).Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to delete group", apiErrorDetail(err))
	}
}

func (r *groupResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
