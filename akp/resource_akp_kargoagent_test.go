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
	"github.com/hashicorp/terraform-plugin-testing/terraform"
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
	kargoInstanceId        string
	kargoInstanceName      string
	kargoVersion           string
	kargoInstanceOwned     bool
	kargoInstanceOnce      sync.Once
	kargoInstanceMu        sync.RWMutex
	kargoSentinelAgentId   string
	kargoSentinelAgentName = "sentinel-default-agent"
	kargoSentinelAgentMu   sync.RWMutex
)

func getKargoSentinelAgentId() string {
	kargoSentinelAgentMu.RLock()
	defer kargoSentinelAgentMu.RUnlock()
	return kargoSentinelAgentId
}

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

		ensureKargoSentinelAgent(ctx, akpCli, workspace.Id)
	})

	return kargoInstanceId
}

func findExistingSentinelAgent(ctx context.Context, akpCli *AkpCli, workspaceID string) string {
	agentsResp, err := retryWithBackoff(ctx, func(ctx context.Context) (*kargov1.ListKargoInstanceAgentsResponse, error) {
		return akpCli.KargoCli.ListKargoInstanceAgents(ctx, &kargov1.ListKargoInstanceAgentsRequest{
			OrganizationId: akpCli.OrgId,
			InstanceId:     kargoInstanceId,
			WorkspaceId:    workspaceID,
		})
	}, "list kargo agents for sentinel")
	if err != nil {
		panic(fmt.Sprintf("Failed to list kargo agents while resolving sentinel: %v", err))
	}
	for _, a := range agentsResp.GetAgents() {
		if a.GetName() == kargoSentinelAgentName {
			return a.GetId()
		}
	}
	return ""
}

func ensureKargoSentinelAgent(ctx context.Context, akpCli *AkpCli, workspaceID string) {
	sentinelID := findExistingSentinelAgent(ctx, akpCli, workspaceID)
	if sentinelID == "" {
		agentStruct, err := structpb.NewStruct(map[string]any{
			"apiVersion": "kargo.akuity.io/v1alpha1",
			"kind":       "KargoAgent",
			"metadata": map[string]any{
				"name":      kargoSentinelAgentName,
				"namespace": "akuity",
			},
			"spec": map[string]any{
				"data": map[string]any{
					"akuityManaged": true,
				},
			},
		})
		if err != nil {
			panic(fmt.Sprintf("Failed to build sentinel agent struct: %v", err))
		}

		applyReq := &kargov1.ApplyKargoInstanceRequest{
			OrganizationId: akpCli.OrgId,
			Id:             kargoInstanceId,
			IdType:         idv1.Type_ID,
			WorkspaceId:    workspaceID,
			Agents:         []*structpb.Struct{agentStruct},
		}
		_, err = retryWithBackoff(ctx, func(ctx context.Context) (*kargov1.ApplyKargoInstanceResponse, error) {
			return akpCli.KargoCli.ApplyKargoInstance(ctx, applyReq)
		}, "create sentinel kargo agent")
		if err != nil {
			panic(fmt.Sprintf("Failed to create sentinel kargo agent: %v", err))
		}

		sentinelID = findExistingSentinelAgent(ctx, akpCli, workspaceID)
		if sentinelID == "" {
			panic(fmt.Sprintf("Sentinel agent %q not found after apply", kargoSentinelAgentName))
		}
	}

	patchStruct, err := structpb.NewStruct(map[string]any{
		"spec": map[string]any{
			"defaultShardAgent": sentinelID,
		},
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to build sentinel default-shard patch: %v", err))
	}
	_, err = retryWithBackoff(ctx, func(ctx context.Context) (*kargov1.PatchKargoInstanceResponse, error) {
		return akpCli.KargoCli.PatchKargoInstance(ctx, &kargov1.PatchKargoInstanceRequest{
			OrganizationId: akpCli.OrgId,
			Id:             kargoInstanceId,
			Patch:          patchStruct,
		})
	}, "pin sentinel as default shard agent")
	if err != nil {
		panic(fmt.Sprintf("Failed to pin sentinel as default shard agent: %v", err))
	}

	kargoSentinelAgentMu.Lock()
	kargoSentinelAgentId = sentinelID
	kargoSentinelAgentMu.Unlock()
}

// restoreKargoSentinelAsDefault re-points the shared Kargo instance's
// defaultShardAgent back at the sentinel agent. Tests that intentionally
// move the default to one of their own agents must call this in t.Cleanup
// so subsequent parallel tests do not autoSet their own agent as default.
func restoreKargoSentinelAsDefault() error {
	sentinelID := getKargoSentinelAgentId()
	if sentinelID == "" {
		return nil
	}

	akpCli := getTestAkpCli()
	if akpCli == nil {
		return nil
	}
	ctx := context.Background()
	ctx = httpctx.SetAuthorizationHeader(ctx, akpCli.Cred.Scheme(), akpCli.Cred.Credential())

	patchStruct, err := structpb.NewStruct(map[string]any{
		"spec": map[string]any{
			"defaultShardAgent": sentinelID,
		},
	})
	if err != nil {
		return fmt.Errorf("build sentinel restore patch: %w", err)
	}
	_, err = akpCli.KargoCli.PatchKargoInstance(ctx, &kargov1.PatchKargoInstanceRequest{
		OrganizationId: akpCli.OrgId,
		Id:             getKargoInstanceId(),
		Patch:          patchStruct,
	})
	if err != nil {
		return fmt.Errorf("restore sentinel as default shard: %w", err)
	}
	return nil
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
			ensureKargoSentinelAgent(ctx, akpCli, instance.GetWorkspaceId())
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

func runKargoAgentResource(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("kargoagent-%s", acctest.RandString(10))

	// Restore the sentinel as the default shard agent so subsequent parallel
	// tests do not re-trigger autoSetDefaultShardAgent on the shared instance.
	t.Cleanup(func() {
		_ = restoreKargoSentinelAsDefault()
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
					resource.TestCheckResourceAttr("akp_kargo_agent.test", "spec.data.self_managed_argocd_url", "https://argocd.example.com"),
					resource.TestCheckResourceAttr("akp_kargo_agent.test", "remove_agent_resources_on_destroy", "true"),
					// --- Data Sources ---
					resource.TestCheckResourceAttr("data.akp_kargo_agent.test", "name", name),
					resource.TestCheckResourceAttrSet("data.akp_kargo_agent.test", "id"),
					resource.TestCheckResourceAttr("data.akp_kargo_agent.test", "spec.data.size", "small"),
					resource.TestCheckResourceAttr("data.akp_kargo_agent.test", "spec.data.auto_upgrade_disabled", "true"),
					resource.TestCheckResourceAttr("data.akp_kargo_agent.test", "spec.data.self_managed_argocd_url", "https://argocd.example.com"),
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
      remote_argocd           = %q
      akuity_managed          = false
      self_managed_argocd_url = "https://argocd.example.com"
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

	for attempt := range 5 {
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

func runKargoAgentResourceAutosize(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("kargoagent-autosize-%s", acctest.RandString(10))
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccKargoAgentResourceConfigAutosize(name, getKargoInstanceId()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("akp_kargo_agent.test", "id"),
					resource.TestCheckResourceAttr("akp_kargo_agent.test", "spec.data.size", "auto"),
					resource.TestCheckResourceAttr("akp_kargo_agent.test", "spec.data.autoscaler_config.kargo_controller.resource_minimum.mem", "1Gi"),
					resource.TestCheckResourceAttr("akp_kargo_agent.test", "spec.data.autoscaler_config.kargo_controller.resource_minimum.cpu", "500m"),
					resource.TestCheckResourceAttr("akp_kargo_agent.test", "spec.data.autoscaler_config.kargo_controller.resource_maximum.mem", "4Gi"),
					resource.TestCheckResourceAttr("akp_kargo_agent.test", "spec.data.autoscaler_config.kargo_controller.resource_maximum.cpu", "2000m"),
				),
			},
			{
				Config: providerConfig + testAccKargoAgentResourceConfigAutosize(name, getKargoInstanceId()),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
			testAccKargoAgentImportStateStep(getKargoInstanceId(), name, testAccKargoAgentAutosizeImportStateVerifyIgnore...),
		},
	})
}

func testAccKargoAgentResourceConfigAutosize(name, kargoInstanceId string) string {
	return fmt.Sprintf(`
resource "akp_kargo_agent" "test" {
  instance_id = %q
  name        = %q
  namespace   = "test"
  spec = {
    description = "Autosize test kargo agent"
    data = {
      size          = "auto"
      remote_argocd = %q
      akuity_managed = false
      autoscaler_config = {
        kargo_controller = {
          resource_minimum = {
            mem = "1Gi"
            cpu = "500m"
          }
          resource_maximum = {
            mem = "4Gi"
            cpu = "2000m"
          }
        }
      }
    }
  }
  remove_agent_resources_on_destroy = true
}
`, kargoInstanceId, name, getInstanceId())
}

func runKargoAgent_DefaultShardDeleteRejected(t *testing.T) {
	name := fmt.Sprintf("kargoagent-defrej-%s", acctest.RandString(8))
	instanceID := getKargoInstanceId()

	var agentID string
	t.Cleanup(func() {
		_ = restoreKargoSentinelAsDefault()
		if agentID == "" {
			return
		}
		akpCli := getTestAkpCli()
		if akpCli == nil {
			return
		}
		ctx := context.Background()
		ctx = httpctx.SetAuthorizationHeader(ctx, akpCli.Cred.Scheme(), akpCli.Cred.Credential())
		_, _ = akpCli.KargoCli.DeleteInstanceAgent(ctx, &kargov1.DeleteInstanceAgentRequest{
			OrganizationId: akpCli.OrgId,
			InstanceId:     instanceID,
			Id:             agentID,
		})
	})

	pinAgentAsDefault := func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources["akp_kargo_agent.test"]
		if !ok {
			return fmt.Errorf("resource akp_kargo_agent.test not found in state")
		}
		agentID = rs.Primary.Attributes["id"]
		if agentID == "" {
			return fmt.Errorf("resource akp_kargo_agent.test has empty id")
		}
		akpCli := getTestAkpCli()
		ctx := context.Background()
		ctx = httpctx.SetAuthorizationHeader(ctx, akpCli.Cred.Scheme(), akpCli.Cred.Credential())
		patchStruct, err := structpb.NewStruct(map[string]any{
			"spec": map[string]any{
				"defaultShardAgent": agentID,
			},
		})
		if err != nil {
			return fmt.Errorf("build pin patch: %w", err)
		}
		_, err = akpCli.KargoCli.PatchKargoInstance(ctx, &kargov1.PatchKargoInstanceRequest{
			OrganizationId: akpCli.OrgId,
			Id:             instanceID,
			Patch:          patchStruct,
		})
		if err != nil {
			return fmt.Errorf("pin agent as default: %w", err)
		}
		return nil
	}

	verifyAgentStillPresent := func(s *terraform.State) error {
		akpCli := getTestAkpCli()
		ctx := context.Background()
		ctx = httpctx.SetAuthorizationHeader(ctx, akpCli.Cred.Scheme(), akpCli.Cred.Credential())
		resp, err := akpCli.KargoCli.GetKargoInstanceAgent(ctx, &kargov1.GetKargoInstanceAgentRequest{
			OrganizationId: akpCli.OrgId,
			InstanceId:     instanceID,
			Id:             agentID,
		})
		if err != nil {
			return fmt.Errorf("expected agent %s to still exist after blocked destroy: %w", agentID, err)
		}
		if resp.GetAgent().GetId() != agentID {
			return fmt.Errorf("expected agent id %s, got %s", agentID, resp.GetAgent().GetId())
		}
		instancesResp, err := akpCli.KargoCli.ListKargoInstances(ctx, &kargov1.ListKargoInstancesRequest{
			OrganizationId: akpCli.OrgId,
		})
		if err != nil {
			return fmt.Errorf("list kargo instances: %w", err)
		}
		for _, instance := range instancesResp.GetInstances() {
			if instance.GetId() != instanceID {
				continue
			}
			if got := instance.GetSpec().GetDefaultShardAgent(); got != agentID {
				return fmt.Errorf("expected defaultShardAgent %s, got %q", agentID, got)
			}
			return nil
		}
		return fmt.Errorf("kargo instance %s not found", instanceID)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: create the agent. autoSetDefaultShardAgent is suppressed
			// by the sentinel, so we explicitly PATCH the instance's default
			// to this agent in the Check.
			{
				Config: providerConfig + testAccKargoAgentResourceConfig("small", name, "default-shard delete rejection", instanceID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("akp_kargo_agent.test", "id"),
					pinAgentAsDefault,
				),
			},
			// Step 2: destroy via empty config. The preflight check in
			// kargoAgentDelete (and the API guard behind it) must reject the
			// delete with the exact error wording, leaving the agent and the
			// instance's defaultShardAgent reference intact.
			{
				Config:      providerConfig,
				Destroy:     true,
				ExpectError: regexp.MustCompile(`cannot delete default shard agent`),
				Check:       verifyAgentStillPresent,
			},
			{
				PreConfig: func() {
					if err := restoreKargoSentinelAsDefault(); err != nil {
						t.Fatalf("restore sentinel default for post-test destroy: %v", err)
					}
				},
				Config: providerConfig + testAccKargoAgentResourceConfig("small", name, "default-shard delete rejection", instanceID),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("akp_kargo_agent.test", "id"),
				),
			},
		},
	})
}
