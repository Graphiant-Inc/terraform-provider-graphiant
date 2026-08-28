package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	sdk "github.com/Graphiant-Inc/graphiant-sdk-go"
)

var (
	_ datasource.DataSource              = &prefixSetDataSource{}
	_ datasource.DataSourceWithConfigure = &prefixSetDataSource{}
)

func NewPrefixSetDataSource() datasource.DataSource {
	return &prefixSetDataSource{}
}

type prefixSetDataSource struct {
	pd *providerData
}

var prefixSetAttrTypes = map[string]attrType{
	"id":           types.Int64Type,
	"global_id":    types.Int64Type,
	"name":         types.StringType,
	"description":  types.StringType,
	"mode":         types.StringType,
	"status":       types.StringType,
	"policy_count": types.Int64Type,
}

type prefixSetModel struct {
	ID          types.Int64  `tfsdk:"id"`
	GlobalID    types.Int64  `tfsdk:"global_id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Mode        types.String `tfsdk:"mode"`
	Status      types.String `tfsdk:"status"`
	PolicyCount types.Int64  `tfsdk:"policy_count"`
}

type prefixSetDataSourceModel struct {
	IDs        types.List `tfsdk:"ids"`
	PrefixSets types.List `tfsdk:"prefix_sets"`
}

func (d *prefixSetDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_prefix_set"
}

func (d *prefixSetDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Global prefix sets, looked up by id. This is read-only: the API has no create/update/" +
			"delete endpoint for prefix sets anywhere, only this lookup-by-id-list. Individual prefix entries " +
			"and attached-policy references are not yet exposed — only top-level prefix set metadata.",
		Attributes: map[string]schema.Attribute{
			"ids": schema.ListAttribute{
				Required:    true,
				ElementType: types.Int64Type,
			},
			"prefix_sets": schema.ListAttribute{
				Computed:    true,
				ElementType: types.ObjectType{AttrTypes: prefixSetAttrTypes},
			},
		},
	}
}

func (d *prefixSetDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.pd = configurePD(req.ProviderData, &resp.Diagnostics)
}

func (d *prefixSetDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg prefixSetDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var ids []int64
	resp.Diagnostics.Append(cfg.IDs.ElementsAs(ctx, &ids, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, httpResp, err := d.pd.api.DefaultAPI.V1GlobalPrefixSetsPost(ctx).
		Authorization(d.pd.token).
		V1GlobalPrefixSetsPostRequest(sdk.V1GlobalPrefixSetsPostRequest{Ids: ids}).
		Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read prefix sets", apiErrorDetail(err))
		return
	}

	models := make([]prefixSetModel, 0)
	if out != nil {
		for _, p := range out.PrefixSets {
			models = append(models, prefixSetModel{
				ID:          types.Int64PointerValue(p.Id),
				GlobalID:    types.Int64PointerValue(p.GlobalId),
				Name:        types.StringPointerValue(p.Name),
				Description: types.StringPointerValue(p.Description),
				Mode:        types.StringPointerValue(p.Mode),
				Status:      types.StringPointerValue(p.Status),
				PolicyCount: types.Int64PointerValue(intPtr32To64(p.PolicyCount)),
			})
		}
	}
	list, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: prefixSetAttrTypes}, models)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg.PrefixSets = list

	resp.Diagnostics.Append(resp.State.Set(ctx, &cfg)...)
}
