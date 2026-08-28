package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &domainCategoriesDataSource{}
	_ datasource.DataSourceWithConfigure = &domainCategoriesDataSource{}
)

func NewDomainCategoriesDataSource() datasource.DataSource {
	return &domainCategoriesDataSource{}
}

type domainCategoriesDataSource struct {
	pd *providerData
}

var domainCategoryAttrTypes = map[string]attrType{
	"id":          types.Int64Type,
	"name":        types.StringType,
	"description": types.StringType,
	"type":        types.StringType,
}

type domainCategoryModel struct {
	ID          types.Int64  `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Type        types.String `tfsdk:"type"`
}

type domainCategoriesDataSourceModel struct {
	DomainCategories types.List `tfsdk:"domain_categories"`
}

func (d *domainCategoriesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_domain_categories"
}

func (d *domainCategoriesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The platform-defined domain category catalog, used by graphiant_content_filter's rules[].domain_category_id.",
		Attributes: map[string]schema.Attribute{
			"domain_categories": schema.ListAttribute{
				Computed:    true,
				ElementType: types.ObjectType{AttrTypes: domainCategoryAttrTypes},
			},
		},
	}
}

func (d *domainCategoriesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.pd = configurePD(req.ProviderData, &resp.Diagnostics)
}

func (d *domainCategoriesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg domainCategoriesDataSourceModel

	out, httpResp, err := d.pd.api.DefaultAPI.V1GlobalDomainCategoriesGet(ctx).Authorization(d.pd.token).Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read domain categories", apiErrorDetail(err))
		return
	}

	models := make([]domainCategoryModel, 0)
	if out != nil {
		for _, c := range out.DomainCategories {
			models = append(models, domainCategoryModel{
				ID:          types.Int64PointerValue(c.Id),
				Name:        types.StringPointerValue(c.Name),
				Description: types.StringPointerValue(c.Description),
				Type:        types.StringPointerValue(c.Type),
			})
		}
	}
	list, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: domainCategoryAttrTypes}, models)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg.DomainCategories = list

	resp.Diagnostics.Append(resp.State.Set(ctx, &cfg)...)
}
