//go:build !unit

package akp

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
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
					resource.TestCheckResourceAttr("data.akp_cluster.test", "instance_id", "6pzhawvy4echbd8x"),
					resource.TestCheckResourceAttr("data.akp_cluster.test", "id", "nyc6s87mrlh4s2af"),
					resource.TestCheckResourceAttr("data.akp_cluster.test", "name", "data-source-cluster"),
					resource.TestCheckResourceAttr("data.akp_cluster.test", "namespace", "akuity"),
					resource.TestCheckResourceAttr("data.akp_cluster.test", "labels.test-label", "test"),
					resource.TestCheckResourceAttr("data.akp_cluster.test", "annotations.test-annotation", "false"),
					// spec
					resource.TestCheckResourceAttr("data.akp_cluster.test", "spec.description", "Cluster Description"),
					resource.TestCheckResourceAttr("data.akp_cluster.test", "spec.namespace_scoped", "false"),
					// spec.data
					resource.TestCheckResourceAttr("data.akp_cluster.test", "spec.data.size", "small"),
					resource.TestCheckResourceAttr("data.akp_cluster.test", "spec.data.auto_upgrade_disabled", "true"),
					resource.TestCheckResourceAttr("data.akp_cluster.test", "spec.data.kustomization", `apiVersion: kustomize.config.k8s.io/v1beta1
images:
- name: quay.io/akuityio/agent
  newName: test.io/agent
kind: Kustomization
`),
					resource.TestCheckResourceAttr("data.akp_cluster.test", "spec.data.app_replication", "false"),
					resource.TestCheckResourceAttr("data.akp_cluster.test", "spec.data.target_version", "0.4.3"),
					resource.TestCheckResourceAttr("data.akp_cluster.test", "spec.data.redis_tunneling", "true"),
				),
			},
		},
	})
}

const testAccClusterDataSourceConfig = `
data "akp_cluster" "test" {
  instance_id = "6pzhawvy4echbd8x"
  name = "data-source-cluster"
}
`
