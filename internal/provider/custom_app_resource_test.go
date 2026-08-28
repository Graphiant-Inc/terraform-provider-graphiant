package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccCustomAppResource(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-custom-app")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccCustomAppResourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graphiant_custom_app.test", "name", name),
					resource.TestCheckResourceAttr("graphiant_custom_app.test", "ip_prefixes.0", "10.10.0.0/16"),
					resource.TestCheckResourceAttrSet("graphiant_custom_app.test", "id"),
				),
			},
			{
				ResourceName:      "graphiant_custom_app.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccCustomAppResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "graphiant_custom_app" "test" {
  name        = %[1]q
  ip_prefixes = ["10.10.0.0/16"]
}
`, name)
}
