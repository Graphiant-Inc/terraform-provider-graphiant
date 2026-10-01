package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &playbookModuleSlotsDataSource{}
	_ datasource.DataSourceWithConfigure = &playbookModuleSlotsDataSource{}
)

func NewPlaybookModuleSlotsDataSource() datasource.DataSource {
	return &playbookModuleSlotsDataSource{}
}

type playbookModuleSlotsDataSource struct {
	pd *providerData
}

var playbookModuleSlotAttrTypes = map[string]attrType{
	"module_key":         types.StringType,
	"name":               types.StringType,
	"description":        types.StringType,
	"file_distinguisher": types.StringType,
	"is_required":        types.BoolType,
}

type playbookModuleSlotModel struct {
	ModuleKey         types.String `tfsdk:"module_key"`
	Name              types.String `tfsdk:"name"`
	Description       types.String `tfsdk:"description"`
	FileDistinguisher types.String `tfsdk:"file_distinguisher"`
	IsRequired        types.Bool   `tfsdk:"is_required"`
}

type playbookModuleSlotsDataSourceModel struct {
	BundleKey types.String `tfsdk:"bundle_key"`
	Modules   types.List   `tfsdk:"modules"`
}

func (d *playbookModuleSlotsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_playbook_module_slots"
}

func (d *playbookModuleSlotsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Module slots of an Automation Workflows catalog bundle: the module_key values a " +
			"graphiant_playbook_config's files can fill, and which of them are required.",
		Attributes: map[string]schema.Attribute{
			"bundle_key": schema.StringAttribute{
				Required:    true,
				Description: "Catalog bundle key, from graphiant_playbook_bundles.",
			},
			"modules": schema.ListAttribute{
				Computed:    true,
				ElementType: types.ObjectType{AttrTypes: playbookModuleSlotAttrTypes},
			},
		},
	}
}

func (d *playbookModuleSlotsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.pd = configurePD(req.ProviderData, &resp.Diagnostics)
}

func (d *playbookModuleSlotsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg playbookModuleSlotsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, httpResp, err := d.pd.api.DefaultAPI.V1SdkAutomationPlaybookModuleSlotsGet(ctx).
		Authorization(d.pd.token).
		BundleKey(cfg.BundleKey.ValueString()).
		Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read playbook module slots", apiErrorDetail(err))
		return
	}

	models := make([]playbookModuleSlotModel, 0)
	if out != nil {
		for _, m := range out.Modules {
			models = append(models, playbookModuleSlotModel{
				ModuleKey:         types.StringPointerValue(m.ModuleKey),
				Name:              types.StringPointerValue(m.Name),
				Description:       types.StringPointerValue(m.Description),
				FileDistinguisher: types.StringPointerValue(m.FileDistinguisher),
				IsRequired:        types.BoolPointerValue(m.IsRequired),
			})
		}
	}
	list, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: playbookModuleSlotAttrTypes}, models)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg.Modules = list

	resp.Diagnostics.Append(resp.State.Set(ctx, &cfg)...)
}
