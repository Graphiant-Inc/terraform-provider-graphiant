package provider

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	sdk "github.com/Graphiant-Inc/graphiant-sdk-go"
)

var (
	_ resource.Resource                = &deviceConfigResource{}
	_ resource.ResourceWithConfigure   = &deviceConfigResource{}
	_ resource.ResourceWithImportState = &deviceConfigResource{}
)

func NewDeviceConfigResource() resource.Resource {
	return &deviceConfigResource{}
}

type deviceConfigResource struct {
	pd *providerData
}

type deviceConfigResourceModel struct {
	ID                     types.String `tfsdk:"id"`
	DeviceID               types.Int64  `tfsdk:"device_id"`
	DeviceType             types.String `tfsdk:"device_type"`
	MaintenanceMode        types.Bool   `tfsdk:"maintenance_mode"`
	Region                 types.String `tfsdk:"region"`
	Description            types.String `tfsdk:"description"`
	LocalWebServerPassword types.String `tfsdk:"local_web_server_password"`
	Replace                types.Bool   `tfsdk:"replace"`
	BgpEnabled             types.Bool   `tfsdk:"bgp_enabled"`
	DhcpServerEnabled      types.Bool   `tfsdk:"dhcp_server_enabled"`
	IpfixEnabled           types.Bool   `tfsdk:"ipfix_enabled"`
	LldpEnabled            types.Bool   `tfsdk:"lldp_enabled"`
	Ospfv2Enabled          types.Bool   `tfsdk:"ospfv2_enabled"`
	Ospfv3Enabled          types.Bool   `tfsdk:"ospfv3_enabled"`
	StaticRoutesEnabled    types.Bool   `tfsdk:"static_routes_enabled"`
	VrrpEnabled            types.Bool   `tfsdk:"vrrp_enabled"`
}

func (r *deviceConfigResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_device_config"
}

func (r *deviceConfigResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Pushes device configuration via the single generic V1DevicesDeviceIdConfigPut endpoint " +
			"(PUT /v1/devices/{deviceId}/config), the same endpoint graphiant-playbooks' Ansible modules use for " +
			"BGP, interfaces, LAG, DHCP relay, NTP, OSPFv2, static routes, VRRP, MACsec, NAT/security/traffic " +
			"policy, and site-to-site VPN. This resource ONLY covers maintenance_mode and, for edge devices, the " +
			"*_enabled toggles (both round-trip cleanly through ManaV2Device on read); region is write-only, see " +
			"its own description. It does NOT expose those other config domains: each is a large nested " +
			"map on ManaV2CoreDeviceConfig/ManaV2EdgeDeviceConfig that hasn't been individually verified and would " +
			"need its own dedicated resource. The underlying PUT is an async job; this resource polls " +
			"V1DevicesDeviceIdJobsJobIdGet (bounded, ~30s) for completion. Update waits 1 minute before pushing, " +
			"since the device can still be settling from a prior job (rejecting a new PUT with \"forbidden from " +
			"its current state\") right after that job reports complete. There is no \"unconfigure\" endpoint, so " +
			"Delete only removes the resource from Terraform state. Import uses \"<device_id>:<device_type>\" " +
			"since device_type can't be derived from the API alone.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"device_id": schema.Int64Attribute{
				Required:      true,
				PlanModifiers: []planmodifier.Int64{int64planmodifier.RequiresReplace()},
			},
			"device_type": schema.StringAttribute{
				Required:      true,
				Description:   "Either \"core\" or \"edge\" — determines whether the PUT body's core or edge config object is populated.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"maintenance_mode": schema.BoolAttribute{
				Optional: true,
				Computed: true,
			},
			"region": schema.StringAttribute{
				Optional:    true,
				Description: "Not refreshed from the API: ManaV2Device.Region resolves to an object, and it's not confirmed which of its fields (name vs. ISO code) matches what this write-side string field expects, so the configured value is preserved as-is rather than guessed at.",
			},
			"description": schema.StringAttribute{
				Optional: true,
			},
			"local_web_server_password": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
			},
			"replace": schema.BoolAttribute{
				Optional:    true,
				Description: "Passed through to the API as-is. Its behavior is undocumented anywhere in the SDK or the raw OpenAPI spec — use with caution.",
			},
			"bgp_enabled":           schema.BoolAttribute{Optional: true, Computed: true, Description: "Edge devices only."},
			"dhcp_server_enabled":   schema.BoolAttribute{Optional: true, Computed: true, Description: "Edge devices only."},
			"ipfix_enabled":         schema.BoolAttribute{Optional: true, Computed: true, Description: "Edge devices only."},
			"lldp_enabled":          schema.BoolAttribute{Optional: true, Computed: true, Description: "Edge devices only."},
			"ospfv2_enabled":        schema.BoolAttribute{Optional: true, Computed: true, Description: "Edge devices only."},
			"ospfv3_enabled":        schema.BoolAttribute{Optional: true, Computed: true, Description: "Edge devices only."},
			"static_routes_enabled": schema.BoolAttribute{Optional: true, Computed: true, Description: "Edge devices only."},
			"vrrp_enabled":          schema.BoolAttribute{Optional: true, Computed: true, Description: "Edge devices only."},
		},
	}
}

func (r *deviceConfigResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.pd = configurePD(req.ProviderData, &resp.Diagnostics)
}

// pollJob waits for an async device-config job to complete. JobState's valid values
// are undocumented anywhere in the SDK/spec, so completion is detected via CompletedAt
// being set rather than string-matching an unconfirmed enum.
func (r *deviceConfigResource) pollJob(ctx context.Context, deviceID, jobID int64) error {
	const maxAttempts = 10
	const delay = 3 * time.Second
	for attempt := 0; attempt < maxAttempts; attempt++ {
		out, httpResp, err := r.pd.api.DefaultAPI.V1DevicesDeviceIdJobsJobIdGet(ctx, deviceID, jobID).
			Authorization(r.pd.token).
			Execute()
		closeBody(httpResp)
		if err != nil {
			return fmt.Errorf("polling job status: %s", apiErrorDetail(err))
		}
		if out != nil && out.JobStatus != nil {
			status := out.JobStatus
			if status.Error != nil && *status.Error != "" {
				return fmt.Errorf("job failed: %s", *status.Error)
			}
			if status.CompletedAt != nil {
				return nil
			}
		}
		if attempt < maxAttempts-1 {
			time.Sleep(delay)
		}
	}
	return fmt.Errorf("job did not complete after %d attempts (%ds)", maxAttempts, maxAttempts*int(delay.Seconds()))
}

func (r *deviceConfigResource) put(ctx context.Context, m *deviceConfigResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	body := sdk.V1DevicesDeviceIdConfigPutRequest{
		Description:            m.Description.ValueStringPointer(),
		LocalWebServerPassword: m.LocalWebServerPassword.ValueStringPointer(),
		Replace:                m.Replace.ValueBoolPointer(),
	}

	switch m.DeviceType.ValueString() {
	case "core":
		body.Core = &sdk.ManaV2CoreDeviceConfig{
			MaintenanceMode: m.MaintenanceMode.ValueBoolPointer(),
			Region:          m.Region.ValueStringPointer(),
		}
	case "edge":
		body.Edge = &sdk.ManaV2EdgeDeviceConfig{
			MaintenanceMode:     m.MaintenanceMode.ValueBoolPointer(),
			Region:              m.Region.ValueStringPointer(),
			BgpEnabled:          m.BgpEnabled.ValueBoolPointer(),
			DhcpServerEnabled:   m.DhcpServerEnabled.ValueBoolPointer(),
			IpfixEnabled:        m.IpfixEnabled.ValueBoolPointer(),
			LldpEnabled:         m.LldpEnabled.ValueBoolPointer(),
			Ospfv2Enabled:       m.Ospfv2Enabled.ValueBoolPointer(),
			Ospfv3Enabled:       m.Ospfv3Enabled.ValueBoolPointer(),
			StaticRoutesEnabled: m.StaticRoutesEnabled.ValueBoolPointer(),
			VrrpEnabled:         m.VrrpEnabled.ValueBoolPointer(),
		}
	default:
		diags.AddError("Invalid device_type", fmt.Sprintf(`device_type must be "core" or "edge", got %q`, m.DeviceType.ValueString()))
		return diags
	}

	deviceID := m.DeviceID.ValueInt64()
	out, httpResp, err := r.pd.api.DefaultAPI.V1DevicesDeviceIdConfigPut(ctx, deviceID).
		Authorization(r.pd.token).
		V1DevicesDeviceIdConfigPutRequest(body).
		Execute()
	closeBody(httpResp)
	if err != nil {
		diags.AddError("Unable to update device config", apiErrorDetail(err))
		return diags
	}
	if out == nil || out.JobId == nil {
		diags.AddError("Unable to update device config", "API returned an empty response")
		return diags
	}

	if err := r.pollJob(ctx, deviceID, *out.JobId); err != nil {
		diags.AddError("Device config job did not complete", err.Error())
	}
	return diags
}

func (r *deviceConfigResource) readDevice(ctx context.Context, deviceID int64) (*sdk.ManaV2Device, bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	out, httpResp, err := r.pd.api.DefaultAPI.V1DevicesDeviceIdGet(ctx, deviceID).Authorization(r.pd.token).Execute()
	if err != nil {
		if isNotFound(httpResp) {
			closeBody(httpResp)
			return nil, false, diags
		}
		closeBody(httpResp)
		diags.AddError("Unable to read device", apiErrorDetail(err))
		return nil, false, diags
	}
	closeBody(httpResp)
	if out == nil || out.Device == nil {
		return nil, false, diags
	}
	return out.Device, true, diags
}

// setBool applies v to *dst, except it leaves an already-known *dst alone when
// v is nil. The API has been observed (on a sibling alertservice endpoint)
// omitting boolean fields entirely rather than sending false explicitly,
// which is indistinguishable from "not returned" — overwriting unconditionally
// would null out a value we just explicitly set via Create/Update and trip
// Terraform's inconsistent-result check. But *dst is Unknown (not yet a known
// value) whenever the attribute is Computed and wasn't set in config, and
// Terraform requires every Computed attribute to resolve to a known value
// after apply — Null counts as known, Unknown does not — so in that case v's
// nil must still resolve to BoolNull() rather than being left alone.
func setBool(dst *types.Bool, v *bool) {
	if v != nil {
		*dst = types.BoolPointerValue(v)
	} else if dst.IsUnknown() {
		*dst = types.BoolNull()
	}
}

// applyDevice deliberately leaves m.Region untouched — see the region attribute's
// schema description for why it can't be safely refreshed from ManaV2Device.
func (m *deviceConfigResourceModel) applyDevice(dev *sdk.ManaV2Device) {
	m.ID = types.StringValue(int64ID(m.DeviceID.ValueInt64()))
	setBool(&m.MaintenanceMode, dev.MaintenanceMode)
	if m.DeviceType.ValueString() == "edge" {
		setBool(&m.BgpEnabled, dev.BgpEnabled)
		setBool(&m.DhcpServerEnabled, dev.DhcpServerEnabled)
		setBool(&m.IpfixEnabled, dev.IpfixEnabled)
		setBool(&m.LldpEnabled, dev.LldpEnabled)
		setBool(&m.Ospfv2Enabled, dev.Ospfv2Enabled)
		setBool(&m.Ospfv3Enabled, dev.Ospfv3Enabled)
		setBool(&m.StaticRoutesEnabled, dev.StaticRoutesEnabled)
		setBool(&m.VrrpEnabled, dev.VrrpEnabled)
	}
}

func (r *deviceConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan deviceConfigResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.put(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	dev, found, diags := r.readDevice(ctx, plan.DeviceID.ValueInt64())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.Diagnostics.AddError("Unable to read device after config update", "device was not found immediately after the config job completed")
		return
	}
	plan.applyDevice(dev)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *deviceConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state deviceConfigResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	dev, found, diags := r.readDevice(ctx, state.DeviceID.ValueInt64())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}
	state.applyDevice(dev)

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *deviceConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan deviceConfigResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// The device can still be settling from a prior config job (observed as a 500,
	// "forbidden from its current state") right after that job's completion is
	// reported; give it time to leave that state before pushing again.
	time.Sleep(1 * time.Minute)

	resp.Diagnostics.Append(r.put(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	dev, found, diags := r.readDevice(ctx, plan.DeviceID.ValueInt64())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.Diagnostics.AddError("Unable to read device after config update", "device was not found immediately after the config job completed")
		return
	}
	plan.applyDevice(dev)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *deviceConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddWarning(
		"Device configuration unchanged on the server",
		"There is no API endpoint to unconfigure these fields; this resource is only being removed from Terraform state.",
	)
}

func (r *deviceConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid import ID",
			`Expected "<device_id>:<device_type>", e.g. "12345:edge" — device_type can't be derived from the API alone.`,
		)
		return
	}
	deviceID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", "device_id must be an integer: "+err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("device_id"), deviceID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("device_type"), parts[1])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), int64ID(deviceID))...)
}
