package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccContentFilterResource exercises the full graphiant_content_filter
// lifecycle against a live Graphiant tenant: create, read back, import, and
// an in-place update. It scopes the filter with use_all_sites rather than a
// domain-category rule, since valid domain category IDs are tenant-specific
// catalog data (see graphiant_content_filters or the portal for real IDs to
// use in rules). Run with:
//
//	TF_ACC=1 GRAPHIANT_ACCESS_TOKEN=... go test ./internal/provider/ -run TestAccContentFilterResource -v
func TestAccContentFilterResource(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-content-filter")
	updatedName := name + "-updated"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and read back.
			{
				Config: testAccContentFilterResourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graphiant_content_filter.test", "name", name),
					resource.TestCheckResourceAttr("graphiant_content_filter.test", "use_all_sites", "true"),
					resource.TestCheckResourceAttrSet("graphiant_content_filter.test", "id"),
					resource.TestCheckResourceAttrSet("graphiant_content_filter.test", "created_at"),
					// graphiant_content_filter data source, looked up by the resource's id.
					resource.TestCheckResourceAttrPair("data.graphiant_content_filter.test", "id", "graphiant_content_filter.test", "id"),
					resource.TestCheckResourceAttrPair("data.graphiant_content_filter.test", "name", "graphiant_content_filter.test", "name"),
					// graphiant_content_filters data source should list at least the filter just created.
					resource.TestCheckResourceAttrSet("data.graphiant_content_filters.test", "content_filters.#"),
				),
			},
			// ImportState.
			{
				ResourceName:      "graphiant_content_filter.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update name in place (no replacement expected).
			{
				Config: testAccContentFilterResourceConfig(updatedName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graphiant_content_filter.test", "name", updatedName),
				),
			},
		},
	})
}

func testAccContentFilterResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "graphiant_content_filter" "test" {
  name          = %[1]q
  use_all_sites = true
}

data "graphiant_content_filter" "test" {
  id = graphiant_content_filter.test.id
}

data "graphiant_content_filters" "test" {
  depends_on = [graphiant_content_filter.test]
}
`, name)
}
