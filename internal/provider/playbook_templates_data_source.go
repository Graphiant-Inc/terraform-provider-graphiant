package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &playbookTemplatesDataSource{}
	_ datasource.DataSourceWithConfigure = &playbookTemplatesDataSource{}
)

func NewPlaybookTemplatesDataSource() datasource.DataSource {
	return &playbookTemplatesDataSource{}
}

type playbookTemplatesDataSource struct {
	pd *providerData
}

var playbookTemplateFileAttrTypes = map[string]attrType{
	"module_key":  types.StringType,
	"filename":    types.StringType,
	"source_path": types.StringType,
	"content":     types.StringType,
}

type playbookTemplateFileModel struct {
	ModuleKey  types.String `tfsdk:"module_key"`
	Filename   types.String `tfsdk:"filename"`
	SourcePath types.String `tfsdk:"source_path"`
	Content    types.String `tfsdk:"content"`
}

type playbookTemplatesDataSourceModel struct {
	BundleKey         types.String `tfsdk:"bundle_key"`
	ModuleKeys        types.List   `tfsdk:"module_keys"`
	CollectionVersion types.String `tfsdk:"collection_version"`
	Files             types.List   `tfsdk:"files"`
}

func (d *playbookTemplatesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_playbook_templates"
}

func (d *playbookTemplatesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Starter playbook and sample module config templates for an Automation Workflows bundle, " +
			"as published by the graphiant-playbooks collection version the service runs.",
		Attributes: map[string]schema.Attribute{
			"bundle_key": schema.StringAttribute{
				Required:    true,
				Description: "Catalog bundle key, from graphiant_playbook_bundles.",
			},
			"module_keys": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Narrow the response to these module keys; omit for the whole bundle (playbook plus every module sample).",
			},
			"collection_version": schema.StringAttribute{
				Computed:    true,
				Description: "graphiant-playbooks collection version the templates come from.",
			},
			"files": schema.ListAttribute{
				Computed:    true,
				ElementType: types.ObjectType{AttrTypes: playbookTemplateFileAttrTypes},
			},
		},
	}
}

func (d *playbookTemplatesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.pd = configurePD(req.ProviderData, &resp.Diagnostics)
}

func (d *playbookTemplatesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg playbookTemplatesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq := d.pd.api.DefaultAPI.V1SdkAutomationPlaybookTemplatesGet(ctx).
		Authorization(d.pd.token).
		BundleKey(cfg.BundleKey.ValueString())
	if !cfg.ModuleKeys.IsNull() {
		var moduleKeys []string
		resp.Diagnostics.Append(cfg.ModuleKeys.ElementsAs(ctx, &moduleKeys, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		apiReq = apiReq.ModuleKeys(moduleKeys)
	}
	out, httpResp, err := apiReq.Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read playbook templates", apiErrorDetail(err))
		return
	}

	models := make([]playbookTemplateFileModel, 0)
	cfg.CollectionVersion = types.StringNull()
	if out != nil {
		cfg.CollectionVersion = types.StringPointerValue(out.CollectionVersion)
		for _, f := range out.Files {
			models = append(models, playbookTemplateFileModel{
				ModuleKey:  types.StringPointerValue(f.ModuleKey),
				Filename:   types.StringPointerValue(f.Filename),
				SourcePath: types.StringPointerValue(f.SourcePath),
				Content:    types.StringPointerValue(f.Content),
			})
		}
	}
	list, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: playbookTemplateFileAttrTypes}, models)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg.Files = list

	resp.Diagnostics.Append(resp.State.Set(ctx, &cfg)...)
}
