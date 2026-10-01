package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &playbookJobsDataSource{}
	_ datasource.DataSourceWithConfigure = &playbookJobsDataSource{}
)

func NewPlaybookJobsDataSource() datasource.DataSource {
	return &playbookJobsDataSource{}
}

type playbookJobsDataSource struct {
	pd *providerData
}

type playbookJobsDataSourceModel struct {
	NameContains types.String `tfsdk:"name_contains"`
	Jobs         types.List   `tfsdk:"jobs"`
}

func (d *playbookJobsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_playbook_jobs"
}

func (d *playbookJobsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Automation Workflows playbook job history, for runs started from Terraform, the Portal, " +
			"the API or AI agents alike. The jobs endpoint has no paging parameters, so only the first page the " +
			"API returns is available; a warning is emitted when the API reports more. Narrow the list with " +
			"name_contains, or use graphiant_playbook_job to look up a config's latest job.",
		Attributes: map[string]schema.Attribute{
			"name_contains": schema.StringAttribute{
				Optional:    true,
				Description: "Case-insensitive substring match on the owning config's name.",
			},
			"jobs": schema.ListAttribute{
				Computed:    true,
				ElementType: types.ObjectType{AttrTypes: playbookJobAttrTypes},
			},
		},
	}
}

func (d *playbookJobsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.pd = configurePD(req.ProviderData, &resp.Diagnostics)
}

func (d *playbookJobsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg playbookJobsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq := d.pd.api.DefaultAPI.V1SdkAutomationPlaybookJobsGet(ctx).Authorization(d.pd.token)
	if !cfg.NameContains.IsNull() {
		apiReq = apiReq.NameContains(cfg.NameContains.ValueString())
	}
	out, httpResp, err := apiReq.Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read playbook jobs", apiErrorDetail(err))
		return
	}

	models := make([]playbookJobModel, 0)
	if out != nil {
		for i := range out.Jobs {
			models = append(models, playbookJobModelFrom(&out.Jobs[i]))
		}
		if out.PageInfo.GetHasNextPage() {
			resp.Diagnostics.AddWarning("Playbook job list is truncated",
				fmt.Sprintf("The API returned %d jobs and reports more, but the jobs endpoint has no paging "+
					"parameters. Narrow the list with name_contains.", len(out.Jobs)))
		}
	}
	list, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: playbookJobAttrTypes}, models)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg.Jobs = list

	resp.Diagnostics.Append(resp.State.Set(ctx, &cfg)...)
}
