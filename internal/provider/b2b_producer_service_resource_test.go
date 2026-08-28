package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccB2bProducerServiceResource(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-b2b-producer")
	lanSegmentName := acctest.RandomWithPrefix("tf-acc-b2b-producer-lan")
	siteName := acctest.RandomWithPrefix("tf-acc-b2b-producer-site")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccB2bProducerServiceResourceConfig(name, lanSegmentName, siteName, "peering description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graphiant_b2b_producer_service.test", "service_name", name),
					resource.TestCheckResourceAttr("graphiant_b2b_producer_service.test", "policy.description", "peering description"),
					resource.TestCheckResourceAttrSet("graphiant_b2b_producer_service.test", "id"),
				),
			},
			{
				ResourceName:      "graphiant_b2b_producer_service.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccB2bProducerServiceResourceConfig(name, lanSegmentName, siteName, "updated description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graphiant_b2b_producer_service.test", "policy.description", "updated description"),
				),
			},
		},
	})
}

// service_lan_segment and sites both reference throwaway graphiant_lan_segment/
// graphiant_site resources created in this same config, rather than hardcoded ids.
func testAccB2bProducerServiceResourceConfig(name, lanSegmentName, siteName, description string) string {
	return fmt.Sprintf(`
resource "graphiant_lan_segment" "test" {
  name = %[2]q
}

resource "graphiant_site" "test" {
  name  = %[3]q
  notes = "created by terraform acceptance tests"

  location {
    city         = "San Jose"
    state_code   = "CA"
    country_code = "US"
  }
}

resource "graphiant_b2b_producer_service" "test" {
  service_name = %[1]q
  service_type = "peering_service"

  policy = {
    description         = %[4]q
    service_lan_segment = graphiant_lan_segment.test.id

    sites = [
      { sites = [graphiant_site.test.id] },
    ]

    prefix_tags = [
      { prefix = "10.40.0.0/16", tag = "shared" },
    ]

    nat_translation_mode = {
      peer_to_peer = [
        { prefix = "10.40.0.0/16" },
      ]
    }
  }
}
`, name, lanSegmentName, siteName, description)
}
