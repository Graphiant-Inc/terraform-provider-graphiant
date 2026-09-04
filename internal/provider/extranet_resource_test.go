package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccExtranetResource(t *testing.T) {
	name := acctest.RandomWithPrefix("tf-acc-extranet")
	sharedLanSegmentName := acctest.RandomWithPrefix("tf-acc-extranet-shared")
	targetLanSegmentName := acctest.RandomWithPrefix("tf-acc-extranet-target")
	siteName := acctest.RandomWithPrefix("tf-acc-extranet-site")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccExtranetResourceConfig(name, sharedLanSegmentName, targetLanSegmentName, siteName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("graphiant_extranet.test", "name", name),
					resource.TestCheckResourceAttrSet("graphiant_extranet.test", "id"),
				),
			},
			{
				ResourceName:      "graphiant_extranet.test",
				ImportState:       true,
				ImportStateVerify: true,
				// The read model doesn't echo "auto" back for this policy type 
				ImportStateVerifyIgnore: []string{"auto"},
			},
		},
	})
}

// source/shared_segment/target_segments are schema-Optional but the API rejects
// this policy without them. shared_segment/target_segments use throwaway
// graphiant_lan_segment resources created in this same config (see
// gateway_resource_test.go/public_vif_resource_test.go for the same pattern).
// source.sites uses a throwaway graphiant_site the same way (see
// testAccSiteResourceConfig in site_resource_test.go, and
// graphiant_site_devices/graphiant_troubleshooting_site in data_sources_test.go
// for the same pattern).
func testAccExtranetResourceConfig(name, sharedLanSegmentName, targetLanSegmentName, siteName string) string {
	return testAccSiteResourceConfig(siteName) + fmt.Sprintf(`
resource "graphiant_lan_segment" "shared" {
  name = %[2]q
}

resource "graphiant_lan_segment" "target" {
  name = %[3]q
}

resource "graphiant_extranet" "test" {
  name = %[1]q

  auto = {
    auto_propagate = true
  }

  source = {
    sites = [graphiant_site.test.id]
  }

  shared_segment = graphiant_lan_segment.shared.id

  target_segments = [graphiant_lan_segment.target.id]
}
`, name, sharedLanSegmentName, targetLanSegmentName)
}
