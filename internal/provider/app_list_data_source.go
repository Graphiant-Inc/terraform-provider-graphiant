package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	graphiant "github.com/Graphiant-Inc/graphiant-sdk-go"
	"github.com/Graphiant-Inc/terraform-provider-graphiant/internal/provider/generated/datasource_app_list"
)

var (
	_ datasource.DataSource              = &appListDataSource{}
	_ datasource.DataSourceWithConfigure = &appListDataSource{}
)

func NewAppListDataSource() datasource.DataSource {
	return &appListDataSource{}
}

type appListDataSource struct {
	client *gClient
}

type appListDataSourceModel struct {
	Id                   types.Int64          `tfsdk:"id"`
	Name                 types.String         `tfsdk:"name"`
	Description          types.String         `tfsdk:"description"`
	Apps                 []appIdentifierModel `tfsdk:"apps"`
	AppCount             types.Int64          `tfsdk:"app_count"`
	PolicyReferenceCount types.Int64          `tfsdk:"policy_reference_count"`
}

func (d *appListDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app_list"
}

// Schema is generated from the OpenAPI spec (see api/generator_config.yml,
// data_sources.app_list) via `make generate-schemas`; app_count/policy_reference_count
// are appended by hand for the same reason as the resource (see app_list_resource.go).
func (d *appListDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = datasource_app_list.AppListDataSourceSchema(ctx)
	resp.Schema.Description = "Looks up a single Graphiant global app list by ID."
	resp.Schema.Attributes["id"] = schema.Int64Attribute{Required: true, Description: "App list identifier."}
	resp.Schema.Attributes["app_count"] = schema.Int64Attribute{Computed: true, Description: "Number of apps in this app list."}
	resp.Schema.Attributes["policy_reference_count"] = schema.Int64Attribute{Computed: true, Description: "Number of policies referencing this app list."}
}

func (d *appListDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *appListDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config appListDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	wantId := config.Id.ValueInt64()
	tflog.Debug(ctx, "reading app list data source", map[string]any{"id": wantId})

	idOut, httpRes, err := d.client.api.DefaultAPI.V1GlobalAppsAppListsAppListIdGet(ctx, wantId).Authorization(d.client.authHeader()).Execute()
	closeBody(httpRes)
	if err != nil {
		resp.Diagnostics.AddError("Error reading app list", apiErrorDetail(err))
		return
	}
	if idOut == nil || idOut.AppListConfig == nil {
		resp.Diagnostics.AddError("App list not found", fmt.Sprintf("no app list with id %d was found", wantId))
		return
	}
	cfg := idOut.AppListConfig

	listOut, httpRes, err := d.client.api.DefaultAPI.V1GlobalAppsAppListsGet(ctx).Authorization(d.client.authHeader()).Execute()
	closeBody(httpRes)
	if err != nil {
		resp.Diagnostics.AddError("Error reading app list counters", apiErrorDetail(err))
		return
	}
	var entry *graphiant.V1GlobalAppsAppListsGetResponseEntry
	if listOut != nil {
		for _, e := range listOut.GetEntries() {
			if e.AppList != nil && e.AppList.Identifier != nil && e.AppList.Identifier.Id != nil && *e.AppList.Identifier.Id == wantId {
				entry = &e
				break
			}
		}
	}

	config.Id = int64Value(&wantId)
	config.Name = strValue(cfg.Name)
	config.Description = strValue(cfg.Description)
	config.Apps = flattenAppIdentifiers(cfg.Apps)
	if entry != nil {
		config.AppCount = int32Value(entry.AppCount)
		config.PolicyReferenceCount = int32Value(entry.PolicyReferenceCount)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
