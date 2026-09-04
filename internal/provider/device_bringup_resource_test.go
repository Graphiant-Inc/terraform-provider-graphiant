package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// device_ids is a placeholder that only resolves on a specific test tenant with
// a device pending bringup; override via GRAPHIANT_ACC_DEVICE_BRINGUP_ID for
// your own tenant, or source via graphiant_edges.
func TestAccDeviceBringupResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckHardcoded(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDeviceBringupResourceConfig(),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graphiant_device_bringup.test", "status", "Allowed"),
					resource.TestCheckResourceAttrSet("graphiant_device_bringup.test", "id"),
				),
			},
			// No ImportState step: this resource doesn't implement ResourceWithImportState
			// (it's action-shaped — see its schema description).
		},
	})
}

func testAccDeviceBringupResourceConfig() string {
	return fmt.Sprintf(`
resource "graphiant_device_bringup" "test" {
  device_ids = [%[1]s]
  status     = "Allowed"
}
`, testAccEnvOrDefault("GRAPHIANT_ACC_DEVICE_BRINGUP_ID", "12345"))
}
