package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAssuranceClassifiedApplicationResource(t *testing.T) {
	appName := acctest.RandomWithPrefix("tf-acc-app")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckDisabled(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAssuranceClassifiedApplicationResourceConfig(appName, "tcp"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graphiant_assurance_classified_application.test", "app_name", appName),
					resource.TestCheckResourceAttrSet("graphiant_assurance_classified_application.test", "id"),
				),
			},
			{
				ResourceName:      "graphiant_assurance_classified_application.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccAssuranceClassifiedApplicationResourceConfig(appName, "udp"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graphiant_assurance_classified_application.test", "protocol_list.0", "udp"),
				),
			},
		},
	})
}

func testAccAssuranceClassifiedApplicationResourceConfig(appName, protocol string) string {
	return fmt.Sprintf(`
resource "graphiant_assurance_classified_application" "test" {
  app_name       = %[1]q
  ip_prefix_list = ["10.0.0.0/24"]
  port_list      = ["443"]
  protocol_list  = [%[2]q]
}
`, appName, protocol)
}
