package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	graphiant "github.com/Graphiant-Inc/graphiant-sdk-go"
)

var (
	_ datasource.DataSource              = &groupDataSource{}
	_ datasource.DataSourceWithConfigure = &groupDataSource{}
)

func NewGroupDataSource() datasource.DataSource {
	return &groupDataSource{}
}

type groupDataSource struct {
	client *gClient
}

type groupDataSourceModel struct {
	Id                 types.String      `tfsdk:"id"`
	Name               types.String      `tfsdk:"name"`
	Description        types.String      `tfsdk:"description"`
	GroupType          types.String      `tfsdk:"group_type"`
	ManagesEnterprises types.Bool        `tfsdk:"manages_enterprises"`
	TimeWindowStart    types.Int64       `tfsdk:"time_window_start"`
	TimeWindowEnd      types.Int64       `tfsdk:"time_window_end"`
	Permissions        *permissionsModel `tfsdk:"permissions"`
	EnterpriseIds      types.List        `tfsdk:"enterprise_ids"`
}

func groupDataSourceAttributes(idRequired bool) map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Required:    idRequired,
			Computed:    !idRequired,
			Description: "Group identifier.",
		},
		"name":                schema.StringAttribute{Computed: true, Description: "Group name."},
		"description":         schema.StringAttribute{Computed: true, Description: "Group description."},
		"group_type":          schema.StringAttribute{Computed: true, Description: "Group type."},
		"manages_enterprises": schema.BoolAttribute{Computed: true, Description: "Whether members of this group can manage sub-enterprises."},
		"time_window_start":   schema.Int64Attribute{Computed: true, Description: "Unix timestamp for the start of the access time window, if the group is time-restricted."},
		"time_window_end":     schema.Int64Attribute{Computed: true, Description: "Unix timestamp for the end of the access time window, if the group is time-restricted."},
		"permissions":         permissionsDataSourceAttribute(),
		"enterprise_ids": schema.ListAttribute{
			Computed:    true,
			ElementType: types.Int64Type,
			Description: "Enterprises this group has access to.",
		},
	}
}

func permissionsDataSourceAttribute() schema.SingleNestedAttribute {
	attrs := make(map[string]schema.Attribute, len(permissionsFields))
	for _, name := range permissionsFields {
		attrs[name] = schema.StringAttribute{
			Computed:    true,
			Description: "Access level for this permission area (e.g. \"none\", \"read\", \"write\").",
		}
	}
	return schema.SingleNestedAttribute{
		Computed:    true,
		Attributes:  attrs,
		Description: "Per-area role permissions.",
	}
}

func flattenGroupDataSource(g *graphiant.IamGroup, m *groupDataSourceModel, ctx context.Context) diag.Diagnostics {
	var diags diag.Diagnostics
	m.Id = strValue(g.Id)
	m.Name = strValue(g.Name)
	m.Description = strValue(g.Description)
	m.GroupType = strValue(g.GroupType)
	m.TimeWindowStart = int64Value(g.TimeWindowStart)
	m.TimeWindowEnd = int64Value(g.TimeWindowEnd)
	m.Permissions = flattenPermissions(g.Permissions)

	ids := g.EnterpriseIds
	if ids == nil {
		ids = []int64{}
	}
	list, d := types.ListValueFrom(ctx, types.Int64Type, ids)
	diags.Append(d...)
	m.EnterpriseIds = list
	return diags
}

func (d *groupDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group"
}

func (d *groupDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Looks up a single Graphiant IAM group by ID.",
		Attributes:  groupDataSourceAttributes(true),
	}
}

func (d *groupDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *groupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config groupDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, httpRes, err := d.client.api.DefaultAPI.V1GroupsGet(ctx).Authorization(d.client.authHeader()).Execute()
	if httpRes != nil {
		defer httpRes.Body.Close()
	}
	if err != nil {
		resp.Diagnostics.AddError("Error reading group", err.Error())
		return
	}

	wantId := config.Id.ValueString()
	var found *graphiant.IamGroup
	if out != nil {
		for _, g := range out.GetGroups() {
			if g.GetId() == wantId {
				found = &g
				break
			}
		}
	}
	if found == nil {
		resp.Diagnostics.AddError("Group not found", fmt.Sprintf("no group with id %q was found", wantId))
		return
	}

	resp.Diagnostics.Append(flattenGroupDataSource(found, &config, ctx)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
