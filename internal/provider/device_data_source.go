package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	graphiant "github.com/Graphiant-Inc/graphiant-sdk-go"
)

var (
	_ datasource.DataSource              = &deviceDataSource{}
	_ datasource.DataSourceWithConfigure = &deviceDataSource{}
)

func NewDeviceDataSource() datasource.DataSource {
	return &deviceDataSource{}
}

type deviceDataSource struct {
	client *gClient
}

// deviceDataSourceModel exposes a read-only summary of manaV2Device. It
// intentionally omits the device's full network configuration (circuits,
// interfaces, routing policies, IPsec tunnels, etc.), which is out of scope
// for this provider.
type deviceDataSourceModel struct {
	Id                  types.Int64  `tfsdk:"id"`
	SerialNumber        types.String `tfsdk:"serial_number"`
	Hostname            types.String `tfsdk:"hostname"`
	Platform            types.String `tfsdk:"platform"`
	Role                types.String `tfsdk:"role"`
	Status              types.String `tfsdk:"status"`
	SoftwareVersion     types.String `tfsdk:"software_version"`
	SiteId              types.Int64  `tfsdk:"site_id"`
	SiteName            types.String `tfsdk:"site_name"`
	MaintenanceMode     types.Bool   `tfsdk:"maintenance_mode"`
	OperStaled          types.Bool   `tfsdk:"oper_staled"`
	BgpEnabled          types.Bool   `tfsdk:"bgp_enabled"`
	Ospfv2Enabled       types.Bool   `tfsdk:"ospfv2_enabled"`
	Ospfv3Enabled       types.Bool   `tfsdk:"ospfv3_enabled"`
	LldpEnabled         types.Bool   `tfsdk:"lldp_enabled"`
	DhcpServerEnabled   types.Bool   `tfsdk:"dhcp_server_enabled"`
	IpfixEnabled        types.Bool   `tfsdk:"ipfix_enabled"`
	StaticRoutesEnabled types.Bool   `tfsdk:"static_routes_enabled"`
	CreatedAt           types.String `tfsdk:"created_at"`
	LastBootedAt        types.String `tfsdk:"last_booted_at"`
}

func deviceDataSourceAttributes(idRequired bool) map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.Int64Attribute{
			Required:    idRequired,
			Computed:    !idRequired,
			Description: "Device identifier.",
		},
		"serial_number":         schema.StringAttribute{Computed: true, Description: "Device serial number."},
		"hostname":              schema.StringAttribute{Computed: true, Description: "Device hostname."},
		"platform":              schema.StringAttribute{Computed: true, Description: "Device hardware/software platform."},
		"role":                  schema.StringAttribute{Computed: true, Description: "Device role (e.g. \"edge\", \"gateway\")."},
		"status":                schema.StringAttribute{Computed: true, Description: "Current device status."},
		"software_version":      schema.StringAttribute{Computed: true, Description: "Running software version."},
		"site_id":               schema.Int64Attribute{Computed: true, Description: "ID of the site this device is onboarded at."},
		"site_name":             schema.StringAttribute{Computed: true, Description: "Name of the site this device is onboarded at."},
		"maintenance_mode":      schema.BoolAttribute{Computed: true, Description: "Whether the device is in maintenance mode."},
		"oper_staled":           schema.BoolAttribute{Computed: true, Description: "Whether the device's operational state is stale."},
		"bgp_enabled":           schema.BoolAttribute{Computed: true, Description: "Whether BGP is enabled on the device."},
		"ospfv2_enabled":        schema.BoolAttribute{Computed: true, Description: "Whether OSPFv2 is enabled on the device."},
		"ospfv3_enabled":        schema.BoolAttribute{Computed: true, Description: "Whether OSPFv3 is enabled on the device."},
		"lldp_enabled":          schema.BoolAttribute{Computed: true, Description: "Whether LLDP is enabled on the device."},
		"dhcp_server_enabled":   schema.BoolAttribute{Computed: true, Description: "Whether the DHCP server is enabled on the device."},
		"ipfix_enabled":         schema.BoolAttribute{Computed: true, Description: "Whether IPFIX export is enabled on the device."},
		"static_routes_enabled": schema.BoolAttribute{Computed: true, Description: "Whether static routes are enabled on the device."},
		"created_at":            schema.StringAttribute{Computed: true, Description: "Creation timestamp (RFC3339, UTC)."},
		"last_booted_at":        schema.StringAttribute{Computed: true, Description: "Timestamp of the device's last boot (RFC3339, UTC)."},
	}
}

func flattenDeviceDataSource(dev *graphiant.ManaV2Device, m *deviceDataSourceModel) {
	m.Id = int64Value(dev.Id)
	m.SerialNumber = strValue(dev.SerialNumber)
	m.Hostname = strValue(dev.Hostname)
	m.Platform = strValue(dev.Platform)
	m.Role = strValue(dev.Role)
	m.Status = strValue(dev.Status)
	m.SoftwareVersion = strValue(dev.SoftwareVersion)
	if dev.Site != nil {
		m.SiteId = int64Value(dev.Site.Id)
		m.SiteName = strValue(dev.Site.Name)
	} else {
		m.SiteId = types.Int64Null()
		m.SiteName = types.StringNull()
	}
	m.MaintenanceMode = boolValue(dev.MaintenanceMode)
	m.OperStaled = boolValue(dev.OperStaled)
	m.BgpEnabled = boolValue(dev.BgpEnabled)
	m.Ospfv2Enabled = boolValue(dev.Ospfv2Enabled)
	m.Ospfv3Enabled = boolValue(dev.Ospfv3Enabled)
	m.LldpEnabled = boolValue(dev.LldpEnabled)
	m.DhcpServerEnabled = boolValue(dev.DhcpServerEnabled)
	m.IpfixEnabled = boolValue(dev.IpfixEnabled)
	m.StaticRoutesEnabled = boolValue(dev.StaticRoutesEnabled)
	m.CreatedAt = timestampValue(dev.CreatedAt)
	m.LastBootedAt = timestampValue(dev.LastBootedAt)
}

func (d *deviceDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_device"
}

func (d *deviceDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up a single onboarded Graphiant device by ID. Read-only; this provider does not manage device configuration.",
		Attributes:  deviceDataSourceAttributes(true),
	}
}

func (d *deviceDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*gClient)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Data Source Configure Type", fmt.Sprintf("expected *gClient, got: %T", req.ProviderData))
		return
	}
	d.client = client
}

func (d *deviceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config deviceDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "reading device data source", map[string]any{"id": config.Id.ValueInt64()})

	out, httpRes, err := d.client.api.DefaultAPI.V1DevicesDeviceIdGet(ctx, config.Id.ValueInt64()).Authorization(d.client.authHeader()).Execute()
	defer closeBody(httpRes)
	if err != nil {
		resp.Diagnostics.AddError("Error reading device", apiErrorDetail(err))
		return
	}
	if out == nil || out.Device == nil {
		resp.Diagnostics.AddError("Device not found", fmt.Sprintf("no device with id %d was found", config.Id.ValueInt64()))
		return
	}

	flattenDeviceDataSource(out.Device, &config)
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
