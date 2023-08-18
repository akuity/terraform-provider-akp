//go:build !unit

package akp

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
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
					resource.TestCheckResourceAttr("data.akp_clusters.test", "instance_id", "6pzhawvy4echbd8x"),
					resource.TestCheckResourceAttr("data.akp_clusters.test", "id", "6pzhawvy4echbd8x"),
					resource.TestCheckResourceAttr("data.akp_clusters.test", "clusters.#", "1"),
					resource.TestCheckResourceAttr("data.akp_clusters.test", "clusters.0.id", "nyc6s87mrlh4s2af"),
					resource.TestCheckResourceAttr("data.akp_clusters.test", "clusters.0.instance_id", "6pzhawvy4echbd8x"),
					resource.TestCheckResourceAttr("data.akp_clusters.test", "clusters.0.name", "data-source-cluster"),
					resource.TestCheckResourceAttr("data.akp_clusters.test", "clusters.0.namespace", "akuity"),
					resource.TestCheckResourceAttr("data.akp_clusters.test", "clusters.0.labels.test-label", "test"),
					resource.TestCheckResourceAttr("data.akp_clusters.test", "clusters.0.annotations.test-annotation", "false"),
					// spec
					resource.TestCheckResourceAttr("data.akp_clusters.test", "clusters.0.spec.description", "Cluster Description"),
					resource.TestCheckResourceAttr("data.akp_clusters.test", "clusters.0.spec.namespace_scoped", "false"),
					// spec.data
					resource.TestCheckResourceAttr("data.akp_clusters.test", "clusters.0.spec.data.size", "small"),
					resource.TestCheckResourceAttr("data.akp_clusters.test", "clusters.0.spec.data.auto_upgrade_disabled", "true"),
					resource.TestCheckResourceAttr("data.akp_clusters.test", "clusters.0.spec.data.kustomization", `apiVersion: kustomize.config.k8s.io/v1beta1
images:
- name: quay.io/akuityio/agent
  newName: test.io/agent
kind: Kustomization
`),
					resource.TestCheckResourceAttr("data.akp_clusters.test", "clusters.0.spec.data.app_replication", "false"),
					resource.TestCheckResourceAttr("data.akp_clusters.test", "clusters.0.spec.data.target_version", "0.4.3"),
					resource.TestCheckResourceAttr("data.akp_clusters.test", "clusters.0.spec.data.redis_tunneling", "true"),
				),
			},
		},
	})
}

const testAccClustersDataSourceConfig = `
data "akp_clusters" "test" {
  instance_id = "6pzhawvy4echbd8x"
}
`
