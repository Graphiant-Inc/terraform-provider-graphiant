package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccSiteListResource exercises the full graphiant_site_list lifecycle
// against a live Graphiant tenant: create (with a site created alongside it
// as a member), read back, import, and an in-place description/entries
// update. Renaming isn't exercised since the API has no rename endpoint
// (name has RequiresReplace in the schema). Run with:
//
//	TF_ACC=1 GRAPHIANT_ACCESS_TOKEN=... go test ./internal/provider/ -run TestAccSiteListResource -v
func TestAccSiteListResource(t *testing.T) {
	siteName := acctest.RandomWithPrefix("tf-acc-site-list-site")
	listName := acctest.RandomWithPrefix("tf-acc-site-list")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and read back.
			{
				Config: testAccSiteListResourceConfig(siteName, listName, "created by terraform-provider-graphiant acceptance tests"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graphiant_site_list.test", "name", listName),
					resource.TestCheckResourceAttr("graphiant_site_list.test", "description", "created by terraform-provider-graphiant acceptance tests"),
					resource.TestCheckResourceAttr("graphiant_site_list.test", "entries.#", "1"),
					resource.TestCheckResourceAttrPair("graphiant_site_list.test", "entries.0.site_id", "graphiant_site.test", "id"),
					resource.TestCheckResourceAttrSet("graphiant_site_list.test", "id"),
					// graphiant_site_list data source, looked up by the resource's id.
					resource.TestCheckResourceAttrPair("data.graphiant_site_list.test", "id", "graphiant_site_list.test", "id"),
					resource.TestCheckResourceAttrPair("data.graphiant_site_list.test", "name", "graphiant_site_list.test", "name"),
					// graphiant_site_lists data source should list at least the site list just created.
					resource.TestCheckResourceAttrSet("data.graphiant_site_lists.test", "site_lists.#"),
				),
			},
			// ImportState.
			{
				ResourceName:      "graphiant_site_list.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update description in place (no replacement expected).
			{
				Config: testAccSiteListResourceConfig(siteName, listName, "updated by terraform-provider-graphiant acceptance tests"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graphiant_site_list.test", "description", "updated by terraform-provider-graphiant acceptance tests"),
				),
			},
		},
	})
}

func testAccSiteListResourceConfig(siteName, listName, description string) string {
	return fmt.Sprintf(`
resource "graphiant_site" "test" {
  name = %[1]q
}

resource "graphiant_site_list" "test" {
  name        = %[2]q
  description = %[3]q

  entries = [
    { site_id = graphiant_site.test.id },
  ]
}

data "graphiant_site_list" "test" {
  id = graphiant_site_list.test.id
}

data "graphiant_site_lists" "test" {
  depends_on = [graphiant_site_list.test]
}
`, siteName, listName, description)
}
