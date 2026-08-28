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

// testAccPreCheckHardcoded gates tests that reference a hardcoded id/serial for an
// object this provider has no way to create on demand (a physical device, a
// platform-fixed catalog entry with no write path, or the caller's own enterprise
// identity) — so, unlike every other acceptance test, they cannot be made
// self-contained with a random name. They therefore never run automatically in CI
// (see the acceptance job in .github/workflows/test.yml, which never sets this
// var) and only run locally once a maintainer has edited the placeholder id in the
// test file to point at a real object in their own test tenant and opted in here.
func testAccPreCheckHardcoded(t *testing.T) {
	t.Helper()
	testAccPreCheck(t)
	if os.Getenv("GRAPHIANT_ACC_HARDCODED_IDS") == "" {
		t.Skip("This test references a hardcoded id/serial for an object this provider can't create on demand " +
			"(see the comment at the top of this test file). It never runs in CI. To run it locally: edit the " +
			"placeholder to a real object in your own test tenant, then set GRAPHIANT_ACC_HARDCODED_IDS=1.")
	}
}

// testAccPreCheckDisabled gates tests that are temporarily disabled pending
// investigation (see CONTRIBUTING.md's "Acceptance tests" section). Like
// testAccPreCheckHardcoded, they never run automatically in CI (test.yml's
// acceptance job never sets GRAPHIANT_ACC_RUN_DISABLED) and only run locally
// once a maintainer has opted in via that env var.
func testAccPreCheckDisabled(t *testing.T) {
	t.Helper()
	testAccPreCheck(t)
	if os.Getenv("GRAPHIANT_ACC_RUN_DISABLED") == "" {
		t.Skip("This test is temporarily disabled (see CONTRIBUTING.md). It never runs in CI. " +
			"To run it locally, set GRAPHIANT_ACC_RUN_DISABLED=1.")
	}
}
