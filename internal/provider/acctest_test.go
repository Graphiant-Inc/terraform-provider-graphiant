package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// testAccProtoV6ProviderFactories instantiates the provider for acceptance
// testing. resource.Test calls this factory for every Terraform CLI command
// (plan, apply, destroy, etc.) it runs during a test.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"graphiant": providerserver.NewProtocol6WithError(New("acctest")()),
}

// testAccPreCheck skips acceptance tests when no credentials are configured,
// so a fork's pull_request run (which never receives repository secrets)
// reports a clear skip instead of an authentication failure. resource.Test
// itself already gates on TF_ACC=1 separately.
func testAccPreCheck(t *testing.T) {
	t.Helper()
	if os.Getenv("GRAPHIANT_ACCESS_TOKEN") != "" {
		return
	}
	if os.Getenv("GRAPHIANT_USERNAME") != "" && os.Getenv("GRAPHIANT_PASSWORD") != "" {
		return
	}
	t.Skip("Acceptance tests require GRAPHIANT_ACCESS_TOKEN, or GRAPHIANT_USERNAME + GRAPHIANT_PASSWORD, to be set")
}
