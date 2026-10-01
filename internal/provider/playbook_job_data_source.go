package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	sdk "github.com/Graphiant-Inc/graphiant-sdk-go"
)

var (
	_ datasource.DataSource              = &playbookJobDataSource{}
	_ datasource.DataSourceWithConfigure = &playbookJobDataSource{}
)

func NewPlaybookJobDataSource() datasource.DataSource {
	return &playbookJobDataSource{}
}

type playbookJobDataSource struct {
	pd *providerData
}

func (d *playbookJobDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_playbook_job"
}

func (d *playbookJobDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	attrs := make(map[string]schema.Attribute, len(playbookJobAttrTypes))
	for name, t := range playbookJobAttrTypes {
		switch t {
		case types.Int64Type:
			attrs[name] = schema.Int64Attribute{Computed: true}
		case types.BoolType:
			attrs[name] = schema.BoolAttribute{Computed: true}
		default:
			attrs[name] = schema.StringAttribute{Computed: true}
		}
	}
	exactlyOne := stringvalidator.ExactlyOneOf(path.MatchRoot("job_id"), path.MatchRoot("config_id"))
	attrs["job_id"] = schema.StringAttribute{
		Optional:    true,
		Computed:    true,
		Description: "Job to look up. Exactly one of job_id or config_id is required.",
		Validators:  []validator.String{exactlyOne},
	}
	attrs["config_id"] = schema.StringAttribute{
		Optional:    true,
		Computed:    true,
		Description: "Look up the latest dry-run/pipeline job for this config instead of a specific job.",
		Validators:  []validator.String{exactlyOne},
	}

	resp.Schema = schema.Schema{
		Description: "A single Automation Workflows playbook job, by job id or as the latest job for a config. " +
			"Timestamps are RFC 3339 (UTC).",
		Attributes: attrs,
	}
}

func (d *playbookJobDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.pd = configurePD(req.ProviderData, &resp.Diagnostics)
}

func (d *playbookJobDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg playbookJobModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var job *sdk.SdkAutomationPlaybookJob
	if !cfg.JobID.IsNull() {
		j, found, err := getPlaybookJob(ctx, d.pd, cfg.JobID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Unable to read playbook job", err.Error())
			return
		}
		if !found {
			resp.Diagnostics.AddError("Playbook job not found", fmt.Sprintf("no job with id %q", cfg.JobID.ValueString()))
			return
		}
		job = j
	} else {
		out, httpResp, err := d.pd.api.DefaultAPI.V1SdkAutomationPlaybookConfigsConfigIdDryRunGet(ctx, cfg.ConfigID.ValueString()).
			Authorization(d.pd.token).
			Execute()
		closeBody(httpResp)
		if err != nil {
			resp.Diagnostics.AddError("Unable to read latest playbook job for config", apiErrorDetail(err))
			return
		}
		if out == nil || out.Job == nil {
			resp.Diagnostics.AddError("Playbook job not found", fmt.Sprintf("config %q has no jobs", cfg.ConfigID.ValueString()))
			return
		}
		job = out.Job
	}

	model := playbookJobModelFrom(job)
	if model.JobID.IsNull() {
		model.JobID = cfg.JobID
	}
	if model.ConfigID.IsNull() {
		model.ConfigID = cfg.ConfigID
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &model)...)
}
