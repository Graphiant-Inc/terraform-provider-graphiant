package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// rule_id_list references pre-existing alert rule ids (see graphiant_alert_rules
// data source) that only resolve on a specific test tenant; adjust for your own.
func TestAccAlertNotificationResource(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-notification")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAlertNotificationResourceConfig(name, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graphiant_alert_notification.test", "notification_name", name),
					resource.TestCheckResourceAttr("graphiant_alert_notification.test", "enabled", "true"),
					resource.TestCheckResourceAttrSet("graphiant_alert_notification.test", "id"),
				),
			},
			{
				ResourceName:      "graphiant_alert_notification.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccAlertNotificationResourceConfig(name, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graphiant_alert_notification.test", "enabled", "false"),
				),
			},
		},
	})
}

func testAccAlertNotificationResourceConfig(name string, enabled bool) string {
	return fmt.Sprintf(`
resource "graphiant_alert_notification" "test" {
  notification_name = %[1]q
  rule_id_list       = ["rule-tf-acc-test"]
  enabled            = %[2]t
  recipient_list     = ["tf-acc-test@example.com"]
}
`, name, enabled)
}
