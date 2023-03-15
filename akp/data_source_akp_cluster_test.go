package akp

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccClusterDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: providerConfig + testAccClusterDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.akp_cluster.test", "instance_id", "jszoyttk16rocq66"),
					resource.TestCheckResourceAttr("data.akp_cluster.test", "name", "existing-cluster"),
					resource.TestCheckResourceAttr("data.akp_cluster.test", "id", "modizax44zfw3usr"),
					resource.TestCheckResourceAttr("data.akp_cluster.test", "namespace", "akuity"),
					resource.TestCheckResourceAttr("data.akp_cluster.test", "namespace_scoped", "false"),
					resource.TestCheckResourceAttr("data.akp_cluster.test", "description", "Cluster Description"),
					resource.TestCheckResourceAttr("data.akp_cluster.test", "size", "small"),
					resource.TestCheckResourceAttr("data.akp_cluster.test", "auto_upgrade_disabled", "false"),
					resource.TestCheckResourceAttr("data.akp_cluster.test", "labels.test-label", "test"),
					resource.TestCheckResourceAttr("data.akp_cluster.test", "annotations.test-annotation", "test"),
					resource.TestCheckResourceAttrSet("data.akp_cluster.test", "manifests"),
				),
			},
		},
	})
}

const testAccClusterDataSourceConfig = `
data "akp_cluster" "test" {
  instance_id = "jszoyttk16rocq66"
  name = "existing-cluster"
}
`
