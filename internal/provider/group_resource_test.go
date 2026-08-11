package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccGroupResource exercises the full graphiant_group lifecycle against a
// live Graphiant tenant: create with permissions, read back, update, import,
// and delete. It also exercises the graphiant_group and graphiant_groups
// data sources against the same resource. Run with:
//
//	TF_ACC=1 GRAPHIANT_ACCESS_TOKEN=... go test ./internal/provider/ -run TestAccGroupResource -v
func TestAccGroupResource(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-group")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and read back.
			{
				Config: testAccGroupResourceConfig(name, "created by terraform-provider-graphiant acceptance tests", "read"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graphiant_group.test", "name", name),
					resource.TestCheckResourceAttr("graphiant_group.test", "description", "created by terraform-provider-graphiant acceptance tests"),
					resource.TestCheckResourceAttr("graphiant_group.test", "permissions.network_configuration", "read"),
					resource.TestCheckResourceAttrSet("graphiant_group.test", "id"),
					// graphiant_group data source, looked up by the resource's id.
					resource.TestCheckResourceAttrPair("data.graphiant_group.test", "id", "graphiant_group.test", "id"),
					resource.TestCheckResourceAttrPair("data.graphiant_group.test", "name", "graphiant_group.test", "name"),
					// graphiant_groups data source should list at least the group just created.
					resource.TestCheckResourceAttrSet("data.graphiant_groups.test", "groups.#"),
				),
			},
			// ImportState.
			{
				ResourceName:      "graphiant_group.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update description and widen a permission.
			{
				Config: testAccGroupResourceConfig(name, "updated by terraform-provider-graphiant acceptance tests", "write"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graphiant_group.test", "description", "updated by terraform-provider-graphiant acceptance tests"),
					resource.TestCheckResourceAttr("graphiant_group.test", "permissions.network_configuration", "write"),
				),
			},
		},
	})
}

func testAccGroupResourceConfig(name, description, networkConfigPermission string) string {
	return fmt.Sprintf(`
resource "graphiant_group" "test" {
  name        = %[1]q
  description = %[2]q

  permissions = {
    network_configuration           = %[3]q
    monitoring_and_troubleshooting  = "read"
  }
}

data "graphiant_group" "test" {
  id = graphiant_group.test.id
}

data "graphiant_groups" "test" {
  depends_on = [graphiant_group.test]
}
`, name, description, networkConfigPermission)
}
