package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccLanSegmentResource(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-lan-segment")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccLanSegmentResourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graphiant_lan_segment.test", "name", name),
					resource.TestCheckResourceAttrSet("graphiant_lan_segment.test", "id"),
				),
			},
			{
				ResourceName:      "graphiant_lan_segment.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccLanSegmentResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "graphiant_lan_segment" "test" {
  name        = %[1]q
  description = "created by terraform acceptance tests"
}
`, name)
}
