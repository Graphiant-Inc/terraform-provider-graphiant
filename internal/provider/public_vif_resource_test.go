package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPublicVifResource(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-pvif")
	producerLan := acctest.RandomWithPrefix("tf-acc-pvif-producer-lan")
	consumerLan := acctest.RandomWithPrefix("tf-acc-pvif-consumer-lan")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPublicVifResourceConfig(name, producerLan, consumerLan, "aws"),
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
				Config: testAccPublicVifResourceConfig(name, producerLan, consumerLan, "azure"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graphiant_public_vif.test", "storage_provider", "azure"),
				),
			},
		},
	})
}

// lan_segment_id and the consumer_lan_segments key both reference throwaway
// graphiant_lan_segment resources created in this same config, rather than
// hardcoded ids. region_id is looked up from the platform-wide region catalog
// (graphiant_regions), which isn't tenant-created.
func testAccPublicVifResourceConfig(name, producerLan, consumerLan, storageProvider string) string {
	return fmt.Sprintf(`
data "graphiant_regions" "all" {}

resource "graphiant_lan_segment" "producer" {
  name = %[2]q
}

resource "graphiant_lan_segment" "consumer" {
  name = %[3]q
}

resource "graphiant_public_vif" "test" {
  service_name     = %[1]q
  lan_segment_id   = graphiant_lan_segment.producer.id
  region_id        = data.graphiant_regions.all.regions[0].id
  storage_provider = %[4]q

  consumer_lan_segments = {
    (graphiant_lan_segment.consumer.id) = {
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
`, name, producerLan, consumerLan, storageProvider)
}
