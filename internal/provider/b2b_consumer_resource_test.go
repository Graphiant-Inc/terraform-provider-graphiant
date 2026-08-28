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
		PreCheck:                 func() { testAccPreCheck(t) },
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

// service_lan_segment/lan_segment/the consumer_lan_segments key all reference
// throwaway graphiant_lan_segment resources created in this same config,
// rather than hardcoded ids.
func testAccB2bConsumerResourceConfig(prefix string) string {
	return fmt.Sprintf(`
resource "graphiant_lan_segment" "producer" {
  name = "%[1]s-producer-lan"
}

resource "graphiant_lan_segment" "consumer" {
  name = "%[1]s-consumer-lan"
}

resource "graphiant_b2b_producer_service" "test" {
  service_name = "%[1]s-svc"
  service_type = "peering_service"

  policy = {
    service_lan_segment = graphiant_lan_segment.producer.id
  }
}

resource "graphiant_b2b_customer" "test" {
  name = "%[1]s-customer"
  type = "non-graphiant"

  invite = {
    admin_emails = ["admin@tf-acc-test.example"]
  }
}

resource "graphiant_b2b_match" "test" {
  customer_id = graphiant_b2b_customer.test.id

  match = {
    service_id        = graphiant_b2b_producer_service.test.id
    lan_segment       = graphiant_lan_segment.producer.id
    consumer_prefixes = ["10.50.0.0/16"]
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
  }
}
`, prefix)
}
