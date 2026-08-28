package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// device_serials is a placeholder that only resolves on a specific test tenant;
// adjust for your own.
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
	return `
resource "graphiant_device_decommission" "test" {
  device_serials = ["TFACCTEST0001"]
}
`
}
