package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &alertRulesDataSource{}
	_ datasource.DataSourceWithConfigure = &alertRulesDataSource{}
)

func NewAlertRulesDataSource() datasource.DataSource {
	return &alertRulesDataSource{}
}

type alertRulesDataSource struct {
	pd *providerData
}

var alertRuleAttrTypes = map[string]attrType{
	"rule_id":     types.StringType,
	"rule_name":   types.StringType,
	"category":    types.StringType,
	"plane":       types.StringType,
	"priority":    types.StringType,
	"enabled":     types.BoolType,
	"alarm_set":   types.StringType,
	"alarm_clear": types.StringType,
	"allow_count": types.Int64Type,
}

type alertRuleModel struct {
	RuleID     types.String `tfsdk:"rule_id"`
	RuleName   types.String `tfsdk:"rule_name"`
	Category   types.String `tfsdk:"category"`
	Plane      types.String `tfsdk:"plane"`
	Priority   types.String `tfsdk:"priority"`
	Enabled    types.Bool   `tfsdk:"enabled"`
	AlarmSet   types.String `tfsdk:"alarm_set"`
	AlarmClear types.String `tfsdk:"alarm_clear"`
	AllowCount types.Int64  `tfsdk:"allow_count"`
}

type alertRulesDataSourceModel struct {
	Rules types.List `tfsdk:"rules"`
}

func (d *alertRulesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_alert_rules"
}

func (d *alertRulesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The fixed catalog of alert rules and their current enabled state. Rules are not created " +
			"via the API (there is no create/update/delete for rules, only this list and a bulk enable/disable " +
			"action which this provider doesn't yet expose).",
		Attributes: map[string]schema.Attribute{
			"rules": schema.ListAttribute{
				Computed:    true,
				ElementType: types.ObjectType{AttrTypes: alertRuleAttrTypes},
			},
		},
	}
}

func (d *alertRulesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.pd = configurePD(req.ProviderData, &resp.Diagnostics)
}

func (d *alertRulesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg alertRulesDataSourceModel

	out, httpResp, err := d.pd.api.DefaultAPI.V2RulelistPost(ctx).Authorization(d.pd.token).Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read alert rules", apiErrorDetail(err))
		return
	}

	models := make([]alertRuleModel, 0)
	if out != nil {
		for _, rule := range out.RuleList {
			models = append(models, alertRuleModel{
				RuleID:     types.StringPointerValue(rule.RuleId),
				RuleName:   types.StringPointerValue(rule.RuleName),
				Category:   types.StringPointerValue(rule.Category),
				Plane:      types.StringPointerValue(rule.Plane),
				Priority:   types.StringPointerValue(rule.Priority),
				Enabled:    types.BoolPointerValue(rule.Enabled),
				AlarmSet:   types.StringPointerValue(rule.AlarmSet),
				AlarmClear: types.StringPointerValue(rule.AlarmClear),
				AllowCount: types.Int64PointerValue(rule.AllowCount),
			})
		}
	}
	list, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: alertRuleAttrTypes}, models)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg.Rules = list

	resp.Diagnostics.Append(resp.State.Set(ctx, &cfg)...)
}
