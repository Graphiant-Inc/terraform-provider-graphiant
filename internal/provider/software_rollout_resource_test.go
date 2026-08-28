package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccSoftwareRolloutResource(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-rollout")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSoftwareRolloutResourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graphiant_software_rollout.test", "name", name),
					resource.TestCheckResourceAttrSet("graphiant_software_rollout.test", "id"),
				),
			},
			{
				ResourceName:      "graphiant_software_rollout.test",
				ImportState:       true,
				ImportStateVerify: true,
				// trigger_now is a fire-once action flag, not persisted server-side.
				ImportStateVerifyIgnore: []string{"trigger_now"},
			},
		},
	})
}

// action/release are placeholders that only resolve on a specific test tenant.
func testAccSoftwareRolloutResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "graphiant_software_rollout" "test" {
  action  = "upgrade"
  name    = %[1]q
  release = "26.8.0"
}
`, name)
}
