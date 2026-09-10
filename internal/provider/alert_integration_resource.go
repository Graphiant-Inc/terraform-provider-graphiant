package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"

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
	_ resource.Resource                = &alertIntegrationResource{}
	_ resource.ResourceWithConfigure   = &alertIntegrationResource{}
	_ resource.ResourceWithImportState = &alertIntegrationResource{}
)

func NewAlertIntegrationResource() resource.Resource {
	return &alertIntegrationResource{}
}

type alertIntegrationResource struct {
	pd *providerData
}

var alertZendeskDetailsAttrTypes = map[string]attrType{
	"zendesk_assignee_id":   types.StringType,
	"zendesk_base_url":      types.StringType,
	"zendesk_client_id":     types.StringType,
	"zendesk_client_secret": types.StringType,
}

type alertZendeskDetailsModel struct {
	ZendeskAssigneeID   types.String `tfsdk:"zendesk_assignee_id"`
	ZendeskBaseURL      types.String `tfsdk:"zendesk_base_url"`
	ZendeskClientID     types.String `tfsdk:"zendesk_client_id"`
	ZendeskClientSecret types.String `tfsdk:"zendesk_client_secret"`
}

var alertIntegrationDetailsAttrTypes = map[string]attrType{
	"opsgenie_key":          types.StringType,
	"opsramp_details":       types.StringType,
	"pagerduty_routing_key": types.StringType,
	"webhook_url":           types.StringType,
	"zendesk_details":       types.ObjectType{AttrTypes: alertZendeskDetailsAttrTypes},
}

type alertIntegrationDetailsModel struct {
	OpsgenieKey         types.String `tfsdk:"opsgenie_key"`
	OpsrampDetails      types.String `tfsdk:"opsramp_details"`
	PagerdutyRoutingKey types.String `tfsdk:"pagerduty_routing_key"`
	WebhookURL          types.String `tfsdk:"webhook_url"`
	ZendeskDetails      types.Object `tfsdk:"zendesk_details"`
}

type alertIntegrationResourceModel struct {
	ID              types.String `tfsdk:"id"`
	Enterprise      types.Int64  `tfsdk:"enterprise"`
	IntegrationType types.String `tfsdk:"integration_type"`
	NickName        types.String `tfsdk:"nick_name"`
	IsActive        types.Bool   `tfsdk:"is_active"`
	Details         types.Object `tfsdk:"details"`
}

func (r *alertIntegrationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_alert_integration"
}

func (r *alertIntegrationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "An alert delivery integration (Zendesk/Slack webhook/PagerDuty/Opsgenie/Opsramp). Only one " +
			"field under details should be set, matching integration_type — the API validates this server-side, " +
			"there is no closed enum for integration_type in the SDK. Import uses \"<enterprise>:<id>\" since Read " +
			"needs the owning enterprise id, which the API doesn't echo back on the integration object itself.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"enterprise": schema.Int64Attribute{
				Required: true,
			},
			"integration_type": schema.StringAttribute{
				Required: true,
			},
			"nick_name": schema.StringAttribute{
				Required: true,
			},
			"is_active": schema.BoolAttribute{
				Optional: true,
				Computed: true,
			},
			"details": schema.SingleNestedAttribute{
				Optional: true,
				Attributes: map[string]schema.Attribute{
					"opsgenie_key":          schema.StringAttribute{Optional: true, Sensitive: true},
					"opsramp_details":       schema.StringAttribute{Optional: true, Sensitive: true},
					"pagerduty_routing_key": schema.StringAttribute{Optional: true, Sensitive: true},
					"webhook_url":           schema.StringAttribute{Optional: true},
					"zendesk_details": schema.SingleNestedAttribute{
						Optional: true,
						Attributes: map[string]schema.Attribute{
							"zendesk_assignee_id":   schema.StringAttribute{Optional: true},
							"zendesk_base_url":      schema.StringAttribute{Optional: true},
							"zendesk_client_id":     schema.StringAttribute{Optional: true},
							"zendesk_client_secret": schema.StringAttribute{Optional: true, Sensitive: true},
						},
					},
				},
			},
		},
	}
}

func (r *alertIntegrationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.pd = configurePD(req.ProviderData, &resp.Diagnostics)
}

func buildAlertIntegrationDetails(ctx context.Context, obj types.Object) (*sdk.AlertserviceIntegrationDetails, diag.Diagnostics) {
	var diags diag.Diagnostics
	if obj.IsNull() || obj.IsUnknown() {
		return nil, diags
	}
	var m alertIntegrationDetailsModel
	diags.Append(obj.As(ctx, &m, objectAsOptions)...)
	if diags.HasError() {
		return nil, diags
	}
	details := &sdk.AlertserviceIntegrationDetails{
		OpsgenieKey:         m.OpsgenieKey.ValueStringPointer(),
		OpsrampDetails:      m.OpsrampDetails.ValueStringPointer(),
		PagerdutyRoutingKey: m.PagerdutyRoutingKey.ValueStringPointer(),
		WebhookUrl:          m.WebhookURL.ValueStringPointer(),
	}
	if !m.ZendeskDetails.IsNull() && !m.ZendeskDetails.IsUnknown() {
		var z alertZendeskDetailsModel
		diags.Append(m.ZendeskDetails.As(ctx, &z, objectAsOptions)...)
		details.ZendeskDetails = &sdk.AlertserviceZendeskDetails{
			ZendeskAssigneeId:   z.ZendeskAssigneeID.ValueString(),
			ZendeskBaseUrl:      z.ZendeskBaseURL.ValueString(),
			ZendeskClientId:     z.ZendeskClientID.ValueString(),
			ZendeskClientSecret: z.ZendeskClientSecret.ValueString(),
		}
	}
	if diags.HasError() {
		return nil, diags
	}
	return details, diags
}

func applyAlertIntegrationDetails(ctx context.Context, details *sdk.AlertserviceIntegrationDetails) (types.Object, diag.Diagnostics) {
	if details == nil {
		return types.ObjectNull(alertIntegrationDetailsAttrTypes), nil
	}
	var diags diag.Diagnostics
	m := alertIntegrationDetailsModel{
		OpsgenieKey:         types.StringPointerValue(details.OpsgenieKey),
		OpsrampDetails:      types.StringPointerValue(details.OpsrampDetails),
		PagerdutyRoutingKey: types.StringPointerValue(details.PagerdutyRoutingKey),
		WebhookURL:          types.StringPointerValue(details.WebhookUrl),
		ZendeskDetails:      types.ObjectNull(alertZendeskDetailsAttrTypes),
	}
	if details.ZendeskDetails != nil {
		z := details.ZendeskDetails
		obj, d := types.ObjectValueFrom(ctx, alertZendeskDetailsAttrTypes, alertZendeskDetailsModel{
			ZendeskAssigneeID:   types.StringValue(z.ZendeskAssigneeId),
			ZendeskBaseURL:      types.StringValue(z.ZendeskBaseUrl),
			ZendeskClientID:     types.StringValue(z.ZendeskClientId),
			ZendeskClientSecret: types.StringValue(z.ZendeskClientSecret),
		})
		diags.Append(d...)
		m.ZendeskDetails = obj
	}
	if diags.HasError() {
		return types.ObjectNull(alertIntegrationDetailsAttrTypes), diags
	}
	return types.ObjectValueFrom(ctx, alertIntegrationDetailsAttrTypes, m)
}

func (m *alertIntegrationResourceModel) applyIntegration(ctx context.Context, in *sdk.AlertserviceIntegration) diag.Diagnostics {
	if in.Id != nil {
		m.ID = types.StringValue(int64ID(*in.Id))
	}
	m.IntegrationType = types.StringPointerValue(in.Type)
	m.NickName = types.StringPointerValue(in.NickName)
	// This alertservice API has been observed (on the sibling alert_notification
	// endpoint) omitting boolean fields entirely rather than sending false
	// explicitly — don't let that null out a value we just set via Create/Update.
	// But is_active is Optional+Computed: when left out of config, m.IsActive is
	// Unknown here, and Terraform requires every Computed attribute to resolve to
	// a known value after apply (Null counts as known, Unknown does not), so a
	// nil in.IsActive must still resolve to BoolNull() in that case.
	if in.IsActive != nil {
		m.IsActive = types.BoolPointerValue(in.IsActive)
	} else if m.IsActive.IsUnknown() {
		m.IsActive = types.BoolNull()
	}

	details, diags := applyAlertIntegrationDetails(ctx, in.Details)
	if diags.HasError() {
		return diags
	}
	m.Details = details
	return diags
}

func (r *alertIntegrationResource) findByID(ctx context.Context, enterpriseID, id int64) (*sdk.AlertserviceIntegration, bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	out, httpResp, err := r.pd.api.DefaultAPI.V2IntegrationGetallEnterpriseIdGet(ctx, enterpriseID).Authorization(r.pd.token).Execute()
	closeBody(httpResp)
	if err != nil {
		diags.AddError("Unable to list alert integrations", apiErrorDetail(err))
		return nil, false, diags
	}
	if out == nil {
		return nil, false, diags
	}
	for i := range out.Integrations {
		if out.Integrations[i].Id != nil && *out.Integrations[i].Id == id {
			return &out.Integrations[i], true, diags
		}
	}
	return nil, false, diags
}

// findByNickName locates a just-created integration by nick_name rather than
// the id V2IntegrationPost's response reports — that id has been observed
// pointing at a pre-existing, unrelated integration instead of the new one,
// so it can't be trusted immediately after create.
func (r *alertIntegrationResource) findByNickName(ctx context.Context, enterpriseID int64, nickName string) (*sdk.AlertserviceIntegration, diag.Diagnostics) {
	var diags diag.Diagnostics
	out, httpResp, err := r.pd.api.DefaultAPI.V2IntegrationGetallEnterpriseIdGet(ctx, enterpriseID).Authorization(r.pd.token).Execute()
	closeBody(httpResp)
	if err != nil {
		diags.AddError("Unable to list alert integrations", apiErrorDetail(err))
		return nil, diags
	}
	if out == nil {
		diags.AddError("Unable to find created alert integration", "integration list came back empty")
		return nil, diags
	}
	var matches []*sdk.AlertserviceIntegration
	for i := range out.Integrations {
		if out.Integrations[i].NickName != nil && *out.Integrations[i].NickName == nickName {
			matches = append(matches, &out.Integrations[i])
		}
	}
	switch len(matches) {
	case 1:
		return matches[0], diags
	case 0:
		diags.AddError("Unable to find created alert integration", "no integration in the list matched the submitted nick_name")
		return nil, diags
	default:
		diags.AddError(
			"Ambiguous alert integration lookup",
			fmt.Sprintf("%d integrations matched nick_name %q; the API's create response id can't be trusted, "+
				"so this provider cannot disambiguate. Use a unique nick_name.", len(matches), nickName),
		)
		return nil, diags
	}
}

func (r *alertIntegrationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan alertIntegrationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	details, diags := buildAlertIntegrationDetails(ctx, plan.Details)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := sdk.V2IntegrationPostRequest{
		IntegrationBody: sdk.AlertserviceCreateIntegrationBody{
			Enterprise:      plan.Enterprise.ValueInt64(),
			IntegrationType: plan.IntegrationType.ValueString(),
			NickName:        plan.NickName.ValueString(),
			Details:         details,
			IsActive:        plan.IsActive.ValueBoolPointer(),
		},
	}
	out, httpResp, err := r.pd.api.DefaultAPI.V2IntegrationPost(ctx).
		Authorization(r.pd.token).
		V2IntegrationPostRequest(body).
		Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create alert integration", apiErrorDetail(err))
		return
	}
	if out == nil || out.Integration == nil {
		resp.Diagnostics.AddError("Unable to create alert integration", "API returned an empty response")
		return
	}

	// The create response's id has been observed pointing at a pre-existing,
	// unrelated integration rather than the one just created, so it can't be
	// trusted — locate the new record by nick_name instead, same as
	// alert_notification's Create works around its own unreliable response.
	got, diags2 := r.findByNickName(ctx, plan.Enterprise.ValueInt64(), plan.NickName.ValueString())
	resp.Diagnostics.Append(diags2...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(plan.applyIntegration(ctx, got)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *alertIntegrationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state alertIntegrationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := parseInt64ID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid alert integration id", err.Error())
		return
	}

	got, found, diags := r.findByID(ctx, state.Enterprise.ValueInt64(), id)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(state.applyIntegration(ctx, got)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *alertIntegrationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan alertIntegrationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := parseInt64ID(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid alert integration id", err.Error())
		return
	}

	details, diags := buildAlertIntegrationDetails(ctx, plan.Details)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := sdk.V2IntegrationIntegrationIdPutRequest{
		IntegrationBody: sdk.AlertserviceUpdateIntegrationBody{
			Enterprise:      plan.Enterprise.ValueInt64(),
			IntegrationType: plan.IntegrationType.ValueString(),
			NickName:        plan.NickName.ValueString(),
			Details:         details,
			IsActive:        plan.IsActive.ValueBoolPointer(),
		},
	}
	_, httpResp, err := r.pd.api.DefaultAPI.V2IntegrationIntegrationIdPut(ctx, id).
		Authorization(r.pd.token).
		V2IntegrationIntegrationIdPutRequest(body).
		Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update alert integration", apiErrorDetail(err))
		return
	}

	got, found, diags2 := r.findByID(ctx, plan.Enterprise.ValueInt64(), id)
	resp.Diagnostics.Append(diags2...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.Diagnostics.AddError("Unable to update alert integration", "integration no longer exists")
		return
	}

	resp.Diagnostics.Append(plan.applyIntegration(ctx, got)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *alertIntegrationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state alertIntegrationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := parseInt64ID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid alert integration id", err.Error())
		return
	}

	_, httpResp, err := r.pd.api.DefaultAPI.V2IntegrationIntegrationIdDelete(ctx, id).Authorization(r.pd.token).Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to delete alert integration", apiErrorDetail(err))
	}
}

// ImportState uses "<enterprise>:<id>" since Read must know the owning enterprise
// to call V2IntegrationGetallEnterpriseIdGet — there is no get-by-id-alone endpoint.
func (r *alertIntegrationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			`Expected "<enterprise>:<id>", e.g. "12345:67" — Read needs the owning enterprise id, which isn't derivable from the integration id alone.`,
		)
		return
	}
	enterprise, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", "enterprise must be an integer: "+err.Error())
		return
	}
	if _, err := strconv.ParseInt(parts[1], 10, 64); err != nil {
		resp.Diagnostics.AddError("Invalid import ID", "id must be an integer: "+err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("enterprise"), enterprise)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
}
