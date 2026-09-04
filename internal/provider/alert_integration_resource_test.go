package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

// enterprise is a placeholder that only resolves on a specific test tenant. It's
// not replaced with a freshly-created graphiant_enterprise here because it's
// unverified whether alert integrations can be scoped to an arbitrary enterprise
// or only the caller's own — adjust the placeholder for your own tenant.
func TestAccAlertIntegrationResource(t *testing.T) {
	nickName := acctest.RandomWithPrefix("tf-acc-integration")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckHardcoded(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAlertIntegrationResourceConfig(nickName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graphiant_alert_integration.test", "nick_name", nickName),
					resource.TestCheckResourceAttrSet("graphiant_alert_integration.test", "id"),
				),
			},
			{
				ResourceName:      "graphiant_alert_integration.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["graphiant_alert_integration.test"]
					if !ok {
						return "", fmt.Errorf("resource not found in state")
					}
					return fmt.Sprintf("%s:%s", rs.Primary.Attributes["enterprise"], rs.Primary.ID), nil
				},
			},
		},
	})
}

func testAccAlertIntegrationResourceConfig(nickName string) string {
	return fmt.Sprintf(`
resource "graphiant_alert_integration" "test" {
  enterprise       = 1
  integration_type = "webhook_url"
  nick_name        = %[1]q
  is_active        = true

  details = {
    webhook_url = "https://hooks.example.com/tf-acc-test"
  }
}
`, nickName)
}
