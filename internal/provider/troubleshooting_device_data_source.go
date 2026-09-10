package provider

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	sdk "github.com/Graphiant-Inc/graphiant-sdk-go"
)

var (
	_ datasource.DataSource              = &troubleshootingDeviceDataSource{}
	_ datasource.DataSourceWithConfigure = &troubleshootingDeviceDataSource{}
)

func NewTroubleshootingDeviceDataSource() datasource.DataSource {
	return &troubleshootingDeviceDataSource{}
}

type troubleshootingDeviceDataSource struct {
	pd *providerData
}

type troubleshootingDeviceDataSourceModel struct {
	DeviceID        types.Int64  `tfsdk:"device_id"`
	Status          types.String `tfsdk:"status"`
	LifecycleStatus types.String `tfsdk:"lifecycle_status"`
	SwVersion       types.String `tfsdk:"sw_version"`
	MaintenanceMode types.Bool   `tfsdk:"maintenance_mode"`
	ColrActive      types.Bool   `tfsdk:"colr_active"`
}

func (d *troubleshootingDeviceDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_troubleshooting_device"
}

func (d *troubleshootingDeviceDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A device health snapshot (troubleshooting domain). control_plane/data_plane/system_plane/" +
			"issues sub-objects are not yet exposed — only top-level scalar status fields.",
		Attributes: map[string]schema.Attribute{
			"device_id": schema.Int64Attribute{
				Required: true,
			},
			"status": schema.StringAttribute{
				Computed: true,
			},
			"lifecycle_status": schema.StringAttribute{
				Computed: true,
			},
			"sw_version": schema.StringAttribute{
				Computed: true,
			},
			"maintenance_mode": schema.BoolAttribute{
				Computed: true,
			},
			"colr_active": schema.BoolAttribute{
				Computed: true,
			},
		},
	}
}

func (d *troubleshootingDeviceDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.pd = configurePD(req.ProviderData, &resp.Diagnostics)
}

func (d *troubleshootingDeviceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg troubleshootingDeviceDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	now := time.Now()
	recentTs := sdk.GoogleProtobufTimestamp{Seconds: sdk.PtrInt64(now.Unix())}
	oldTs := sdk.GoogleProtobufTimestamp{Seconds: sdk.PtrInt64(now.Add(-1 * time.Hour).Unix())}
	timeWindow := sdk.StatsmonTroubleshootingTimeWindow{
		RecentTs: &recentTs,
		OldTs:    &oldTs,
	}

	out, httpResp, err := d.pd.api.DefaultAPI.V1TroubleshootingDeviceDeviceIdPost(ctx, cfg.DeviceID.ValueInt64()).
		Authorization(d.pd.token).
		V1TroubleshootingDeviceDeviceIdPostRequest(sdk.V1TroubleshootingDeviceDeviceIdPostRequest{TimeWindow: &timeWindow}).
		Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to read device troubleshooting snapshot", apiErrorDetail(err))
		return
	}
	if out == nil {
		resp.Diagnostics.AddError("Unable to read device troubleshooting snapshot", "API returned an empty response")
		return
	}

	cfg.Status = types.StringPointerValue(out.Status)
	cfg.LifecycleStatus = types.StringPointerValue(out.LifecycleStatus)
	cfg.SwVersion = types.StringPointerValue(out.SwVersion)
	cfg.MaintenanceMode = types.BoolPointerValue(out.MaintenanceMode)
	cfg.ColrActive = types.BoolPointerValue(out.ColrActive)

	resp.Diagnostics.Append(resp.State.Set(ctx, &cfg)...)
}
