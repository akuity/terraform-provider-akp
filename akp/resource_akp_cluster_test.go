package akp

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"testing"

	hashitype "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"

	"time"

	"github.com/akuity/api-client-go/pkg/api/gateway/accesscontrol"
	gwoption "github.com/akuity/api-client-go/pkg/api/gateway/option"
	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	kargov1 "github.com/akuity/api-client-go/pkg/api/gen/kargo/v1"
	orgcv1 "github.com/akuity/api-client-go/pkg/api/gen/organization/v1"
	idv1 "github.com/akuity/api-client-go/pkg/api/gen/types/id/v1"
	healthv1 "github.com/akuity/api-client-go/pkg/api/gen/types/status/health/v1"
	httpctx "github.com/akuity/grpc-gateway-client/pkg/http/context"
	"github.com/akuity/terraform-provider-akp/akp/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	instanceId string
	testAkpCli *AkpCli
)

func getInstanceId() string {
	if instanceId == "" {
		if v := os.Getenv("AKUITY_INSTANCE_ID"); v == "" {
			// Create a new instance for testing
			instanceId = createTestInstance()
		} else {
			instanceId = v
		}
	}

	return instanceId
}

func getTestAkpCli() *AkpCli {
	if testAkpCli != nil {
		return testAkpCli
	}

	ctx := context.Background()

	serverUrl := os.Getenv("AKUITY_SERVER_URL")
	if serverUrl == "" {
		serverUrl = "https://akuity.cloud"
	}

	apiKeyID := os.Getenv("AKUITY_API_KEY_ID")
	apiKeySecret := os.Getenv("AKUITY_API_KEY_SECRET")

	if apiKeyID == "" || apiKeySecret == "" {
		panic("API key credentials are required")
	}

	// Create client following the same logic as the provider
	cred := accesscontrol.NewAPIKeyCredential(apiKeyID, apiKeySecret)
	ctx = httpctx.SetAuthorizationHeader(ctx, cred.Scheme(), cred.Credential())
	gwc := gwoption.NewClient(serverUrl, false)
	orgc := orgcv1.NewOrganizationServiceGatewayClient(gwc)

	// Get Organization ID by name
	res, err := orgc.GetOrganization(ctx, &orgcv1.GetOrganizationRequest{
		Id:     orgName,
		IdType: idv1.Type_NAME,
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to get organization: %v", err))
	}

	orgID := res.Organization.Id

	// Create service clients
	argoc := argocdv1.NewArgoCDServiceGatewayClient(gwc)
	kargoc := kargov1.NewKargoServiceGatewayClient(gwc)

	testAkpCli = &AkpCli{
		Cli:      argoc,
		KargoCli: kargoc,
		Cred:     cred,
		OrgId:    orgID,
		OrgCli:   orgc,
	}
	return testAkpCli
}

func createTestInstance() string {
	akpCli := getTestAkpCli()
	ctx := context.Background()
	ctx = httpctx.SetAuthorizationHeader(ctx, akpCli.Cred.Scheme(), akpCli.Cred.Credential())
	instanceName := fmt.Sprintf("test-cluster-provider-%s", acctest.RandString(8))

	createReq := &argocdv1.CreateInstanceRequest{
		OrganizationId: akpCli.OrgId,
		Name:           instanceName,
		Version:        "v3.0.0",
	}
	instance, err := akpCli.Cli.CreateInstance(ctx, createReq)
	if err != nil {
		panic(fmt.Sprintf("Failed to create instance: %v", err))
	}

	getResourceFunc := func(ctx context.Context) (*argocdv1.GetInstanceResponse, error) {
		return akpCli.Cli.GetInstance(ctx, &argocdv1.GetInstanceRequest{
			OrganizationId: akpCli.OrgId,
			Id:             instance.GetInstance().Id,
			IdType:         idv1.Type_ID,
		})
	}

	getStatusFunc := func(resp *argocdv1.GetInstanceResponse) healthv1.StatusCode {
		if resp == nil || resp.Instance == nil {
			return healthv1.StatusCode_STATUS_CODE_UNKNOWN
		}
		return resp.Instance.GetHealthStatus().GetCode()
	}

	err = waitForStatus(
		ctx,
		getResourceFunc,
		getStatusFunc,
		[]healthv1.StatusCode{healthv1.StatusCode_STATUS_CODE_HEALTHY},
		10*time.Second,
		5*time.Minute,
		fmt.Sprintf("Test instance %s", instanceName),
		"health",
	)

	if err != nil {
		panic(fmt.Sprintf("Test instance did not become healthy: %v", err))
	}

	return instance.Instance.Id
}

func cleanupTestInstance() {
	if instanceId == "" || testAkpCli == nil {
		return
	}

	ctx := context.Background()
	ctx = httpctx.SetAuthorizationHeader(ctx, testAkpCli.Cred.Scheme(), testAkpCli.Cred.Credential())

	// Delete the instance
	_, _ = testAkpCli.Cli.DeleteInstance(ctx, &argocdv1.DeleteInstanceRequest{
		Id:             instanceId,
		OrganizationId: testAkpCli.OrgId,
	})
}

func TestAccClusterResource(t *testing.T) {
	name := fmt.Sprintf("cluster-%s", acctest.RandString(10))
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + testAccClusterResourceConfig("small", name, "test one", getInstanceId()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("akp_cluster.test", "id"),
					resource.TestCheckResourceAttr("akp_cluster.test", "namespace", "test"),
					resource.TestCheckResourceAttr("akp_cluster.test", "labels.test-label", "true"),
					resource.TestCheckResourceAttr("akp_cluster.test", "annotations.test-annotation", "false"),
					// spec
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.description", "test one"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.namespace_scoped", "true"),
					// spec.data
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.size", "small"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.auto_upgrade_disabled", "true"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.kustomization", `  apiVersion: kustomize.config.k8s.io/v1beta1
  kind: Kustomization
  patches:
    - patch: |-
        apiVersion: apps/v1
        kind: Deployment
        metadata:
          name: argocd-repo-server
        spec:
          template:
            spec:
              containers:
              - name: argocd-repo-server
                resources:
                  limits:
                    memory: 2Gi
                  requests:
                    cpu: 750m
                    memory: 1Gi
      target:
        kind: Deployment
        name: argocd-repo-server
`),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.app_replication", "false"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.redis_tunneling", "false"),
					resource.TestCheckResourceAttr("akp_cluster.test", "remove_agent_resources_on_destroy", "true"),
				),
			},
			// Update and Read testing
			{
				Config: providerConfig + testAccClusterResourceConfig("medium", name, "test two", getInstanceId()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.description", "test two"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.size", "medium"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccClusterResourceIPv6(t *testing.T) {
	name := fmt.Sprintf("cluster-ipv6-%s", acctest.RandString(10))
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccClusterResourceConfigIPv6(name, getInstanceId()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("akp_cluster.test", "id"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.compatibility.ipv6_only", "true"),
				),
			},
		},
	})
}

func TestAccClusterResourceArgoCDNotifications(t *testing.T) {
	name := fmt.Sprintf("cluster-notifications-%s", acctest.RandString(10))
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccClusterResourceConfigNotifications(name, getInstanceId()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("akp_cluster.test", "id"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.argocd_notifications_settings.in_cluster_settings", "true"),
				),
			},
		},
	})
}

func TestAccClusterResourceCustomAgentSize(t *testing.T) {
	name := fmt.Sprintf("cluster-custom-size-%s", acctest.RandString(10))
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccClusterResourceConfigCustomAgentSize(name, getInstanceId()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("akp_cluster.test", "id"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.custom_agent_size_config.application_controller.memory", "2Gi"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.custom_agent_size_config.application_controller.cpu", "1000m"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.custom_agent_size_config.repo_server.memory", "4Gi"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.custom_agent_size_config.repo_server.cpu", "2000m"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.custom_agent_size_config.repo_server.replicas", "3"),
				),
			},
		},
	})
}

func TestAccClusterResourceManagedCluster(t *testing.T) {
	name := fmt.Sprintf("cluster-managed-%s", acctest.RandString(10))
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccClusterResourceConfigManagedCluster(name, getInstanceId()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("akp_cluster.test", "id"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.managed_cluster_config.secret_name", "test-secret"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.managed_cluster_config.secret_key", "kubeconfig"),
				),
			},
		},
	})
}

func TestAccClusterResourceFeatures(t *testing.T) {
	name := fmt.Sprintf("cluster-features-%s", acctest.RandString(10))
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccClusterResourceConfigFeatures(name, getInstanceId()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("akp_cluster.test", "id"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.multi_cluster_k8s_dashboard_enabled", "true"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.eks_addon_enabled", "true"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.datadog_annotations_enabled", "true"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.redis_tunneling", "true"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.app_replication", "true"),
				),
			},
		},
	})
}

func TestAccClusterResourceReapplyManifests(t *testing.T) {
	name := fmt.Sprintf("cluster-reapply-%s", acctest.RandString(10))
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccClusterResourceConfigReapplyManifests(name, "test initial", getInstanceId()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("akp_cluster.test", "id"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.description", "test initial"),
					resource.TestCheckResourceAttr("akp_cluster.test", "reapply_manifests_on_update", "true"),
				),
			},
			{
				Config: providerConfig + testAccClusterResourceConfigReapplyManifests(name, "test updated", getInstanceId()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.description", "test updated"),
					resource.TestCheckResourceAttr("akp_cluster.test", "reapply_manifests_on_update", "true"),
				),
			},
		},
	})
}

func TestAccClusterResourceNamespaceScoped(t *testing.T) {
	name := fmt.Sprintf("cluster-ns-scoped-%s", acctest.RandString(10))
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccClusterResourceConfigNamespaceScoped(name, true, getInstanceId()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("akp_cluster.test", "id"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.namespace_scoped", "true"),
				),
			},
			{
				Config: providerConfig + testAccClusterResourceConfigNamespaceScoped(name, false, getInstanceId()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.namespace_scoped", "false"),
				),
			},
		},
	})
}

func TestAccClusterResourceProject(t *testing.T) {
	name := fmt.Sprintf("cluster-project-%s", acctest.RandString(10))
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccClusterResourceConfigProject(name, "test-project", getInstanceId()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("akp_cluster.test", "id"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.project", "test-project"),
				),
			},
		},
	})
}

func testAccClusterResourceConfig(size, name, description, instanceId string) string {
	return fmt.Sprintf(`
resource "akp_cluster" "test" {
  instance_id = %q
  name      = %q
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
      kustomization         = <<EOF
  apiVersion: kustomize.config.k8s.io/v1beta1
  kind: Kustomization
  patches:
    - patch: |-
        apiVersion: apps/v1
        kind: Deployment
        metadata:
          name: argocd-repo-server
        spec:
          template:
            spec:
              containers:
              - name: argocd-repo-server
                resources:
                  limits:
                    memory: 2Gi
                  requests:
                    cpu: 750m
                    memory: 1Gi
      target:
        kind: Deployment
        name: argocd-repo-server
EOF
    }
  }
}
`, instanceId, name, description, size)
}

func testAccClusterResourceConfigIPv6(name, instanceId string) string {
	return fmt.Sprintf(`
resource "akp_cluster" "test" {
  instance_id = %q
  name      = %q
  namespace = "test"
  spec = {
    namespace_scoped = true
    description      = "IPv6 test cluster"
    data = {
      size = "small"
      compatibility = {
        ipv6_only = true
      }
    }
  }
  remove_agent_resources_on_destroy = true
}
`, instanceId, name)
}

func testAccClusterResourceConfigNotifications(name, instanceId string) string {
	return fmt.Sprintf(`
resource "akp_cluster" "test" {
  instance_id = %q
  name      = %q
  namespace = "test"
  spec = {
    namespace_scoped = true
    description      = "ArgoCD notifications test cluster"
    data = {
      size = "small"
      argocd_notifications_settings = {
        in_cluster_settings = true
      }
    }
  }
  remove_agent_resources_on_destroy = true
}
`, instanceId, name)
}

func testAccClusterResourceConfigCustomAgentSize(name, instanceId string) string {
	return fmt.Sprintf(`
resource "akp_cluster" "test" {
  instance_id = %q
  name      = %q
  namespace = "test"
  spec = {
    namespace_scoped = true
    description      = "Custom agent size test cluster"
    data = {
      size = "custom"
      custom_agent_size_config = {
        application_controller = {
          memory = "2Gi"
          cpu    = "1000m"
        }
        repo_server = {
          memory   = "4Gi"
          cpu      = "2000m"
          replicas = 3
        }
      }
    }
  }
  remove_agent_resources_on_destroy = true
}
`, instanceId, name)
}

func testAccClusterResourceConfigManagedCluster(name, instanceId string) string {
	return fmt.Sprintf(`
resource "akp_cluster" "test" {
  instance_id = %q
  name      = %q
  namespace = "test"
  spec = {
    namespace_scoped = true
    description      = "Managed cluster test"
    data = {
      size = "small"
      managed_cluster_config = {
        secret_name = "test-secret"
        secret_key  = "kubeconfig"
      }
    }
  }
  remove_agent_resources_on_destroy = true
}
`, instanceId, name)
}

func testAccClusterResourceConfigFeatures(name, instanceId string) string {
	return fmt.Sprintf(`
resource "akp_cluster" "test" {
  instance_id = %q
  name      = %q
  namespace = "test"
  spec = {
    namespace_scoped = true
    description      = "Feature flags test cluster"
    data = {
      size                               = "small"
      multi_cluster_k8s_dashboard_enabled = true
      eks_addon_enabled                  = true
      datadog_annotations_enabled        = true
      redis_tunneling                    = true
      app_replication                    = true
    }
  }
  remove_agent_resources_on_destroy = true
}
`, instanceId, name)
}

func testAccClusterResourceConfigReapplyManifests(name, description, instanceId string) string {
	return fmt.Sprintf(`
resource "akp_cluster" "test" {
  instance_id = %q
  name      = %q
  namespace = "test"
  spec = {
    namespace_scoped = true
    description      = %q
    data = {
      size = "small"
    }
  }
  reapply_manifests_on_update       = true
  remove_agent_resources_on_destroy = true
}
`, instanceId, name, description)
}

func testAccClusterResourceConfigNamespaceScoped(name string, namespaceScoped bool, instanceId string) string {
	return fmt.Sprintf(`
resource "akp_cluster" "test" {
  instance_id = %q
  name      = %q
  namespace = "test"
  spec = {
    namespace_scoped = %t
    description      = "Namespace scoped test cluster"
    data = {
      size = "small"
    }
  }
  remove_agent_resources_on_destroy = true
}
`, instanceId, name, namespaceScoped)
}

func testAccClusterResourceConfigProject(name, project, instanceId string) string {
	return fmt.Sprintf(`
resource "akp_cluster" "test" {
  instance_id = %q
  name      = %q
  namespace = "test"
  spec = {
    namespace_scoped = true
    description      = "Project assignment test cluster"
    data = {
      size    = "small"
      project = %q
    }
  }
  remove_agent_resources_on_destroy = true
}
`, instanceId, name, project)
}

func TestAccClusterResourceKubeconfig(t *testing.T) {
	name := fmt.Sprintf("cluster-kubeconfig-%s", acctest.RandString(10))
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      providerConfig + testAccClusterResourceConfigKubeconfig(name, getInstanceId()),
				ExpectError: regexp.MustCompile("unable to apply manifests"),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Verify that cluster was automatically cleaned up and no state was committed
					testCheckClusterCleanedUp(name, getInstanceId()),
				),
			},
		},
	})
}

func testCheckClusterCleanedUp(clusterName, instanceId string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Check that the cluster was automatically cleaned up by the provider
		akpCli := getTestAkpCli()
		ctx := context.Background()
		ctx = httpctx.SetAuthorizationHeader(ctx, akpCli.Cred.Scheme(), akpCli.Cred.Credential())

		clusterReq := &argocdv1.GetInstanceClusterRequest{
			OrganizationId: akpCli.OrgId,
			InstanceId:     instanceId,
			Id:             clusterName,
			IdType:         idv1.Type_NAME,
		}

		_, err := akpCli.Cli.GetInstanceCluster(ctx, clusterReq)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				// This is what we expect - the cluster should not exist
				// Check that no resource exists in Terraform state
				for name := range s.RootModule().Resources {
					if name == "akp_cluster.test" {
						return fmt.Errorf("cluster resource should not exist in Terraform state")
					}
				}
				return nil
			}
			return fmt.Errorf("unexpected error when checking cluster: %v", err)
		}

		return fmt.Errorf("cluster %s should have been automatically cleaned up but still exists in API", clusterName)
	}
}

func testAccClusterResourceConfigKubeconfig(name, instanceId string) string {
	return fmt.Sprintf(`
resource "akp_cluster" "test" {
  instance_id = %q
  name      = %q
  namespace = "test"
  spec = {
    namespace_scoped = true
    description      = "Kubeconfig test cluster"
    data = {
      size = "small"
    }
  }
  kube_config = {
    host     = "https://test-cluster.example.com"
    insecure = true
    token    = "test-token"
  }
  remove_agent_resources_on_destroy = true
}
`, instanceId, name)
}

func TestAkpClusterResource_applyInstance(t *testing.T) {
	type args struct {
		plan             *types.Cluster
		apiReq           *argocdv1.ApplyInstanceRequest
		isCreate         bool
		applyInstance    func(context.Context, *argocdv1.ApplyInstanceRequest) (*argocdv1.ApplyInstanceResponse, error)
		upsertKubeConfig func(ctx context.Context, plan *types.Cluster) error
	}
	tests := []struct {
		name  string
		args  args
		want  *types.Cluster
		error error
	}{
		{
			name: "happy path, no kubeconfig",
			args: args{
				plan:     &types.Cluster{},
				isCreate: true,
				applyInstance: func(ctx context.Context, request *argocdv1.ApplyInstanceRequest) (*argocdv1.ApplyInstanceResponse, error) {
					return &argocdv1.ApplyInstanceResponse{}, nil
				},
				upsertKubeConfig: func(ctx context.Context, plan *types.Cluster) error {
					return errors.New("this should not be called")
				},
			},
			want:  &types.Cluster{},
			error: nil,
		},
		{
			name: "error path, no kubeconfig",
			args: args{
				plan:     &types.Cluster{},
				isCreate: true,
				applyInstance: func(ctx context.Context, request *argocdv1.ApplyInstanceRequest) (*argocdv1.ApplyInstanceResponse, error) {
					return &argocdv1.ApplyInstanceResponse{}, errors.New("some error")
				},
				upsertKubeConfig: func(ctx context.Context, plan *types.Cluster) error {
					return errors.New("this should not be called")
				},
			},
			want:  nil,
			error: fmt.Errorf("unable to create Argo CD instance: some error"),
		},
		{
			name: "happy path, with kubeconfig",
			args: args{
				plan: &types.Cluster{
					Kubeconfig: &types.Kubeconfig{
						Host: hashitype.StringValue("some-host"),
					},
				},
				applyInstance: func(ctx context.Context, request *argocdv1.ApplyInstanceRequest) (*argocdv1.ApplyInstanceResponse, error) {
					return &argocdv1.ApplyInstanceResponse{}, nil
				},
				isCreate: true,
				upsertKubeConfig: func(ctx context.Context, plan *types.Cluster) error {
					assert.Equal(t, &types.Cluster{
						Kubeconfig: &types.Kubeconfig{
							Host: hashitype.StringValue("some-host"),
						},
					}, plan)
					return nil
				},
			},
			want: &types.Cluster{
				Kubeconfig: &types.Kubeconfig{
					Host: hashitype.StringValue("some-host"),
				},
			},
			error: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &AkpClusterResource{}
			ctx := context.Background()
			got, err := r.applyInstance(ctx, tt.args.plan, tt.args.apiReq, tt.args.isCreate, tt.args.applyInstance, tt.args.upsertKubeConfig)
			assert.Equal(t, tt.error, err)
			assert.Equalf(t, tt.want, got, "applyInstance(%v, %v, %v)", tt.args.plan, tt.args.apiReq, tt.args.isCreate)
		})
	}
}

func TestAkpClusterResource_reApplyManifests(t *testing.T) {
	type args struct {
		plan             *types.Cluster
		apiReq           *argocdv1.ApplyInstanceRequest
		applyInstance    func(context.Context, *argocdv1.ApplyInstanceRequest) (*argocdv1.ApplyInstanceResponse, error)
		upsertKubeConfig func(ctx context.Context, plan *types.Cluster) error
	}
	tests := []struct {
		name  string
		args  args
		want  *types.Cluster
		error error
	}{
		{
			name: "error path, with kubeconfig",
			args: args{
				plan: &types.Cluster{
					Kubeconfig: &types.Kubeconfig{
						Host: hashitype.StringValue("some-host"),
					},
					ReapplyManifestsOnUpdate: hashitype.BoolValue(true),
				},
				applyInstance: func(ctx context.Context, request *argocdv1.ApplyInstanceRequest) (*argocdv1.ApplyInstanceResponse, error) {
					return &argocdv1.ApplyInstanceResponse{}, nil
				},
				upsertKubeConfig: func(ctx context.Context, plan *types.Cluster) error {
					return errors.New("some kube apply error")
				},
			},
			want:  &types.Cluster{},
			error: fmt.Errorf("unable to apply manifests: some kube apply error"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &AkpClusterResource{}
			ctx := context.Background()
			_, err := r.applyInstance(ctx, tt.args.plan, tt.args.apiReq, false, tt.args.applyInstance, tt.args.upsertKubeConfig)
			assert.Equal(t, tt.error, err)
		})
	}
}
