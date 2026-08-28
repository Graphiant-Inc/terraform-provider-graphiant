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
	_ resource.Resource                = &routeTagResource{}
	_ resource.ResourceWithConfigure   = &routeTagResource{}
	_ resource.ResourceWithImportState = &routeTagResource{}
)

func NewRouteTagResource() resource.Resource {
	return &routeTagResource{}
}

type routeTagResource struct {
	pd *providerData
}

type routeTagResourceModel struct {
	ID        types.String `tfsdk:"id"`
	LevelZero types.String `tfsdk:"level_zero"`
	LevelOne  types.String `tfsdk:"level_one"`
	LevelTwo  types.String `tfsdk:"level_two"`
}

func (r *routeTagResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_route_tag"
}

func (r *routeTagResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "An enterprise route tag (a 3-level hierarchical tag used to scope policies to sites/segments). " +
			"There is no update endpoint, so every field is force-new. The read endpoint returns tags as a " +
			"recursive tree (not a flat level_zero/one/two record), so Read only confirms the tag id still exists " +
			"in that tree — level_zero/one/two are preserved from configuration/prior state rather than refreshed.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"level_zero": schema.StringAttribute{
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"level_one": schema.StringAttribute{
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"level_two": schema.StringAttribute{
				Optional:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
		},
	}
}

func (r *routeTagResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.pd = configurePD(req.ProviderData, &resp.Diagnostics)
}

// findTagID recursively searches the route-tag tree for a matching id.
func findTagID(elements []sdk.ManaV2RouteTagElement, id int64) bool {
	for _, e := range elements {
		if e.Id != nil && *e.Id == id {
			return true
		}
		if findTagID(e.NextSet, id) {
			return true
		}
	}
	return false
}

func (r *routeTagResource) exists(ctx context.Context, id int64) (bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	out, httpResp, err := r.pd.api.DefaultAPI.V1PolicyRouteTagSetsTagsGet(ctx).Authorization(r.pd.token).Execute()
	closeBody(httpResp)
	if err != nil {
		diags.AddError("Unable to list route tags", apiErrorDetail(err))
		return false, diags
	}
	if out == nil {
		return false, diags
	}
	return findTagID(out.Tags, id), diags
}

func (r *routeTagResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan routeTagResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tag := &sdk.ManaV2RouteTag{
		LevelZero: plan.LevelZero.ValueStringPointer(),
		LevelOne:  plan.LevelOne.ValueStringPointer(),
		LevelTwo:  plan.LevelTwo.ValueStringPointer(),
	}
	body := sdk.V1PolicyRouteTagSetsPostRequest{Tag: tag}
	out, httpResp, err := r.pd.api.DefaultAPI.V1PolicyRouteTagSetsPost(ctx).
		Authorization(r.pd.token).
		V1PolicyRouteTagSetsPostRequest(body).
		Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create route tag", apiErrorDetail(err))
		return
	}
	if out == nil || out.Id == nil {
		resp.Diagnostics.AddError("Unable to create route tag", "API returned an empty response")
		return
	}

	plan.ID = types.StringValue(int64ID(*out.Id))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *routeTagResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state routeTagResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := parseInt64ID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid route tag id", err.Error())
		return
	}

	found, diags := r.exists(ctx, id)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

// Update should never fire in practice since every attribute is RequiresReplace;
// it exists only to satisfy the resource.Resource interface.
func (r *routeTagResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan routeTagResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *routeTagResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state routeTagResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := parseInt64ID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid route tag id", err.Error())
		return
	}

	_, httpResp, err := r.pd.api.DefaultAPI.V1PolicyRouteTagSetsIdDelete(ctx, id).Authorization(r.pd.token).Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to delete route tag", apiErrorDetail(err))
	}
}

func (r *routeTagResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
