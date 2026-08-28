package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccEnterpriseResource(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-enterprise")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheckDisabled(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccEnterpriseResourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graphiant_enterprise.test", "company_name", name),
					resource.TestCheckResourceAttr("graphiant_enterprise.test", "account_type", "enterprise"),
					resource.TestCheckResourceAttrSet("graphiant_enterprise.test", "id"),
				),
			},
			{
				ResourceName:      "graphiant_enterprise.test",
				ImportState:       true,
				ImportStateVerify: true,
				// IamEnterprise (the read model) has no EnterpriseContract field at all, so it's
				// preserved from config/prior state rather than refreshed — see the resource's
				// applyEnterprise doc comment. Import starts from a blank slate, so this field
				// can't match what was configured.
				ImportStateVerifyIgnore: []string{"enterprise_contract"},
			},
		},
	})
}

func testAccEnterpriseResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "graphiant_enterprise" "test" {
  account_type = "enterprise"
  company_name = %[1]q

  enterprise_contract = {
    contracted_credits = 100
  }
}
`, name)
}
