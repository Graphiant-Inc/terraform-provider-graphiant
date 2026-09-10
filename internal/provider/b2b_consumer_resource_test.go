package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccB2bConsumerResource(t *testing.T) {
	prefix := acctest.RandomWithPrefix("tf-acc-b2b-consumer")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckDisabled(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccB2bConsumerResourceConfig(prefix),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("graphiant_b2b_consumer.test", "id"),
					resource.TestCheckResourceAttrSet("graphiant_b2b_consumer.test", "match_id"),
				),
			},
			// No ImportState step: the resource doesn't implement ResourceWithImportState
			// (see its schema description — no get-by-consumer-id endpoint exists).
		},
	})
}

// service_lan_segment/lan_segment/the consumer_lan_segments key/sites all
// reference throwaway graphiant_lan_segment/graphiant_site resources created
// in this same config, rather than hardcoded ids.
func testAccB2bConsumerResourceConfig(prefix string) string {
	return fmt.Sprintf(`
resource "graphiant_lan_segment" "producer" {
  name = "%[1]s-producer-lan"
}

resource "graphiant_lan_segment" "consumer" {
  name = "%[1]s-consumer-lan"
}

resource "graphiant_site" "producer" {
  name  = "%[1]s-producer-site"
  notes = "created by terraform acceptance tests"

  location {
    address_line1 = "123 Main St"
    city          = "San Jose"
    state_code    = "CA"
    country_code  = "US"
  }
}

resource "graphiant_site" "consumer" {
  name  = "%[1]s-consumer-site"
  notes = "created by terraform acceptance tests"

  location {
    address_line1 = "456 Market St"
    city          = "San Francisco"
    state_code    = "CA"
    country_code  = "US"
  }
}

resource "graphiant_b2b_producer_service" "test" {
  service_name = "%[1]s-svc"
  service_type = "peering_service"

  policy = {
    service_lan_segment = graphiant_lan_segment.producer.id

    sites = [
      { sites = [graphiant_site.producer.id] },
    ]

    prefix_tags = [
      { prefix = "10.50.0.0/16", tag = "shared" },
    ]
  }
}

resource "graphiant_b2b_customer" "test" {
  name = "%[1]s-customer"
  type = "graphiant_peer"

  invite = {
    admin_emails = ["admin_2_test@graphiant.com"]
  }
}

resource "graphiant_b2b_match" "test" {
  customer_id = graphiant_b2b_customer.test.id

  match = {
    service_id  = graphiant_b2b_producer_service.test.id
    lan_segment = graphiant_lan_segment.consumer.id

    service_prefixes = [
      { prefix = "10.50.0.0/16", tag = "shared" },
    ]

    nat_translation_mode = {
      peer_to_peer = [
        { prefix = "10.50.0.0/16" },
      ]
    }
  }
}

resource "graphiant_b2b_consumer" "test" {
  customer_id = graphiant_b2b_customer.test.id
  match_id    = graphiant_b2b_match.test.id
  service_id  = graphiant_b2b_producer_service.test.id

  policy = {
    consumer_lan_segments = {
      (graphiant_lan_segment.consumer.id) = {
        consumer_prefixes = ["10.50.0.0/16"]
      }
    }

    sites = [
      { sites = [graphiant_site.consumer.id] },
    ]
  }
}
`, prefix)
}
