package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccGroupResource(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-group")
	descriptionUpdated := "updated by terraform acceptance tests"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccGroupResourceConfig(name, "created by terraform acceptance tests"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graphiant_group.test", "name", name),
					resource.TestCheckResourceAttr("graphiant_group.test", "permissions.reports", "read"),
					resource.TestCheckResourceAttrSet("graphiant_group.test", "id"),
				),
			},
			{
				ResourceName:      "graphiant_group.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccGroupResourceConfig(name, descriptionUpdated),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graphiant_group.test", "description", descriptionUpdated),
				),
			},
		},
	})
}

func testAccGroupResourceConfig(name, description string) string {
	return fmt.Sprintf(`
resource "graphiant_group" "test" {
  name        = %[1]q
  description = %[2]q

  permissions {
    reports = "read"
  }
}
`, name, description)
}
