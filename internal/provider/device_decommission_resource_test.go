package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// device_serials is a placeholder that only resolves on a specific test tenant;
// override via GRAPHIANT_ACC_DEVICE_DECOMMISSION_SERIAL for your own.
func TestAccDeviceDecommissionResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckHardcoded(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDeviceDecommissionResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("graphiant_device_decommission.test", "id"),
				),
			},
			// No ImportState step: this resource doesn't implement ResourceWithImportState
			// (it's action-shaped — see its schema description).
		},
	})
}

func testAccDeviceDecommissionResourceConfig() string {
	return fmt.Sprintf(`
resource "graphiant_device_decommission" "test" {
  device_serials = ["%[1]s"]
}
`, testAccEnvOrDefault("GRAPHIANT_ACC_DEVICE_DECOMMISSION_SERIAL", ""))
}
