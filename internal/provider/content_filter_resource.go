package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/boolvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	sdk "github.com/Graphiant-Inc/graphiant-sdk-go"
)

var (
	_ resource.Resource                = &contentFilterResource{}
	_ resource.ResourceWithConfigure   = &contentFilterResource{}
	_ resource.ResourceWithImportState = &contentFilterResource{}
)

func NewContentFilterResource() resource.Resource {
	return &contentFilterResource{}
}

type contentFilterResource struct {
	pd *providerData
}

type contentFilterRuleModel struct {
	DomainCategoryID   types.Int64 `tfsdk:"domain_category_id"`
	ExceptionWildcards types.List  `tfsdk:"exception_wildcards"`
}

var contentFilterRuleAttrTypes = map[string]attrType{
	"domain_category_id":  types.Int64Type,
	"exception_wildcards": types.ListType{ElemType: types.StringType},
}

type contentFilterResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	LanNames    types.List   `tfsdk:"lan_names"`
	Rules       types.List   `tfsdk:"rules"`
	SiteListID  types.Int64  `tfsdk:"site_list_id"`
	UseAllSites types.Bool   `tfsdk:"use_all_sites"`
}

func (r *contentFilterResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_content_filter"
}

func (r *contentFilterResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A global content filter: a set of domain-category rules applied either to all sites or to a specific site list.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"lan_names": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "LAN segments this filter applies on.",
			},
			"site_list_id": schema.Int64Attribute{
				Optional:    true,
				Description: "Site list this filter applies to. Mutually exclusive with use_all_sites.",
				Validators: []validator.Int64{
					int64validator.ConflictsWith(path.Expressions{path.MatchRoot("use_all_sites")}...),
				},
			},
			"use_all_sites": schema.BoolAttribute{
				Optional:    true,
				Description: "Apply this filter to every site in the tenant. Mutually exclusive with site_list_id.",
				Validators: []validator.Bool{
					boolvalidator.ConflictsWith(path.Expressions{path.MatchRoot("site_list_id")}...),
				},
			},
			"rules": schema.ListNestedAttribute{
				Optional: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"domain_category_id": schema.Int64Attribute{
							Required:    true,
							Description: "ID of the category whose traffic is blocked.",
						},
						"exception_wildcards": schema.ListAttribute{
							Optional:    true,
							ElementType: types.StringType,
						},
					},
				},
			},
		},
	}
}

func (r *contentFilterResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.pd = configurePD(req.ProviderData, &resp.Diagnostics)
}

func (m *contentFilterResourceModel) buildConfig(ctx context.Context) (*sdk.ManaV2GlobalContentFilterConfig, diag.Diagnostics) {
	cfg := &sdk.ManaV2GlobalContentFilterConfig{
		Name:        m.Name.ValueStringPointer(),
		SiteListId:  m.SiteListID.ValueInt64Pointer(),
		UseAllSites: m.UseAllSites.ValueBoolPointer(),
	}

	if !m.LanNames.IsNull() && !m.LanNames.IsUnknown() {
		diags := m.LanNames.ElementsAs(ctx, &cfg.LanNames, false)
		if diags.HasError() {
			return nil, diags
		}
	}

	if !m.Rules.IsNull() && !m.Rules.IsUnknown() {
		var rules []contentFilterRuleModel
		diags := m.Rules.ElementsAs(ctx, &rules, false)
		if diags.HasError() {
			return nil, diags
		}
		for _, rule := range rules {
			r := sdk.ManaV2GlobalContentFilterRule{DomainCategoryId: rule.DomainCategoryID.ValueInt64Pointer()}
			if !rule.ExceptionWildcards.IsNull() && !rule.ExceptionWildcards.IsUnknown() {
				diags := rule.ExceptionWildcards.ElementsAs(ctx, &r.ExceptionWildcards, false)
				if diags.HasError() {
					return nil, diags
				}
			}
			cfg.Rules = append(cfg.Rules, r)
		}
	}

	return cfg, nil
}

func (m *contentFilterResourceModel) applyConfig(ctx context.Context, cfg *sdk.ManaV2GlobalContentFilterConfig) diag.Diagnostics {
	m.Name = types.StringPointerValue(cfg.Name)
	m.SiteListID = types.Int64PointerValue(cfg.SiteListId)
	m.UseAllSites = types.BoolPointerValue(cfg.UseAllSites)

	lanNames, diags := types.ListValueFrom(ctx, types.StringType, cfg.LanNames)
	if diags.HasError() {
		return diags
	}
	m.LanNames = lanNames

	if len(cfg.Rules) == 0 {
		m.Rules = types.ListNull(types.ObjectType{AttrTypes: contentFilterRuleAttrTypes})
	} else {
		rules := make([]contentFilterRuleModel, 0, len(cfg.Rules))
		for _, rule := range cfg.Rules {
			wildcards, d := types.ListValueFrom(ctx, types.StringType, rule.ExceptionWildcards)
			if d.HasError() {
				return d
			}
			rules = append(rules, contentFilterRuleModel{
				DomainCategoryID:   types.Int64PointerValue(rule.DomainCategoryId),
				ExceptionWildcards: wildcards,
			})
		}
		rulesList, diags2 := types.ListValueFrom(ctx, types.ObjectType{AttrTypes: contentFilterRuleAttrTypes}, rules)
		if diags2.HasError() {
			return diags2
		}
		m.Rules = rulesList
	}

	return nil
}

func (r *contentFilterResource) readByID(ctx context.Context, id int64) (*sdk.ManaV2GlobalContentFilterConfig, bool, diag.Diagnostics) {
	var diags diag.Diagnostics
	out, httpResp, err := r.pd.api.DefaultAPI.V1GlobalContentFiltersGlobalContentFilterIdGet(ctx, id).Authorization(r.pd.token).Execute()
	if err != nil {
		if isNotFound(httpResp) {
			closeBody(httpResp)
			return nil, false, diags
		}
		closeBody(httpResp)
		diags.AddError("Unable to read content filter", apiErrorDetail(err))
		return nil, false, diags
	}
	closeBody(httpResp)
	if out == nil || out.Config == nil {
		return nil, false, diags
	}
	return out.Config, true, diags
}

func (r *contentFilterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan contentFilterResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg, diags := plan.buildConfig(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := sdk.V1GlobalContentFiltersPostRequest{Config: cfg}
	out, httpResp, err := r.pd.api.DefaultAPI.V1GlobalContentFiltersPost(ctx).
		Authorization(r.pd.token).
		V1GlobalContentFiltersPostRequest(body).
		Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create content filter", apiErrorDetail(err))
		return
	}
	if out == nil || out.GlobalContentFilterId == nil {
		resp.Diagnostics.AddError("Unable to create content filter", "API returned an empty response")
		return
	}

	created, found, diags2 := r.readByID(ctx, *out.GlobalContentFilterId)
	resp.Diagnostics.Append(diags2...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.Diagnostics.AddError("Unable to read created content filter", "content filter was created but could not be read back")
		return
	}

	plan.ID = types.StringValue(int64ID(*out.GlobalContentFilterId))
	resp.Diagnostics.Append(plan.applyConfig(ctx, created)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *contentFilterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state contentFilterResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := parseInt64ID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid content filter id", err.Error())
		return
	}

	cfg, found, diags := r.readByID(ctx, id)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(state.applyConfig(ctx, cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *contentFilterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan contentFilterResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg, diags := plan.buildConfig(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := parseInt64ID(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid content filter id", err.Error())
		return
	}

	body := sdk.V1GlobalContentFiltersGlobalContentFilterIdPutRequest{Config: cfg}
	_, httpResp, err := r.pd.api.DefaultAPI.V1GlobalContentFiltersGlobalContentFilterIdPut(ctx, id).
		Authorization(r.pd.token).
		V1GlobalContentFiltersGlobalContentFilterIdPutRequest(body).
		Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to update content filter", apiErrorDetail(err))
		return
	}

	updated, found, diags2 := r.readByID(ctx, id)
	resp.Diagnostics.Append(diags2...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !found {
		resp.Diagnostics.AddError("Unable to update content filter", "content filter no longer exists")
		return
	}

	resp.Diagnostics.Append(plan.applyConfig(ctx, updated)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *contentFilterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state contentFilterResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := parseInt64ID(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid content filter id", err.Error())
		return
	}

	_, httpResp, err := r.pd.api.DefaultAPI.V1GlobalContentFiltersGlobalContentFilterIdDelete(ctx, id).Authorization(r.pd.token).Execute()
	closeBody(httpResp)
	if err != nil {
		resp.Diagnostics.AddError("Unable to delete content filter", apiErrorDetail(err))
	}
}

func (r *contentFilterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
