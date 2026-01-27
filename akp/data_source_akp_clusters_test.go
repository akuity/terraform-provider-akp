//go:build !unit

package akp

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccClustersDataSource(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("ds-clusters-%s", acctest.RandString(8))
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + getAccClustersDataSourceConfig(getInstanceId(), name),
				Check: resource.ComposeAggregateTestCheckFunc(
					testAccCheckClusterAttributes("data.akp_clusters.test", name),
				),
			},
		},
	})
}

func getAccClustersDataSourceConfig(instanceId, name string) string {
	return fmt.Sprintf(`
resource "akp_cluster" "test" {
  instance_id = %q
  name        = %q
  namespace   = "akuity"
  labels = {
    test-label = "test"
  }
  annotations = {
    test-annotation = "false"
  }
  spec = {
    namespace_scoped = false
    description      = "Cluster Description"
    data = {
      size                  = "small"
      auto_upgrade_disabled = true
      kustomization         = <<EOF
apiVersion: kustomize.config.k8s.io/v1beta1
images:
- name: quay.io/akuity/agent
  newName: test.io/agent
kind: Kustomization
EOF
      app_replication = false
      redis_tunneling = true
    }
  }
}

data "akp_clusters" "test" {
  instance_id = akp_cluster.test.instance_id
}
`, instanceId, name)
}

func testAccCheckClusterAttributes(dataSourceName, targetClusterName string) resource.TestCheckFunc {
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
				if err := resource.TestCheckResourceAttrSet(dataSourceName, fmt.Sprintf("clusters.%d.instance_id", i))(s); err != nil {
					return err
				}
				if err := resource.TestCheckResourceAttrSet(dataSourceName, fmt.Sprintf("clusters.%d.id", i))(s); err != nil {
					return err
				}
				if err := resource.TestCheckResourceAttr(dataSourceName, fmt.Sprintf("clusters.%d.name", i), targetClusterName)(s); err != nil {
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
