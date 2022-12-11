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
				Config: providerConfig + testAccClusterResourceConfig("small", "test one"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_cluster.test", "name", "new-cluster"),
					resource.TestCheckResourceAttr("akp_cluster.test", "description", "test one"),
					resource.TestCheckResourceAttr("akp_cluster.test", "namespace", "akuity"),
					resource.TestCheckResourceAttr("akp_cluster.test", "namespace_scoped", "false"),
					resource.TestCheckResourceAttr("akp_cluster.test", "auto_upgrade_disabled", "false"),
					resource.TestCheckResourceAttr("akp_cluster.test", "size", "small"),
					resource.TestCheckResourceAttr("akp_cluster.test", "labels.label_1", "test-label"),
					resource.TestCheckResourceAttr("akp_cluster.test", "annotations.ann_1", "test-annotation"),
					resource.TestCheckResourceAttrSet("akp_cluster.test", "manifests"),
				),
			},
			// Update and Read testing
			{
				Config: providerConfig + testAccClusterResourceConfig("medium","test two"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_cluster.test", "name", "new-cluster"),
					resource.TestCheckResourceAttr("akp_cluster.test", "description", "test two"),
					resource.TestCheckResourceAttr("akp_cluster.test", "namespace", "akuity"),
					resource.TestCheckResourceAttr("akp_cluster.test", "namespace_scoped", "false"),
					resource.TestCheckResourceAttr("akp_cluster.test", "size", "medium"),
					resource.TestCheckResourceAttr("akp_cluster.test", "labels.label_1", "test-label"),
					resource.TestCheckResourceAttr("akp_cluster.test", "annotations.ann_1", "test-annotation"),
					resource.TestCheckResourceAttrSet("akp_cluster.test", "manifests"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccClusterResourceConfig(size string, description string) string {
	return fmt.Sprintf(`
resource "akp_cluster" "test" {
  name = "new-cluster"
  size = %q
  description = %q
  instance_id = "gnjajx9dkszyyp55"
  namespace = "akuity"
  labels = {
	label_1 = "test-label"
  }
  annotations = {
	ann_1 = "test-annotation"
  }
}
`, size, description)
}
