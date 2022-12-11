package akp

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccClustersDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: providerConfig + testAccClustersDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.akp_clusters.test", "instance_id", "gnjajx9dkszyyp55"),
					resource.TestCheckResourceAttr("data.akp_clusters.test", "clusters.#", "1"),
					resource.TestCheckResourceAttr("data.akp_clusters.test", "clusters.0.id", "k7up9v9cseynv3vc"),
					resource.TestCheckResourceAttr("data.akp_clusters.test", "clusters.0.instance_id", "gnjajx9dkszyyp55"),
					resource.TestCheckResourceAttr("data.akp_clusters.test", "clusters.0.name", "existing-cluster"),
					resource.TestCheckResourceAttr("data.akp_clusters.test", "clusters.0.namespace", "akuity"),
					resource.TestCheckResourceAttr("data.akp_clusters.test", "clusters.0.namespace_scoped", "false"),
					resource.TestCheckResourceAttr("data.akp_clusters.test", "clusters.0.description", "Cluster Description"),
					resource.TestCheckResourceAttr("data.akp_clusters.test", "clusters.0.size", "small"),
					resource.TestCheckResourceAttr("data.akp_clusters.test", "clusters.0.auto_upgrade_disabled", "false"),
					resource.TestCheckResourceAttr("data.akp_clusters.test", "clusters.0.labels.test-label", "test"),
					resource.TestCheckResourceAttr("data.akp_clusters.test", "clusters.0.annotations.test-annotation", "test"),
					resource.TestCheckResourceAttrSet("data.akp_clusters.test", "clusters.0.manifests"),
				),
			},
		},
	})
}

const testAccClustersDataSourceConfig = `
data "akp_clusters" "test" {
  instance_id = "gnjajx9dkszyyp55"
}
`
