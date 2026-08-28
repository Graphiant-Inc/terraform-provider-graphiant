package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// device_id is a placeholder that only resolves on a specific test tenant;
// adjust for your own.
func TestAccDeviceConfigResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDeviceConfigResourceConfig(true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graphiant_device_config.test", "maintenance_mode", "true"),
					resource.TestCheckResourceAttrSet("graphiant_device_config.test", "id"),
				),
			},
			{
				ResourceName:      "graphiant_device_config.test",
				ImportState:       true,
				ImportStateId:     "12345:edge",
				ImportStateVerify: true,
			},
			{
				Config: testAccDeviceConfigResourceConfig(false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graphiant_device_config.test", "maintenance_mode", "false"),
				),
			},
		},
	})
}

func testAccDeviceConfigResourceConfig(maintenanceMode bool) string {
	return fmt.Sprintf(`
resource "graphiant_device_config" "test" {
  device_id   = 12345
  device_type = "edge"

  maintenance_mode = %[1]t
}
`, maintenanceMode)
}
