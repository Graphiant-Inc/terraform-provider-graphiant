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
		PreCheck:                 func() { testAccPreCheckDisabled(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccB2bProducerServiceResourceConfig(name, lanSegmentName, siteName, "peering description", false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graphiant_b2b_producer_service.test", "service_name", name),
					resource.TestCheckResourceAttr("graphiant_b2b_producer_service.test", "policy.description", "peering description"),
					resource.TestCheckResourceAttr("graphiant_b2b_producer_service.test", "policy.prefix_tags.#", "1"),
					resource.TestCheckResourceAttrSet("graphiant_b2b_producer_service.test", "id"),
				),
			},
			{
				ResourceName:      "graphiant_b2b_producer_service.test",
				ImportState:       true,
				ImportStateVerify: true,
				// policy.nat_translation_mode isn't echoed back by the read endpoint,
				// so it's preserved from prior state rather than refreshed from the
				// API — but import starts from empty state, so there's nothing to
				// preserve it from.
				ImportStateVerifyIgnore: []string{"policy.nat_translation_mode"},
			},
			{
				// For peering_service, existing policy fields (description,
				// service_lan_segment, sites, and existing prefix_tags entries) are
				// immutable once created — the backend rejects any change to them
				// with a 500 ("X cannot be modified for peering_service"). The only
				// supported update is appending a new prefix_tags entry, so this step
				// adds a second entry rather than modifying the first.
				Config: testAccB2bProducerServiceResourceConfig(name, lanSegmentName, siteName, "peering description", true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graphiant_b2b_producer_service.test", "policy.prefix_tags.#", "2"),
					resource.TestCheckResourceAttr("graphiant_b2b_producer_service.test", "policy.prefix_tags.1.prefix", "10.41.0.0/16"),
				),
			},
		},
	})
}

// service_lan_segment and sites both reference throwaway graphiant_lan_segment/
// graphiant_site resources created in this same config, rather than hardcoded ids.
func testAccB2bProducerServiceResourceConfig(name, lanSegmentName, siteName, description string, addSecondPrefix bool) string {
	secondPrefixTag := ""
	if addSecondPrefix {
		secondPrefixTag = `{ prefix = "10.41.0.0/16", tag = "shared-2" },`
	}
	return fmt.Sprintf(`
resource "graphiant_lan_segment" "test" {
  name = %[2]q
}

resource "graphiant_site" "test" {
  name  = %[3]q
  notes = "created by terraform acceptance tests"

  location {
    address_line1 = "123 Main St"
    city          = "San Jose"
    state_code    = "CA"
    country_code  = "US"
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
      %[5]s
    ]

    nat_translation_mode = {
      peer_to_peer = [
        { prefix = "10.40.0.0/16" },
      ]
    }
  }
}
`, name, lanSegmentName, siteName, description, secondPrefixTag)
}
