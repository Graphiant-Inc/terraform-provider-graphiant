package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

// testAccProtoV6ProviderFactories is passed to resource.TestCase.ProtoV6ProviderFactories
// for every acceptance test in this package.
var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"graphiant": providerserver.NewProtocol6WithError(New("acctest")()),
}

// testAccPreCheck is called by every acceptance test's PreCheck. It skips the
// test (rather than failing it) when no Graphiant credentials are
// configured, matching the convention documented in CONTRIBUTING.md and used
// by graphiant-sdk-go's own test suite: either GRAPHIANT_ACCESS_TOKEN, or
// both GRAPHIANT_USERNAME and GRAPHIANT_PASSWORD, must be set. GRAPHIANT_HOST
// is optional and defaults to the provider's built-in default (production).
func testAccPreCheck(t *testing.T) {
	t.Helper()

	hasToken := os.Getenv("GRAPHIANT_ACCESS_TOKEN") != ""
	hasUserPass := os.Getenv("GRAPHIANT_USERNAME") != "" && os.Getenv("GRAPHIANT_PASSWORD") != ""

	if !hasToken && !hasUserPass {
		t.Skip("acceptance test skipped: set GRAPHIANT_ACCESS_TOKEN, or both GRAPHIANT_USERNAME and GRAPHIANT_PASSWORD, to run tests against a live Graphiant tenant")
	}
}
