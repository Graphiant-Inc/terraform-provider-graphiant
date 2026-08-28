package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccSiteListResource(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-site-list")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSiteListResourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graphiant_site_list.test", "name", name),
					resource.TestCheckResourceAttrSet("graphiant_site_list.test", "id"),
				),
			},
			{
				ResourceName:      "graphiant_site_list.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccSiteListResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "graphiant_site_list" "test" {
  name        = %[1]q
  description = "created by terraform acceptance tests"
}
`, name)
}
