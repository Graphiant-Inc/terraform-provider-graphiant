package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

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

// rule_id_list references a rule from the fixed platform-wide alert rule
// catalog (graphiant_alert_rules) rather than a hardcoded id, since that
// catalog is not tenant-created and its first entry always exists.
func testAccAlertNotificationResourceConfig(name string, enabled bool) string {
	return fmt.Sprintf(`
data "graphiant_alert_rules" "all" {}

resource "graphiant_alert_notification" "test" {
  notification_name = %[1]q
  rule_id_list       = [data.graphiant_alert_rules.all.rules[0].rule_id]
  enabled            = %[2]t
  recipient_list     = ["tf-acc-test@example.com"]
}
`, name, enabled)
}
