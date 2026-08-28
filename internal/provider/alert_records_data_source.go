package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	sdk "github.com/Graphiant-Inc/graphiant-sdk-go"
)

var (
	_ datasource.DataSource              = &alertRecordsDataSource{}
	_ datasource.DataSourceWithConfigure = &alertRecordsDataSource{}
)

func NewAlertRecordsDataSource() datasource.DataSource {
	return &alertRecordsDataSource{}
}

type alertRecordsDataSource struct {
	pd *providerData
}

var alertRecordAttrTypes = map[string]attrType{
	"alert_id":      types.StringType,
	"rule_id":       types.StringType,
	"entity":        types.StringType,
	"type":          types.StringType,
	"severity":      types.StringType,
	"status":        types.StringType,
	"plane":         types.StringType,
	"start_time":    types.Int64Type,
	"end_time":      types.Int64Type,
	"occurrences":   types.Int64Type,
	"allow_listed":  types.BoolType,
	"mute_listed":   types.BoolType,
	"enterprise_id": types.StringType,
	"device_id":     types.StringType,
	"site_id":       types.StringType,
}

type alertRecordModel struct {
	AlertID      types.String `tfsdk:"alert_id"`
	RuleID       types.String `tfsdk:"rule_id"`
	Entity       types.String `tfsdk:"entity"`
	Type         types.String `tfsdk:"type"`
	Severity     types.String `tfsdk:"severity"`
	Status       types.String `tfsdk:"status"`
	Plane        types.String `tfsdk:"plane"`
	StartTime    types.Int64  `tfsdk:"start_time"`
	EndTime      types.Int64  `tfsdk:"end_time"`
	Occurrences  types.Int64  `tfsdk:"occurrences"`
	AllowListed  types.Bool   `tfsdk:"allow_listed"`
	MuteListed   types.Bool   `tfsdk:"mute_listed"`
	EnterpriseID types.String `tfsdk:"enterprise_id"`
	DeviceID     types.String `tfsdk:"device_id"`
	SiteID       types.String `tfsdk:"site_id"`
}

type alertRecordsDataSourceModel struct {
	Alerts types.List `tfsdk:"alerts"`
}

func (d *alertRecordsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_alert_records"
}

func (d *alertRecordsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Current top-level (parent) alert records. Child alerts, acknowledgement fields, and a " +
			"caller-supplied historical time window are not yet exposed — this always reads the API's default " +
			"(unfiltered) current alert list.",
		Attributes: map[string]schema.Attribute{
			"alerts": schema.ListAttribute{
				Computed:    true,
				ElementType: types.ObjectType{AttrTypes: alertRecordAttrTypes},
			},
		},
	}
}

func (d *alertRecordsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.pd = configurePD(req.ProviderData, &resp.Diagnostics)
}

func (d *alertRecordsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg alertRecordsDataSourceModel

	out, httpResp, err := d.pd.api.DefaultAPI.V2ParentalertlistPost(ctx).
		Authorization(d.pd.token).
		V2ParentalertlistPostRequest(sdk.V2ParentalertlistPostRequest{}).
		Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read alert records", apiErrorDetail(err))
		return
	}

	models := make([]alertRecordModel, 0)
	if out != nil {
		for _, a := range out.AlertList {
			models = append(models, alertRecordModel{
				AlertID:      types.StringPointerValue(a.AlertId),
				RuleID:       types.StringPointerValue(a.RuleId),
				Entity:       types.StringPointerValue(a.Entity),
				Type:         types.StringPointerValue(a.Type),
				Severity:     types.StringPointerValue(a.Severity),
				Status:       types.StringPointerValue(a.Status),
				Plane:        types.StringPointerValue(a.Plane),
				StartTime:    types.Int64PointerValue(a.StartTime),
				EndTime:      types.Int64PointerValue(a.EndTime),
				Occurrences:  types.Int64PointerValue(a.Occurrences),
				AllowListed:  types.BoolPointerValue(a.AllowListed),
				MuteListed:   types.BoolPointerValue(a.MuteListed),
				EnterpriseID: types.StringPointerValue(a.EnterpriseId),
				DeviceID:     types.StringPointerValue(a.DeviceId),
				SiteID:       types.StringPointerValue(a.SiteId),
			})
		}
	}
	list, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: alertRecordAttrTypes}, models)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg.Alerts = list

	resp.Diagnostics.Append(resp.State.Set(ctx, &cfg)...)
}
