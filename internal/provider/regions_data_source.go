package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &regionsDataSource{}
	_ datasource.DataSourceWithConfigure = &regionsDataSource{}
)

func NewRegionsDataSource() datasource.DataSource {
	return &regionsDataSource{}
}

type regionsDataSource struct {
	pd *providerData
}

var regionAttrTypes = map[string]attrType{
	"id":              types.Int64Type,
	"name":            types.StringType,
	"region_iso_code": types.StringType,
	"unavailable":     types.BoolType,
}

type regionModel struct {
	ID            types.Int64  `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	RegionIsoCode types.String `tfsdk:"region_iso_code"`
	Unavailable   types.Bool   `tfsdk:"unavailable"`
}

type regionsDataSourceModel struct {
	Regions types.List `tfsdk:"regions"`
}

func (d *regionsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_regions"
}

func (d *regionsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Graphiant regions, used by graphiant_gateway/graphiant_public_vif's region_id and " +
			"graphiant_device_config's region. Coordinates are not yet exposed.",
		Attributes: map[string]schema.Attribute{
			"regions": schema.ListAttribute{
				Computed:    true,
				ElementType: types.ObjectType{AttrTypes: regionAttrTypes},
			},
		},
	}
}

func (d *regionsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.pd = configurePD(req.ProviderData, &resp.Diagnostics)
}

func (d *regionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg regionsDataSourceModel

	out, httpResp, err := d.pd.api.DefaultAPI.V1RegionsGet(ctx).Authorization(d.pd.token).Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read regions", apiErrorDetail(err))
		return
	}

	models := make([]regionModel, 0)
	if out != nil {
		for _, r := range out.Regions {
			models = append(models, regionModel{
				ID:            types.Int64PointerValue(intPtr32To64(r.Id)),
				Name:          types.StringPointerValue(r.Name),
				RegionIsoCode: types.StringPointerValue(r.RegionIsoCode),
				Unavailable:   types.BoolPointerValue(r.Unavailable),
			})
		}
	}
	list, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: regionAttrTypes}, models)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg.Regions = list

	resp.Diagnostics.Append(resp.State.Set(ctx, &cfg)...)
}
