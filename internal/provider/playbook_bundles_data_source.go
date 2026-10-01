package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &playbookBundlesDataSource{}
	_ datasource.DataSourceWithConfigure = &playbookBundlesDataSource{}
)

func NewPlaybookBundlesDataSource() datasource.DataSource {
	return &playbookBundlesDataSource{}
}

type playbookBundlesDataSource struct {
	pd *providerData
}

var playbookBundleAttrTypes = map[string]attrType{
	"key":         types.StringType,
	"name":        types.StringType,
	"description": types.StringType,
}

type playbookBundleModel struct {
	Key         types.String `tfsdk:"key"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
}

type playbookBundlesDataSourceModel struct {
	Bundles types.List `tfsdk:"bundles"`
}

func (d *playbookBundlesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_playbook_bundles"
}

func (d *playbookBundlesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Automation Workflows catalog bundles (System, Network, Routing, ...). A bundle's key is the " +
			"bundle_key used by graphiant_playbook_config and the module-slot/template data sources.",
		Attributes: map[string]schema.Attribute{
			"bundles": schema.ListAttribute{
				Computed:    true,
				ElementType: types.ObjectType{AttrTypes: playbookBundleAttrTypes},
			},
		},
	}
}

func (d *playbookBundlesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.pd = configurePD(req.ProviderData, &resp.Diagnostics)
}

func (d *playbookBundlesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg playbookBundlesDataSourceModel

	out, httpResp, err := d.pd.api.DefaultAPI.V1SdkAutomationPlaybookBundlesGet(ctx).Authorization(d.pd.token).Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read playbook bundles", apiErrorDetail(err))
		return
	}

	models := make([]playbookBundleModel, 0)
	if out != nil {
		for _, b := range out.Bundles {
			models = append(models, playbookBundleModel{
				Key:         types.StringPointerValue(b.Key),
				Name:        types.StringPointerValue(b.Name),
				Description: types.StringPointerValue(b.Description),
			})
		}
	}
	list, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: playbookBundleAttrTypes}, models)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg.Bundles = list

	resp.Diagnostics.Append(resp.State.Set(ctx, &cfg)...)
}
