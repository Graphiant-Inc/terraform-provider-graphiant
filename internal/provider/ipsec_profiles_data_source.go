package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &ipsecProfilesDataSource{}
	_ datasource.DataSourceWithConfigure = &ipsecProfilesDataSource{}
)

func NewIpsecProfilesDataSource() datasource.DataSource {
	return &ipsecProfilesDataSource{}
}

type ipsecProfilesDataSource struct {
	pd *providerData
}

var ipsecProfileAttrTypes = map[string]attrType{
	"id":                 types.Int64Type,
	"ipsec_profile_name": types.StringType,
	"count":              types.Int64Type,
}

type ipsecProfileModel struct {
	ID               types.Int64  `tfsdk:"id"`
	IpsecProfileName types.String `tfsdk:"ipsec_profile_name"`
	Count            types.Int64  `tfsdk:"count"`
}

type ipsecProfilesDataSourceModel struct {
	IpsecProfiles types.List `tfsdk:"ipsec_profiles"`
}

func (d *ipsecProfilesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ipsec_profiles"
}

func (d *ipsecProfilesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Global IPsec profiles and how many gateway configs currently reference each one, used by graphiant_gateway's ipsec_gateway.vpn_profile.",
		Attributes: map[string]schema.Attribute{
			"ipsec_profiles": schema.ListAttribute{
				Computed:    true,
				ElementType: types.ObjectType{AttrTypes: ipsecProfileAttrTypes},
			},
		},
	}
}

func (d *ipsecProfilesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.pd = configurePD(req.ProviderData, &resp.Diagnostics)
}

func (d *ipsecProfilesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg ipsecProfilesDataSourceModel

	out, httpResp, err := d.pd.api.DefaultAPI.V1GlobalIpsecProfileGet(ctx).Authorization(d.pd.token).Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read IPsec profiles", apiErrorDetail(err))
		return
	}

	models := make([]ipsecProfileModel, 0)
	if out != nil {
		for _, p := range out.IpsecProfiles {
			models = append(models, ipsecProfileModel{
				ID:               types.Int64PointerValue(p.Id),
				IpsecProfileName: types.StringPointerValue(p.IpsecProfileName),
				Count:            types.Int64PointerValue(intPtr32To64(p.Count)),
			})
		}
	}
	list, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: ipsecProfileAttrTypes}, models)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg.IpsecProfiles = list

	resp.Diagnostics.Append(resp.State.Set(ctx, &cfg)...)
}
