package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// This file covers the provider's 14 data sources. Data sources have no create/
// destroy lifecycle and no import, so each test is a single Config+Check step.
// The eight that take no input are exercised as-is; the six that look up a
// specific object by id use placeholder ids that only resolve on a specific
// test tenant — adjust for your own.

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
		PreCheck:                 func() { testAccPreCheck(t) },
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

// device_id/site_id/prefix set & routing policy ids below are placeholders that
// only resolve on a specific test tenant; adjust for your own.

func TestAccDeviceDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "graphiant_device" "test" { id = 12345 }`,
				Check:  resource.TestCheckResourceAttrSet("data.graphiant_device.test", "hostname"),
			},
		},
	})
}

func TestAccSiteDevicesDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "graphiant_site_devices" "test" { site_id = 12345 }`,
				Check:  resource.TestCheckResourceAttrSet("data.graphiant_site_devices.test", "devices.#"),
			},
		},
	})
}

func TestAccTroubleshootingDeviceDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "graphiant_troubleshooting_device" "test" { device_id = 12345 }`,
				Check:  resource.TestCheckResourceAttrSet("data.graphiant_troubleshooting_device.test", "status"),
			},
		},
	})
}

func TestAccTroubleshootingSiteDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "graphiant_troubleshooting_site" "test" { site_id = 12345 }`,
				Check:  resource.TestCheckResourceAttrSet("data.graphiant_troubleshooting_site.test", "site_name"),
			},
		},
	})
}

func TestAccPrefixSetDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
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
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "graphiant_routing_policy" "test" { ids = [12345] }`,
				Check:  resource.TestCheckResourceAttrSet("data.graphiant_routing_policy.test", "policies.#"),
			},
		},
	})
}
