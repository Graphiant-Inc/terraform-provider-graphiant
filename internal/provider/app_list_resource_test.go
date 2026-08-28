package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAppListResource(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-app-list")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAppListResourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graphiant_app_list.test", "name", name),
					resource.TestCheckResourceAttrSet("graphiant_app_list.test", "id"),
				),
			},
			{
				ResourceName:      "graphiant_app_list.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccAppListResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "graphiant_app_list" "test" {
  name        = %[1]q
  description = "created by terraform acceptance tests"
}
`, name)
}
