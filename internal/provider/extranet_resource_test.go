package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccExtranetResource(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-extranet")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckDisabled(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccExtranetResourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graphiant_extranet.test", "name", name),
					resource.TestCheckResourceAttrSet("graphiant_extranet.test", "id"),
				),
			},
			{
				ResourceName:      "graphiant_extranet.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccExtranetResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "graphiant_extranet" "test" {
  name = %[1]q

  auto = {
    auto_propagate = true
  }
}
`, name)
}
