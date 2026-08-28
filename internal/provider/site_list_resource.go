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
	_ resource.Resource                = &siteListResource{}
	_ resource.ResourceWithConfigure   = &siteListResource{}
	_ resource.ResourceWithImportState = &siteListResource{}
)

func NewSiteListResource() resource.Resource {
	return &siteListResource{}
}

type siteListResource struct {
	pd *providerData
}

type siteListRouteTagModel struct {
	LevelZero types.Int64 `tfsdk:"level_zero"`
	LevelOne  types.Int64 `tfsdk:"level_one"`
	LevelTwo  types.Int64 `tfsdk:"level_two"`
}

var siteListRouteTagAttrTypes = map[string]attrType{
	"level_zero": types.Int64Type,
	"level_one":  types.Int64Type,
	"level_two":  types.Int64Type,
}

type siteListEntryModel struct {
	SiteID   types.Int64  `tfsdk:"site_id"`
	RouteTag types.Object `tfsdk:"route_tag"`
}

var siteListEntryAttrTypes = map[string]attrType{
	"site_id":   types.Int64Type,
	"route_tag": types.ObjectType{AttrTypes: siteListRouteTagAttrTypes},
}

var siteListEntryListType = types.ObjectType{AttrTypes: siteListEntryAttrTypes}

type siteListResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Entries     types.List   `tfsdk:"entries"`
}

func (r *siteListResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_site_list"
}

func (r *siteListResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A named list of sites (by ID or route tag), used as a scope target in policies. " +
			"The update endpoint has no name field, so name forces recreation on change.",
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
				Optional: true,
			},
			"entries": schema.ListNestedAttribute{
				Optional: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"site_id": schema.Int64Attribute{
							Optional:    true,
							Description: "Direct site ID. Mutually exclusive with route_tag within a single entry.",
						},
						"route_tag": schema.SingleNestedAttribute{
							Optional:    true,
							Description: "Match sites by route tag instead of a direct ID.",
							Attributes: map[string]schema.Attribute{
								"level_zero": schema.Int64Attribute{Optional: true},
								"level_one":  schema.Int64Attribute{Optional: true},
								"level_two":  schema.Int64Attribute{Optional: true},
							},
						},
					},
				},
			},
		},
	}
}

func (r *siteListResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.pd = configurePD(req.ProviderData, &resp.Diagnostics)
}

func buildSiteListEntries(ctx context.Context, list types.List) ([]sdk.ManaV2SiteListEntry, diag.Diagnostics) {
	if list.IsNull() || list.IsUnknown() {
		return nil, nil
	}
	var models []siteListEntryModel
	diags := list.ElementsAs(ctx, &models, false)
	if diags.HasError() {
		return nil, diags
	}

	entries := make([]sdk.ManaV2SiteListEntry, 0, len(models))
	for _, m := range models {
		e := sdk.ManaV2SiteListEntry{Regular: m.SiteID.ValueInt64Pointer()}
		if !m.RouteTag.IsNull() && !m.RouteTag.IsUnknown() {
			var rt siteListRouteTagModel
			d := m.RouteTag.As(ctx, &rt, objectAsOptions)
			if d.HasError() {
				return nil, d
			}
			e.Tag = &sdk.ManaV2RouteTagId{
				LevelZero: rt.LevelZero.ValueInt64Pointer(),
				LevelOne:  rt.LevelOne.ValueInt64Pointer(),
				LevelTwo:  rt.LevelTwo.ValueInt64Pointer(),
			}
		}
		entries = append(entries, e)
	}
	return entries, nil
}

func siteListEntriesToList(ctx context.Context, entries []sdk.ManaV2SiteListEntry) (types.List, diag.Diagnostics) {
	models := make([]siteListEntryModel, 0, len(entries))
	for _, e := range entries {
		m := siteListEntryModel{SiteID: types.Int64PointerValue(e.Regular)}
		if e.Tag != nil {
			rt := siteListRouteTagModel{
				LevelZero: types.Int64PointerValue(e.Tag.LevelZero),
				LevelOne:  types.Int64PointerValue(e.Tag.LevelOne),
				LevelTwo:  types.Int64PointerValue(e.Tag.LevelTwo),
			}
			obj, diags := types.ObjectValueFrom(ctx, siteListRouteTagAttrTypes, rt)
			if diags.HasError() {
				return types.ListNull(siteListEntryListType), diags
			}
			m.RouteTag = obj
		} else {
			m.RouteTag = types.ObjectNull(siteListRouteTagAttrTypes)
		}
		models = append(models, m)
	}
	return types.ListValueFrom(ctx, siteListEntryListType, models)
}

// readByID fetches description+entries (the id-scoped GET doesn't return name).
func (r *siteListResource) readByID(ctx context.Context, id int64) (*sdk.V1GlobalSiteListsIdGetResponse, bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	out, httpResp, err := r.pd.api.DefaultAPI.V1GlobalSiteListsIdGet(ctx, id).Authorization(r.pd.token).Execute()
	if err != nil {
		if isNotFound(httpResp) {
			closeBody(httpResp)
			return nil, false, diags
		}
		closeBody(httpResp)
		diags.AddError("Unable to read site list", apiErrorDetail(err))
		return nil, false, diags
	}
	closeBody(httpResp)
	return out, true, diags
}

// findName recovers name from the list endpoint, the only one that reports it.
func (r *siteListResource) findName(ctx context.Context, id int64) (string, bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	out, httpResp, err := r.pd.api.DefaultAPI.V1GlobalSiteListsGet(ctx).Authorization(r.pd.token).Execute()
	closeBody(httpResp)
	if err != nil {
		diags.AddError("Unable to list site lists", apiErrorDetail(err))
		return "", false, diags
	}
	if out == nil {
		return "", false, diags
	}
	for _, e := range out.Entries {
		if int64PtrID(e.Id) == int64ID(id) {
			if e.Name == nil {
				return "", true, diags
			}
			return *e.Name, true, diags
		}
	}
	return "", false, diags
}

func (r *siteListResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan siteListResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	entries, diags := buildSiteListEntries(ctx, plan.Entries)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := sdk.V1GlobalSiteListsPostRequest{
		Name:        plan.Name.ValueStringPointer(),
		Description: plan.Description.ValueStringPointer(),
		Entries:     entries,
	}

	out, httpResp, err := r.pd.api.DefaultAPI.V1GlobalSiteListsPost(ctx).
		Authorization(r.pd.token).
		V1GlobalSiteListsPostRequest(body).
		Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create site list", apiErrorDetail(err))
		return
	}
	if out == nil || out.Id == nil {
		resp.Diagnostics.AddError("Unable to create site list", "API returned an empty response")
		return
	}

	got, found, diags2 := r.readByID(ctx, *out.Id)
	resp.Diagnostics.Append(diags2...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.Diagnostics.AddError("Unable to read created site list", "site list was created but could not be read back")
		return
	}

	plan.ID = types.StringValue(int64ID(*out.Id))
	plan.Description = types.StringPointerValue(got.Description)
	entriesList, diags3 := siteListEntriesToList(ctx, got.Entries)
	resp.Diagnostics.Append(diags3...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.Entries = entriesList
	// plan.Name is left as configured — it was just submitted verbatim and isn't echoed by readByID.

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *siteListResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state siteListResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := parseInt64ID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid site list id", err.Error())
		return
	}

	got, found, diags := r.readByID(ctx, id)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	name, nameFound, diags2 := r.findName(ctx, id)
	resp.Diagnostics.Append(diags2...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !nameFound {
		resp.State.RemoveResource(ctx)
		return
	}

	state.Name = types.StringValue(name)
	state.Description = types.StringPointerValue(got.Description)
	entriesList, diags3 := siteListEntriesToList(ctx, got.Entries)
	resp.Diagnostics.Append(diags3...)
	if resp.Diagnostics.HasError() {
		return
	}
	state.Entries = entriesList

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *siteListResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan siteListResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	entries, diags := buildSiteListEntries(ctx, plan.Entries)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := parseInt64ID(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid site list id", err.Error())
		return
	}

	body := sdk.V1GlobalSiteListsIdPutRequest{
		Description: plan.Description.ValueStringPointer(),
		Entries:     entries,
	}
	_, httpResp, err := r.pd.api.DefaultAPI.V1GlobalSiteListsIdPut(ctx, id).
		Authorization(r.pd.token).
		V1GlobalSiteListsIdPutRequest(body).
		Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update site list", apiErrorDetail(err))
		return
	}

	got, found, diags2 := r.readByID(ctx, id)
	resp.Diagnostics.Append(diags2...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.Diagnostics.AddError("Unable to update site list", "site list no longer exists")
		return
	}

	plan.Description = types.StringPointerValue(got.Description)
	entriesList, diags3 := siteListEntriesToList(ctx, got.Entries)
	resp.Diagnostics.Append(diags3...)
	if resp.Diagnostics.HasError() {
		return
	}
	plan.Entries = entriesList

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *siteListResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state siteListResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := parseInt64ID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid site list id", err.Error())
		return
	}

	httpResp, err := r.pd.api.DefaultAPI.V1GlobalSiteListsIdDelete(ctx, id).Authorization(r.pd.token).Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to delete site list", apiErrorDetail(err))
	}
}

func (r *siteListResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
