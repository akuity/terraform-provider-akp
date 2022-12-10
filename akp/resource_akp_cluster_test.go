package akp

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccClusterResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + testAccClusterResourceConfig("test one"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_cluster.test", "name", "new-cluster"),
					resource.TestCheckResourceAttr("akp_cluster.test", "description", "test one"),
					resource.TestCheckResourceAttr("akp_cluster.test", "namespace", "akuity"),
					resource.TestCheckResourceAttr("akp_cluster.test", "namespace_scoped", "false"),
					resource.TestCheckResourceAttrSet("akp_cluster.test", "manifests"),
				),
			},
			// Update and Read testing
			{
				Config: providerConfig + testAccClusterResourceConfig("test two"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_cluster.test", "name", "new-cluster"),
					resource.TestCheckResourceAttr("akp_cluster.test", "description", "test two"),
					resource.TestCheckResourceAttr("akp_cluster.test", "namespace", "akuity"),
					resource.TestCheckResourceAttr("akp_cluster.test", "namespace_scoped", "false"),
					resource.TestCheckResourceAttrSet("akp_cluster.test", "manifests"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccClusterResourceConfig(description string) string {
	return fmt.Sprintf(`
resource "akp_cluster" "test" {
  name = "new-cluster"
  description = %q
  instance_id = "gnjajx9dkszyyp55"
  namespace = "akuity"
}
`, description)
}
