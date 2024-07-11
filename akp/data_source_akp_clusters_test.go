//go:build !unit

package akp

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccClustersDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccClustersDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckClusterAttributes("data.akp_clusters.test", "data-source-cluster"),
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

func testAccCheckClusterAttributes(dataSourceName string, targetClusterName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[dataSourceName]
		if !ok {
			return fmt.Errorf("data source %s not found", dataSourceName)
		}
		clusters := rs.Primary.Attributes
		for i := 0; ; i++ {
			if clusters[fmt.Sprintf("clusters.%d.name", i)] == "" {
				break
			}
			if clusters[fmt.Sprintf("clusters.%d.name", i)] == targetClusterName {
				if err := resource.TestCheckResourceAttr(dataSourceName, fmt.Sprintf("clusters.%d.instance_id", i), "6pzhawvy4echbd8x")(s); err != nil {
					return err
				}
				if err := resource.TestCheckResourceAttr(dataSourceName, fmt.Sprintf("clusters.%d.id", i), "nyc6s87mrlh4s2af")(s); err != nil {
					return err
				}
				if err := resource.TestCheckResourceAttr(dataSourceName, fmt.Sprintf("clusters.%d.name", i), "data-source-cluster")(s); err != nil {
					return err
				}
				if err := resource.TestCheckResourceAttr(dataSourceName, fmt.Sprintf("clusters.%d.namespace", i), "akuity")(s); err != nil {
					return err
				}
				if err := resource.TestCheckResourceAttr(dataSourceName, fmt.Sprintf("clusters.%d.labels.test-label", i), "test")(s); err != nil {
					return err
				}
				if err := resource.TestCheckResourceAttr(dataSourceName, fmt.Sprintf("clusters.%d.annotations.test-annotation", i), "false")(s); err != nil {
					return err
				}
				if err := resource.TestCheckResourceAttr(dataSourceName, fmt.Sprintf("clusters.%d.spec.description", i), "Cluster Description")(s); err != nil {
					return err
				}
				if err := resource.TestCheckResourceAttr(dataSourceName, fmt.Sprintf("clusters.%d.spec.namespace_scoped", i), "false")(s); err != nil {
					return err
				}
				if err := resource.TestCheckResourceAttr(dataSourceName, fmt.Sprintf("clusters.%d.spec.data.size", i), "small")(s); err != nil {
					return err
				}
				if err := resource.TestCheckResourceAttr(dataSourceName, fmt.Sprintf("clusters.%d.spec.data.auto_upgrade_disabled", i), "true")(s); err != nil {
					return err
				}
				if err := resource.TestCheckResourceAttr(dataSourceName, fmt.Sprintf("clusters.%d.spec.data.kustomization", i), `apiVersion: kustomize.config.k8s.io/v1beta1
images:
- name: quay.io/akuity/agent
  newName: test.io/agent
kind: Kustomization
`)(s); err != nil {
					return err
				}
				if err := resource.TestCheckResourceAttr(dataSourceName, fmt.Sprintf("clusters.%d.spec.data.app_replication", i), "false")(s); err != nil {
					return err
				}
				if err := resource.TestCheckResourceAttr(dataSourceName, fmt.Sprintf("clusters.%d.spec.data.redis_tunneling", i), "true")(s); err != nil {
					return err
				}
				return nil
			}
		}
		return fmt.Errorf("target cluster %s not found", targetClusterName)
	}
}
