package akp

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccKargoDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccKargoDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.akp_kargo_instance.test-instance", "id", "5gjcg0rh8fjemhc0"),
					resource.TestCheckResourceAttr("data.akp_kargo_instance.test-instance", "name", "test-instance"),
					// spec
					resource.TestCheckResourceAttr("data.akp_kargo_instance.test-instance", "kargo.spec.version", "v1.2.2"),
					resource.TestCheckResourceAttr("data.akp_kargo_instance.test-instance", "kargo.spec.kargo_instance_spec.ip_allow_list.#", "0"),
					resource.TestCheckResourceAttr("data.akp_kargo_instance.test-instance", "kargo.spec.kargo_instance_spec.global_credentials_ns.#", "2"),
					resource.TestCheckResourceAttr("data.akp_kargo_instance.test-instance", "kargo.spec.kargo_instance_spec.global_service_account_ns.#", "1"),
					resource.TestCheckResourceAttr("data.akp_kargo_instance.test-instance", "kargo.spec.kargo_instance_spec.default_shard_agent", "kgbgel4pst55klf9"),
					// cm
					resource.TestCheckResourceAttr("data.akp_kargo_instance.test-instance", "kargo_cm.%", "2"),
				),
			},
		},
	})
}

func TestAccKargoInstanceDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: providerConfig + testAccKargoInstanceDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.akp_kargo.test", "id", "test-kargo"),
					resource.TestCheckResourceAttr("data.akp_kargo.test", "name", "test-kargo"),
					resource.TestCheckResourceAttr("data.akp_kargo.test", "workspace", "test-workspace"),

					// Test Kargo Resources
					resource.TestCheckResourceAttr("data.akp_kargo.test", "kargo_resources.#", "3"),
					// Test Project resource
					resource.TestCheckResourceAttr("data.akp_kargo.test", "kargo_resources.0", `{
						"apiVersion": "kargo.akuity.io/v1alpha1",
						"kind": "Project",
						"metadata": {
							"name": "test-project",
							"namespace": "kargo-system"
						},
						"spec": {
							"description": "Test project for Kargo"
						}
					}`),
					// Test Warehouse resource
					resource.TestCheckResourceAttr("data.akp_kargo.test", "kargo_resources.1", `{
						"apiVersion": "kargo.akuity.io/v1alpha1",
						"kind": "Warehouse",
						"metadata": {
							"name": "test-warehouse",
							"namespace": "kargo-system"
						},
						"spec": {
							"freight": {
								"repos": [
									{
										"repo": "https://github.com/akuity/kargo-example-apps",
										"branch": "main"
									}
								]
							}
						}
					}`),
					// Test Stage resource
					resource.TestCheckResourceAttr("data.akp_kargo.test", "kargo_resources.2", `{
						"apiVersion": "kargo.akuity.io/v1alpha1",
						"kind": "Stage",
						"metadata": {
							"name": "test-stage",
							"namespace": "kargo-system"
						},
						"spec": {
							"subscribes": {
								"warehouse": "test-warehouse"
							},
							"promotionMechanisms": {
								"gitRepoUpdates": [
									{
										"repoURL": "https://github.com/akuity/kargo-example-apps",
										"readBranch": "main",
										"writeBranch": "staging"
									}
								]
							}
						}
					}`),
				),
			},
		},
	})
}

const testAccKargoDataSourceConfig = `
data "akp_kargo_instance" "test-instance" {
  name = "test-instance"
}
`

const testAccKargoInstanceDataSourceConfig = `
data "akp_kargo" "test" {
	name = "test-kargo"
}
`
