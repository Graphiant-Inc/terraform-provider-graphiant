package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccAssuranceGlobalResource(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-assurance")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckDisabled(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccAssuranceGlobalResourceConfig(name, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graphiant_assurance_global.test", "name", name),
					resource.TestCheckResourceAttr("graphiant_assurance_global.test", "use_all_sites", "true"),
					resource.TestCheckResourceAttrSet("graphiant_assurance_global.test", "id"),
				),
			},
			{
				ResourceName:      "graphiant_assurance_global.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				Config: testAccAssuranceGlobalResourceConfig(name, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graphiant_assurance_global.test", "use_all_sites", "false"),
				),
			},
		},
	})
}

func testAccAssuranceGlobalResourceConfig(name string, useAllSites bool) string {
	return fmt.Sprintf(`
resource "graphiant_assurance_global" "test" {
  name          = %[1]q
  flex_algo     = "default"
  use_all_sites = %[2]t
}
`, name, useAllSites)
}
