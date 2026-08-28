package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGatewayResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGatewayResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graphiant_gateway.test", "region_id", "1"),
					resource.TestCheckResourceAttrSet("graphiant_gateway.test", "id"),
				),
			},
			{
				ResourceName:      "graphiant_gateway.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// region_id/vrf_id are placeholders that only resolve on a specific test tenant;
// adjust them (or source from graphiant_regions / a real segment) for your own.
func testAccGatewayResourceConfig() string {
	return `
resource "graphiant_gateway" "test" {
  region_id = 1
  vrf_id    = 100
}
`
}
