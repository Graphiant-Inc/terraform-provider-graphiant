package provider

import (
	"context"
	"strconv"
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
	_ resource.Resource              = &deviceBringupResource{}
	_ resource.ResourceWithConfigure = &deviceBringupResource{}
)

func NewDeviceBringupResource() resource.Resource {
	return &deviceBringupResource{}
}

type deviceBringupResource struct {
	pd *providerData
}

type deviceBringupSummaryModel struct {
	DeviceID           types.Int64  `tfsdk:"device_id"`
	State              types.String `tfsdk:"state"`
	Status             types.String `tfsdk:"status"`
	IsNew              types.Bool   `tfsdk:"is_new"`
	DiscoveredLocation types.String `tfsdk:"discovered_location"`
	IPDetected         types.String `tfsdk:"ip_detected"`
}

var deviceBringupSummaryAttrTypes = map[string]attrType{
	"device_id":           types.Int64Type,
	"state":               types.StringType,
	"status":              types.StringType,
	"is_new":              types.BoolType,
	"discovered_location": types.StringType,
	"ip_detected":         types.StringType,
}

type deviceBringupResourceModel struct {
	ID        types.String `tfsdk:"id"`
	DeviceIDs types.List   `tfsdk:"device_ids"`
	Status    types.String `tfsdk:"status"`
	Devices   types.List   `tfsdk:"devices"`
}

func (r *deviceBringupResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_device_bringup"
}

func (r *deviceBringupResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Triggers a bulk device bringup/activation status transition (V1DevicesBringupPut). This is " +
			"action-shaped, not a persistent object: there is no \"un-bringup\" endpoint, so Delete only removes the " +
			"resource from Terraform state — it does not change the devices' actual status on the server. The API " +
			"does not document valid status values; pass through whatever string your deployment uses.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Derived from the sorted device_ids; not a server-assigned identifier.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"device_ids": schema.ListAttribute{
				Required:    true,
				ElementType: types.Int64Type,
			},
			"status": schema.StringAttribute{
				Required: true,
			},
			"devices": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"device_id":           schema.Int64Attribute{Computed: true},
						"state":               schema.StringAttribute{Computed: true},
						"status":              schema.StringAttribute{Computed: true},
						"is_new":              schema.BoolAttribute{Computed: true},
						"discovered_location": schema.StringAttribute{Computed: true},
						"ip_detected":         schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (r *deviceBringupResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.pd = configurePD(req.ProviderData, &resp.Diagnostics)
}

func bringupID(ids []int64) string {
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = strconv.FormatInt(id, 10)
	}
	return strings.Join(parts, ",")
}

func (r *deviceBringupResource) refresh(ctx context.Context, m *deviceBringupResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	var ids []int64
	d := m.DeviceIDs.ElementsAs(ctx, &ids, false)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}
	m.ID = types.StringValue(bringupID(ids))

	body := sdk.V1DevicesBringupPostRequest{DeviceIds: ids}
	out, httpResp, err := r.pd.api.DefaultAPI.V1DevicesBringupPost(ctx).
		Authorization(r.pd.token).
		V1DevicesBringupPostRequest(body).
		Execute()
	closeBody(httpResp)
	if err != nil {
		diags.AddError("Unable to read device bringup status", apiErrorDetail(err))
		return diags
	}

	entries := make([]deviceBringupSummaryModel, 0)
	if out != nil {
		for _, s := range out.Summaries {
			entries = append(entries, deviceBringupSummaryModel{
				DeviceID:           types.Int64PointerValue(s.DeviceId),
				State:              types.StringPointerValue(s.State),
				Status:             types.StringPointerValue(s.Status),
				IsNew:              types.BoolPointerValue(s.IsNew),
				DiscoveredLocation: types.StringPointerValue(s.DiscoveredLocation),
				IPDetected:         types.StringPointerValue(s.IpDetected),
			})
		}
	}
	devices, d2 := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: deviceBringupSummaryAttrTypes}, entries)
	diags.Append(d2...)
	if diags.HasError() {
		return diags
	}
	m.Devices = devices
	return diags
}

func (r *deviceBringupResource) apply(ctx context.Context, m *deviceBringupResourceModel, resp *diag.Diagnostics) {
	var ids []int64
	resp.Append(m.DeviceIDs.ElementsAs(ctx, &ids, false)...)
	if resp.HasError() {
		return
	}

	body := sdk.V1DevicesBringupPutRequest{DeviceIds: ids, Status: m.Status.ValueStringPointer()}
	httpResp, err := r.pd.api.DefaultAPI.V1DevicesBringupPut(ctx).
		Authorization(r.pd.token).
		V1DevicesBringupPutRequest(body).
		Execute()
	closeBody(httpResp)
	if err != nil {
		resp.AddError("Unable to trigger device bringup", apiErrorDetail(err))
		return
	}

	resp.Append(r.refresh(ctx, m)...)
}

func (r *deviceBringupResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan deviceBringupResourceModel
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

func (r *deviceBringupResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state deviceBringupResourceModel
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

func (r *deviceBringupResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan deviceBringupResourceModel
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

func (r *deviceBringupResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	resp.Diagnostics.AddWarning(
		"Device bringup status unchanged on the server",
		"There is no API endpoint to reverse a device bringup action; this resource is only being removed from Terraform state.",
	)
}
