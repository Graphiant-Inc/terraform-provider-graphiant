package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &assuranceFlexAlgosDataSource{}
	_ datasource.DataSourceWithConfigure = &assuranceFlexAlgosDataSource{}
)

func NewAssuranceFlexAlgosDataSource() datasource.DataSource {
	return &assuranceFlexAlgosDataSource{}
}

type assuranceFlexAlgosDataSource struct {
	pd *providerData
}

var flexAlgoAttrTypes = map[string]attrType{
	"id":          types.Int64Type,
	"name":        types.StringType,
	"description": types.StringType,
}

type flexAlgoModel struct {
	ID          types.Int64  `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
}

type assuranceFlexAlgosDataSourceModel struct {
	FlexAlgos types.List `tfsdk:"flex_algos"`
}

func (d *assuranceFlexAlgosDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_assurance_flex_algos"
}

func (d *assuranceFlexAlgosDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The platform-defined flex algo reference list used by graphiant_assurance_global's flex_algo attribute.",
		Attributes: map[string]schema.Attribute{
			"flex_algos": schema.ListAttribute{
				Computed:    true,
				ElementType: types.ObjectType{AttrTypes: flexAlgoAttrTypes},
			},
		},
	}
}

func (d *assuranceFlexAlgosDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.pd = configurePD(req.ProviderData, &resp.Diagnostics)
}

func (d *assuranceFlexAlgosDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg assuranceFlexAlgosDataSourceModel

	out, httpResp, err := d.pd.api.DefaultAPI.V1DataAssuranceFlexAlgosGet(ctx).Authorization(d.pd.token).Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read flex algos", apiErrorDetail(err))
		return
	}

	models := make([]flexAlgoModel, 0)
	if out != nil {
		for _, e := range out.Entries {
			models = append(models, flexAlgoModel{
				ID:          types.Int64PointerValue(intPtr32To64(e.Id)),
				Name:        types.StringPointerValue(e.Name),
				Description: types.StringPointerValue(e.Description),
			})
		}
	}
	list, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: flexAlgoAttrTypes}, models)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg.FlexAlgos = list

	resp.Diagnostics.Append(resp.State.Set(ctx, &cfg)...)
}
