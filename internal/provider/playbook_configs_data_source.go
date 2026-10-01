package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &playbookConfigsDataSource{}
	_ datasource.DataSourceWithConfigure = &playbookConfigsDataSource{}
)

func NewPlaybookConfigsDataSource() datasource.DataSource {
	return &playbookConfigsDataSource{}
}

type playbookConfigsDataSource struct {
	pd *providerData
}

var playbookConfigSummaryAttrTypes = map[string]attrType{
	"config_id":          types.StringType,
	"name":               types.StringType,
	"status":             types.StringType,
	"job_status":         types.StringType,
	"run_id":             types.StringType,
	"run_number":         types.Int64Type,
	"created_by_user_id": types.StringType,
	"started_by_user_id": types.StringType,
	"started_at":         types.StringType,
	"updated_at":         types.StringType,
}

type playbookConfigSummaryModel struct {
	ConfigID        types.String `tfsdk:"config_id"`
	Name            types.String `tfsdk:"name"`
	Status          types.String `tfsdk:"status"`
	JobStatus       types.String `tfsdk:"job_status"`
	RunID           types.String `tfsdk:"run_id"`
	RunNumber       types.Int64  `tfsdk:"run_number"`
	CreatedByUserID types.String `tfsdk:"created_by_user_id"`
	StartedByUserID types.String `tfsdk:"started_by_user_id"`
	StartedAt       types.String `tfsdk:"started_at"`
	UpdatedAt       types.String `tfsdk:"updated_at"`
}

type playbookConfigsDataSourceModel struct {
	NameContains types.String `tfsdk:"name_contains"`
	Statuses     types.List   `tfsdk:"statuses"`
	Configs      types.List   `tfsdk:"configs"`
}

func (d *playbookConfigsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_playbook_configs"
}

func (d *playbookConfigsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Automation Workflows playbook configs (the Portal's Pending Executions table), with each " +
			"config's latest job status. All pages are fetched.",
		Attributes: map[string]schema.Attribute{
			"name_contains": schema.StringAttribute{
				Optional:    true,
				Description: "Case-insensitive substring match on config name.",
			},
			"statuses": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Only return configs in these statuses (e.g. STAGED, PAUSED).",
			},
			"configs": schema.ListAttribute{
				Computed:    true,
				ElementType: types.ObjectType{AttrTypes: playbookConfigSummaryAttrTypes},
			},
		},
	}
}

func (d *playbookConfigsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.pd = configurePD(req.ProviderData, &resp.Diagnostics)
}

func (d *playbookConfigsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg playbookConfigsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var statuses []string
	if !cfg.Statuses.IsNull() {
		resp.Diagnostics.Append(cfg.Statuses.ElementsAs(ctx, &statuses, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	models := make([]playbookConfigSummaryModel, 0)
	after := ""
	for {
		apiReq := d.pd.api.DefaultAPI.V1SdkAutomationPlaybookConfigsGet(ctx).Authorization(d.pd.token)
		if !cfg.NameContains.IsNull() {
			apiReq = apiReq.NameContains(cfg.NameContains.ValueString())
		}
		if len(statuses) > 0 {
			apiReq = apiReq.Status(statuses)
		}
		if after != "" {
			apiReq = apiReq.After(after)
		}
		out, httpResp, err := apiReq.Execute()
		closeBody(httpResp)
		if err != nil {
			resp.Diagnostics.AddError("Unable to read playbook configs", apiErrorDetail(err))
			return
		}
		if out == nil {
			break
		}
		for _, c := range out.Configs {
			models = append(models, playbookConfigSummaryModel{
				ConfigID:        types.StringPointerValue(c.ConfigId),
				Name:            types.StringPointerValue(c.Name),
				Status:          types.StringPointerValue(c.Status),
				JobStatus:       types.StringPointerValue(c.JobStatus),
				RunID:           types.StringPointerValue(c.RunId),
				RunNumber:       types.Int64PointerValue(intPtr32To64(c.RunNumber)),
				CreatedByUserID: types.StringPointerValue(c.CreatedByUserId),
				StartedByUserID: types.StringPointerValue(c.StartedByUserId),
				StartedAt:       timestampValue(c.StartedAt),
				UpdatedAt:       timestampValue(c.UpdatedAt),
			})
		}
		// Stop unless the API says there's another page and gives a new cursor to reach it.
		if out.PageInfo == nil || !out.PageInfo.GetHasNextPage() {
			break
		}
		next := out.PageInfo.GetEndCursor()
		if next == "" || next == after {
			break
		}
		after = next
	}

	list, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: playbookConfigSummaryAttrTypes}, models)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg.Configs = list

	resp.Diagnostics.Append(resp.State.Set(ctx, &cfg)...)
}
