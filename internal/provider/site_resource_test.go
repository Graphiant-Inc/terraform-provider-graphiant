package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccSiteResource(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-site")
	nameUpdated := name + "-updated"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSiteResourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graphiant_site.test", "name", name),
					resource.TestCheckResourceAttr("graphiant_site.test", "location.city", "San Jose"),
					resource.TestCheckResourceAttrSet("graphiant_site.test", "id"),
				),
			},
			{
				ResourceName:      "graphiant_site.test",
				ImportState:       true,
				ImportStateVerify: true,
				// ManaV2Site (the read model) doesn't echo enterprise_id back, so it's
				// preserved from config/prior state rather than refreshed — see the resource's
				// applySite doc comment. Import starts from a blank slate, so this field
				// can't match what was configured — same situation as enterprise_contract in
				// enterprise_resource_test.go.
				ImportStateVerifyIgnore: []string{"enterprise_id"},
			},
			{
				Config: testAccSiteResourceConfig(nameUpdated),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graphiant_site.test", "name", nameUpdated),
				),
			},
		},
	})
}

// enterprise_id is a placeholder that only resolves on a specific test tenant;
// override via GRAPHIANT_ACC_ENTERPRISE_ID for your own — needed for an
// MSP-scoped token with access to more than one enterprise, where it's
// unverified whether omitting it resolves to the right one (same ambiguity
// noted in alert_integration_resource_test.go).
func testAccSiteResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "graphiant_site" "test" {
  name          = %[1]q
  notes         = "created by terraform acceptance tests"
  enterprise_id = %[2]s

  location {
    address_line1 = "2077 Gateway Pl"
    city          = "San Jose"
    state         = "California"
    country       = "United States"
  }
}
`, name, testAccEnvOrDefault("GRAPHIANT_ACC_ENTERPRISE_ID", "10000000325"))
}
