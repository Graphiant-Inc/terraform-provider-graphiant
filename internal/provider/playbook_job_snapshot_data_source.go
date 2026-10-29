package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &playbookJobSnapshotDataSource{}
	_ datasource.DataSourceWithConfigure = &playbookJobSnapshotDataSource{}
)

func NewPlaybookJobSnapshotDataSource() datasource.DataSource {
	return &playbookJobSnapshotDataSource{}
}

type playbookJobSnapshotDataSource struct {
	pd *providerData
}

type playbookJobSnapshotDataSourceModel struct {
	JobID       types.String `tfsdk:"job_id"`
	ConfigID    types.String `tfsdk:"config_id"`
	BundleKey   types.String `tfsdk:"bundle_key"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Playbook    types.String `tfsdk:"playbook"`
	Tags        types.String `tfsdk:"tags"`
	Files       types.List   `tfsdk:"files"`
}

func (d *playbookJobSnapshotDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_playbook_job_snapshot"
}

func (d *playbookJobSnapshotDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The config inputs an Automation Workflows playbook job ran with, frozen when the job was created. " +
			"Unlike graphiant_playbook_config, this does not change when the config is edited later. " +
			"name, description, playbook and tags are empty for jobs created before the service started recording them.",
		Attributes: map[string]schema.Attribute{
			"job_id": schema.StringAttribute{
				Required: true,
			},
			"config_id": schema.StringAttribute{
				Computed:    true,
				Description: "Config that owned this run when the snapshot was taken.",
			},
			"bundle_key": schema.StringAttribute{
				Computed: true,
			},
			"name": schema.StringAttribute{
				Computed:    true,
				Description: "Config display name at job creation.",
			},
			"description": schema.StringAttribute{
				Computed:    true,
				Description: "Config notes at job creation.",
			},
			"playbook": schema.StringAttribute{
				Computed:    true,
				Description: "Playbook filename at job creation (e.g. network_bundle.yml).",
			},
			"tags": schema.StringAttribute{
				Computed:    true,
				Description: "Ansible --tags the job ran with; empty when no tags were applied.",
			},
			"files": schema.ListAttribute{
				Computed:    true,
				ElementType: types.ObjectType{AttrTypes: playbookModuleFileAttrTypes},
			},
		},
	}
}

func (d *playbookJobSnapshotDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.pd = configurePD(req.ProviderData, &resp.Diagnostics)
}

func (d *playbookJobSnapshotDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg playbookJobSnapshotDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, httpResp, err := d.pd.api.DefaultAPI.V1SdkAutomationPlaybookJobsJobIdSnapshotGet(ctx, cfg.JobID.ValueString()).
		Authorization(d.pd.token).
		Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read playbook job snapshot", apiErrorDetail(err))
		return
	}
	if out == nil || out.Snapshot == nil {
		resp.Diagnostics.AddError("Playbook job snapshot not found", "the API returned no snapshot for job "+cfg.JobID.ValueString())
		return
	}

	s := out.Snapshot
	cfg.ConfigID = types.StringPointerValue(s.ConfigId)
	cfg.BundleKey = types.StringPointerValue(s.BundleKey)
	cfg.Name = types.StringPointerValue(s.Name)
	cfg.Description = types.StringPointerValue(s.Description)
	cfg.Playbook = types.StringPointerValue(s.Playbook)
	cfg.Tags = types.StringPointerValue(s.Tags)

	files, diags := flattenPlaybookFiles(ctx, s.Files, types.ListNull(types.ObjectType{AttrTypes: playbookModuleFileAttrTypes}))
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg.Files = files

	resp.Diagnostics.Append(resp.State.Set(ctx, &cfg)...)
}
