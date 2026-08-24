package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccAppListResource exercises the full graphiant_app_list lifecycle
// against a live Graphiant tenant: create, read back, import, and an
// in-place update. It scopes membership to a single Graphiant-catalog app
// (type "graphiant") rather than a custom app, since valid catalog app IDs
// are tenant-specific data (see graphiant_app_lists or the portal for real
// IDs). Run with:
//
//	TF_ACC=1 GRAPHIANT_ACCESS_TOKEN=... go test ./internal/provider/ -run TestAccAppListResource -v
func TestAccAppListResource(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-app-list")
	updatedName := name + "-updated"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and read back.
			{
				Config: testAccAppListResourceConfig(name, "created by terraform-provider-graphiant acceptance tests"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graphiant_app_list.test", "name", name),
					resource.TestCheckResourceAttr("graphiant_app_list.test", "description", "created by terraform-provider-graphiant acceptance tests"),
					resource.TestCheckResourceAttr("graphiant_app_list.test", "apps.#", "1"),
					resource.TestCheckResourceAttr("graphiant_app_list.test", "apps.0.type", "graphiant"),
					resource.TestCheckResourceAttrSet("graphiant_app_list.test", "id"),
					// graphiant_app_list data source, looked up by the resource's id.
					resource.TestCheckResourceAttrPair("data.graphiant_app_list.test", "id", "graphiant_app_list.test", "id"),
					resource.TestCheckResourceAttrPair("data.graphiant_app_list.test", "name", "graphiant_app_list.test", "name"),
					// graphiant_app_lists data source should list at least the app list just created.
					resource.TestCheckResourceAttrSet("data.graphiant_app_lists.test", "app_lists.#"),
				),
			},
			// ImportState.
			{
				ResourceName:      "graphiant_app_list.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update description in place (no replacement expected).
			{
				Config: testAccAppListResourceConfig(updatedName, "updated by terraform-provider-graphiant acceptance tests"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graphiant_app_list.test", "name", updatedName),
					resource.TestCheckResourceAttr("graphiant_app_list.test", "description", "updated by terraform-provider-graphiant acceptance tests"),
				),
			},
		},
	})
}

func testAccAppListResourceConfig(name, description string) string {
	return fmt.Sprintf(`
resource "graphiant_app_list" "test" {
  name        = %[1]q
  description = %[2]q

  apps = [
    { id = 1, type = "graphiant" },
  ]
}

data "graphiant_app_list" "test" {
  id = graphiant_app_list.test.id
}

data "graphiant_app_lists" "test" {
  depends_on = [graphiant_app_list.test]
}
`, name, description)
}
