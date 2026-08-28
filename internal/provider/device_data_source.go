package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &deviceDataSource{}
	_ datasource.DataSourceWithConfigure = &deviceDataSource{}
)

func NewDeviceDataSource() datasource.DataSource {
	return &deviceDataSource{}
}

type deviceDataSource struct {
	pd *providerData
}

type deviceDataSourceModel struct {
	ID              types.Int64  `tfsdk:"id"`
	Hostname        types.String `tfsdk:"hostname"`
	Status          types.String `tfsdk:"status"`
	Platform        types.String `tfsdk:"platform"`
	SerialNumber    types.String `tfsdk:"serial_number"`
	SoftwareVersion types.String `tfsdk:"software_version"`
	SiteID          types.Int64  `tfsdk:"site_id"`
	Notes           types.String `tfsdk:"notes"`
	MaintenanceMode types.Bool   `tfsdk:"maintenance_mode"`
	Role            types.String `tfsdk:"role"`
}

func (d *deviceDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_device"
}

func (d *deviceDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up a single device by ID. Deeper device state (circuits, interfaces, BGP, etc.) is not yet exposed.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Required:    true,
				Description: "Device ID.",
			},
			"hostname": schema.StringAttribute{
				Computed: true,
			},
			"status": schema.StringAttribute{
				Computed: true,
			},
			"platform": schema.StringAttribute{
				Computed: true,
			},
			"serial_number": schema.StringAttribute{
				Computed: true,
			},
			"software_version": schema.StringAttribute{
				Computed: true,
			},
			"site_id": schema.Int64Attribute{
				Computed: true,
			},
			"notes": schema.StringAttribute{
				Computed: true,
			},
			"maintenance_mode": schema.BoolAttribute{
				Computed: true,
			},
			"role": schema.StringAttribute{
				Computed: true,
			},
		},
	}
}

func (d *deviceDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.pd = configurePD(req.ProviderData, &resp.Diagnostics)
}

func (d *deviceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg deviceDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, httpResp, err := d.pd.api.DefaultAPI.V1DevicesDeviceIdGet(ctx, cfg.ID.ValueInt64()).
		Authorization(d.pd.token).
		Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read device", apiErrorDetail(err))
		return
	}
	if out == nil || out.Device == nil {
		resp.Diagnostics.AddError("Unable to read device", "API returned an empty response")
		return
	}

	dev := out.Device
	cfg.Hostname = types.StringPointerValue(dev.Hostname)
	cfg.Status = types.StringPointerValue(dev.Status)
	cfg.Platform = types.StringPointerValue(dev.Platform)
	cfg.SerialNumber = types.StringPointerValue(dev.SerialNumber)
	cfg.SoftwareVersion = types.StringPointerValue(dev.SoftwareVersion)
	cfg.Notes = types.StringPointerValue(dev.Notes)
	cfg.MaintenanceMode = types.BoolPointerValue(dev.MaintenanceMode)
	cfg.Role = types.StringPointerValue(dev.Role)
	if dev.Site != nil {
		cfg.SiteID = types.Int64PointerValue(dev.Site.Id)
	} else {
		cfg.SiteID = types.Int64Null()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &cfg)...)
}
