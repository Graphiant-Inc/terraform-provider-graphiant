package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccPlaybookConfigAndJobResource needs module YAML that validates against the
// test tenant (it names real devices/sites), which can't be generated on demand,
// so it's gated behind testAccPreCheckHardcoded. Point it at your tenant with:
//
//	GRAPHIANT_ACC_PLAYBOOK_BUNDLE_KEY  bundle key, e.g. system_bundle
//	GRAPHIANT_ACC_PLAYBOOK_MODULE_KEY  module key of the slot the file fills
//	GRAPHIANT_ACC_PLAYBOOK_FILE        path to a module YAML file that validates
//
// The job is never approved: the test only runs the dry-run, so it makes no
// change to the network.
func TestAccPlaybookConfigAndJobResource(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-playbook")
	bundleKey := testAccEnvOrDefault("GRAPHIANT_ACC_PLAYBOOK_BUNDLE_KEY", "system_bundle")
	moduleKey := testAccEnvOrDefault("GRAPHIANT_ACC_PLAYBOOK_MODULE_KEY", "ntp")
	file := testAccEnvOrDefault("GRAPHIANT_ACC_PLAYBOOK_FILE", "testdata/playbook_module.yaml")

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheckHardcoded(t)
			if _, err := os.Stat(file); err != nil {
				t.Skipf("GRAPHIANT_ACC_PLAYBOOK_FILE %q is not readable: %s", file, err)
			}
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccPlaybookResourceConfig(name, "created by acceptance test", bundleKey, moduleKey, file),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("graphiant_playbook_config.test", "id"),
					resource.TestCheckResourceAttr("graphiant_playbook_config.test", "name", name),
					resource.TestCheckResourceAttrSet("graphiant_playbook_job.test", "id"),
					resource.TestCheckResourceAttrSet("graphiant_playbook_job.test", "dry_run_ended_at"),
					resource.TestCheckNoResourceAttr("graphiant_playbook_job.test", "deploy_started_at"),
					resource.TestCheckResourceAttrPair("data.graphiant_playbook_job.latest", "job_id", "graphiant_playbook_job.test", "id"),
					resource.TestCheckResourceAttrSet("data.graphiant_playbook_job_logs.test", "lines.#"),
				),
			},
			{
				ResourceName:      "graphiant_playbook_config.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				ResourceName:            "graphiant_playbook_job.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"triggers"},
			},
			{
				Config: testAccPlaybookResourceConfig(name, "updated by acceptance test", bundleKey, moduleKey, file),
				Check:  resource.TestCheckResourceAttr("graphiant_playbook_config.test", "description", "updated by acceptance test"),
			},
		},
	})
}

func testAccPlaybookResourceConfig(name, description, bundleKey, moduleKey, file string) string {
	return fmt.Sprintf(`
resource "graphiant_playbook_config" "test" {
  name        = %[1]q
  description = %[2]q
  bundle_key  = %[3]q

  files = [{
    module_key = %[4]q
    filename   = basename(%[5]q)
    content    = file(%[5]q)
  }]
}

resource "graphiant_playbook_job" "test" {
  config_id = graphiant_playbook_config.test.id

  triggers = {
    files = sha256(jsonencode(graphiant_playbook_config.test.files))
  }
}

data "graphiant_playbook_job" "latest" {
  config_id  = graphiant_playbook_config.test.id
  depends_on = [graphiant_playbook_job.test]
}

data "graphiant_playbook_job_logs" "test" {
  job_id = graphiant_playbook_job.test.id
  phase  = "dry_run"
}
`, name, description, bundleKey, moduleKey, file)
}
