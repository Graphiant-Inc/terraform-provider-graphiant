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
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccB2bMatchResourceConfig(prefix, "10.50.0.0/16"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graphiant_b2b_match.test", "match.consumer_prefixes.0", "10.50.0.0/16"),
					resource.TestCheckResourceAttrSet("graphiant_b2b_match.test", "id"),
				),
			},
			{
				ResourceName:      "graphiant_b2b_match.test",
				ImportState:       true,
				ImportStateVerify: true,
				// customer_id isn't echoed back by the read endpoint, so it's preserved
				// from config rather than refreshed (see the resource's schema description).
				ImportStateVerifyIgnore: []string{"customer_id"},
			},
			{
				Config: testAccB2bMatchResourceConfig(prefix, "10.51.0.0/16"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graphiant_b2b_match.test", "match.consumer_prefixes.0", "10.51.0.0/16"),
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

resource "graphiant_b2b_producer_service" "test" {
  service_name = "%[1]s-svc"
  service_type = "peering_service"

  policy = {
    service_lan_segment = graphiant_lan_segment.test.id
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
    lan_segment       = graphiant_lan_segment.test.id
    consumer_prefixes = [%[2]q]
  }
}
`, prefix, consumerPrefix)
}
