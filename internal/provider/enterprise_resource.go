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
	_ resource.Resource                = &enterpriseResource{}
	_ resource.ResourceWithConfigure   = &enterpriseResource{}
	_ resource.ResourceWithImportState = &enterpriseResource{}
)

func NewEnterpriseResource() resource.Resource {
	return &enterpriseResource{}
}

type enterpriseResource struct {
	pd *providerData
}

type enterpriseTimePeriodModel struct {
	Month types.Int64 `tfsdk:"month"`
	Year  types.Int64 `tfsdk:"year"`
}

type enterpriseContractModel struct {
	ContractedCredits types.Float64 `tfsdk:"contracted_credits"`
	ExpirationDate    types.Object  `tfsdk:"expiration_date"`
}

type enterpriseResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	AccountType          types.String `tfsdk:"account_type"`
	CompanyName          types.String `tfsdk:"company_name"`
	EnterpriseContract   types.Object `tfsdk:"enterprise_contract"`
	AdminEmail           types.String `tfsdk:"admin_email"`
	AdminFirstName       types.String `tfsdk:"admin_first_name"`
	AdminLastName        types.String `tfsdk:"admin_last_name"`
	AdminTimeZone        types.String `tfsdk:"admin_time_zone"`
	CloudProvider        types.String `tfsdk:"cloud_provider"`
	Description          types.String `tfsdk:"description"`
	Logo                 types.String `tfsdk:"logo"`
	SmallLogo            types.String `tfsdk:"small_logo"`
	MarketplaceID        types.String `tfsdk:"marketplace_id"`
	CreditLimit          types.Int64  `tfsdk:"credit_limit"`
	ImpersonationEnabled types.Bool   `tfsdk:"impersonation_enabled"`
	PortalBanner         types.String `tfsdk:"portal_banner"`
	ProxyTenantID        types.Int64  `tfsdk:"proxy_tenant_id"`
}

func (r *enterpriseResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_enterprise"
}

func (r *enterpriseResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "An enterprise or MSP tenant. Credits/billing are fields on this resource (enterprise_contract, " +
			"credit_limit) — the API has no standalone credits resource. Create has no response body, so the new " +
			"enterprise is located afterward by matching company_name+account_type in the enterprise list — this " +
			"will fail ambiguously if another enterprise with the same name and type already exists.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"account_type": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"company_name": schema.StringAttribute{
				Required: true,
			},
			"admin_email": schema.StringAttribute{
				Optional: true,
			},
			"admin_first_name": schema.StringAttribute{
				Optional:      true,
				Description:   "Only used at creation; the API has no update endpoint for this field.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"admin_last_name": schema.StringAttribute{
				Optional:      true,
				Description:   "Only used at creation; the API has no update endpoint for this field.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"admin_time_zone": schema.StringAttribute{
				Optional:      true,
				Description:   "Only used at creation; the API has no update endpoint for this field.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"cloud_provider": schema.StringAttribute{
				Optional: true,
			},
			"description": schema.StringAttribute{
				Optional: true,
			},
			"logo": schema.StringAttribute{
				Optional: true,
			},
			"small_logo": schema.StringAttribute{
				Optional: true,
			},
			"marketplace_id": schema.StringAttribute{
				Optional: true,
			},
			"credit_limit": schema.Int64Attribute{
				Optional: true,
			},
			"impersonation_enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Not settable at creation; only updatable afterward.",
			},
			"portal_banner": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Not settable at creation; only updatable afterward.",
			},
			"proxy_tenant_id": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Not settable at creation; only updatable afterward.",
			},
			"enterprise_contract": schema.SingleNestedAttribute{
				Required:    true,
				Description: "Billing contract; contracted_credits is where an enterprise's or MSP pool's credits are set.",
				Attributes: map[string]schema.Attribute{
					"contracted_credits": schema.Float64Attribute{
						Optional: true,
					},
					"expiration_date": schema.SingleNestedAttribute{
						Optional: true,
						Attributes: map[string]schema.Attribute{
							"month": schema.Int64Attribute{Optional: true},
							"year":  schema.Int64Attribute{Optional: true},
						},
					},
				},
			},
		},
	}
}

func (r *enterpriseResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.pd = configurePD(req.ProviderData, &resp.Diagnostics)
}

func buildEnterpriseContract(ctx context.Context, obj types.Object) (sdk.CommonBillingContract, diag.Diagnostics) {
	var contract sdk.CommonBillingContract
	if obj.IsNull() || obj.IsUnknown() {
		return contract, nil
	}
	var m enterpriseContractModel
	diags := obj.As(ctx, &m, objectAsOptions)
	if diags.HasError() {
		return contract, diags
	}
	if v := m.ContractedCredits.ValueFloat64Pointer(); v != nil {
		f := float32(*v)
		contract.ContractedCredits = &f
	}
	if !m.ExpirationDate.IsNull() && !m.ExpirationDate.IsUnknown() {
		var period enterpriseTimePeriodModel
		diags2 := m.ExpirationDate.As(ctx, &period, objectAsOptions)
		diags.Append(diags2...)
		if diags.HasError() {
			return contract, diags
		}
		exp := &sdk.CommonBillingTimePeriod{}
		if v := period.Month.ValueInt64Pointer(); v != nil {
			mo := int32(*v)
			exp.Month = &mo
		}
		if v := period.Year.ValueInt64Pointer(); v != nil {
			yr := int32(*v)
			exp.Year = &yr
		}
		contract.ExpirationDate = exp
	}
	return contract, nil
}

// applyEnterprise deliberately leaves m.EnterpriseContract untouched: IamEnterprise
// (the Get/list response type) has no EnterpriseContract field at all, so whatever
// the model already has (from plan or prior state) is preserved as-is.
func (m *enterpriseResourceModel) applyEnterprise(ctx context.Context, e *sdk.IamEnterprise) diag.Diagnostics {
	if e.EnterpriseId != nil {
		m.ID = types.StringValue(int64ID(*e.EnterpriseId))
	} else {
		m.ID = types.StringNull()
	}
	m.AccountType = types.StringPointerValue(e.AccountType)
	m.CompanyName = types.StringPointerValue(e.CompanyName)
	m.AdminEmail = types.StringPointerValue(e.AdminEmail)
	m.CloudProvider = types.StringPointerValue(e.CloudProvider)
	m.Description = types.StringPointerValue(e.Description)
	m.Logo = types.StringPointerValue(e.Logo)
	m.SmallLogo = types.StringPointerValue(e.SmallLogo)
	m.MarketplaceID = types.StringPointerValue(e.MarketplaceId)
	if e.CreditLimit != nil {
		m.CreditLimit = types.Int64Value(int64(*e.CreditLimit))
	} else {
		m.CreditLimit = types.Int64Null()
	}
	m.ImpersonationEnabled = types.BoolPointerValue(e.ImpersonationEnabled)
	m.PortalBanner = types.StringPointerValue(e.PortalBanner)
	m.ProxyTenantID = types.Int64PointerValue(e.ProxyTenantId)
	return nil
}

func (r *enterpriseResource) findByID(ctx context.Context, id int64) (*sdk.IamEnterprise, bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	out, httpResp, err := r.pd.api.DefaultAPI.V1EnterprisesGet(ctx).EnterpriseIds([]int64{id}).Authorization(r.pd.token).Execute()
	closeBody(httpResp)
	if err != nil {
		diags.AddError("Unable to read enterprise", apiErrorDetail(err))
		return nil, false, diags
	}
	if out == nil {
		return nil, false, diags
	}
	for i := range out.Enterprises {
		if out.Enterprises[i].EnterpriseId != nil && *out.Enterprises[i].EnterpriseId == id {
			return &out.Enterprises[i], true, diags
		}
	}
	return nil, false, diags
}

func (r *enterpriseResource) findByCompanyAndType(ctx context.Context, companyName, accountType string) (*sdk.IamEnterprise, diag.Diagnostics) {
	var diags diag.Diagnostics
	out, httpResp, err := r.pd.api.DefaultAPI.V1EnterprisesGet(ctx).Authorization(r.pd.token).Execute()
	closeBody(httpResp)
	if err != nil {
		diags.AddError("Unable to list enterprises", apiErrorDetail(err))
		return nil, diags
	}
	if out == nil {
		diags.AddError("Unable to find created enterprise", "enterprise list came back empty")
		return nil, diags
	}
	var matches []*sdk.IamEnterprise
	for i := range out.Enterprises {
		e := &out.Enterprises[i]
		if e.CompanyName != nil && *e.CompanyName == companyName && e.AccountType != nil && *e.AccountType == accountType {
			matches = append(matches, e)
		}
	}
	switch len(matches) {
	case 1:
		return matches[0], diags
	case 0:
		diags.AddError("Unable to find created enterprise", "no enterprise in the list matched the submitted company_name and account_type")
		return nil, diags
	default:
		diags.AddError(
			"Ambiguous enterprise lookup",
			fmt.Sprintf("%d enterprises matched company_name %q and account_type %q; the API's create endpoint returns no ID, "+
				"so this provider cannot disambiguate. Use a unique company_name.", len(matches), companyName, accountType),
		)
		return nil, diags
	}
}

func (r *enterpriseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan enterpriseResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	contract, diags := buildEnterpriseContract(ctx, plan.EnterpriseContract)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := sdk.V1EnterprisesPutRequest{
		AccountType:        plan.AccountType.ValueString(),
		CompanyName:        plan.CompanyName.ValueString(),
		EnterpriseContract: contract,
		AdminEmail:         plan.AdminEmail.ValueStringPointer(),
		AdminFirstName:     plan.AdminFirstName.ValueStringPointer(),
		AdminLastName:      plan.AdminLastName.ValueStringPointer(),
		AdminTimeZone:      plan.AdminTimeZone.ValueStringPointer(),
		CloudProvider:      plan.CloudProvider.ValueStringPointer(),
		Description:        plan.Description.ValueStringPointer(),
		Logo:               plan.Logo.ValueStringPointer(),
		MarketplaceId:      plan.MarketplaceID.ValueStringPointer(),
		SmallLogo:          plan.SmallLogo.ValueStringPointer(),
	}
	if v := plan.CreditLimit.ValueInt64Pointer(); v != nil {
		cl := int32(*v)
		body.CreditLimit = &cl
	}

	httpResp, err := r.pd.api.DefaultAPI.V1EnterprisesPut(ctx).
		Authorization(r.pd.token).
		V1EnterprisesPutRequest(body).
		Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create enterprise", apiErrorDetail(err))
		return
	}

	created, diags2 := r.findByCompanyAndType(ctx, plan.CompanyName.ValueString(), plan.AccountType.ValueString())
	resp.Diagnostics.Append(diags2...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(plan.applyEnterprise(ctx, created)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *enterpriseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state enterpriseResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := parseInt64ID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid enterprise id", err.Error())
		return
	}

	e, found, diags := r.findByID(ctx, id)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(state.applyEnterprise(ctx, e)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *enterpriseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan enterpriseResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := parseInt64ID(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid enterprise id", err.Error())
		return
	}

	contract, diags := buildEnterpriseContract(ctx, plan.EnterpriseContract)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := sdk.V1EnterprisesPatchRequest{
		EnterpriseId:         id,
		AdminEmail:           plan.AdminEmail.ValueStringPointer(),
		CloudProvider:        plan.CloudProvider.ValueStringPointer(),
		CompanyName:          plan.CompanyName.ValueStringPointer(),
		Description:          plan.Description.ValueStringPointer(),
		EnterpriseContract:   &contract,
		ImpersonationEnabled: plan.ImpersonationEnabled.ValueBoolPointer(),
		Logo:                 plan.Logo.ValueStringPointer(),
		MarketplaceId:        plan.MarketplaceID.ValueStringPointer(),
		PortalBanner:         plan.PortalBanner.ValueStringPointer(),
		ProxyTenantId:        plan.ProxyTenantID.ValueInt64Pointer(),
		SmallLogo:            plan.SmallLogo.ValueStringPointer(),
	}
	if v := plan.CreditLimit.ValueInt64Pointer(); v != nil {
		cl := int32(*v)
		body.CreditLimit = &cl
	}

	httpResp, err := r.pd.api.DefaultAPI.V1EnterprisesPatch(ctx).
		Authorization(r.pd.token).
		V1EnterprisesPatchRequest(body).
		Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update enterprise", apiErrorDetail(err))
		return
	}

	e, found, diags2 := r.findByID(ctx, id)
	resp.Diagnostics.Append(diags2...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.Diagnostics.AddError("Unable to update enterprise", "enterprise no longer exists")
		return
	}

	resp.Diagnostics.Append(plan.applyEnterprise(ctx, e)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *enterpriseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state enterpriseResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := parseInt64ID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid enterprise id", err.Error())
		return
	}

	httpResp, err := r.pd.api.DefaultAPI.V1EnterprisesEnterpriseIdDelete(ctx, id).Authorization(r.pd.token).Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to delete enterprise", apiErrorDetail(err))
	}
}

func (r *enterpriseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
