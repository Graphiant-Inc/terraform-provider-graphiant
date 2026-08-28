package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccB2bCustomerResource(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-b2b-customer")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccB2bCustomerResourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graphiant_b2b_customer.test", "name", name),
					resource.TestCheckResourceAttr("graphiant_b2b_customer.test", "type", "non-graphiant"),
					resource.TestCheckResourceAttrSet("graphiant_b2b_customer.test", "id"),
				),
			},
			{
				ResourceName:      "graphiant_b2b_customer.test",
				ImportState:       true,
				ImportStateVerify: true,
				// invite.maximum_number_of_sites isn't echoed back by the read endpoint,
				// so it's preserved from config rather than refreshed (see the resource's
				// schema description) — on import there's no config to preserve it from.
				ImportStateVerifyIgnore: []string{"invite.maximum_number_of_sites"},
			},
		},
	})
}

func testAccB2bCustomerResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "graphiant_b2b_customer" "test" {
  name = %[1]q
  type = "non-graphiant"

  invite = {
    admin_emails            = ["admin@tf-acc-test.example"]
    maximum_number_of_sites = 5
  }
}
`, name)
}
