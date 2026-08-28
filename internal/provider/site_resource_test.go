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

func testAccSiteResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "graphiant_site" "test" {
  name  = %[1]q
  notes = "created by terraform acceptance tests"

  location {
    city         = "San Jose"
    state_code   = "CA"
    country_code = "US"
  }
}
`, name)
}
