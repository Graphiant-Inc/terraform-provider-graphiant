package provider

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	graphiant "github.com/Graphiant-Inc/graphiant-sdk-go"
)

// gClient wraps the generated SDK client together with the bearer token used
// for every request. The generated SDK requires callers to pass an
// Authorization header on each call individually, so resources and data
// sources call gClient.authHeader() rather than holding the token themselves.
type gClient struct {
	api   *graphiant.APIClient
	token string
}

func (c *gClient) authHeader() string {
	return c.token
}

// providerConfig mirrors the provider schema and is filled in during Configure.
type providerConfig struct {
	Host         string
	AccessToken  string
	Username     string
	Password     string
	InsecureSkip bool
}

// newClient builds a graphiant-sdk-go client from the resolved provider
// configuration and resolves a bearer token, preferring a static access
// token and falling back to a username/password login (matching the
// behavior of graphiant-sdk-go's AuthorizationBearerFromEnvOrLogin).
func newClient(ctx context.Context, cfg providerConfig) (*gClient, error) {
	sdkCfg := graphiant.NewConfiguration()

	if cfg.Host != "" {
		host := strings.TrimSpace(cfg.Host)
		if !strings.Contains(host, "://") {
			host = "https://" + host
		}
		u, err := url.Parse(host)
		if err != nil || u.Host == "" {
			return nil, fmt.Errorf("invalid host %q: %w", cfg.Host, err)
		}
		sdkCfg.Scheme = u.Scheme
		sdkCfg.Host = u.Host
	}

	if cfg.InsecureSkip {
		sdkCfg.HTTPClient = insecureHTTPClient()
	}

	api := graphiant.NewAPIClient(sdkCfg)

	token := strings.TrimSpace(cfg.AccessToken)
	if token != "" {
		if !strings.HasPrefix(strings.ToLower(token), "bearer ") {
			token = "Bearer " + token
		}
		return &gClient{api: api, token: token}, nil
	}

	if cfg.Username == "" || cfg.Password == "" {
		return nil, fmt.Errorf("either access_token, or both username and password, must be set")
	}

	tflog.Debug(ctx, "authenticating with username/password", map[string]any{"username": cfg.Username})

	authReq := graphiant.NewV1AuthLoginPostRequestWithDefaults()
	authReq.SetUsername(cfg.Username)
	authReq.SetPassword(cfg.Password)

	resp, httpRes, err := api.DefaultAPI.V1AuthLoginPost(ctx).V1AuthLoginPostRequest(*authReq).Execute()
	defer closeBody(httpRes)
	if err != nil {
		return nil, fmt.Errorf("login failed: %s", apiErrorDetail(err))
	}
	if resp == nil || !resp.GetAuth() || resp.GetToken() == "" {
		return nil, fmt.Errorf("login failed: no token returned")
	}

	return &gClient{api: api, token: "Bearer " + resp.GetToken()}, nil
}
