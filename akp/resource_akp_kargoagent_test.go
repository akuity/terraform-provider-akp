//go:build !unit

package akp

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"

	httpctx "github.com/akuity/grpc-gateway-client/pkg/http/context"
	kargov1 "github.com/akuity/api-client-go/pkg/api/gen/kargo/v1"
	idv1 "github.com/akuity/api-client-go/pkg/api/gen/types/id/v1"
	healthv1 "github.com/akuity/api-client-go/pkg/api/gen/types/status/health/v1"
)

var (
	kargoInstanceId    string
	kargoInstanceName  string
	kargoVersion       string
	kargoInstanceOwned bool
	kargoInstanceOnce  sync.Once
	kargoInstanceMu    sync.RWMutex
)

func getKargoInstanceId() string {
	kargoInstanceMu.RLock()
	id := kargoInstanceId
	kargoInstanceMu.RUnlock()

	if id == "" {
		kargoInstanceMu.Lock()
		defer kargoInstanceMu.Unlock()
		// Double-check after acquiring write lock
		if kargoInstanceId == "" {
			if v := os.Getenv("AKUITY_KARGO_INSTANCE_ID"); v == "" {
				// Create a new Kargo instance for testing
				kargoInstanceOwned = true
				kargoInstanceId = createTestKargoInstance()
			} else {
				fmt.Printf("Reusing Kargo test instance %s from AKUITY_KARGO_INSTANCE_ID\n", v)
				kargoInstanceOwned = false
				kargoInstanceId = v
				fetchKargoInstanceDetails(v)
			}
		}
		return kargoInstanceId
	}

	return id
}

func getKargoVersion() string {
	getKargoInstanceId()
	kargoInstanceMu.RLock()
	defer kargoInstanceMu.RUnlock()
	return kargoVersion
}

func getKargoInstanceName() string {
	getKargoInstanceId()
	kargoInstanceMu.RLock()
	defer kargoInstanceMu.RUnlock()
	return kargoInstanceName
}

func createTestKargoInstance() string {
	// Note: Caller (getKargoInstanceId) must hold kargoInstanceMu.Lock() when calling this function
	kargoInstanceOnce.Do(func() {
		if os.Getenv("TF_ACC") != "1" {
			return
		}

		akpCli := getTestAkpCli()
		ctx := context.Background()
		ctx = httpctx.SetAuthorizationHeader(ctx, akpCli.Cred.Scheme(), akpCli.Cred.Credential())
		kargoInstanceName = fmt.Sprintf("test-kargo-instance-%s", acctest.RandString(8))

		// Get default workspace
		workspace, err := getWorkspace(ctx, akpCli.OrgCli, akpCli.OrgId, "")
		if err != nil {
			panic(fmt.Sprintf("Failed to get default workspace: %v", err))
		}

		// Create minimal Kargo instance with required version
		kargoVersion = os.Getenv("AKUITY_KARGO_VERSION")
		if kargoVersion == "" {
			panic("AKUITY_KARGO_VERSION not set! This needs to be set to a valid Kargo version!")
		}

		kargoStruct, err := structpb.NewStruct(map[string]any{
			"metadata": map[string]any{
				"name": kargoInstanceName,
			},
			"spec": map[string]any{
				"version":     kargoVersion,
				"description": "Test Kargo instance for terraform provider tests",
				"kargoInstanceSpec": map[string]any{
					"globalCredentialsNs":    []any{"credentials-ns-1", "credentials-ns-2"},
					"globalServiceAccountNs": []any{"sa-ns-1"},
				},
			},
		})
		if err != nil {
			panic(fmt.Sprintf("Failed to create Kargo struct: %v", err))
		}

		// First create the Kargo instance WITHOUT Projects (can't add Projects until instance is ready)
		applyReq := &kargov1.ApplyKargoInstanceRequest{
			OrganizationId: akpCli.OrgId,
			Id:             kargoInstanceName,
			IdType:         idv1.Type_NAME,
			WorkspaceId:    workspace.Id,
			Kargo:          kargoStruct,
		}
		_, err = retryWithBackoff(ctx, func(ctx context.Context) (*kargov1.ApplyKargoInstanceResponse, error) {
			return akpCli.KargoCli.ApplyKargoInstance(ctx, applyReq)
		}, "create Kargo instance")
		if err != nil {
			panic(fmt.Sprintf("Failed to create Kargo instance: %v", err))
		}

		instanceResponse, err := retryWithBackoff(ctx, func(ctx context.Context) (*kargov1.GetKargoInstanceResponse, error) {
			return akpCli.KargoCli.GetKargoInstance(ctx, &kargov1.GetKargoInstanceRequest{
				OrganizationId: akpCli.OrgId,
				Name:           kargoInstanceName,
			})
		}, "get Kargo instance")
		if err != nil {
			panic(fmt.Sprintf("Failed to get created Kargo instance: %v", err))
		}

		getResourceFunc := func(ctx context.Context) (*kargov1.GetKargoInstanceResponse, error) {
			return akpCli.KargoCli.GetKargoInstance(ctx, &kargov1.GetKargoInstanceRequest{
				OrganizationId: akpCli.OrgId,
				Name:           kargoInstanceName,
			})
		}

		getStatusFunc := func(resp *kargov1.GetKargoInstanceResponse) healthv1.StatusCode {
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
			10*time.Minute,
			fmt.Sprintf("Test Kargo instance %s", kargoInstanceName),
			"health",
		)
		if err != nil {
			panic(fmt.Sprintf("Test Kargo instance did not become healthy: %v", err))
		}

		kargoInstanceId = instanceResponse.Instance.Id

		// Now that the instance is healthy, add the Project
		projectStruct, err := structpb.NewStruct(map[string]any{
			"apiVersion": "kargo.akuity.io/v1alpha1",
			"kind":       "Project",
			"metadata": map[string]any{
				"name": "kargo-demo",
			},
		})
		if err != nil {
			panic(fmt.Sprintf("Failed to create Project struct: %v", err))
		}

		applyProjectReq := &kargov1.ApplyKargoInstanceRequest{
			OrganizationId: akpCli.OrgId,
			Id:             kargoInstanceId,
			IdType:         idv1.Type_ID,
			WorkspaceId:    workspace.Id,
			Projects:       []*structpb.Struct{projectStruct},
		}
		_, err = retryWithBackoff(ctx, func(ctx context.Context) (*kargov1.ApplyKargoInstanceResponse, error) {
			return akpCli.KargoCli.ApplyKargoInstance(ctx, applyProjectReq)
		}, "add Project to Kargo instance")
		if err != nil {
			panic(fmt.Sprintf("Failed to add Project to Kargo instance: %v", err))
		}

		// Now add the Warehouse resource
		warehouseStruct, err := structpb.NewStruct(map[string]any{
			"apiVersion": "kargo.akuity.io/v1alpha1",
			"kind":       "Warehouse",
			"metadata": map[string]any{
				"name":      "kargo-demo",
				"namespace": "kargo-demo",
			},
			"spec": map[string]any{
				"subscriptions": []any{
					map[string]any{
						"image": map[string]any{
							"repoURL":          "public.ecr.aws/nginx/nginx",
							"semverConstraint": "^1.28.0",
							"discoveryLimit":   5,
						},
					},
				},
			},
		})
		if err != nil {
			panic(fmt.Sprintf("Failed to create Warehouse struct: %v", err))
		}

		applyWarehouseReq := &kargov1.ApplyKargoInstanceRequest{
			OrganizationId: akpCli.OrgId,
			Id:             kargoInstanceId,
			IdType:         idv1.Type_ID,
			WorkspaceId:    workspace.Id,
			Projects:       []*structpb.Struct{projectStruct},
			Warehouses:     []*structpb.Struct{warehouseStruct},
		}
		_, err = retryWithBackoff(ctx, func(ctx context.Context) (*kargov1.ApplyKargoInstanceResponse, error) {
			return akpCli.KargoCli.ApplyKargoInstance(ctx, applyWarehouseReq)
		}, "add Warehouse to Kargo instance")
		if err != nil {
			panic(fmt.Sprintf("Failed to add Warehouse to Kargo instance: %v", err))
		}
	})

	return kargoInstanceId
}

func fetchKargoInstanceDetails(instanceId string) {
	if os.Getenv("TF_ACC") != "1" {
		return
	}

	akpCli := getTestAkpCli()
	ctx := context.Background()
	ctx = httpctx.SetAuthorizationHeader(ctx, akpCli.Cred.Scheme(), akpCli.Cred.Credential())

	listResp, err := akpCli.KargoCli.ListKargoInstances(ctx, &kargov1.ListKargoInstancesRequest{
		OrganizationId: akpCli.OrgId,
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to list Kargo instances: %v", err))
	}

	for _, instance := range listResp.GetInstances() {
		if instance.GetId() == instanceId {
			kargoInstanceName = instance.GetName()
			kargoVersion = instance.GetVersion()
			return
		}
	}

	panic(fmt.Sprintf("Kargo instance with ID %q not found", instanceId))
}

func cleanupTestKargoInstance() {
	kargoInstanceMu.RLock()
	id := kargoInstanceId
	owned := kargoInstanceOwned
	kargoInstanceMu.RUnlock()

	testAkpCliMu.Lock()
	akpCli := testAkpCli
	testAkpCliMu.Unlock()

	if id == "" || akpCli == nil || !owned {
		return
	}

	ctx := context.Background()
	ctx = httpctx.SetAuthorizationHeader(ctx, akpCli.Cred.Scheme(), akpCli.Cred.Credential())

	// Delete the Kargo instance
	_, _ = akpCli.KargoCli.DeleteInstance(ctx, &kargov1.DeleteInstanceRequest{
		Id:             id,
		OrganizationId: akpCli.OrgId,
	})
}

func clearDefaultShardAgent(instanceId string) error {
	if instanceId == "" {
		return nil
	}

	akpCli := getTestAkpCli()
	ctx := context.Background()
	ctx = httpctx.SetAuthorizationHeader(ctx, akpCli.Cred.Scheme(), akpCli.Cred.Credential())

	// Get the Kargo instance by name
	instanceResp, err := akpCli.KargoCli.GetKargoInstance(ctx, &kargov1.GetKargoInstanceRequest{
		OrganizationId: akpCli.OrgId,
		Name:           kargoInstanceName,
	})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil // Instance doesn't exist, nothing to clear
		}
		return fmt.Errorf("failed to get Kargo instance: %v", err)
	}

	// Export the Kargo instance to get the full spec
	exportResp, err := akpCli.KargoCli.ExportKargoInstance(ctx, &kargov1.ExportKargoInstanceRequest{
		OrganizationId: akpCli.OrgId,
		Id:             instanceResp.Instance.Id,
		WorkspaceId:    instanceResp.Instance.WorkspaceId,
	})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil
		}
		return fmt.Errorf("failed to export Kargo instance: %v", err)
	}

	// Check if there's a default shard agent set
	if exportResp.Kargo == nil {
		return nil
	}

	specMap := exportResp.Kargo.AsMap()
	if spec, ok := specMap["spec"].(map[string]any); ok {
		if kargoInstanceSpec, ok := spec["kargoInstanceSpec"].(map[string]any); ok {
			if defaultShardAgent, ok := kargoInstanceSpec["defaultShardAgent"].(string); ok && defaultShardAgent != "" {
				// Clear the default shard agent
				kargoInstanceSpec["defaultShardAgent"] = ""
				updatedKargo, err := structpb.NewStruct(specMap)
				if err != nil {
					return fmt.Errorf("failed to create updated Kargo struct: %v", err)
				}

				updateReq := &kargov1.ApplyKargoInstanceRequest{
					OrganizationId: akpCli.OrgId,
					Id:             kargoInstanceName,
					IdType:         idv1.Type_NAME,
					Kargo:          updatedKargo,
				}
				_, err = akpCli.KargoCli.ApplyKargoInstance(ctx, updateReq)
				if err != nil {
					return fmt.Errorf("failed to clear default shard agent: %v", err)
				}
			}
		}
	}

	return nil
}

func runKargoAgentResource(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("kargoagent-%s", acctest.RandString(10))

	// Ensure we clear the default shard agent before tests clean up
	t.Cleanup(func() {
		_ = clearDefaultShardAgent(getKargoInstanceId())
	})

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: providerConfig + testAccKargoAgentResourceConfig("small", name, "test kargo agent", getKargoInstanceId()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("akp_kargo_agent.test", "id"),
					resource.TestCheckResourceAttr("akp_kargo_agent.test", "namespace", "test"),
					resource.TestCheckResourceAttr("akp_kargo_agent.test", "labels.test-label", "true"),
					resource.TestCheckResourceAttr("akp_kargo_agent.test", "annotations.test-annotation", "false"),
					// spec
					resource.TestCheckResourceAttr("akp_kargo_agent.test", "spec.description", "test kargo agent"),
					// spec.data
					resource.TestCheckResourceAttr("akp_kargo_agent.test", "spec.data.size", "small"),
					resource.TestCheckResourceAttr("akp_kargo_agent.test", "spec.data.auto_upgrade_disabled", "true"),
					resource.TestCheckResourceAttr("akp_kargo_agent.test", "spec.data.kustomization", `  apiVersion: kustomize.config.k8s.io/v1beta1
  kind: Kustomization
  patches:
    - patch: |-
        apiVersion: apps/v1
        kind: Deployment
        metadata:
          name: kargo-agent
        spec:
          template:
            spec:
              containers:
              - name: kargo-agent
                resources:
                  limits:
                    memory: 2Gi
                  requests:
                    cpu: 750m
                    memory: 1Gi
      target:
        kind: Deployment
        name: kargo-agent
`),
					resource.TestCheckResourceAttr("akp_kargo_agent.test", "spec.data.remote_argocd", getInstanceId()),
					resource.TestCheckResourceAttr("akp_kargo_agent.test", "spec.data.akuity_managed", "false"),
					resource.TestCheckResourceAttr("akp_kargo_agent.test", "remove_agent_resources_on_destroy", "true"),
					// --- Data Sources ---
					resource.TestCheckResourceAttr("data.akp_kargo_agent.test", "name", name),
					resource.TestCheckResourceAttrSet("data.akp_kargo_agent.test", "id"),
					resource.TestCheckResourceAttr("data.akp_kargo_agent.test", "spec.data.size", "small"),
					resource.TestCheckResourceAttr("data.akp_kargo_agent.test", "spec.data.auto_upgrade_disabled", "true"),
					resource.TestCheckResourceAttr("data.akp_kargo_agent.test", "remove_agent_resources_on_destroy", "true"),
					resource.TestCheckResourceAttr("data.akp_kargo_agent.test", "reapply_manifests_on_update", "false"),
					// kargo agents list data source
					resource.TestCheckResourceAttrSet("data.akp_kargo_agents.test", "agents.#"),
				),
			},
			// Update and Read testing
			{
				Config: providerConfig + testAccKargoAgentResourceConfig("medium", name, "updated kargo agent", getKargoInstanceId()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_kargo_agent.test", "spec.description", "updated kargo agent"),
					resource.TestCheckResourceAttr("akp_kargo_agent.test", "spec.data.size", "medium"),
					resource.TestCheckResourceAttr("akp_kargo_agent.test", "spec.data.remote_argocd", getInstanceId()),
					resource.TestCheckResourceAttr("akp_kargo_agent.test", "spec.data.akuity_managed", "false"),
				),
			},
			testAccKargoAgentImportStateStep(getKargoInstanceId(), name, testAccKargoAgentKustomizationImportStateVerifyIgnore...),
			// Delete testing automatically occurs in TestCase
		},
	})
}

func runKargoAgentResourceRemoteArgoCD(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("kargoagent-remote-%s", acctest.RandString(10))
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccKargoAgentResourceConfigRemoteArgoCD(name, getKargoInstanceId(), getInstanceId()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("akp_kargo_agent.test", "id"),
					resource.TestCheckResourceAttr("akp_kargo_agent.test", "spec.data.remote_argocd", getInstanceId()),
					resource.TestCheckResourceAttr("akp_kargo_agent.test", "spec.data.akuity_managed", "false"),
				),
			},
			testAccKargoAgentImportStateStep(getKargoInstanceId(), name, testAccKargoAgentCommonImportStateVerifyIgnore...),
		},
	})
}

func runKargoAgentResourceCustomNamespace(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("kargoagent-ns-%s", acctest.RandString(10))
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      providerConfig + testAccKargoAgentResourceConfigCustomNamespace(name, getKargoInstanceId()),
				ExpectError: regexp.MustCompile(`Invalid argocd_namespace`),
			},
		},
	})
}

func runKargoAgentResourceReapplyManifests(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("kargoagent-reapply-%s", acctest.RandString(10))
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccKargoAgentResourceConfigReapplyManifests(name, "test initial", getKargoInstanceId()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("akp_kargo_agent.test", "id"),
					resource.TestCheckResourceAttr("akp_kargo_agent.test", "spec.description", "test initial"),
					resource.TestCheckResourceAttr("akp_kargo_agent.test", "reapply_manifests_on_update", "true"),
				),
			},
			{
				Config: providerConfig + testAccKargoAgentResourceConfigReapplyManifests(name, "test updated", getKargoInstanceId()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_kargo_agent.test", "spec.description", "test updated"),
					resource.TestCheckResourceAttr("akp_kargo_agent.test", "reapply_manifests_on_update", "true"),
				),
			},
			testAccKargoAgentImportStateStep(getKargoInstanceId(), name, testAccKargoAgentReapplyImportStateVerifyIgnore...),
		},
	})
}

func runKargoAgentResourceTargetVersion(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("kargoagent-version-%s", acctest.RandString(10))
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccKargoAgentResourceConfigTargetVersion(name, "", getKargoInstanceId()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("akp_kargo_agent.test", "id"),
					resource.TestCheckResourceAttr("akp_kargo_agent.test", "spec.data.auto_upgrade_disabled", "false"),
					resource.TestCheckResourceAttr("akp_kargo_agent.test", "spec.data.remote_argocd", getInstanceId()),
					resource.TestCheckResourceAttr("akp_kargo_agent.test", "spec.data.akuity_managed", "false"),
				),
			},
			{
				Config: providerConfig + testAccKargoAgentResourceConfigTargetVersion(name, "", getKargoInstanceId()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_kargo_agent.test", "spec.data.remote_argocd", getInstanceId()),
					resource.TestCheckResourceAttr("akp_kargo_agent.test", "spec.data.akuity_managed", "false"),
				),
			},
			testAccKargoAgentImportStateStep(getKargoInstanceId(), name, testAccKargoAgentCommonImportStateVerifyIgnore...),
		},
	})
}

func testAccKargoAgentResourceConfig(size, name, description, kargoInstanceId string) string {
	return fmt.Sprintf(`
resource "akp_kargo_agent" "test" {
  instance_id = %q
  name        = %q
  namespace   = "test"
  labels = {
    test-label = "true"
  }
  annotations = {
    test-annotation = "false"
  }
  spec = {
    description = %q
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
          name: kargo-agent
        spec:
          template:
            spec:
              containers:
              - name: kargo-agent
                resources:
                  limits:
                    memory: 2Gi
                  requests:
                    cpu: 750m
                    memory: 1Gi
      target:
        kind: Deployment
        name: kargo-agent
EOF
      remote_argocd         = %q
      akuity_managed        = false
    }
  }
  remove_agent_resources_on_destroy = true
}

data "akp_kargo_agent" "test" {
  instance_id = akp_kargo_agent.test.instance_id
  name        = akp_kargo_agent.test.name
}

data "akp_kargo_agents" "test" {
  instance_id = akp_kargo_agent.test.instance_id
}
`, kargoInstanceId, name, description, size, getInstanceId())
}

func testAccKargoAgentResourceConfigRemoteArgoCD(name, kargoInstanceId, remoteArgoCDId string) string {
	return fmt.Sprintf(`
resource "akp_kargo_agent" "test" {
  instance_id = %q
  name        = %q
  namespace   = "test"
  spec = {
    description = "Remote ArgoCD test kargo agent"
    data = {
      size         = "small"
      remote_argocd = %q
      akuity_managed = false
    }
  }
  remove_agent_resources_on_destroy = true
}
`, kargoInstanceId, name, remoteArgoCDId)
}

func testAccKargoAgentResourceConfigCustomNamespace(name, kargoInstanceId string) string {
	return fmt.Sprintf(`
resource "akp_kargo_agent" "test" {
  instance_id = %q
  name        = %q
  namespace   = "test"
  spec = {
    description = "Custom namespace test kargo agent"
    data = {
      size            = "small"
      argocd_namespace = "custom-argocd"
      remote_argocd   = %q
      akuity_managed  = false
    }
  }
  remove_agent_resources_on_destroy = true
}
`, kargoInstanceId, name, getInstanceId())
}

func testAccKargoAgentResourceConfigReapplyManifests(name, description, kargoInstanceId string) string {
	return fmt.Sprintf(`
resource "akp_kargo_agent" "test" {
  instance_id = %q
  name        = %q
  namespace   = "test"
  spec = {
    description = %q
    data = {
      size = "small"
      remote_argocd = %q
      akuity_managed = false
    }
  }
  reapply_manifests_on_update       = true
  remove_agent_resources_on_destroy = true
}
`, kargoInstanceId, name, description, getInstanceId())
}

func testAccKargoAgentResourceConfigTargetVersion(name, _, kargoInstanceId string) string {
	return fmt.Sprintf(`
resource "akp_kargo_agent" "test" {
  instance_id = %q
  name        = %q
  namespace   = "test"
  spec = {
    description = "Target version test kargo agent"
    data = {
      size                  = "small"
      auto_upgrade_disabled = false
      remote_argocd         = %q
      akuity_managed        = false
    }
  }
  remove_agent_resources_on_destroy = true
}
`, kargoInstanceId, name, getInstanceId())
}

func runKargoAgentResourceKubeconfig(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("kargoagent-kubeconfig-%s", acctest.RandString(10))
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:             providerConfig + testAccKargoAgentResourceConfigKubeconfig(name, getKargoInstanceId()),
				ExpectError:        regexp.MustCompile("unable to apply kargo manifests"),
				ExpectNonEmptyPlan: true,
			},
		},
	})

	assert.NoError(t, testCheckKargoAgentCleanedUp(name, getKargoInstanceId()))
}

func testCheckKargoAgentCleanedUp(agentName, kargoInstanceId string) error {
	akpCli := getTestAkpCli()
	ctx := context.Background()
	ctx = httpctx.SetAuthorizationHeader(ctx, akpCli.Cred.Scheme(), akpCli.Cred.Credential())

	for attempt := 0; attempt < 5; attempt++ {
		if attempt > 0 {
			time.Sleep(2 * time.Second)
		}

		agents, err := akpCli.KargoCli.ListKargoInstanceAgents(ctx, &kargov1.ListKargoInstanceAgentsRequest{
			OrganizationId: akpCli.OrgId,
			InstanceId:     kargoInstanceId,
		})
		if err != nil && (status.Code(err) == codes.NotFound || status.Code(err) == codes.PermissionDenied) {
			return nil
		}

		found := false
		for _, agent := range agents.GetAgents() {
			if agent.GetName() == agentName {
				found = true
				break
			}
		}
		if !found {
			return nil
		}
	}

	return fmt.Errorf("kargo agent %s should have been automatically cleaned up but still exists in API after retries", agentName)
}

func testAccKargoAgentResourceConfigKubeconfig(name, kargoInstanceId string) string {
	return fmt.Sprintf(`
resource "akp_kargo_agent" "test" {
  instance_id = %q
  name        = %q
  namespace   = "test"
  spec = {
    description = "Kubeconfig test kargo agent"
    data = {
      size = "small"
      remote_argocd = %q
      akuity_managed = false
    }
  }
  kube_config = {
    host     = "https://test-cluster.example.com"
    insecure = true
    token    = "test-token"
  }
  remove_agent_resources_on_destroy = true
}
`, kargoInstanceId, name, getInstanceId())
}

func runKargoAgentResourceAllowedJobSA(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("kargoagent-jobsa-%s", acctest.RandString(10))
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccKargoAgentResourceConfigAllowedJobSA(name, getKargoInstanceId()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("akp_kargo_agent.test", "id"),
					resource.TestCheckResourceAttr("akp_kargo_agent.test", "spec.data.allowed_job_sa.#", "2"),
					resource.TestCheckResourceAttr("akp_kargo_agent.test", "spec.data.allowed_job_sa.0", "job-runner"),
					resource.TestCheckResourceAttr("akp_kargo_agent.test", "spec.data.allowed_job_sa.1", "analysis-runner"),
				),
			},
			testAccKargoAgentImportStateStep(getKargoInstanceId(), name, testAccKargoAgentAllowedJobSAImportStateVerifyIgnore...),
		},
	})
}

func testAccKargoAgentResourceConfigAllowedJobSA(name, kargoInstanceId string) string {
	return fmt.Sprintf(`
resource "akp_kargo_agent" "test" {
  instance_id = %q
  name        = %q
  namespace   = "test"
  spec = {
    description = "Allowed job SA test kargo agent"
    data = {
      size           = "small"
      remote_argocd  = %q
      akuity_managed = false
      allowed_job_sa = ["job-runner", "analysis-runner"]
    }
  }
  remove_agent_resources_on_destroy = true
}
`, kargoInstanceId, name, getInstanceId())
}

func runKargoAgentResourceMaintenanceMode(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("kargoagent-maint-%s", acctest.RandString(10))
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccKargoAgentResourceConfigMaintenanceMode(name, getKargoInstanceId()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("akp_kargo_agent.test", "id"),
					resource.TestCheckResourceAttr("akp_kargo_agent.test", "spec.data.maintenance_mode", "true"),
					resource.TestCheckResourceAttr("akp_kargo_agent.test", "spec.data.maintenance_mode_expiry", "2030-12-31T23:59:59Z"),
				),
			},
			testAccKargoAgentImportStateStep(getKargoInstanceId(), name, testAccKargoAgentCommonImportStateVerifyIgnore...),
		},
	})
}

func testAccKargoAgentResourceConfigMaintenanceMode(name, kargoInstanceId string) string {
	return fmt.Sprintf(`
resource "akp_kargo_agent" "test" {
  instance_id = %q
  name        = %q
  namespace   = "test"
  spec = {
    description = "Maintenance mode test kargo agent"
    data = {
      size                    = "small"
      remote_argocd           = %q
      akuity_managed          = false
      maintenance_mode        = true
      maintenance_mode_expiry = "2030-12-31T23:59:59Z"
    }
  }
  remove_agent_resources_on_destroy = true
}
`, kargoInstanceId, name, getInstanceId())
}

func runKargoAgentResourceMaintenanceModeTransitions(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("kargoagent-maint-tr-%s", acctest.RandString(10))
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccKargoAgentResourceConfigMaintenanceModeToggle(name, getKargoInstanceId(), true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("akp_kargo_agent.test", "id"),
					resource.TestCheckResourceAttr("akp_kargo_agent.test", "spec.data.maintenance_mode", "true"),
					resource.TestCheckResourceAttr("akp_kargo_agent.test", "spec.data.maintenance_mode_expiry", "2030-12-31T23:59:59Z"),
				),
			},
			{
				Config: providerConfig + testAccKargoAgentResourceConfigMaintenanceModeToggle(name, getKargoInstanceId(), false),
			},
			{
				Config: providerConfig + testAccKargoAgentResourceConfigMaintenanceModeToggle(name, getKargoInstanceId(), true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_kargo_agent.test", "spec.data.maintenance_mode", "true"),
				),
			},
			testAccKargoAgentImportStateStep(getKargoInstanceId(), name, testAccKargoAgentCommonImportStateVerifyIgnore...),
		},
	})
}

func testAccKargoAgentResourceConfigMaintenanceModeToggle(name, kargoInstanceId string, maintenanceMode bool) string {
	if maintenanceMode {
		return fmt.Sprintf(`
resource "akp_kargo_agent" "test" {
  instance_id = %q
  name        = %q
  namespace   = "test"
  spec = {
    description = "Maintenance mode toggle test"
    data = {
      size                    = "small"
      remote_argocd           = %q
      akuity_managed          = false
      maintenance_mode        = true
      maintenance_mode_expiry = "2030-12-31T23:59:59Z"
    }
  }
  remove_agent_resources_on_destroy = true
}
`, kargoInstanceId, name, getInstanceId())
	}
	return fmt.Sprintf(`
resource "akp_kargo_agent" "test" {
  instance_id = %q
  name        = %q
  namespace   = "test"
  spec = {
    description = "Maintenance mode toggle test"
    data = {
      size           = "small"
      remote_argocd  = %q
      akuity_managed = false
    }
  }
  remove_agent_resources_on_destroy = true
}
`, kargoInstanceId, name, getInstanceId())
}

func runKargoAgentResourceAllowedJobSATransitions(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("kargoagent-jobsa-tr-%s", acctest.RandString(10))
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccKargoAgentResourceConfigJobSAList(name, getKargoInstanceId(), `["job-runner", "analysis-runner"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("akp_kargo_agent.test", "id"),
					resource.TestCheckResourceAttr("akp_kargo_agent.test", "spec.data.allowed_job_sa.#", "2"),
				),
			},
			{
				Config: providerConfig + testAccKargoAgentResourceConfigJobSAList(name, getKargoInstanceId(), `["job-runner", "analysis-runner", "build-runner"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_kargo_agent.test", "spec.data.allowed_job_sa.#", "3"),
					resource.TestCheckResourceAttr("akp_kargo_agent.test", "spec.data.allowed_job_sa.2", "build-runner"),
				),
			},
			{
				Config: providerConfig + testAccKargoAgentResourceConfigJobSAList(name, getKargoInstanceId(), `["job-runner"]`),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_kargo_agent.test", "spec.data.allowed_job_sa.#", "1"),
					resource.TestCheckResourceAttr("akp_kargo_agent.test", "spec.data.allowed_job_sa.0", "job-runner"),
				),
			},
			testAccKargoAgentImportStateStep(getKargoInstanceId(), name, testAccKargoAgentAllowedJobSAImportStateVerifyIgnore...),
		},
	})
}

func testAccKargoAgentResourceConfigJobSAList(name, kargoInstanceId, saList string) string {
	return fmt.Sprintf(`
resource "akp_kargo_agent" "test" {
  instance_id = %q
  name        = %q
  namespace   = "test"
  spec = {
    description = "Job SA transitions test"
    data = {
      size           = "small"
      remote_argocd  = %q
      akuity_managed = false
      allowed_job_sa = %s
    }
  }
  remove_agent_resources_on_destroy = true
}
`, kargoInstanceId, name, getInstanceId(), saList)
}

func runKargoAgentResourceIdempotentReapply(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("kargoagent-idempotent-%s", acctest.RandString(10))
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccKargoAgentResourceConfigIdempotent(name, getKargoInstanceId()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("akp_kargo_agent.test", "id"),
					resource.TestCheckResourceAttr("akp_kargo_agent.test", "spec.description", "idempotent test"),
					resource.TestCheckResourceAttr("akp_kargo_agent.test", "spec.data.size", "small"),
				),
			},
			{
				Config: providerConfig + testAccKargoAgentResourceConfigIdempotent(name, getKargoInstanceId()),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
			testAccKargoAgentImportStateStep(getKargoInstanceId(), name, testAccKargoAgentAllowedJobSAImportStateVerifyIgnore...),
		},
	})
}

func testAccKargoAgentResourceConfigIdempotent(name, kargoInstanceId string) string {
	return fmt.Sprintf(`
resource "akp_kargo_agent" "test" {
  instance_id = %q
  name        = %q
  namespace   = "test"
  spec = {
    description = "idempotent test"
    data = {
      size                  = "small"
      auto_upgrade_disabled = true
      remote_argocd         = %q
      akuity_managed        = false
      allowed_job_sa        = ["job-runner"]
    }
  }
  remove_agent_resources_on_destroy = true
}
`, kargoInstanceId, name, getInstanceId())
}

func runKargoAgent_MinimalNestedImport(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("kargoagent-minimal-%s", acctest.RandString(10))
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccKargoAgentMinimalNestedConfig(name, getKargoInstanceId()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("akp_kargo_agent.test", "id"),
					resource.TestCheckResourceAttr("akp_kargo_agent.test", "name", name),
					resource.TestCheckResourceAttr("akp_kargo_agent.test", "spec.data.size", "small"),
					resource.TestCheckResourceAttr("akp_kargo_agent.test", "spec.data.auto_upgrade_disabled", "true"),
				),
			},
			{
				Config: providerConfig + testAccKargoAgentMinimalNestedConfig(name, getKargoInstanceId()),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
			testAccKargoAgentImportStateStep(getKargoInstanceId(), name, testAccKargoAgentMinimalImportStateVerifyIgnore...),
		},
	})
}

func testAccKargoAgentMinimalNestedConfig(name, kargoInstanceId string) string {
	return fmt.Sprintf(`
resource "akp_kargo_agent" "test" {
  instance_id = %q
  name        = %q
  namespace   = "test"
  spec = {
    description = "Minimal nested import test"
    data = {
      size                  = "small"
      auto_upgrade_disabled = true
      remote_argocd         = %q
      akuity_managed        = false
    }
  }
  remove_agent_resources_on_destroy = true
}
`, kargoInstanceId, name, getInstanceId())
}

func runKargoAgent_PartialDataImport(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("kargoagent-partial-%s", acctest.RandString(10))
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccKargoAgentPartialDataConfig(name, getKargoInstanceId()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("akp_kargo_agent.test", "id"),
					resource.TestCheckResourceAttr("akp_kargo_agent.test", "spec.data.size", "small"),
					resource.TestCheckResourceAttr("akp_kargo_agent.test", "spec.data.auto_upgrade_disabled", "true"),
					resource.TestCheckResourceAttrSet("akp_kargo_agent.test", "spec.data.kustomization"),
				),
			},
			{
				Config: providerConfig + testAccKargoAgentPartialDataConfig(name, getKargoInstanceId()),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
			testAccKargoAgentImportStateStep(getKargoInstanceId(), name, testAccKargoAgentPartialDataImportStateVerifyIgnore...),
		},
	})
}

func testAccKargoAgentPartialDataConfig(name, kargoInstanceId string) string {
	return fmt.Sprintf(`
resource "akp_kargo_agent" "test" {
  instance_id = %q
  name        = %q
  namespace   = "test"
  spec = {
    description = "Partial data import test"
    data = {
      size                  = "small"
      auto_upgrade_disabled = true
      remote_argocd         = %q
      akuity_managed        = false
      kustomization         = "apiVersion: kustomize.config.k8s.io/v1beta1\nkind: Kustomization\n"
    }
  }
  remove_agent_resources_on_destroy = true
}
`, kargoInstanceId, name, getInstanceId())
}

func runKargoAgentResourcePodInheritMetadata(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("kargoagent-pod-inherit-%s", acctest.RandString(10))
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccKargoAgentResourceConfigPodInheritMetadata(name, getKargoInstanceId()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("akp_kargo_agent.test", "id"),
					resource.TestCheckResourceAttr("akp_kargo_agent.test", "spec.data.pod_inherit_metadata", "true"),
				),
			},
			testAccKargoAgentImportStateStep(getKargoInstanceId(), name, testAccKargoAgentCommonImportStateVerifyIgnore...),
		},
	})
}

func testAccKargoAgentResourceConfigPodInheritMetadata(name, kargoInstanceId string) string {
	return fmt.Sprintf(`
resource "akp_kargo_agent" "test" {
  instance_id = %q
  name        = %q
  namespace   = "test"
  spec = {
    description = "Pod inherit metadata test kargo agent"
    data = {
      size                 = "small"
      remote_argocd        = %q
      akuity_managed       = false
      pod_inherit_metadata = true
    }
  }
  remove_agent_resources_on_destroy = true
}
`, kargoInstanceId, name, getInstanceId())
}
