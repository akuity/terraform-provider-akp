//go:build !unit

package akp

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccClusterDataSource(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("ds-cluster-%s", acctest.RandString(8))
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: providerConfig + getAccClusterDataSourceConfig(getInstanceId(), name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.akp_cluster.test", "instance_id", getInstanceId()),
					resource.TestCheckResourceAttrSet("data.akp_cluster.test", "id"),
					resource.TestCheckResourceAttr("data.akp_cluster.test", "name", name),
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
- name: quay.io/akuity/agent
  newName: test.io/agent
kind: Kustomization
`),
					resource.TestCheckResourceAttr("data.akp_cluster.test", "spec.data.app_replication", "false"),
					resource.TestCheckResourceAttr("data.akp_cluster.test", "spec.data.redis_tunneling", "true"),
				),
			},
		},
	})
}

func getAccClusterDataSourceConfig(instanceId, name string) string {
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

data "akp_cluster" "test" {
  instance_id = akp_cluster.test.instance_id
  name        = akp_cluster.test.name
}
`, instanceId, name)
}
