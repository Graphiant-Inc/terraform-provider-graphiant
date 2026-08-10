package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure GraphiantProvider satisfies the expected interfaces.
var _ provider.Provider = &GraphiantProvider{}

// GraphiantProvider is the Terraform provider implementation for Graphiant NaaS,
// backed by github.com/Graphiant-Inc/graphiant-sdk-go.
type GraphiantProvider struct {
	// version is set to the provider version at release build time.
	version string
}

// graphiantProviderModel maps the provider configuration block.
type graphiantProviderModel struct {
	Host               types.String `tfsdk:"host"`
	AccessToken        types.String `tfsdk:"access_token"`
	Username           types.String `tfsdk:"username"`
	Password           types.String `tfsdk:"password"`
	InsecureSkipVerify types.Bool   `tfsdk:"insecure_skip_verify"`
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &GraphiantProvider{version: version}
	}
}

func (p *GraphiantProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "graphiant"
	resp.Version = p.version
}

func (p *GraphiantProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The Graphiant provider manages Graphiant Network-as-a-Service (NaaS) resources: sites, IAM users and groups, and read-only access to onboarded devices. It is backed by the graphiant-sdk-go client.",
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				Optional:    true,
				Description: "Graphiant API base URL. Defaults to https://api.graphiant.com. Can also be set via the GRAPHIANT_API_HOST or GRAPHIANT_HOST environment variables.",
			},
			"access_token": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Bearer token issued by the Graphiant portal/CLI. Can also be set via the GRAPHIANT_ACCESS_TOKEN environment variable. Takes precedence over username/password.",
			},
			"username": schema.StringAttribute{
				Optional:    true,
				Description: "Graphiant username, used to log in when access_token is not set. Can also be set via the GRAPHIANT_USERNAME environment variable.",
			},
			"password": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Graphiant password, used to log in when access_token is not set. Can also be set via the GRAPHIANT_PASSWORD environment variable.",
			},
			"insecure_skip_verify": schema.BoolAttribute{
				Optional:    true,
				Description: "Skip TLS certificate verification when calling the Graphiant API. Only use this against a trusted on-prem/lab controller.",
			},
		},
	}
}

func (p *GraphiantProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data graphiantProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg := providerConfig{
		Host:         firstNonEmpty(data.Host.ValueString(), os.Getenv("GRAPHIANT_API_HOST"), os.Getenv("GRAPHIANT_HOST")),
		AccessToken:  firstNonEmpty(data.AccessToken.ValueString(), os.Getenv("GRAPHIANT_ACCESS_TOKEN")),
		Username:     firstNonEmpty(data.Username.ValueString(), os.Getenv("GRAPHIANT_USERNAME")),
		Password:     firstNonEmpty(data.Password.ValueString(), os.Getenv("GRAPHIANT_PASSWORD")),
		InsecureSkip: data.InsecureSkipVerify.ValueBool(),
	}

	client, err := newClient(ctx, cfg)
	if err != nil {
		resp.Diagnostics.AddError("Unable to create Graphiant API client", err.Error())
		return
	}

	resp.DataSourceData = client
	resp.ResourceData = client
}

func (p *GraphiantProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewSiteResource,
		NewGroupResource,
		NewUserResource,
	}
}

func (p *GraphiantProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewSiteDataSource,
		NewSitesDataSource,
		NewGroupDataSource,
		NewGroupsDataSource,
		NewUserDataSource,
		NewUsersDataSource,
		NewDeviceDataSource,
		NewDevicesDataSource,
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
