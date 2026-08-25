package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccCustomAppResource exercises the full graphiant_custom_app lifecycle
// against a live Graphiant tenant: create, read back, import, and an
// in-place update. Run with:
//
//	TF_ACC=1 GRAPHIANT_ACCESS_TOKEN=... go test ./internal/provider/ -run TestAccCustomAppResource -v
func TestAccCustomAppResource(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-custom-app")
	updatedName := name + "-updated"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and read back.
			{
				Config: testAccCustomAppResourceConfig(name, "created by terraform-provider-graphiant acceptance tests"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graphiant_custom_app.test", "name", name),
					resource.TestCheckResourceAttr("graphiant_custom_app.test", "description", "created by terraform-provider-graphiant acceptance tests"),
					resource.TestCheckResourceAttr("graphiant_custom_app.test", "ip_protocol", "tcp"),
					resource.TestCheckResourceAttr("graphiant_custom_app.test", "port_ranges.#", "1"),
					resource.TestCheckResourceAttr("graphiant_custom_app.test", "port_ranges.0.lower", "8080"),
					resource.TestCheckResourceAttr("graphiant_custom_app.test", "port_ranges.0.upper", "8080"),
					resource.TestCheckResourceAttrSet("graphiant_custom_app.test", "id"),
					// graphiant_custom_app data source, looked up by the resource's id.
					resource.TestCheckResourceAttrPair("data.graphiant_custom_app.test", "id", "graphiant_custom_app.test", "id"),
					resource.TestCheckResourceAttrPair("data.graphiant_custom_app.test", "name", "graphiant_custom_app.test", "name"),
					// graphiant_custom_apps data source should list at least the app just created.
					resource.TestCheckResourceAttrSet("data.graphiant_custom_apps.test", "custom_apps.#"),
				),
			},
			// ImportState.
			{
				ResourceName:      "graphiant_custom_app.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update name/description in place (no replacement expected).
			{
				Config: testAccCustomAppResourceConfig(updatedName, "updated by terraform-provider-graphiant acceptance tests"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graphiant_custom_app.test", "name", updatedName),
					resource.TestCheckResourceAttr("graphiant_custom_app.test", "description", "updated by terraform-provider-graphiant acceptance tests"),
				),
			},
		},
	})
}

func testAccCustomAppResourceConfig(name, description string) string {
	return fmt.Sprintf(`
resource "graphiant_custom_app" "test" {
  name        = %[1]q
  description = %[2]q
  ip_protocol = "tcp"

  port_ranges = [
    { lower = 8080, upper = 8080 },
  ]
}

data "graphiant_custom_app" "test" {
  id = graphiant_custom_app.test.id
}

data "graphiant_custom_apps" "test" {
  depends_on = [graphiant_custom_app.test]
}
`, name, description)
}
