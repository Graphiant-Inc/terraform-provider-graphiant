package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccB2bMatchResource(t *testing.T) {
	prefix := acctest.RandomWithPrefix("tf-acc-b2b-match")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckDisabled(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccB2bMatchResourceConfig(prefix, "10.50.0.0/16"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graphiant_b2b_match.test", "match.nat_translation_mode.peer_to_peer.0.prefix", "10.50.0.0/16"),
					resource.TestCheckResourceAttrSet("graphiant_b2b_match.test", "id"),
				),
			},
			{
				ResourceName:      "graphiant_b2b_match.test",
				ImportState:       true,
				ImportStateVerify: true,
				// customer_id and match.lan_segment aren't echoed back by the read
				// endpoint, so they're preserved from prior state rather than
				// refreshed from the API — but import starts from empty state, so
				// there's nothing to preserve match.lan_segment from.
				ImportStateVerifyIgnore: []string{"customer_id", "match.lan_segment"},
			},
			{
				Config: testAccB2bMatchResourceConfig(prefix, "10.51.0.0/16"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graphiant_b2b_match.test", "match.nat_translation_mode.peer_to_peer.0.prefix", "10.51.0.0/16"),
				),
			},
		},
	})
}

// service_lan_segment/lan_segment both reference a throwaway graphiant_lan_segment
// created in this same config, rather than a hardcoded id.
func testAccB2bMatchResourceConfig(prefix, consumerPrefix string) string {
	return fmt.Sprintf(`
resource "graphiant_lan_segment" "test" {
  name = "%[1]s-lan"
}

resource "graphiant_site" "test" {
  name  = "%[1]s-site"
  notes = "created by terraform acceptance tests"

  location {
    address_line1 = "123 Main St"
    city          = "San Jose"
    state_code    = "CA"
    country_code  = "US"
  }
}

resource "graphiant_b2b_producer_service" "test" {
  service_name = "%[1]s-svc"
  service_type = "peering_service"

  policy = {
    service_lan_segment = graphiant_lan_segment.test.id

    sites = [
      { sites = [graphiant_site.test.id] },
    ]

    prefix_tags = [
      { prefix = "10.50.0.0/16", tag = "shared" },
      { prefix = "10.51.0.0/16", tag = "shared" },
    ]
  }
}

resource "graphiant_b2b_customer" "test" {
  name = "%[1]s-customer"
  type = "non_graphiant_peer"

  invite = {
    admin_emails = ["admin@tf-acc-test.example"]
  }
}

resource "graphiant_b2b_match" "test" {
  customer_id = graphiant_b2b_customer.test.id

  match = {
    service_id  = graphiant_b2b_producer_service.test.id
    lan_segment = graphiant_lan_segment.test.id

    service_prefixes = [
      { prefix = %[2]q, tag = "shared" },
    ]

    nat_translation_mode = {
      peer_to_peer = [
        { prefix = %[2]q },
      ]
    }
  }
}
`, prefix, consumerPrefix)
}
