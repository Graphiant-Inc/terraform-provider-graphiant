package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &playbookJobLogsDataSource{}
	_ datasource.DataSourceWithConfigure = &playbookJobLogsDataSource{}
)

func NewPlaybookJobLogsDataSource() datasource.DataSource {
	return &playbookJobLogsDataSource{}
}

type playbookJobLogsDataSource struct {
	pd *providerData
}

type playbookJobLogsDataSourceModel struct {
	JobID types.String `tfsdk:"job_id"`
	Phase types.String `tfsdk:"phase"`
	Lines types.List   `tfsdk:"lines"`
}

func (d *playbookJobLogsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_playbook_job_logs"
}

func (d *playbookJobLogsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Ansible logs for an Automation Workflows playbook job, with secrets masked by the service.",
		Attributes: map[string]schema.Attribute{
			"job_id": schema.StringAttribute{
				Required: true,
			},
			"phase": schema.StringAttribute{
				Optional:    true,
				Description: "Only return logs for this phase (dry_run, deploy or post_deploy_check). Omit for all phases.",
			},
			"lines": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (d *playbookJobLogsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.pd = configurePD(req.ProviderData, &resp.Diagnostics)
}

func (d *playbookJobLogsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg playbookJobLogsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq := d.pd.api.DefaultAPI.V1SdkAutomationPlaybookJobsJobIdLogsGet(ctx, cfg.JobID.ValueString()).Authorization(d.pd.token)
	if !cfg.Phase.IsNull() {
		apiReq = apiReq.Phase(cfg.Phase.ValueString())
	}
	out, httpResp, err := apiReq.Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read playbook job logs", apiErrorDetail(err))
		return
	}

	lines := make([]string, 0)
	if out != nil {
		lines = append(lines, out.Lines...)
	}
	list, diags := types.ListValueFrom(ctx, types.StringType, lines)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg.Lines = list

	resp.Diagnostics.Append(resp.State.Set(ctx, &cfg)...)
}
