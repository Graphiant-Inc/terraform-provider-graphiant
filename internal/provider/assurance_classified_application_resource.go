package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	sdk "github.com/Graphiant-Inc/graphiant-sdk-go"
)

var (
	_ resource.Resource                = &assuranceClassifiedApplicationResource{}
	_ resource.ResourceWithConfigure   = &assuranceClassifiedApplicationResource{}
	_ resource.ResourceWithImportState = &assuranceClassifiedApplicationResource{}
)

func NewAssuranceClassifiedApplicationResource() resource.Resource {
	return &assuranceClassifiedApplicationResource{}
}

type assuranceClassifiedApplicationResource struct {
	pd *providerData
}

type assuranceClassifiedApplicationResourceModel struct {
	ID           types.String `tfsdk:"id"`
	AppName      types.String `tfsdk:"app_name"`
	IPPrefixList types.List   `tfsdk:"ip_prefix_list"`
	PortList     types.List   `tfsdk:"port_list"`
	ProtocolList types.List   `tfsdk:"protocol_list"`
}

func (r *assuranceClassifiedApplicationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_assurance_classified_application"
}

func (r *assuranceClassifiedApplicationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A custom application classification rule for data assurance (matches traffic by IP prefix/port/protocol).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Server-assigned classification entry ID (string, not numeric).",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"app_name": schema.StringAttribute{
				Required: true,
			},
			"ip_prefix_list": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
			},
			"port_list": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
			},
			"protocol_list": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
			},
		},
	}
}

func (r *assuranceClassifiedApplicationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.pd = configurePD(req.ProviderData, &resp.Diagnostics)
}

func (m *assuranceClassifiedApplicationResourceModel) applyEntry(ctx context.Context, e *sdk.AssuranceClassifiedApplication) diag.Diagnostics {
	var diags diag.Diagnostics
	m.AppName = types.StringPointerValue(e.AppName)

	ipPrefixList, d := types.ListValueFrom(ctx, types.StringType, e.IpPrefixList)
	diags.Append(d...)
	m.IPPrefixList = ipPrefixList

	portList, d2 := types.ListValueFrom(ctx, types.StringType, e.PortList)
	diags.Append(d2...)
	m.PortList = portList

	protocolList, d3 := types.ListValueFrom(ctx, types.StringType, e.ProtocolList)
	diags.Append(d3...)
	m.ProtocolList = protocolList

	return diags
}

func (r *assuranceClassifiedApplicationResource) findByID(ctx context.Context, id string) (*sdk.AssuranceClassifiedApplication, bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	out, httpResp, err := r.pd.api.DefaultAPI.V2AssuranceGetclassifiedapplicationlistGet(ctx).Authorization(r.pd.token).Execute()
	closeBody(httpResp)
	if err != nil {
		diags.AddError("Unable to list classified applications", apiErrorDetail(err))
		return nil, false, diags
	}
	if out == nil {
		return nil, false, diags
	}
	for i := range out.ClassifiedApplicationList {
		e := &out.ClassifiedApplicationList[i]
		if e.ClassificationEntryId != nil && *e.ClassificationEntryId == id {
			return e, true, diags
		}
	}
	return nil, false, diags
}

func (r *assuranceClassifiedApplicationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan assuranceClassifiedApplicationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := sdk.V2AssuranceCreateclassifiedapplicationPostRequest{AppName: plan.AppName.ValueStringPointer()}
	if !plan.IPPrefixList.IsNull() && !plan.IPPrefixList.IsUnknown() {
		resp.Diagnostics.Append(plan.IPPrefixList.ElementsAs(ctx, &body.IpPrefixList, false)...)
	}
	if !plan.PortList.IsNull() && !plan.PortList.IsUnknown() {
		resp.Diagnostics.Append(plan.PortList.ElementsAs(ctx, &body.PortList, false)...)
	}
	if !plan.ProtocolList.IsNull() && !plan.ProtocolList.IsUnknown() {
		resp.Diagnostics.Append(plan.ProtocolList.ElementsAs(ctx, &body.ProtocolList, false)...)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	out, httpResp, err := r.pd.api.DefaultAPI.V2AssuranceCreateclassifiedapplicationPost(ctx).
		Authorization(r.pd.token).
		V2AssuranceCreateclassifiedapplicationPostRequest(body).
		Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create classified application", apiErrorDetail(err))
		return
	}
	if out == nil || out.ClassificationEntryId == nil {
		resp.Diagnostics.AddError("Unable to create classified application", "API returned an empty response")
		return
	}

	got, found, diags := r.findByID(ctx, *out.ClassificationEntryId)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.Diagnostics.AddError("Unable to read created classified application", "entry was created but could not be read back")
		return
	}

	plan.ID = types.StringValue(*out.ClassificationEntryId)
	resp.Diagnostics.Append(plan.applyEntry(ctx, got)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *assuranceClassifiedApplicationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state assuranceClassifiedApplicationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	got, found, diags := r.findByID(ctx, state.ID.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(state.applyEntry(ctx, got)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *assuranceClassifiedApplicationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan assuranceClassifiedApplicationResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := plan.ID.ValueString()
	entry := sdk.AssuranceClassifiedApplication{
		ClassificationEntryId: &id,
		AppName:               plan.AppName.ValueStringPointer(),
	}
	if !plan.IPPrefixList.IsNull() && !plan.IPPrefixList.IsUnknown() {
		resp.Diagnostics.Append(plan.IPPrefixList.ElementsAs(ctx, &entry.IpPrefixList, false)...)
	}
	if !plan.PortList.IsNull() && !plan.PortList.IsUnknown() {
		resp.Diagnostics.Append(plan.PortList.ElementsAs(ctx, &entry.PortList, false)...)
	}
	if !plan.ProtocolList.IsNull() && !plan.ProtocolList.IsUnknown() {
		resp.Diagnostics.Append(plan.ProtocolList.ElementsAs(ctx, &entry.ProtocolList, false)...)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	body := sdk.V2AssuranceUpdateclassifiedapplicationPostRequest{ClassifiedApplicationList: []sdk.AssuranceClassifiedApplication{entry}}
	_, httpResp, err := r.pd.api.DefaultAPI.V2AssuranceUpdateclassifiedapplicationPost(ctx).
		Authorization(r.pd.token).
		V2AssuranceUpdateclassifiedapplicationPostRequest(body).
		Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update classified application", apiErrorDetail(err))
		return
	}

	got, found, diags := r.findByID(ctx, id)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.Diagnostics.AddError("Unable to update classified application", "entry no longer exists")
		return
	}

	resp.Diagnostics.Append(plan.applyEntry(ctx, got)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *assuranceClassifiedApplicationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state assuranceClassifiedApplicationResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	_, httpResp, err := r.pd.api.DefaultAPI.V2AssuranceDeleteclassifiedapplicationDelete(ctx).
		ClassificationEntryIdList([]string{state.ID.ValueString()}).
		Authorization(r.pd.token).
		Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to delete classified application", apiErrorDetail(err))
	}
}

func (r *assuranceClassifiedApplicationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
