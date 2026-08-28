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
	_ resource.Resource                = &lanSegmentResource{}
	_ resource.ResourceWithConfigure   = &lanSegmentResource{}
	_ resource.ResourceWithImportState = &lanSegmentResource{}
)

func NewLanSegmentResource() resource.Resource {
	return &lanSegmentResource{}
}

type lanSegmentResource struct {
	pd *providerData
}

type lanSegmentResourceModel struct {
	ID                   types.String `tfsdk:"id"`
	Name                 types.String `tfsdk:"name"`
	Description          types.String `tfsdk:"description"`
	AssociatedInterfaces types.Int64  `tfsdk:"associated_interfaces"`
	EdgeReferences       types.Int64  `tfsdk:"edge_references"`
	SiteListReferences   types.Int64  `tfsdk:"site_list_references"`
}

func (r *lanSegmentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_lan_segment"
}

func (r *lanSegmentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A global LAN segment. There is no update endpoint, so every field is force-new.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"description": schema.StringAttribute{
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"associated_interfaces": schema.Int64Attribute{
				Computed: true,
			},
			"edge_references": schema.Int64Attribute{
				Computed: true,
			},
			"site_list_references": schema.Int64Attribute{
				Computed: true,
			},
		},
	}
}

func (r *lanSegmentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.pd = configurePD(req.ProviderData, &resp.Diagnostics)
}

func (m *lanSegmentResourceModel) applyEntry(e *sdk.V1GlobalLanSegmentsGetResponseEntry) {
	if e.Id != nil {
		m.ID = types.StringValue(int64ID(*e.Id))
	}
	m.Name = types.StringPointerValue(e.Name)
	m.Description = types.StringPointerValue(e.Description)
	m.AssociatedInterfaces = types.Int64PointerValue(intPtr32To64(e.AssociatedInterfaces))
	m.EdgeReferences = types.Int64PointerValue(intPtr32To64(e.EdgeReferences))
	m.SiteListReferences = types.Int64PointerValue(intPtr32To64(e.SiteListReferences))
}

func (r *lanSegmentResource) findByID(ctx context.Context, id string) (*sdk.V1GlobalLanSegmentsGetResponseEntry, bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	out, httpResp, err := r.pd.api.DefaultAPI.V1GlobalLanSegmentsGet(ctx).Authorization(r.pd.token).Execute()
	closeBody(httpResp)
	if err != nil {
		diags.AddError("Unable to list LAN segments", apiErrorDetail(err))
		return nil, false, diags
	}
	if out == nil {
		return nil, false, diags
	}
	for i := range out.Entries {
		if int64PtrID(out.Entries[i].Id) == id {
			return &out.Entries[i], true, diags
		}
	}
	return nil, false, diags
}

func (r *lanSegmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan lanSegmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := sdk.V1GlobalLanSegmentsPostRequest{
		Name:        plan.Name.ValueStringPointer(),
		Description: plan.Description.ValueStringPointer(),
	}
	out, httpResp, err := r.pd.api.DefaultAPI.V1GlobalLanSegmentsPost(ctx).
		Authorization(r.pd.token).
		V1GlobalLanSegmentsPostRequest(body).
		Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create LAN segment", apiErrorDetail(err))
		return
	}
	if out == nil || out.Id == nil {
		resp.Diagnostics.AddError("Unable to create LAN segment", "API returned an empty response")
		return
	}

	entry, found, diags := r.findByID(ctx, int64ID(*out.Id))
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.Diagnostics.AddError("Unable to read created LAN segment", "LAN segment was created but could not be read back")
		return
	}

	plan.applyEntry(entry)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *lanSegmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state lanSegmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	entry, found, diags := r.findByID(ctx, state.ID.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	state.applyEntry(entry)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update should never fire in practice since every attribute is RequiresReplace;
// it exists only to satisfy the resource.Resource interface.
func (r *lanSegmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan lanSegmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *lanSegmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state lanSegmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := parseInt64ID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid LAN segment id", err.Error())
		return
	}

	httpResp, err := r.pd.api.DefaultAPI.V1GlobalLanSegmentsIdDelete(ctx, id).Authorization(r.pd.token).Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to delete LAN segment", apiErrorDetail(err))
	}
}

func (r *lanSegmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
