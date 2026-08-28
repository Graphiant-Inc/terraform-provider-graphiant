package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// lan_segment_id/region_id are placeholders that only resolve on a specific
// test tenant; adjust for your own, or source via graphiant_regions data source.
func TestAccPublicVifResource(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-pvif")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPublicVifResourceConfig(name, "aws"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graphiant_public_vif.test", "service_name", name),
					resource.TestCheckResourceAttr("graphiant_public_vif.test", "storage_provider", "aws"),
					resource.TestCheckResourceAttrSet("graphiant_public_vif.test", "id"),
				),
			},
			{
				ResourceName:      "graphiant_public_vif.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccPublicVifResourceConfig(name, "azure"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graphiant_public_vif.test", "storage_provider", "azure"),
				),
			},
		},
	})
}

func testAccPublicVifResourceConfig(name, storageProvider string) string {
	return fmt.Sprintf(`
resource "graphiant_public_vif" "test" {
  service_name     = %[1]q
  lan_segment_id   = 100
  region_id        = 1
  storage_provider = %[2]q

  consumer_lan_segments = {
    "200" = {
      consumer_prefixes = ["10.20.0.0/16"]
    }
  }

  gateway_bgp_neighbors = {
    "1" = {
      enabled        = true
      peer_asn       = 65001
      remote_address = "203.0.113.1"
      local_address  = "203.0.113.2"
    }
  }

  nat_prefix_strategy = {
    decentralized = {
      prefixes = {
        "1" = "10.30.0.0/16"
      }
    }
  }
}
`, name, storageProvider)
}
