package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &siteDevicesDataSource{}
	_ datasource.DataSourceWithConfigure = &siteDevicesDataSource{}
)

func NewSiteDevicesDataSource() datasource.DataSource {
	return &siteDevicesDataSource{}
}

type siteDevicesDataSource struct {
	pd *providerData
}

var topologyDeviceAttrTypes = map[string]attrType{
	"device_id":        types.Int64Type,
	"hostname":         types.StringType,
	"location":         types.StringType,
	"maintenance_mode": types.BoolType,
	"management_ip":    types.StringType,
	"model":            types.StringType,
	"role":             types.StringType,
	"serial_number":    types.StringType,
	"software_version": types.StringType,
	"staging_mode":     types.BoolType,
	"vrrp_interface":   types.StringType,
	"vrrp_state":       types.StringType,
}

type topologyDeviceModel struct {
	DeviceID        types.Int64  `tfsdk:"device_id"`
	Hostname        types.String `tfsdk:"hostname"`
	Location        types.String `tfsdk:"location"`
	MaintenanceMode types.Bool   `tfsdk:"maintenance_mode"`
	ManagementIP    types.String `tfsdk:"management_ip"`
	Model           types.String `tfsdk:"model"`
	Role            types.String `tfsdk:"role"`
	SerialNumber    types.String `tfsdk:"serial_number"`
	SoftwareVersion types.String `tfsdk:"software_version"`
	StagingMode     types.Bool   `tfsdk:"staging_mode"`
	VrrpInterface   types.String `tfsdk:"vrrp_interface"`
	VrrpState       types.String `tfsdk:"vrrp_state"`
}

type siteDevicesDataSourceModel struct {
	SiteID  types.Int64 `tfsdk:"site_id"`
	Devices types.List  `tfsdk:"devices"`
}

func (d *siteDevicesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_site_devices"
}

func (d *siteDevicesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Per-site device list with maintenance/VRRP/staging state. This is the closest real proxy " +
			"for \"site health\" in the API — there is no dedicated site-health endpoint (V1SitesDetailsGet's " +
			"\"site wide status\" doc comment does not correspond to any actual response field).",
		Attributes: map[string]schema.Attribute{
			"site_id": schema.Int64Attribute{
				Required: true,
			},
			"devices": schema.ListAttribute{
				Computed:    true,
				ElementType: types.ObjectType{AttrTypes: topologyDeviceAttrTypes},
			},
		},
	}
}

func (d *siteDevicesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.pd = configurePD(req.ProviderData, &resp.Diagnostics)
}

func (d *siteDevicesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg siteDevicesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, httpResp, err := d.pd.api.DefaultAPI.V1SitesSiteIdDevicesGet(ctx, cfg.SiteID.ValueInt64()).
		Authorization(d.pd.token).
		Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read site devices", apiErrorDetail(err))
		return
	}

	models := make([]topologyDeviceModel, 0)
	if out != nil {
		for _, dev := range out.Device {
			models = append(models, topologyDeviceModel{
				DeviceID:        types.Int64PointerValue(dev.DeviceId),
				Hostname:        types.StringPointerValue(dev.Hostname),
				Location:        types.StringPointerValue(dev.Location),
				MaintenanceMode: types.BoolPointerValue(dev.MaintenanceMode),
				ManagementIP:    types.StringPointerValue(dev.ManagementIp),
				Model:           types.StringPointerValue(dev.Model),
				Role:            types.StringPointerValue(dev.Role),
				SerialNumber:    types.StringPointerValue(dev.SerialNumber),
				SoftwareVersion: types.StringPointerValue(dev.SoftwareVersion),
				StagingMode:     types.BoolPointerValue(dev.StagingMode),
				VrrpInterface:   types.StringPointerValue(dev.VrrpInterface),
				VrrpState:       types.StringPointerValue(dev.VrrpState),
			})
		}
	}
	list, diags := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: topologyDeviceAttrTypes}, models)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg.Devices = list

	resp.Diagnostics.Append(resp.State.Set(ctx, &cfg)...)
}
