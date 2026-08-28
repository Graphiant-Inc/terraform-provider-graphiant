package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	sdk "github.com/Graphiant-Inc/graphiant-sdk-go"
)

var (
	_ datasource.DataSource              = &edgesDataSource{}
	_ datasource.DataSourceWithConfigure = &edgesDataSource{}
)

func NewEdgesDataSource() datasource.DataSource {
	return &edgesDataSource{}
}

type edgesDataSource struct {
	pd *providerData
}

var edgeSummaryAttrTypes = map[string]attrType{
	"device_id":       types.Int64Type,
	"hostname":        types.StringType,
	"status":          types.StringType,
	"portal_status":   types.StringType,
	"role":            types.StringType,
	"site_id":         types.Int64Type,
	"site":            types.StringType,
	"region":          types.StringType,
	"serial_num":      types.StringType,
	"model":           types.StringType,
	"sw_version":      types.StringType,
	"is_new":          types.BoolType,
	"is_hardware":     types.BoolType,
	"stale":           types.BoolType,
	"tt_conn_count":   types.Int64Type,
	"enterprise_id":   types.Int64Type,
	"enterprise_name": types.StringType,
}

type edgeSummaryModel struct {
	DeviceID       types.Int64  `tfsdk:"device_id"`
	Hostname       types.String `tfsdk:"hostname"`
	Status         types.String `tfsdk:"status"`
	PortalStatus   types.String `tfsdk:"portal_status"`
	Role           types.String `tfsdk:"role"`
	SiteID         types.Int64  `tfsdk:"site_id"`
	Site           types.String `tfsdk:"site"`
	Region         types.String `tfsdk:"region"`
	SerialNum      types.String `tfsdk:"serial_num"`
	Model          types.String `tfsdk:"model"`
	SwVersion      types.String `tfsdk:"sw_version"`
	IsNew          types.Bool   `tfsdk:"is_new"`
	IsHardware     types.Bool   `tfsdk:"is_hardware"`
	Stale          types.Bool   `tfsdk:"stale"`
	TtConnCount    types.Int64  `tfsdk:"tt_conn_count"`
	EnterpriseID   types.Int64  `tfsdk:"enterprise_id"`
	EnterpriseName types.String `tfsdk:"enterprise_name"`
}

type edgesDataSourceModel struct {
	EnterpriseID   types.Int64 `tfsdk:"enterprise_id"`
	IsRequested    types.Bool  `tfsdk:"is_requested"`
	UpgradeSummary types.Bool  `tfsdk:"upgrade_summary"`
	DeviceIDs      types.List  `tfsdk:"device_ids"`
	Roles          types.List  `tfsdk:"roles"`
	Statuses       types.List  `tfsdk:"statuses"`
	Edges          types.List  `tfsdk:"edges"`
}

func (d *edgesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_edges"
}

func (d *edgesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Current edge device summary list (onboarding/status snapshot). If device_ids/roles/statuses " +
			"are set, uses the filtered POST variant; otherwise uses the plain GET with enterprise_id/is_requested/" +
			"upgrade_summary. Location and upgrade-summary detail sub-objects are not yet exposed.",
		Attributes: map[string]schema.Attribute{
			"enterprise_id":   schema.Int64Attribute{Optional: true},
			"is_requested":    schema.BoolAttribute{Optional: true},
			"upgrade_summary": schema.BoolAttribute{Optional: true},
			"device_ids":      schema.ListAttribute{Optional: true, ElementType: types.Int64Type},
			"roles":           schema.ListAttribute{Optional: true, ElementType: types.StringType},
			"statuses":        schema.ListAttribute{Optional: true, ElementType: types.StringType},
			"edges": schema.ListAttribute{
				Computed:    true,
				ElementType: types.ObjectType{AttrTypes: edgeSummaryAttrTypes},
			},
		},
	}
}

func (d *edgesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.pd = configurePD(req.ProviderData, &resp.Diagnostics)
}

func edgeSummariesToList(ctx context.Context, in []sdk.SearchEdgeSummary) (types.List, diag.Diagnostics) {
	models := make([]edgeSummaryModel, 0, len(in))
	for _, e := range in {
		models = append(models, edgeSummaryModel{
			DeviceID:       types.Int64PointerValue(e.DeviceId),
			Hostname:       types.StringPointerValue(e.Hostname),
			Status:         types.StringPointerValue(e.Status),
			PortalStatus:   types.StringPointerValue(e.PortalStatus),
			Role:           types.StringPointerValue(e.Role),
			SiteID:         types.Int64PointerValue(e.SiteId),
			Site:           types.StringPointerValue(e.Site),
			Region:         types.StringPointerValue(e.Region),
			SerialNum:      types.StringPointerValue(e.SerialNum),
			Model:          types.StringPointerValue(e.Model),
			SwVersion:      types.StringPointerValue(e.SwVersion),
			IsNew:          types.BoolPointerValue(e.IsNew),
			IsHardware:     types.BoolPointerValue(e.IsHardware),
			Stale:          types.BoolPointerValue(e.Stale),
			TtConnCount:    types.Int64PointerValue(intPtr32To64(e.TtConnCount)),
			EnterpriseID:   types.Int64PointerValue(e.EnterpriseId),
			EnterpriseName: types.StringPointerValue(e.EnterpriseName),
		})
	}
	return types.ListValueFrom(ctx, types.ObjectType{AttrTypes: edgeSummaryAttrTypes}, models)
}

func (d *edgesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg edgesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	hasFilter := !cfg.DeviceIDs.IsNull() && !cfg.DeviceIDs.IsUnknown() ||
		!cfg.Roles.IsNull() && !cfg.Roles.IsUnknown() ||
		!cfg.Statuses.IsNull() && !cfg.Statuses.IsUnknown()

	var edges []sdk.SearchEdgeSummary
	if hasFilter {
		filter := &sdk.V1EdgesSummaryPostRequestFilter{}
		if !cfg.DeviceIDs.IsNull() && !cfg.DeviceIDs.IsUnknown() {
			resp.Diagnostics.Append(cfg.DeviceIDs.ElementsAs(ctx, &filter.DeviceIds, false)...)
		}
		if !cfg.Roles.IsNull() && !cfg.Roles.IsUnknown() {
			resp.Diagnostics.Append(cfg.Roles.ElementsAs(ctx, &filter.Roles, false)...)
		}
		if !cfg.Statuses.IsNull() && !cfg.Statuses.IsUnknown() {
			resp.Diagnostics.Append(cfg.Statuses.ElementsAs(ctx, &filter.Statuses, false)...)
		}
		if resp.Diagnostics.HasError() {
			return
		}

		out, httpResp, err := d.pd.api.DefaultAPI.V1EdgesSummaryPost(ctx).
			Authorization(d.pd.token).
			V1EdgesSummaryPostRequest(sdk.V1EdgesSummaryPostRequest{Filter: filter}).
			Execute()
		closeBody(httpResp)
		if err != nil {
			resp.Diagnostics.AddError("Unable to read edge summaries", apiErrorDetail(err))
			return
		}
		if out != nil {
			edges = out.EdgesSummary
		}
	} else {
		call := d.pd.api.DefaultAPI.V1EdgesSummaryGet(ctx).Authorization(d.pd.token)
		if v := cfg.EnterpriseID.ValueInt64Pointer(); v != nil {
			call = call.EnterpriseId(*v)
		}
		if v := cfg.IsRequested.ValueBoolPointer(); v != nil {
			call = call.IsRequested(*v)
		}
		if v := cfg.UpgradeSummary.ValueBoolPointer(); v != nil {
			call = call.UpgradeSummary(*v)
		}
		out, httpResp, err := call.Execute()
		closeBody(httpResp)
		if err != nil {
			resp.Diagnostics.AddError("Unable to read edge summaries", apiErrorDetail(err))
			return
		}
		if out != nil {
			edges = out.EdgesSummary
		}
	}

	list, diags := edgeSummariesToList(ctx, edges)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg.Edges = list

	resp.Diagnostics.Append(resp.State.Set(ctx, &cfg)...)
}
