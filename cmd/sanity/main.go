// Command sanity is a Terraform-independent smoke test: it resolves Graphiant
// credentials the same way the provider does (see internal/provider/provider.go's
// Configure), logs in (or reuses a static access token), and prints the current
// edge summary list. Useful for confirming GRAPHIANT_* env vars and connectivity
// work before reaching for `terraform plan`/`apply` against a local provider build.
//
// Usage:
//
//	export GRAPHIANT_ACCESS_TOKEN="..."   # or GRAPHIANT_USERNAME + GRAPHIANT_PASSWORD
//	export GRAPHIANT_API_HOST="https://api.graphiant.com"   # optional, same default the SDK uses
//	go run ./cmd/sanity
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"text/tabwriter"

	sdk "github.com/Graphiant-Inc/graphiant-sdk-go"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "sanity check failed:", err)
		os.Exit(1)
	}
}

func run() error {
	ctx := context.Background()

	cfg := sdk.NewConfiguration()
	cfg.UserAgent = "terraform-provider-graphiant-sanity/dev"
	sdk.ConfigureHostFromEnv(cfg)
	client := sdk.NewAPIClient(cfg)

	host := os.Getenv(sdk.EnvAPIHost)
	if host == "" {
		host = os.Getenv(sdk.EnvHost)
	}
	if host == "" {
		host = "https://api.graphiant.com (default)"
	}
	fmt.Println("Host:", host)

	token, err := sdk.AuthorizationBearerFromEnvOrLogin(ctx, client)
	if err != nil {
		return fmt.Errorf("authenticating (set %s, or %s + %s): %w",
			sdk.EnvAccessToken, sdk.EnvUsername, sdk.EnvPassword, err)
	}
	fmt.Println("Login OK.")

	out, httpResp, err := client.DefaultAPI.V1EdgesSummaryGet(ctx).Authorization(token).Execute()
	if httpResp != nil {
		defer func() { _ = httpResp.Body.Close() }()
	}
	if err != nil {
		return fmt.Errorf("listing edge summary: %w", err)
	}
	if out == nil {
		return errors.New("listing edge summary: empty response")
	}

	edges := out.EdgesSummary
	fmt.Printf("\n%d edge(s):\n\n", len(edges))

	w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	if _, err := fmt.Fprintln(w, "DEVICE ID\tHOSTNAME\tSTATUS\tROLE\tSITE"); err != nil {
		return err
	}
	for _, e := range edges {
		if _, err := fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%v\n",
			deref(e.DeviceId), deref(e.Hostname), deref(e.Status), deref(e.Role), deref(e.Site)); err != nil {
			return err
		}
	}
	return w.Flush()
}

// deref renders a possibly-nil pointer field as its value or "-" when unset,
// for any of the string/int64 pointer types graphiant-sdk-go generates.
func deref[T any](p *T) any {
	if p == nil {
		return "-"
	}
	return *p
}
