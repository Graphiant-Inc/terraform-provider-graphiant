package provider

import (
	"context"
	"fmt"
	"strings"

	sdk "github.com/Graphiant-Inc/graphiant-sdk-go"
)

// providerData is handed to every resource/data source via req.ProviderData.
// Authorization on this SDK is per-request, not per-client, so callers chain
// .Authorization(pd.token) onto every generated request builder themselves.
type providerData struct {
	api   *sdk.APIClient
	token string
}

// normalizeBearer ensures token is in the "Bearer <value>" form the API expects,
// tolerating a caller who already included the "Bearer " prefix.
func normalizeBearer(token string) string {
	if len(token) >= 7 && strings.EqualFold(token[:7], "bearer ") {
		return token
	}
	return "Bearer " + token
}

// applyHost points cfg at the given API host, defaulting to https:// when the
// caller didn't include a scheme. graphiant_sdk.Configuration.Host/Scheme are
// unused by the generated client; the real base URL lives in cfg.Servers.
func applyHost(cfg *sdk.Configuration, raw string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	cfg.Servers = sdk.ServerConfigurations{{URL: raw}}
}

// loginBearer exchanges a username/password for a bearer token via POST /v1/auth/login,
// mirroring graphiant_sdk.LoginBearerFromEnvCredentials but with explicit credentials
// (that helper only reads GRAPHIANT_USERNAME/GRAPHIANT_PASSWORD from the environment).
func loginBearer(ctx context.Context, client *sdk.APIClient, username, password string) (string, error) {
	authReq := sdk.NewV1AuthLoginPostRequestWithDefaults()
	authReq.SetUsername(username)
	authReq.SetPassword(password)

	resp, httpResp, err := client.DefaultAPI.V1AuthLoginPost(ctx).V1AuthLoginPostRequest(*authReq).Execute()
	if httpResp != nil {
		defer func() { _ = httpResp.Body.Close() }()
	}
	if err != nil {
		return "", fmt.Errorf("login request failed: %s", apiErrorDetail(err))
	}
	if resp == nil {
		return "", fmt.Errorf("login response body missing")
	}
	if !resp.GetAuth() {
		return "", fmt.Errorf("login failed (auth=false)")
	}
	token := resp.GetToken()
	if token == "" {
		return "", fmt.Errorf("login response missing token")
	}
	return "Bearer " + token, nil
}
