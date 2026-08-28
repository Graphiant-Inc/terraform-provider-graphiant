package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccUserResource(t *testing.T) {
	email := fmt.Sprintf("%s@example.com", acctest.RandomWithPrefix("tf-acc-user"))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccUserResourceConfig(email),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graphiant_user.test", "email", email),
					resource.TestCheckResourceAttr("graphiant_user.test", "first_name", "Terraform"),
					resource.TestCheckResourceAttrSet("graphiant_user.test", "id"),
				),
			},
			{
				ResourceName:      "graphiant_user.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccUserResourceConfig(email string) string {
	return fmt.Sprintf(`
resource "graphiant_user" "test" {
  email      = %[1]q
  first_name = "Terraform"
  last_name  = "AccTest"
}
`, email)
}
