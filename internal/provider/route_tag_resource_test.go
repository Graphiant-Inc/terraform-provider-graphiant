package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRouteTagResource(t *testing.T) {
	levelZero := acctest.RandomWithPrefix("tf-acc-route-tag")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccRouteTagResourceConfig(levelZero),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graphiant_route_tag.test", "level_zero", levelZero),
					resource.TestCheckResourceAttrSet("graphiant_route_tag.test", "id"),
				),
			},
			{
				ResourceName:      "graphiant_route_tag.test",
				ImportState:       true,
				ImportStateVerify: true,
				// The read endpoint returns a recursive tag tree, not a flat level_zero/one/two
				// record, so these are preserved from config rather than refreshed — see the
				// resource's schema description. Import starts from a blank slate.
				ImportStateVerifyIgnore: []string{"level_zero", "level_one", "level_two"},
			},
		},
	})
}

func testAccRouteTagResourceConfig(levelZero string) string {
	return fmt.Sprintf(`
resource "graphiant_route_tag" "test" {
  level_zero = %[1]q
}
`, levelZero)
}
