package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	graphiant "github.com/Graphiant-Inc/graphiant-sdk-go"
)

var (
	_ resource.Resource                = &siteResource{}
	_ resource.ResourceWithConfigure   = &siteResource{}
	_ resource.ResourceWithImportState = &siteResource{}
)

func NewSiteResource() resource.Resource {
	return &siteResource{}
}

type siteResource struct {
	client *gClient
}

// siteResourceModel mirrors manaV2Site together with the subset of fields
// that can be set on create/update via manaV2NewSite.
type siteResourceModel struct {
	Id                     types.Int64    `tfsdk:"id"`
	EnterpriseId           types.Int64    `tfsdk:"enterprise_id"`
	Name                   types.String   `tfsdk:"name"`
	Notes                  types.String   `tfsdk:"notes"`
	Location               *locationModel `tfsdk:"location"`
	Address                types.String   `tfsdk:"address"`
	EdgeCount              types.Int64    `tfsdk:"edge_count"`
	SegmentCount           types.Int64    `tfsdk:"segment_count"`
	PolicyReferenceCount   types.Int64    `tfsdk:"policy_reference_count"`
	SiteListReferenceCount types.Int64    `tfsdk:"site_list_reference_count"`
	Tags                   types.List     `tfsdk:"tags"`
	CreatedAt              types.String   `tfsdk:"created_at"`
	UpdatedAt              types.String   `tfsdk:"updated_at"`
}

func (r *siteResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_site"
}

func (r *siteResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Graphiant site (a physical or logical location containing one or more edge devices).",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				Computed:    true,
				Description: "Site identifier assigned by the Graphiant controller.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"enterprise_id": schema.Int64Attribute{
				Optional:    true,
				Description: "Enterprise to create the site under. Only meaningful for reseller/multi-tenant accounts; cannot be changed after creation.",
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Site name.",
			},
			"notes": schema.StringAttribute{
				Optional:    true,
				Description: "Free-form notes about the site.",
			},
			"location":                  locationSchemaAttribute(false),
			"address":                   schema.StringAttribute{Computed: true, Description: "Resolved postal address for the site location."},
			"edge_count":                schema.Int64Attribute{Computed: true, Description: "Number of edge devices onboarded at this site."},
			"segment_count":             schema.Int64Attribute{Computed: true, Description: "Number of LAN segments configured at this site."},
			"policy_reference_count":    schema.Int64Attribute{Computed: true, Description: "Number of policies referencing this site."},
			"site_list_reference_count": schema.Int64Attribute{Computed: true, Description: "Number of site lists referencing this site."},
			"tags": schema.ListAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Tags applied to the site.",
			},
			"created_at": schema.StringAttribute{Computed: true, Description: "Creation timestamp (RFC3339, UTC)."},
			"updated_at": schema.StringAttribute{Computed: true, Description: "Last update timestamp (RFC3339, UTC)."},
		},
	}
}

func (r *siteResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*gClient)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("expected *gClient, got: %T", req.ProviderData))
		return
	}
	r.client = client
}

func (r *siteResource) expandNewSite(ctx context.Context, m *siteResourceModel) *graphiant.ManaV2NewSite {
	site := graphiant.NewManaV2NewSiteWithDefaults()
	if v := strPtr(m.Name); v != nil {
		site.SetName(*v)
	}
	if v := strPtr(m.Notes); v != nil {
		site.SetNotes(*v)
	}
	if m.Location != nil {
		site.SetLocation(*expandLocation(m.Location))
	}
	return site
}

func (r *siteResource) flatten(ctx context.Context, site *graphiant.ManaV2Site, m *siteResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics
	m.Id = int64Value(site.Id)
	m.Name = strValue(site.Name)
	m.Notes = strValue(site.Notes)
	m.Location = flattenLocation(site.Location)
	m.Address = strValue(site.Address)
	m.EdgeCount = int32Value(site.EdgeCount)
	m.SegmentCount = int32Value(site.SegmentCount)
	m.PolicyReferenceCount = int32Value(site.PolicyReferenceCount)
	m.SiteListReferenceCount = int32Value(site.SiteListReferenceCount)
	m.CreatedAt = timestampValue(site.CreatedAt)
	m.UpdatedAt = timestampValue(site.UpdatedAt)

	tags, d := stringListValue(ctx, site.Tags)
	diags.Append(d...)
	m.Tags = tags
	return diags
}

func (r *siteResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan siteResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := graphiant.NewV1SitesPostRequestWithDefaults()
	body.SetSite(*r.expandNewSite(ctx, &plan))
	if v := int64Ptr(plan.EnterpriseId); v != nil {
		body.SetEnterpriseId(*v)
	}

	out, httpRes, err := r.client.api.DefaultAPI.V1SitesPost(ctx).Authorization(r.client.authHeader()).V1SitesPostRequest(*body).Execute()
	if httpRes != nil {
		defer httpRes.Body.Close()
	}
	if err != nil {
		resp.Diagnostics.AddError("Error creating site", err.Error())
		return
	}
	if out == nil || out.Site == nil {
		resp.Diagnostics.AddError("Error creating site", "the API did not return the created site")
		return
	}

	resp.Diagnostics.Append(r.flatten(ctx, out.Site, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// findSite looks up a site by ID. There is no get-by-id endpoint for sites,
// so this lists every site and filters client-side.
func (r *siteResource) findSite(ctx context.Context, id int64) (*graphiant.ManaV2Site, error) {
	out, httpRes, err := r.client.api.DefaultAPI.V1SitesGet(ctx).Authorization(r.client.authHeader()).Execute()
	if httpRes != nil {
		defer httpRes.Body.Close()
	}
	if err != nil {
		return nil, err
	}
	if out == nil {
		return nil, nil
	}
	for _, s := range out.GetSites() {
		if s.GetId() == id {
			return &s, nil
		}
	}
	return nil, nil
}

func (r *siteResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state siteResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	site, err := r.findSite(ctx, state.Id.ValueInt64())
	if err != nil {
		resp.Diagnostics.AddError("Error reading site", err.Error())
		return
	}
	if site == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(r.flatten(ctx, site, &state)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *siteResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state siteResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	body := graphiant.NewV1SitesSiteIdPostRequestWithDefaults()
	body.SetSite(*r.expandNewSite(ctx, &plan))

	out, httpRes, err := r.client.api.DefaultAPI.V1SitesSiteIdPost(ctx, state.Id.ValueInt64()).Authorization(r.client.authHeader()).V1SitesSiteIdPostRequest(*body).Execute()
	if httpRes != nil {
		defer httpRes.Body.Close()
	}
	if err != nil {
		resp.Diagnostics.AddError("Error updating site", err.Error())
		return
	}
	if out == nil || out.Site == nil {
		resp.Diagnostics.AddError("Error updating site", "the API did not return the updated site")
		return
	}

	resp.Diagnostics.Append(r.flatten(ctx, out.Site, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *siteResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state siteResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	httpRes, err := r.client.api.DefaultAPI.V1SitesSiteIdDelete(ctx, state.Id.ValueInt64()).Authorization(r.client.authHeader()).Execute()
	if httpRes != nil {
		defer httpRes.Body.Close()
	}
	if err != nil {
		resp.Diagnostics.AddError("Error deleting site", err.Error())
	}
}

func (r *siteResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
