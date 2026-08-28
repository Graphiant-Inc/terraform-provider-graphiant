package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	sdk "github.com/Graphiant-Inc/graphiant-sdk-go"
)

var (
	_ datasource.DataSource              = &routingPolicyDataSource{}
	_ datasource.DataSourceWithConfigure = &routingPolicyDataSource{}
)

func NewRoutingPolicyDataSource() datasource.DataSource {
	return &routingPolicyDataSource{}
}

type routingPolicyDataSource struct {
	pd *providerData
}

var routingPolicyAttrTypes = map[string]attrType{
	"id":             types.Int64Type,
	"global_id":      types.Int64Type,
	"name":           types.StringType,
	"description":    types.StringType,
	"attach_point":   types.StringType,
	"default_action": types.StringType,
	"status":         types.StringType,
}

type routingPolicyModel struct {
	ID            types.Int64  `tfsdk:"id"`
	GlobalID      types.Int64  `tfsdk:"global_id"`
	Name          types.String `tfsdk:"name"`
	Description   types.String `tfsdk:"description"`
	AttachPoint   types.String `tfsdk:"attach_point"`
	DefaultAction types.String `tfsdk:"default_action"`
	Status        types.String `tfsdk:"status"`
}

type routingPolicyDataSourceModel struct {
	IDs      types.List `tfsdk:"ids"`
	Policies types.List `tfsdk:"policies"`
}

func (d *routingPolicyDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_routing_policy"
}

func (d *routingPolicyDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Global routing policies, looked up by id. This is read-only: the API has no create/update/" +
			"delete endpoint for routing policies anywhere, only this lookup-by-id-list. Policy statement " +
			"match/action rules are not yet exposed — only top-level policy metadata.",
		Attributes: map[string]schema.Attribute{
			"ids": schema.ListAttribute{
				Required:    true,
				ElementType: types.Int64Type,
			},
			"policies": schema.ListAttribute{
				Computed:    true,
				ElementType: types.ObjectType{AttrTypes: routingPolicyAttrTypes},
			},
		},
	}
}

func (d *routingPolicyDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.pd = configurePD(req.ProviderData, &resp.Diagnostics)
}

func (d *routingPolicyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg routingPolicyDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var ids []int64
	resp.Diagnostics.Append(cfg.IDs.ElementsAs(ctx, &ids, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, httpResp, err := d.pd.api.DefaultAPI.V1GlobalRoutingPoliciesPost(ctx).
		Authorization(d.pd.token).
		V1GlobalRoutingPoliciesPostRequest(sdk.V1GlobalRoutingPoliciesPostRequest{Ids: ids}).
		Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read routing policies", apiErrorDetail(err))
		return
	}

	models := make([]routingPolicyModel, 0)
	if out != nil {
		for _, p := range out.RoutingPolicies {
			models = append(models, routingPolicyModel{
				ID:            types.Int64PointerValue(p.Id),
				GlobalID:      types.Int64PointerValue(p.GlobalId),
				Name:          types.StringPointerValue(p.Name),
				Description:   types.StringPointerValue(p.Description),
				AttachPoint:   types.StringPointerValue(p.AttachPoint),
				DefaultAction: types.StringPointerValue(p.DefaultAction),
				Status:        types.StringPointerValue(p.Status),
			})
		}
	}
	list, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: routingPolicyAttrTypes}, models)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg.Policies = list

	resp.Diagnostics.Append(resp.State.Set(ctx, &cfg)...)
}
