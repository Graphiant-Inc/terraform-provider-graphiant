package provider

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	sdk "github.com/Graphiant-Inc/graphiant-sdk-go"
)

var (
	_ resource.Resource              = &deviceDecommissionResource{}
	_ resource.ResourceWithConfigure = &deviceDecommissionResource{}
)

func NewDeviceDecommissionResource() resource.Resource {
	return &deviceDecommissionResource{}
}

type deviceDecommissionResource struct {
	pd *providerData
}

type deviceDecommissionResourceModel struct {
	ID            types.String `tfsdk:"id"`
	DeviceSerials types.List   `tfsdk:"device_serials"`
	Approve       types.Bool   `tfsdk:"approve"`
	Clear         types.Bool   `tfsdk:"clear"`
	Requested     types.Bool   `tfsdk:"requested"`
}

func (r *deviceDecommissionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_device_decommission"
}

func (r *deviceDecommissionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Drives the hardware-return decommission workflow (request → approve → clear), keyed by " +
			"device serial number rather than device id (a different identity space than every other resource in " +
			"this provider). Action-shaped: Delete performs a best-effort clear-return and does not fail if the " +
			"server has already cleared it.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Derived from device_serials; not a server-assigned identifier.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"device_serials": schema.ListAttribute{
				Required:    true,
				ElementType: types.StringType,
			},
			"approve": schema.BoolAttribute{
				Optional:    true,
				Description: "When true, also calls the approve-return step.",
			},
			"clear": schema.BoolAttribute{
				Optional:    true,
				Description: "When true, also calls the clear-return step.",
			},
			"requested": schema.BoolAttribute{
				Computed:    true,
				Description: "Whether the server currently shows a return requested for these serials (from inventory).",
			},
		},
	}
}

func (r *deviceDecommissionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.pd = configurePD(req.ProviderData, &resp.Diagnostics)
}

func (r *deviceDecommissionResource) refresh(ctx context.Context, m *deviceDecommissionResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	var serials []string
	diags.Append(m.DeviceSerials.ElementsAs(ctx, &serials, false)...)
	if diags.HasError() {
		return diags
	}
	m.ID = types.StringValue(strings.Join(serials, ","))

	out, httpResp, err := r.pd.api.DefaultAPI.V1DevicesInventoryGet(ctx).Authorization(r.pd.token).Execute()
	closeBody(httpResp)
	if err != nil {
		diags.AddError("Unable to read device inventory", apiErrorDetail(err))
		return diags
	}

	requested := false
	if out != nil {
		wanted := make(map[string]bool, len(serials))
		for _, s := range serials {
			wanted[s] = true
		}
		for _, inv := range out.Inventory {
			if inv.DeviceSerial != nil && wanted[*inv.DeviceSerial] && inv.IsRequested != nil && *inv.IsRequested {
				requested = true
				break
			}
		}
	}
	m.Requested = types.BoolValue(requested)
	return diags
}

func (r *deviceDecommissionResource) apply(ctx context.Context, m *deviceDecommissionResourceModel, diags *diag.Diagnostics) {
	var serials []string
	diags.Append(m.DeviceSerials.ElementsAs(ctx, &serials, false)...)
	if diags.HasError() {
		return
	}

	reqBody := sdk.V1DevicesInventoryRequestReturnPostRequest{DeviceSerials: serials}
	_, httpResp, err := r.pd.api.DefaultAPI.V1DevicesInventoryRequestReturnPost(ctx).
		Authorization(r.pd.token).
		V1DevicesInventoryRequestReturnPostRequest(reqBody).
		Execute()
	closeBody(httpResp)
	if err != nil {
		diags.AddError("Unable to request device return", apiErrorDetail(err))
		return
	}

	if m.Approve.ValueBool() {
		approveBody := sdk.V1DevicesInventoryApproveReturnPostRequest{DeviceSerials: serials}
		_, httpResp, err := r.pd.api.DefaultAPI.V1DevicesInventoryApproveReturnPost(ctx).
			Authorization(r.pd.token).
			V1DevicesInventoryApproveReturnPostRequest(approveBody).
			Execute()
		closeBody(httpResp)
		if err != nil {
			diags.AddError("Unable to approve device return", apiErrorDetail(err))
			return
		}
	}

	if m.Clear.ValueBool() {
		clearBody := sdk.V1DevicesInventoryClearReturnPostRequest{DeviceSerials: serials}
		_, httpResp, err := r.pd.api.DefaultAPI.V1DevicesInventoryClearReturnPost(ctx).
			Authorization(r.pd.token).
			V1DevicesInventoryClearReturnPostRequest(clearBody).
			Execute()
		closeBody(httpResp)
		if err != nil {
			diags.AddError("Unable to clear device return", apiErrorDetail(err))
			return
		}
	}

	diags.Append(r.refresh(ctx, m)...)
}

func (r *deviceDecommissionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan deviceDecommissionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.apply(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *deviceDecommissionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state deviceDecommissionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(r.refresh(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *deviceDecommissionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan deviceDecommissionResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.apply(ctx, &plan, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *deviceDecommissionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state deviceDecommissionResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var serials []string
	resp.Diagnostics.Append(state.DeviceSerials.ElementsAs(ctx, &serials, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := sdk.V1DevicesInventoryClearReturnPostRequest{DeviceSerials: serials}
	_, httpResp, err := r.pd.api.DefaultAPI.V1DevicesInventoryClearReturnPost(ctx).
		Authorization(r.pd.token).
		V1DevicesInventoryClearReturnPostRequest(body).
		Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddWarning("Unable to clear device return on delete", apiErrorDetail(err))
	}
}
