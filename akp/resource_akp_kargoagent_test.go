package akp

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"testing"
	"time"

	hashitype "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"

	kargov1 "github.com/akuity/api-client-go/pkg/api/gen/kargo/v1"
	idv1 "github.com/akuity/api-client-go/pkg/api/gen/types/id/v1"
	healthv1 "github.com/akuity/api-client-go/pkg/api/gen/types/status/health/v1"
	httpctx "github.com/akuity/grpc-gateway-client/pkg/http/context"
	"github.com/akuity/terraform-provider-akp/akp/types"
)

var kargoInstanceId string

func getKargoInstanceId() string {
	if kargoInstanceId == "" {
		if v := os.Getenv("AKUITY_KARGO_INSTANCE_ID"); v == "" {
			// Create a new Kargo instance for testing
			kargoInstanceId = createTestKargoInstance()
		} else {
			kargoInstanceId = v
		}
	}

	return kargoInstanceId
}

func createTestKargoInstance() string {
	if os.Getenv("TF_ACC") != "1" {
		return ""
	}

	akpCli := getTestAkpCli()
	ctx := context.Background()
	ctx = httpctx.SetAuthorizationHeader(ctx, akpCli.Cred.Scheme(), akpCli.Cred.Credential())
	kargoInstanceName := fmt.Sprintf("test-kargo-instance-%s", acctest.RandString(8))

	// Get default workspace
	workspace, err := getWorkspace(ctx, akpCli.OrgCli, akpCli.OrgId, "")
	if err != nil {
		panic(fmt.Sprintf("Failed to get default workspace: %v", err))
	}

	// Create minimal Kargo instance with required version
	kargoVersion := os.Getenv("AKUITY_KARGO_VERSION")
	if kargoVersion == "" {
		kargoVersion = "v1.7.4-ak.0"
	}

	kargoStruct, err := structpb.NewStruct(map[string]any{
		"metadata": map[string]any{
			"name": kargoInstanceName,
		},
		"spec": map[string]any{
			"version":     kargoVersion,
			"description": "Test Kargo instance for terraform provider tests",
		},
	})
	if err != nil {
		panic(fmt.Sprintf("Failed to create Kargo struct: %v", err))
	}

	applyReq := &kargov1.ApplyKargoInstanceRequest{
		OrganizationId: akpCli.OrgId,
		Id:             kargoInstanceName,
		IdType:         idv1.Type_NAME,
		WorkspaceId:    workspace.Id,
		Kargo:          kargoStruct,
	}
	_, err = akpCli.KargoCli.ApplyKargoInstance(ctx, applyReq)
	if err != nil {
		panic(fmt.Sprintf("Failed to create Kargo instance: %v", err))
	}

	// Get the created instance to get its ID
	instanceResponse, err := akpCli.KargoCli.GetKargoInstance(ctx, &kargov1.GetKargoInstanceRequest{
		OrganizationId: akpCli.OrgId,
		Name:           kargoInstanceName,
	})
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
		5*time.Minute,
		fmt.Sprintf("Test Kargo instance %s", kargoInstanceName),
		"health",
	)
	if err != nil {
		panic(fmt.Sprintf("Test Kargo instance did not become healthy: %v", err))
	}

	return instanceResponse.Instance.Id
}

func cleanupTestKargoInstance() {
	if kargoInstanceId == "" || testAkpCli == nil {
		return
	}

	ctx := context.Background()
	ctx = httpctx.SetAuthorizationHeader(ctx, testAkpCli.Cred.Scheme(), testAkpCli.Cred.Credential())

	// Delete the Kargo instance
	_, _ = testAkpCli.KargoCli.DeleteInstance(ctx, &kargov1.DeleteInstanceRequest{
		Id:             kargoInstanceId,
		OrganizationId: testAkpCli.OrgId,
	})
}

func TestAccKargoAgentResource(t *testing.T) {
	name := fmt.Sprintf("kargoagent-%s", acctest.RandString(10))
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
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccKargoAgentResourceRemoteArgoCD(t *testing.T) {
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
		},
	})
}

func TestAccKargoAgentResourceCustomNamespace(t *testing.T) {
	name := fmt.Sprintf("kargoagent-ns-%s", acctest.RandString(10))
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccKargoAgentResourceConfigCustomNamespace(name, getKargoInstanceId()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("akp_kargo_agent.test", "id"),
					resource.TestCheckResourceAttr("akp_kargo_agent.test", "spec.data.argocd_namespace", "custom-argocd"),
					resource.TestCheckResourceAttr("akp_kargo_agent.test", "spec.data.remote_argocd", getInstanceId()),
					resource.TestCheckResourceAttr("akp_kargo_agent.test", "spec.data.akuity_managed", "false"),
				),
			},
		},
	})
}

func TestAccKargoAgentResourceReapplyManifests(t *testing.T) {
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
		},
	})
}

func TestAccKargoAgentResourceTargetVersion(t *testing.T) {
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

func TestAccKargoAgentResourceKubeconfig(t *testing.T) {
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
	// Check that the agent was automatically cleaned up by the provider
	akpCli := getTestAkpCli()
	ctx := context.Background()
	ctx = httpctx.SetAuthorizationHeader(ctx, akpCli.Cred.Scheme(), akpCli.Cred.Credential())

	agents, err := akpCli.KargoCli.ListKargoInstanceAgents(ctx, &kargov1.ListKargoInstanceAgentsRequest{
		OrganizationId: akpCli.OrgId,
		InstanceId:     kargoInstanceId,
	})
	if err != nil && (status.Code(err) == codes.NotFound || status.Code(err) == codes.PermissionDenied) {
		return nil
	}

	for _, agent := range agents.GetAgents() {
		if agent.GetName() == agentName {
			return fmt.Errorf("kargo agent %s should have been automatically cleaned up but still exists in API", agentName)
		}
	}

	return nil
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

func TestAkpKargoAgentResource_reApplyManifests(t *testing.T) {
	type args struct {
		plan               *types.KargoAgent
		apiReq             *kargov1.ApplyKargoInstanceRequest
		applyKargoInstance func(context.Context, *kargov1.ApplyKargoInstanceRequest) (*kargov1.ApplyKargoInstanceResponse, error)
		upsertKubeConfig   func(ctx context.Context, plan *types.KargoAgent) error
	}
	tests := []struct {
		name  string
		args  args
		want  *types.KargoAgent
		error error
	}{
		{
			name: "error path, with kubeconfig",
			args: args{
				plan: &types.KargoAgent{
					Kubeconfig: &types.Kubeconfig{
						Host: hashitype.StringValue("some-host"),
					},
					ReapplyManifestsOnUpdate: hashitype.BoolValue(true),
				},
				applyKargoInstance: func(ctx context.Context, request *kargov1.ApplyKargoInstanceRequest) (*kargov1.ApplyKargoInstanceResponse, error) {
					return &kargov1.ApplyKargoInstanceResponse{}, nil
				},
				upsertKubeConfig: func(ctx context.Context, plan *types.KargoAgent) error {
					return errors.New("some kube apply error")
				},
			},
			want:  &types.KargoAgent{},
			error: fmt.Errorf("unable to apply kargo manifests: some kube apply error"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &AkpKargoAgentResource{}
			ctx := context.Background()
			_, err := r.applyKargoInstance(ctx, tt.args.plan, tt.args.apiReq, false, tt.args.applyKargoInstance, tt.args.upsertKubeConfig)
			assert.Equal(t, tt.error, err)
		})
	}
}
