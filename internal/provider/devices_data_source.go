package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
)

var (
	_ datasource.DataSource              = &devicesDataSource{}
	_ datasource.DataSourceWithConfigure = &devicesDataSource{}
)

func NewDevicesDataSource() datasource.DataSource {
	return &devicesDataSource{}
}

type devicesDataSource struct {
	client *gClient
}

type devicesDataSourceModel struct {
	Devices []deviceDataSourceModel `tfsdk:"devices"`
}

func (d *devicesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_devices"
}

func (d *devicesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Lists all onboarded Graphiant devices in the enterprise. Read-only; this provider does not manage device configuration.",
		Attributes: map[string]schema.Attribute{
			"devices": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: deviceDataSourceAttributes(false),
				},
			},
		},
	}
}

func (d *devicesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *devicesDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	out, httpRes, err := d.client.api.DefaultAPI.V1DevicesGet(ctx).Authorization(d.client.authHeader()).Execute()
	if httpRes != nil {
		defer httpRes.Body.Close()
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading devices", err.Error())
		return
	}

	var state devicesDataSourceModel
	if out != nil {
		for _, dev := range out.GetDevices() {
			var m deviceDataSourceModel
			flattenDeviceDataSource(&dev, &m)
			state.Devices = append(state.Devices, m)
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
