package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// The catalog data sources need no ids: module slots and templates look up the
// first bundle graphiant_playbook_bundles returns. graphiant_playbook_job and
// graphiant_playbook_job_logs need a real job and are covered by
// TestAccPlaybookConfigAndJobResource instead.
//
// Disabled by default (see testAccPreCheckDisabled): these haven't been run
// against a live tenant yet, and Automation Workflows may not be enabled on the
// CI test tenant.

func TestAccPlaybookBundlesDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckDisabled(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `data "graphiant_playbook_bundles" "test" {}`,
				Check:  resource.TestCheckResourceAttrSet("data.graphiant_playbook_bundles.test", "bundles.0.key"),
			},
		},
	})
}

func TestAccPlaybookModuleSlotsAndTemplatesDataSources(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckDisabled(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
data "graphiant_playbook_bundles" "all" {}

data "graphiant_playbook_module_slots" "test" {
  bundle_key = data.graphiant_playbook_bundles.all.bundles[0].key
}

data "graphiant_playbook_templates" "test" {
  bundle_key = data.graphiant_playbook_bundles.all.bundles[0].key
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.graphiant_playbook_module_slots.test", "modules.0.module_key"),
					resource.TestCheckResourceAttrSet("data.graphiant_playbook_templates.test", "files.0.content"),
				),
			},
		},
	})
}

func TestAccPlaybookConfigsAndJobsDataSources(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckDisabled(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
data "graphiant_playbook_configs" "test" {}

data "graphiant_playbook_jobs" "test" {}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.graphiant_playbook_configs.test", "configs.#"),
					resource.TestCheckResourceAttrSet("data.graphiant_playbook_jobs.test", "jobs.#"),
				),
			},
		},
	})
}
