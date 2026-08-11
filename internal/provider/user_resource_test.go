package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

// TestAccUserResource exercises the full graphiant_user lifecycle against a
// live Graphiant tenant: create assigned to a graphiant_group, read back,
// update, import, and delete. It also exercises the graphiant_user and
// graphiant_users data sources against the same resource. Run with:
//
//	TF_ACC=1 GRAPHIANT_ACCESS_TOKEN=... go test ./internal/provider/ -run TestAccUserResource -v
func TestAccUserResource(t *testing.T) {
	groupName := acctest.RandomWithPrefix("tf-acc-user-group")
	email := fmt.Sprintf("%s@example.com", acctest.RandomWithPrefix("tf-acc-user"))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and read back.
			{
				Config: testAccUserResourceConfig(groupName, email, "Jane", "Doe"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graphiant_user.test", "email", email),
					resource.TestCheckResourceAttr("graphiant_user.test", "first_name", "Jane"),
					resource.TestCheckResourceAttr("graphiant_user.test", "last_name", "Doe"),
					resource.TestCheckResourceAttrPair("graphiant_user.test", "group_id", "graphiant_group.test", "id"),
					resource.TestCheckResourceAttrSet("graphiant_user.test", "id"),
					// graphiant_user data source, looked up by the resource's id (email).
					resource.TestCheckResourceAttrPair("data.graphiant_user.test", "id", "graphiant_user.test", "id"),
					resource.TestCheckResourceAttrPair("data.graphiant_user.test", "email", "graphiant_user.test", "email"),
					// graphiant_users data source should list at least the user just created.
					resource.TestCheckResourceAttrSet("data.graphiant_users.test", "users.#"),
				),
			},
			// ImportState.
			{
				ResourceName:      "graphiant_user.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update name fields (email cannot change; it forces replacement).
			{
				Config: testAccUserResourceConfig(groupName, email, "Janet", "Doe"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graphiant_user.test", "first_name", "Janet"),
				),
			},
		},
	})
}

func testAccUserResourceConfig(groupName, email, firstName, lastName string) string {
	return fmt.Sprintf(`
resource "graphiant_group" "test" {
  name        = %[1]q
  description = "group for terraform-provider-graphiant acceptance tests"
}

resource "graphiant_user" "test" {
  email      = %[2]q
  first_name = %[3]q
  last_name  = %[4]q
  group_id   = graphiant_group.test.id
}

data "graphiant_user" "test" {
  id = graphiant_user.test.id
}

data "graphiant_users" "test" {
  depends_on = [graphiant_user.test]
}
`, groupName, email, firstName, lastName)
}
