package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccDevicesDataSource exercises the read-only graphiant_devices data
// source against a live Graphiant tenant. Unlike sites/groups/users, devices
// can't be created by this provider (onboarding requires physical/virtual
// hardware), so this only verifies the list call succeeds — it does not
// assert a specific count, and does not exercise the singular graphiant_device
// data source (which needs a real device id). Run with:
//
//	TF_ACC=1 GRAPHIANT_ACCESS_TOKEN=... go test ./internal/provider/ -run TestAccDevicesDataSource -v
func TestAccDevicesDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "graphiant_devices" "test" {}`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.graphiant_devices.test", "devices.#"),
				),
			},
		},
	})
}
