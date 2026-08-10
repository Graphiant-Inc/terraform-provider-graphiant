package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	graphiant "github.com/Graphiant-Inc/graphiant-sdk-go"
)

var (
	_ datasource.DataSource              = &siteDataSource{}
	_ datasource.DataSourceWithConfigure = &siteDataSource{}
)

func NewSiteDataSource() datasource.DataSource {
	return &siteDataSource{}
}

type siteDataSource struct {
	client *gClient
}

type siteDataSourceModel struct {
	Id                     types.Int64    `tfsdk:"id"`
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

func siteDataSourceAttributes(idRequired bool) map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.Int64Attribute{
			Required:    idRequired,
			Computed:    !idRequired,
			Description: "Site identifier.",
		},
		"name":                      schema.StringAttribute{Computed: true, Description: "Site name."},
		"notes":                     schema.StringAttribute{Computed: true, Description: "Free-form notes about the site."},
		"location":                  locationDataSourceAttribute(),
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
	}
}

func locationDataSourceAttribute() schema.SingleNestedAttribute {
	return schema.SingleNestedAttribute{
		Computed: true,
		Attributes: map[string]schema.Attribute{
			"address_line1": schema.StringAttribute{Computed: true},
			"address_line2": schema.StringAttribute{Computed: true},
			"city":          schema.StringAttribute{Computed: true},
			"state":         schema.StringAttribute{Computed: true},
			"state_code":    schema.StringAttribute{Computed: true},
			"province_code": schema.StringAttribute{Computed: true},
			"country":       schema.StringAttribute{Computed: true},
			"country_code":  schema.StringAttribute{Computed: true},
			"latitude":      schema.Float64Attribute{Computed: true},
			"longitude":     schema.Float64Attribute{Computed: true},
			"notes":         schema.StringAttribute{Computed: true},
		},
	}
}

func flattenSiteDataSource(ctx context.Context, site *graphiant.ManaV2Site, m *siteDataSourceModel) diag.Diagnostics {
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

func (d *siteDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_site"
}

func (d *siteDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up a single Graphiant site by ID.",
		Attributes:  siteDataSourceAttributes(true),
	}
}

func (d *siteDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*gClient)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Data Source Configure Type", fmt.Sprintf("expected *gClient, got: %T", req.ProviderData))
		return
	}
	d.client = client
}

func (d *siteDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config siteDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "reading site data source", map[string]any{"id": config.Id.ValueInt64()})

	out, httpRes, err := d.client.api.DefaultAPI.V1SitesGet(ctx).Authorization(d.client.authHeader()).Execute()
	defer closeBody(httpRes)
	if err != nil {
		resp.Diagnostics.AddError("Error reading site", apiErrorDetail(err))
		return
	}

	wantId := config.Id.ValueInt64()
	var found *graphiant.ManaV2Site
	if out != nil {
		for _, s := range out.GetSites() {
			if s.GetId() == wantId {
				found = &s
				break
			}
		}
	}
	if found == nil {
		resp.Diagnostics.AddError("Site not found", fmt.Sprintf("no site with id %d was found", wantId))
		return
	}

	resp.Diagnostics.Append(flattenSiteDataSource(ctx, found, &config)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
