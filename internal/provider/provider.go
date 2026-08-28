// Package provider implements the Graphiant Terraform provider directly on
// top of github.com/Graphiant-Inc/graphiant-sdk-go's generated client and
// models — there is no intermediate codegen step; each resource/data source
// maps Terraform schema fields to SDK request/response structs by hand.
package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	sdk "github.com/Graphiant-Inc/graphiant-sdk-go"
)

// Ensure GraphiantProvider satisfies the provider.Provider interface.
var _ provider.Provider = &GraphiantProvider{}

// GraphiantProvider is the Terraform provider implementation.
type GraphiantProvider struct {
	// version is set to the provider version at release build time (see main.go).
	version string
}

// graphiantProviderModel maps the provider configuration block.
type graphiantProviderModel struct {
	Host        types.String `tfsdk:"host"`
	AccessToken types.String `tfsdk:"access_token"`
	Username    types.String `tfsdk:"username"`
	Password    types.String `tfsdk:"password"`
}

func (p *GraphiantProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "graphiant"
	resp.Version = p.version
}

func (p *GraphiantProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Interact with the Graphiant network platform API.",
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				Optional: true,
				Description: "Graphiant API base URL. Defaults to https://api.graphiant.com. " +
					"Can also be set via the GRAPHIANT_API_HOST or GRAPHIANT_HOST environment variable.",
			},
			"access_token": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
				Description: "Bearer access token for the Graphiant API. Takes precedence over username/password. " +
					"Can also be set via the GRAPHIANT_ACCESS_TOKEN environment variable.",
			},
			"username": schema.StringAttribute{
				Optional: true,
				Description: "Graphiant username. Used with password to obtain an access token when access_token " +
					"is not set. Can also be set via the GRAPHIANT_USERNAME environment variable.",
			},
			"password": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
				Description: "Graphiant password. Used with username to obtain an access token when access_token " +
					"is not set. Can also be set via the GRAPHIANT_PASSWORD environment variable.",
			},
		},
	}
}

func (p *GraphiantProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config graphiantProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg := sdk.NewConfiguration()
	cfg.UserAgent = "terraform-provider-graphiant/" + p.version

	if host := config.Host.ValueString(); host != "" {
		applyHost(cfg, host)
	} else {
		sdk.ConfigureHostFromEnv(cfg)
	}

	client := sdk.NewAPIClient(cfg)

	var token string
	switch {
	case config.AccessToken.ValueString() != "":
		token = normalizeBearer(config.AccessToken.ValueString())
	case config.Username.ValueString() != "" && config.Password.ValueString() != "":
		var err error
		token, err = loginBearer(ctx, client, config.Username.ValueString(), config.Password.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Unable to authenticate with Graphiant", err.Error())
			return
		}
	default:
		var err error
		token, err = sdk.AuthorizationBearerFromEnvOrLogin(ctx, client)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to resolve Graphiant credentials",
				"Set access_token or username/password in the provider configuration block, or set "+
					"GRAPHIANT_ACCESS_TOKEN (or GRAPHIANT_USERNAME + GRAPHIANT_PASSWORD) in the environment: "+err.Error(),
			)
			return
		}
	}

	pd := &providerData{api: client, token: token}
	resp.ResourceData = pd
	resp.DataSourceData = pd
}

func (p *GraphiantProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewSiteResource,
		NewUserResource,
		NewGroupResource,
		NewAppListResource,
		NewContentFilterResource,
		NewCustomAppResource,
		NewSiteListResource,
		NewEnterpriseResource,
		NewGatewayResource,
		NewPublicVifResource,
		NewExtranetResource,
		NewSoftwareRolloutResource,
		NewDeviceBringupResource,
		NewDeviceDecommissionResource,
		NewB2bProducerServiceResource,
		NewB2bCustomerResource,
		NewB2bMatchResource,
		NewB2bConsumerResource,
		NewAssuranceGlobalResource,
		NewAssuranceClassifiedApplicationResource,
		NewAlertIntegrationResource,
		NewAlertNotificationResource,
		NewRouteTagResource,
		NewDeviceConfigResource,
		NewLanSegmentResource,
	}
}

func (p *GraphiantProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewDeviceDataSource,
		NewEdgesDataSource,
		NewSiteDevicesDataSource,
		NewAlertRecordsDataSource,
		NewAlertRulesDataSource,
		NewAssuranceFlexAlgosDataSource,
		NewAssuranceDnsproxyEntriesDataSource,
		NewTroubleshootingDeviceDataSource,
		NewTroubleshootingSiteDataSource,
		NewRoutingPolicyDataSource,
		NewPrefixSetDataSource,
		NewDomainCategoriesDataSource,
		NewRegionsDataSource,
		NewIpsecProfilesDataSource,
	}
}

// New returns the provider factory main.go hands to providerserver.Serve.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &GraphiantProvider{version: version}
	}
}
