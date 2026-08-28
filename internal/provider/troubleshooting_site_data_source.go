package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &troubleshootingSiteDataSource{}
	_ datasource.DataSourceWithConfigure = &troubleshootingSiteDataSource{}
)

func NewTroubleshootingSiteDataSource() datasource.DataSource {
	return &troubleshootingSiteDataSource{}
}

type troubleshootingSiteDataSource struct {
	pd *providerData
}

var troubleshootingEdgeStatusAttrTypes = map[string]attrType{
	"device_id":     types.Int64Type,
	"device_name":   types.StringType,
	"device_status": types.StringType,
}

type troubleshootingEdgeStatusModel struct {
	DeviceID     types.Int64  `tfsdk:"device_id"`
	DeviceName   types.String `tfsdk:"device_name"`
	DeviceStatus types.String `tfsdk:"device_status"`
}

type troubleshootingSiteDataSourceModel struct {
	SiteID       types.Int64  `tfsdk:"site_id"`
	SiteName     types.String `tfsdk:"site_name"`
	SiteStatus   types.String `tfsdk:"site_status"`
	EdgeStatuses types.List   `tfsdk:"edge_statuses"`
}

func (d *troubleshootingSiteDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_troubleshooting_site"
}

func (d *troubleshootingSiteDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A site status snapshot (troubleshooting domain), including per-edge device status.",
		Attributes: map[string]schema.Attribute{
			"site_id": schema.Int64Attribute{
				Required: true,
			},
			"site_name": schema.StringAttribute{
				Computed: true,
			},
			"site_status": schema.StringAttribute{
				Computed: true,
			},
			"edge_statuses": schema.ListAttribute{
				Computed:    true,
				ElementType: types.ObjectType{AttrTypes: troubleshootingEdgeStatusAttrTypes},
			},
		},
	}
}

func (d *troubleshootingSiteDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.pd = configurePD(req.ProviderData, &resp.Diagnostics)
}

func (d *troubleshootingSiteDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg troubleshootingSiteDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, httpResp, err := d.pd.api.DefaultAPI.V1TroubleshootingSiteSiteIdGet(ctx, cfg.SiteID.ValueInt64()).
		Authorization(d.pd.token).
		Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read site troubleshooting snapshot", apiErrorDetail(err))
		return
	}
	if out == nil {
		resp.Diagnostics.AddError("Unable to read site troubleshooting snapshot", "API returned an empty response")
		return
	}

	cfg.SiteName = types.StringPointerValue(out.SiteName)
	cfg.SiteStatus = types.StringPointerValue(out.SiteStatus)

	models := make([]troubleshootingEdgeStatusModel, 0, len(out.EdgeStatuses))
	for _, e := range out.EdgeStatuses {
		models = append(models, troubleshootingEdgeStatusModel{
			DeviceID:     types.Int64PointerValue(e.DeviceId),
			DeviceName:   types.StringPointerValue(e.DeviceName),
			DeviceStatus: types.StringPointerValue(e.DeviceStatus),
		})
	}
	list, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: troubleshootingEdgeStatusAttrTypes}, models)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg.EdgeStatuses = list

	resp.Diagnostics.Append(resp.State.Set(ctx, &cfg)...)
}
