package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	sdk "github.com/Graphiant-Inc/graphiant-sdk-go"
)

var (
	_ resource.Resource                = &alertNotificationResource{}
	_ resource.ResourceWithConfigure   = &alertNotificationResource{}
	_ resource.ResourceWithImportState = &alertNotificationResource{}
)

func NewAlertNotificationResource() resource.Resource {
	return &alertNotificationResource{}
}

type alertNotificationResource struct {
	pd *providerData
}

type alertNotificationResourceModel struct {
	ID               types.String `tfsdk:"id"`
	NotificationName types.String `tfsdk:"notification_name"`
	RuleIDList       types.List   `tfsdk:"rule_id_list"`
	Description      types.String `tfsdk:"description"`
	Duration         types.String `tfsdk:"duration"`
	Enabled          types.Bool   `tfsdk:"enabled"`
	Frequency        types.Int64  `tfsdk:"frequency"`
	MessageBody      types.String `tfsdk:"message_body"`
	OpsgenieList     types.List   `tfsdk:"opsgenie_list"`
	OpsrampList      types.List   `tfsdk:"opsramp_list"`
	PagerdutyList    types.List   `tfsdk:"pagerduty_list"`
	RecipientList    types.List   `tfsdk:"recipient_list"`
	TeamsList        types.List   `tfsdk:"teams_list"`
	WebhookURLList   types.List   `tfsdk:"webhook_url_list"`
	AlertType        types.String `tfsdk:"alert_type"`
	RuleID           types.String `tfsdk:"rule_id"`
}

func (r *alertNotificationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_alert_notification"
}

func (r *alertNotificationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Alert notification routing config: binds a set of rules to delivery channels/recipients. " +
			"Create has no response body, so the new notification is located afterward by matching " +
			"notification_name in the list — this will fail ambiguously if another notification with the same " +
			"name already exists. rule_id_list is force-new: the update endpoint has no field to change which " +
			"rules a notification is bound to.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"notification_name": schema.StringAttribute{
				Required: true,
			},
			"rule_id_list": schema.ListAttribute{
				Required:      true,
				ElementType:   types.StringType,
				PlanModifiers: []planmodifier.List{listplanmodifier.RequiresReplace()},
			},
			"description": schema.StringAttribute{
				Optional: true,
			},
			"duration": schema.StringAttribute{
				Optional: true,
			},
			"enabled": schema.BoolAttribute{
				Optional: true,
				Computed: true,
			},
			"frequency": schema.Int64Attribute{
				Optional: true,
			},
			"message_body": schema.StringAttribute{
				Optional: true,
			},
			"opsgenie_list":    schema.ListAttribute{Optional: true, ElementType: types.StringType},
			"opsramp_list":     schema.ListAttribute{Optional: true, ElementType: types.StringType},
			"pagerduty_list":   schema.ListAttribute{Optional: true, ElementType: types.StringType},
			"recipient_list":   schema.ListAttribute{Optional: true, ElementType: types.StringType},
			"teams_list":       schema.ListAttribute{Optional: true, ElementType: types.StringType},
			"webhook_url_list": schema.ListAttribute{Optional: true, ElementType: types.StringType},
			"alert_type": schema.StringAttribute{
				Computed: true,
			},
			"rule_id": schema.StringAttribute{
				Computed:    true,
				Description: "The rule id this notification reports as bound to (only one is surfaced by the read API even when rule_id_list has more).",
			},
		},
	}
}

func (r *alertNotificationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.pd = configurePD(req.ProviderData, &resp.Diagnostics)
}

func (m *alertNotificationResourceModel) buildBody(ctx context.Context) (sdk.AlertserviceNotificationBody, diag.Diagnostics) {
	body := sdk.AlertserviceNotificationBody{
		NotificationName: m.NotificationName.ValueStringPointer(),
		Description:      m.Description.ValueStringPointer(),
		Duration:         m.Duration.ValueStringPointer(),
		Enabled:          m.Enabled.ValueBoolPointer(),
		Frequency:        m.Frequency.ValueInt64Pointer(),
		MessageBody:      m.MessageBody.ValueStringPointer(),
	}
	var diags diag.Diagnostics
	for _, pair := range []struct {
		list *[]string
		src  types.List
	}{
		{&body.OpsgenieList, m.OpsgenieList},
		{&body.OpsrampList, m.OpsrampList},
		{&body.PagerdutyList, m.PagerdutyList},
		{&body.RecipientList, m.RecipientList},
		{&body.TeamsList, m.TeamsList},
		{&body.WebhookUrlList, m.WebhookURLList},
	} {
		if !pair.src.IsNull() && !pair.src.IsUnknown() {
			diags.Append(pair.src.ElementsAs(ctx, pair.list, false)...)
		}
	}
	return body, diags
}

func (m *alertNotificationResourceModel) applyRecord(ctx context.Context, rec *sdk.AlertserviceNotificationRecord) diag.Diagnostics {
	var diags diag.Diagnostics
	if rec.NotificationId != nil {
		m.ID = types.StringValue(*rec.NotificationId)
	}
	m.NotificationName = types.StringPointerValue(rec.Name)
	m.AlertType = types.StringPointerValue(rec.AlertType)
	m.RuleID = types.StringPointerValue(rec.RuleId)

	if rec.NotificationBody == nil {
		return diags
	}
	b := rec.NotificationBody
	m.Description = types.StringPointerValue(b.Description)
	m.Duration = types.StringPointerValue(b.Duration)
	m.Enabled = types.BoolPointerValue(b.Enabled)
	m.Frequency = types.Int64PointerValue(b.Frequency)
	m.MessageBody = types.StringPointerValue(b.MessageBody)

	for _, pair := range []struct {
		dst *types.List
		src []string
	}{
		{&m.OpsgenieList, b.OpsgenieList},
		{&m.OpsrampList, b.OpsrampList},
		{&m.PagerdutyList, b.PagerdutyList},
		{&m.RecipientList, b.RecipientList},
		{&m.TeamsList, b.TeamsList},
		{&m.WebhookURLList, b.WebhookUrlList},
	} {
		list, d := types.ListValueFrom(ctx, types.StringType, pair.src)
		diags.Append(d...)
		*pair.dst = list
	}
	return diags
}

func (r *alertNotificationResource) findByID(ctx context.Context, id string) (*sdk.AlertserviceNotificationRecord, bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	body := sdk.V2NotificationlistPostRequest{}
	out, httpResp, err := r.pd.api.DefaultAPI.V2NotificationlistPost(ctx).
		Authorization(r.pd.token).
		V2NotificationlistPostRequest(body).
		Execute()
	closeBody(httpResp)
	if err != nil {
		diags.AddError("Unable to list alert notifications", apiErrorDetail(err))
		return nil, false, diags
	}
	if out == nil {
		return nil, false, diags
	}
	for i := range out.NotificationList {
		if out.NotificationList[i].NotificationId != nil && *out.NotificationList[i].NotificationId == id {
			return &out.NotificationList[i], true, diags
		}
	}
	return nil, false, diags
}

func (r *alertNotificationResource) findByName(ctx context.Context, name string) (*sdk.AlertserviceNotificationRecord, diag.Diagnostics) {
	var diags diag.Diagnostics
	body := sdk.V2NotificationlistPostRequest{}
	out, httpResp, err := r.pd.api.DefaultAPI.V2NotificationlistPost(ctx).
		Authorization(r.pd.token).
		V2NotificationlistPostRequest(body).
		Execute()
	closeBody(httpResp)
	if err != nil {
		diags.AddError("Unable to list alert notifications", apiErrorDetail(err))
		return nil, diags
	}
	if out == nil {
		diags.AddError("Unable to find created alert notification", "notification list came back empty")
		return nil, diags
	}
	var matches []*sdk.AlertserviceNotificationRecord
	for i := range out.NotificationList {
		if out.NotificationList[i].Name != nil && *out.NotificationList[i].Name == name {
			matches = append(matches, &out.NotificationList[i])
		}
	}
	switch len(matches) {
	case 1:
		return matches[0], diags
	case 0:
		diags.AddError("Unable to find created alert notification", "no notification in the list matched the submitted notification_name")
		return nil, diags
	default:
		diags.AddError(
			"Ambiguous alert notification lookup",
			fmt.Sprintf("%d notifications matched notification_name %q; the API's create endpoint returns no ID, "+
				"so this provider cannot disambiguate. Use a unique notification_name.", len(matches), name),
		)
		return nil, diags
	}
}

func (r *alertNotificationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan alertNotificationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body, diags := plan.buildBody(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var ruleIDs []string
	resp.Diagnostics.Append(plan.RuleIDList.ElementsAs(ctx, &ruleIDs, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createBody := sdk.V2NotificationCreatePostRequest{NotificationBody: body, RuleIdList: ruleIDs}
	_, httpResp, err := r.pd.api.DefaultAPI.V2NotificationCreatePost(ctx).
		Authorization(r.pd.token).
		V2NotificationCreatePostRequest(createBody).
		Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create alert notification", apiErrorDetail(err))
		return
	}

	rec, diags2 := r.findByName(ctx, plan.NotificationName.ValueString())
	resp.Diagnostics.Append(diags2...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(plan.applyRecord(ctx, rec)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *alertNotificationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state alertNotificationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rec, found, diags := r.findByID(ctx, state.ID.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(state.applyRecord(ctx, rec)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *alertNotificationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan alertNotificationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body, diags := plan.buildBody(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateBody := sdk.V2NotificationUpdatePostRequest{
		NotificationBody:   body,
		NotificationIdList: []string{plan.ID.ValueString()},
	}
	_, httpResp, err := r.pd.api.DefaultAPI.V2NotificationUpdatePost(ctx).
		Authorization(r.pd.token).
		V2NotificationUpdatePostRequest(updateBody).
		Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update alert notification", apiErrorDetail(err))
		return
	}

	rec, found, diags2 := r.findByID(ctx, plan.ID.ValueString())
	resp.Diagnostics.Append(diags2...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.Diagnostics.AddError("Unable to update alert notification", "notification no longer exists")
		return
	}

	resp.Diagnostics.Append(plan.applyRecord(ctx, rec)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *alertNotificationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state alertNotificationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := sdk.V2NotificationDeletePostRequest{NotificationIdList: []string{state.ID.ValueString()}}
	_, httpResp, err := r.pd.api.DefaultAPI.V2NotificationDeletePost(ctx).
		Authorization(r.pd.token).
		V2NotificationDeletePostRequest(body).
		Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to delete alert notification", apiErrorDetail(err))
	}
}

func (r *alertNotificationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
