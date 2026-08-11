package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccSiteResource exercises the full graphiant_site lifecycle against a
// live Graphiant tenant: create, read back computed attributes, update a
// mutable field, import, and (via Terraform's built-in destroy step) delete.
// It also exercises the graphiant_site and graphiant_sites data sources
// against the same resource. Run with:
//
//	TF_ACC=1 GRAPHIANT_ACCESS_TOKEN=... go test ./internal/provider/ -run TestAccSiteResource -v
func TestAccSiteResource(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-site")
	updatedName := name + "-updated"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and read back.
			{
				Config: testAccSiteResourceConfig(name, "created by terraform-provider-graphiant acceptance tests"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graphiant_site.test", "name", name),
					resource.TestCheckResourceAttr("graphiant_site.test", "notes", "created by terraform-provider-graphiant acceptance tests"),
					resource.TestCheckResourceAttr("graphiant_site.test", "location.city", "San Jose"),
					resource.TestCheckResourceAttr("graphiant_site.test", "location.country_code", "US"),
					resource.TestCheckResourceAttrSet("graphiant_site.test", "id"),
					resource.TestCheckResourceAttrSet("graphiant_site.test", "created_at"),
					// graphiant_site data source, looked up by the resource's id.
					resource.TestCheckResourceAttrPair("data.graphiant_site.test", "id", "graphiant_site.test", "id"),
					resource.TestCheckResourceAttrPair("data.graphiant_site.test", "name", "graphiant_site.test", "name"),
					// graphiant_sites data source should list at least the site just created.
					resource.TestCheckResourceAttrSet("data.graphiant_sites.test", "sites.#"),
				),
			},
			// ImportState.
			{
				ResourceName:      "graphiant_site.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update name and notes in place (no replacement expected).
			{
				Config: testAccSiteResourceConfig(updatedName, "updated by terraform-provider-graphiant acceptance tests"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graphiant_site.test", "name", updatedName),
					resource.TestCheckResourceAttr("graphiant_site.test", "notes", "updated by terraform-provider-graphiant acceptance tests"),
				),
			},
		},
	})
}

func testAccSiteResourceConfig(name, notes string) string {
	return fmt.Sprintf(`
resource "graphiant_site" "test" {
  name  = %[1]q
  notes = %[2]q

  location = {
    address_line1 = "1 Graphiant Way"
    city          = "San Jose"
    state_code    = "CA"
    country_code  = "US"
  }
}

data "graphiant_site" "test" {
  id = graphiant_site.test.id
}

data "graphiant_sites" "test" {
  depends_on = [graphiant_site.test]
}
`, name, notes)
}
