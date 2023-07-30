//go:build !unit

package akp

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
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
					resource.TestCheckResourceAttrSet("akp_cluster.test", "id"),
					resource.TestCheckResourceAttr("akp_cluster.test", "name", "test"),
					resource.TestCheckResourceAttr("akp_cluster.test", "namespace", "test"),
					resource.TestCheckResourceAttr("akp_cluster.test", "labels.test-label", "true"),
					resource.TestCheckResourceAttr("akp_cluster.test", "annotations.test-annotation", "false"),
					resource.TestCheckResourceAttrSet("akp_cluster.test", "manifests"),
					// spec
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.description", "test one"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.namespace_scoped", "true"),
					// spec.data
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.size", "small"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.auto_upgrade_disabled", "true"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.kustomization", `  apiVersion: kustomize.config.k8s.io/v1beta1
  kind: Kustomization
  resources:
  - test.yaml
`),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.app_replication", "false"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.target_version", "0.4.0"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.redis_tunneling", "false"),
				),
			},
			// Update and Read testing
			{
				Config: providerConfig + testAccClusterResourceConfig("medium", "test two"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.description", "test two"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.size", "medium"),
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
  instance_id = "kgw15g3hg4ist8vl"
  name      = "test"
  namespace = "test"
  labels = {
    test-label = "true"
  }
  annotations = {
    test-annotation = "false"
  }
  spec = {
    namespace_scoped = true
    description      = %q
    data = {
      size                  = %q
      auto_upgrade_disabled = true
      target_version        = "0.4.0"
      kustomization         = <<EOF
  apiVersion: kustomize.config.k8s.io/v1beta1
  kind: Kustomization
  resources:
  - test.yaml
EOF
    }
  }
}
`, description, size)
}
