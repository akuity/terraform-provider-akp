//go:build !unit

package akp

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccInstanceDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: providerConfig + testAccInstanceDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.akp_instance.test", "id", "6pzhawvy4echbd8x"),
					resource.TestCheckResourceAttr("data.akp_instance.test", "name", "test-cluster"),

					// argocd
					resource.TestCheckResourceAttr("data.akp_instance.test", "argocd.spec.description", "This is used by the terraform provider to test managing clusters."),
					resource.TestCheckResourceAttr("data.akp_instance.test", "argocd.spec.version", "v2.13.1"),
					// argocd.instance_spec
					resource.TestCheckResourceAttr("data.akp_instance.test", "argocd.spec.instance_spec.subdomain", "6pzhawvy4echbd8x"),
					resource.TestCheckResourceAttr("data.akp_instance.test", "argocd.spec.instance_spec.declarative_management_enabled", "false"),
					resource.TestCheckResourceAttr("data.akp_instance.test", "argocd.spec.instance_spec.image_updater_enabled", "false"),
					resource.TestCheckResourceAttr("data.akp_instance.test", "argocd.spec.instance_spec.appset_policy.policy", "sync"),
					resource.TestCheckResourceAttr("data.akp_instance.test", "argocd.spec.instance_spec.appset_policy.override_policy", "false"),
					resource.TestCheckResourceAttr("data.akp_instance.test", "argocd.spec.instance_spec.host_aliases.#", "1"),
					resource.TestCheckResourceAttr("data.akp_instance.test", "argocd.spec.instance_spec.host_aliases.0.ip", "1.2.3.4"),
					resource.TestCheckResourceAttr("data.akp_instance.test", "argocd.spec.instance_spec.host_aliases.0.hostnames.#", "2"),
					resource.TestCheckResourceAttr("data.akp_instance.test", "argocd.spec.instance_spec.host_aliases.0.hostnames.0", "test-1"),
					resource.TestCheckResourceAttr("data.akp_instance.test", "argocd.spec.instance_spec.host_aliases.0.hostnames.1", "test-2"),
					resource.TestCheckResourceAttr("data.akp_instance.test", "argocd.spec.instance_spec.appset_plugins.#", "1"),
					resource.TestCheckResourceAttr("data.akp_instance.test", "argocd.spec.instance_spec.appset_plugins.0.name", "plugin-test"),
					resource.TestCheckResourceAttr("data.akp_instance.test", "argocd.spec.instance_spec.appset_plugins.0.token", "random-token"),
					resource.TestCheckResourceAttr("data.akp_instance.test", "argocd.spec.instance_spec.appset_plugins.0.base_url", "http://random-test.xp"),

					// argocd_cm, all fields should be computed.
					resource.TestCheckResourceAttr("data.akp_instance.test", "argocd_cm.%", "0"),

					// Test Argo Resources
					resource.TestCheckResourceAttr("data.akp_instance.test", "argocd_resources.%", "2"),
					resource.TestCheckResourceAttrWith("data.akp_instance.test", "argocd_resources.argoproj.io/v1alpha1/Application/argocd/app-test", func(value string) error {
						if !strings.Contains(value, "argocd-example-apps.git") {
							return fmt.Errorf("expected to contain repoURL")
						}
						return nil
					}),
					resource.TestCheckResourceAttrWith("data.akp_instance.test", "argocd_resources.argoproj.io/v1alpha1/AppProject/argocd/default", func(value string) error {
						if !strings.Contains(value, "sourceRepos") {
							return fmt.Errorf("expected to contain sourceRepos")
						}
						return nil
					}),
				),
			},
		},
	})
}

const testAccInstanceDataSourceConfig = `
data "akp_instance" "test" {
	name = "test-cluster"
}
`
