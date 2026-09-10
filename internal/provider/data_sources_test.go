package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// This file covers the provider's 14 data sources. Data sources have no create/
// destroy lifecycle and no import, so each test is a single Config+Check step.
// The eight that take no input are exercised as-is. Of the six that look up a
// specific object by id, four (graphiant_device/graphiant_troubleshooting_device/
// graphiant_prefix_set/graphiant_routing_policy) use testAccPreCheckHardcoded and
// never run in CI (see that helper's doc comment): they look up objects this
// provider has no way to create — a device, or a prefix set/routing policy (no
// write path exists for either). graphiant_site_devices/graphiant_troubleshooting_site
// create their own throwaway graphiant_site to look up instead (see
// testAccSiteResourceConfig in site_resource_test.go), now that site creation
// works (it needed address_line1/state/country populated — see that test's
// comment for the history).

func TestAccAlertRecordsDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "graphiant_alert_records" "test" {}`,
				Check:  resource.TestCheckResourceAttrSet("data.graphiant_alert_records.test", "alerts.#"),
			},
		},
	})
}

func TestAccAlertRulesDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckDisabled(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "graphiant_alert_rules" "test" {}`,
				Check:  resource.TestCheckResourceAttrSet("data.graphiant_alert_rules.test", "rules.#"),
			},
		},
	})
}

func TestAccAssuranceDnsproxyEntriesDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "graphiant_assurance_dnsproxy_entries" "test" {}`,
				Check:  resource.TestCheckResourceAttrSet("data.graphiant_assurance_dnsproxy_entries.test", "entries.#"),
			},
		},
	})
}

func TestAccAssuranceFlexAlgosDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "graphiant_assurance_flex_algos" "test" {}`,
				Check:  resource.TestCheckResourceAttrSet("data.graphiant_assurance_flex_algos.test", "flex_algos.#"),
			},
		},
	})
}

func TestAccDomainCategoriesDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "graphiant_domain_categories" "test" {}`,
				Check:  resource.TestCheckResourceAttrSet("data.graphiant_domain_categories.test", "domain_categories.#"),
			},
		},
	})
}

func TestAccEdgesDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "graphiant_edges" "test" {}`,
				Check:  resource.TestCheckResourceAttrSet("data.graphiant_edges.test", "edges.#"),
			},
		},
	})
}

func TestAccIpsecProfilesDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "graphiant_ipsec_profiles" "test" {}`,
				Check:  resource.TestCheckResourceAttrSet("data.graphiant_ipsec_profiles.test", "ipsec_profiles.#"),
			},
		},
	})
}

func TestAccRegionsDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "graphiant_regions" "test" {}`,
				Check:  resource.TestCheckResourceAttrSet("data.graphiant_regions.test", "regions.#"),
			},
		},
	})
}

func TestAccSiteDevicesDataSource(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-site-devices")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSiteResourceConfig(name) + `
data "graphiant_site_devices" "test" {
  site_id = graphiant_site.test.id
}
`,
				Check: resource.TestCheckResourceAttrSet("data.graphiant_site_devices.test", "devices.#"),
			},
		},
	})
}

func TestAccTroubleshootingSiteDataSource(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-troubleshooting-site")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccSiteResourceConfig(name) + `
data "graphiant_troubleshooting_site" "test" {
  site_id = graphiant_site.test.id
}
`,
				Check: resource.TestCheckResourceAttrSet("data.graphiant_troubleshooting_site.test", "site_name"),
			},
		},
	})
}

// device_id/prefix set/routing policy ids below reference objects this provider
// has no way to create (a device, or a prefix set/routing policy — no write path
// exists for either), so these are placeholders that only resolve on a specific
// test tenant; see testAccPreCheckHardcoded. The device id is overridable via
// GRAPHIANT_ACC_DEVICE_ID for your own tenant.

func TestAccDeviceDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckHardcoded(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`data "graphiant_device" "test" { id = %[1]s }`,
					testAccEnvOrDefault("GRAPHIANT_ACC_DEVICE_ID", "12345")),
				Check: resource.TestCheckResourceAttrSet("data.graphiant_device.test", "hostname"),
			},
		},
	})
}

func TestAccTroubleshootingDeviceDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckHardcoded(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`data "graphiant_troubleshooting_device" "test" { device_id = %[1]s }`,
					testAccEnvOrDefault("GRAPHIANT_ACC_DEVICE_ID", "12345")),
				Check: resource.TestCheckResourceAttrSet("data.graphiant_troubleshooting_device.test", "status"),
			},
		},
	})
}

func TestAccPrefixSetDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckHardcoded(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "graphiant_prefix_set" "test" { ids = [12345] }`,
				Check:  resource.TestCheckResourceAttrSet("data.graphiant_prefix_set.test", "prefix_sets.#"),
			},
		},
	})
}

func TestAccRoutingPolicyDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckHardcoded(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "graphiant_routing_policy" "test" { ids = [12345] }`,
				Check:  resource.TestCheckResourceAttrSet("data.graphiant_routing_policy.test", "policies.#"),
			},
		},
	})
}
