package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccContentFilterResource(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-content-filter")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccContentFilterResourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graphiant_content_filter.test", "name", name),
					resource.TestCheckResourceAttr("graphiant_content_filter.test", "use_all_sites", "true"),
					resource.TestCheckResourceAttrSet("graphiant_content_filter.test", "id"),
				),
			},
			{
				ResourceName:      "graphiant_content_filter.test",
				ImportState:       true,
				ImportStateVerify: true,
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
`, name)
}
