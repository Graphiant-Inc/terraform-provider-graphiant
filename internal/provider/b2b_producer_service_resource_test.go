package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// service_lan_segment/sites reference placeholders that only resolve on a
// specific test tenant; adjust for your own.
func TestAccB2bProducerServiceResource(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-b2b-producer")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccB2bProducerServiceResourceConfig(name, "peering description"),
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
				Config: testAccB2bProducerServiceResourceConfig(name, "updated description"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graphiant_b2b_producer_service.test", "policy.description", "updated description"),
				),
			},
		},
	})
}

func testAccB2bProducerServiceResourceConfig(name, description string) string {
	return fmt.Sprintf(`
resource "graphiant_b2b_producer_service" "test" {
  service_name = %[1]q
  service_type = "peering_service"

  policy = {
    description         = %[2]q
    service_lan_segment = 100

    sites = [
      { sites = [12345] },
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
`, name, description)
}
