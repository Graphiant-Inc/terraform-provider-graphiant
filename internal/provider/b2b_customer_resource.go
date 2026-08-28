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
	_ resource.Resource                = &b2bCustomerResource{}
	_ resource.ResourceWithConfigure   = &b2bCustomerResource{}
	_ resource.ResourceWithImportState = &b2bCustomerResource{}
)

func NewB2bCustomerResource() resource.Resource {
	return &b2bCustomerResource{}
}

type b2bCustomerResource struct {
	pd *providerData
}

var b2bCustomerInviteAttrTypes = map[string]attrType{
	"admin_emails":            types.ListType{ElemType: types.StringType},
	"maximum_number_of_sites": types.Int64Type,
}

type b2bCustomerInviteModel struct {
	AdminEmails          types.List  `tfsdk:"admin_emails"`
	MaximumNumberOfSites types.Int64 `tfsdk:"maximum_number_of_sites"`
}

type b2bCustomerResourceModel struct {
	ID       types.String `tfsdk:"id"`
	Name     types.String `tfsdk:"name"`
	Type     types.String `tfsdk:"type"`
	Invite   types.Object `tfsdk:"invite"`
	NumSites types.Int64  `tfsdk:"num_sites"`
	Status   types.String `tfsdk:"status"`
}

func (r *b2bCustomerResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_b2b_customer"
}

func (r *b2bCustomerResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A B2B ('partner') data exchange customer invite. Part of the producer service -> customer -> " +
			"match -> consumer workflow (see graphiant_b2b_producer_service). name/type are force-new: the update " +
			"endpoint only accepts invite changes. invite.maximum_number_of_sites is not echoed back by the read " +
			"endpoint, so it is preserved from configuration rather than refreshed from the API.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:      true,
				Description:   "Partner display name.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"type": schema.StringAttribute{
				Required:      true,
				Description:   "Graphiant peer vs guest (non-Graphiant) partner.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"num_sites": schema.Int64Attribute{
				Computed: true,
			},
			"status": schema.StringAttribute{
				Computed: true,
			},
			"invite": schema.SingleNestedAttribute{
				Required: true,
				Attributes: map[string]schema.Attribute{
					"admin_emails": schema.ListAttribute{
						Optional:    true,
						ElementType: types.StringType,
					},
					"maximum_number_of_sites": schema.Int64Attribute{
						Optional: true,
					},
				},
			},
		},
	}
}

func (r *b2bCustomerResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.pd = configurePD(req.ProviderData, &resp.Diagnostics)
}

func buildB2bCustomerInvite(ctx context.Context, obj types.Object) (sdk.ManaV2ExtranetServiceCustomerInvite, diag.Diagnostics) {
	var out sdk.ManaV2ExtranetServiceCustomerInvite
	var diags diag.Diagnostics
	if obj.IsNull() || obj.IsUnknown() {
		return out, diags
	}
	var m b2bCustomerInviteModel
	diags.Append(obj.As(ctx, &m, objectAsOptions)...)
	if diags.HasError() {
		return out, diags
	}
	if !m.AdminEmails.IsNull() && !m.AdminEmails.IsUnknown() {
		diags.Append(m.AdminEmails.ElementsAs(ctx, &out.AdminEmails, false)...)
	}
	out.MaximumNumberOfSites = int32PtrFromInt64(m.MaximumNumberOfSites)
	return out, diags
}

func (r *b2bCustomerResource) readByID(ctx context.Context, id int64) (*sdk.V1ExtranetB2bCustomersIdDetailsGetResponse, bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	out, httpResp, err := r.pd.api.DefaultAPI.V1ExtranetB2bCustomersIdDetailsGet(ctx, id).Authorization(r.pd.token).Execute()
	if err != nil {
		if isNotFound(httpResp) {
			closeBody(httpResp)
			return nil, false, diags
		}
		closeBody(httpResp)
		diags.AddError("Unable to read B2B customer", apiErrorDetail(err))
		return nil, false, diags
	}
	closeBody(httpResp)
	if out == nil {
		return nil, false, diags
	}
	return out, true, diags
}

// applyFromGet deliberately leaves m.Invite.maximum_number_of_sites untouched: the
// details endpoint doesn't echo it back, only admin_emails.
func (m *b2bCustomerResourceModel) applyFromGet(ctx context.Context, out *sdk.V1ExtranetB2bCustomersIdDetailsGetResponse) diag.Diagnostics {
	var diags diag.Diagnostics
	m.Name = types.StringPointerValue(out.Name)
	m.Type = types.StringPointerValue(out.Type)
	m.Status = types.StringPointerValue(out.Status)
	if out.NumSites != nil {
		m.NumSites = types.Int64Value(int64(*out.NumSites))
	}

	var existing b2bCustomerInviteModel
	if !m.Invite.IsNull() && !m.Invite.IsUnknown() {
		diags.Append(m.Invite.As(ctx, &existing, objectAsOptions)...)
	}
	adminEmails, d := types.ListValueFrom(ctx, types.StringType, out.AdminEmails)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}
	obj, d2 := types.ObjectValueFrom(ctx, b2bCustomerInviteAttrTypes, b2bCustomerInviteModel{
		AdminEmails:          adminEmails,
		MaximumNumberOfSites: existing.MaximumNumberOfSites,
	})
	diags.Append(d2...)
	m.Invite = obj
	return diags
}

func (r *b2bCustomerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan b2bCustomerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	invite, diags := buildB2bCustomerInvite(ctx, plan.Invite)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := sdk.V1ExtranetB2bCustomersPostRequest{
		Invite: invite,
		Name:   plan.Name.ValueString(),
		Type:   plan.Type.ValueString(),
	}
	out, httpResp, err := r.pd.api.DefaultAPI.V1ExtranetB2bCustomersPost(ctx).
		Authorization(r.pd.token).
		V1ExtranetB2bCustomersPostRequest(body).
		Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create B2B customer", apiErrorDetail(err))
		return
	}
	if out == nil || out.Id == nil {
		resp.Diagnostics.AddError("Unable to create B2B customer", "API returned an empty response")
		return
	}
	plan.ID = types.StringValue(int64ID(*out.Id))

	got, found, diags2 := r.readByID(ctx, *out.Id)
	resp.Diagnostics.Append(diags2...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.Diagnostics.AddError("Unable to read created B2B customer", "customer was created but could not be read back")
		return
	}

	resp.Diagnostics.Append(plan.applyFromGet(ctx, got)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *b2bCustomerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state b2bCustomerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := parseInt64ID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid B2B customer id", err.Error())
		return
	}

	got, found, diags := r.readByID(ctx, id)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(state.applyFromGet(ctx, got)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *b2bCustomerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan b2bCustomerResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := parseInt64ID(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid B2B customer id", err.Error())
		return
	}

	invite, diags := buildB2bCustomerInvite(ctx, plan.Invite)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := sdk.V1ExtranetB2bCustomersIdPutRequest{Invite: &invite}
	_, httpResp, err := r.pd.api.DefaultAPI.V1ExtranetB2bCustomersIdPut(ctx, id).
		Authorization(r.pd.token).
		V1ExtranetB2bCustomersIdPutRequest(body).
		Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update B2B customer", apiErrorDetail(err))
		return
	}

	got, found, diags2 := r.readByID(ctx, id)
	resp.Diagnostics.Append(diags2...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.Diagnostics.AddError("Unable to update B2B customer", "customer no longer exists")
		return
	}

	resp.Diagnostics.Append(plan.applyFromGet(ctx, got)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *b2bCustomerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state b2bCustomerResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := parseInt64ID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid B2B customer id", err.Error())
		return
	}

	_, httpResp, err := r.pd.api.DefaultAPI.V1ExtranetB2bCustomersIdDelete(ctx, id).Authorization(r.pd.token).Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to delete B2B customer", apiErrorDetail(err))
	}
}

func (r *b2bCustomerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
