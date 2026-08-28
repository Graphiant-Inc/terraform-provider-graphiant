package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGatewayResource(t *testing.T) {
	lanSegmentName := acctest.RandomWithPrefix("tf-acc-gateway-lan")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckDisabled(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGatewayResourceConfig(lanSegmentName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("graphiant_gateway.test", "region_id"),
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

// region_id is looked up from the platform-wide region catalog (graphiant_regions)
// rather than hardcoded, since that catalog isn't tenant-created. vrf_id
// references a throwaway graphiant_lan_segment created in this same config —
// "VRF" and "LAN segment" are used interchangeably across this provider's
// schema descriptions for the same underlying id space.
func testAccGatewayResourceConfig(lanSegmentName string) string {
	return fmt.Sprintf(`
data "graphiant_regions" "all" {}

resource "graphiant_lan_segment" "test" {
  name = %[1]q
}

resource "graphiant_gateway" "test" {
  region_id = data.graphiant_regions.all.regions[0].id
  vrf_id    = graphiant_lan_segment.test.id
}
`, lanSegmentName)
}
