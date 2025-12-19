package akp

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"sync"
	"testing"
	"time"

	hashitype "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/tfjsonpath"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"

	"github.com/akuity/api-client-go/pkg/api/gateway/accesscontrol"
	gwoption "github.com/akuity/api-client-go/pkg/api/gateway/option"
	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	kargov1 "github.com/akuity/api-client-go/pkg/api/gen/kargo/v1"
	orgcv1 "github.com/akuity/api-client-go/pkg/api/gen/organization/v1"
	idv1 "github.com/akuity/api-client-go/pkg/api/gen/types/id/v1"
	healthv1 "github.com/akuity/api-client-go/pkg/api/gen/types/status/health/v1"
	httpctx "github.com/akuity/grpc-gateway-client/pkg/http/context"
	"github.com/akuity/terraform-provider-akp/akp/types"
)

var (
	instanceId      string
	instanceName    string
	instanceVersion string
	testAkpCli      *AkpCli
	instanceOnce    sync.Once
)

func getInstanceVersion() string {
	getInstanceId()
	return instanceVersion
}

func getInstanceName() string {
	getInstanceId()
	return instanceName
}

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

	if os.Getenv("TF_ACC") != "1" {
		return nil
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

	gwc := gwoption.NewClient(serverUrl, skipTLSVerify)
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
	if instanceId == "" {
		instanceOnce.Do(func() {
			if os.Getenv("TF_ACC") != "1" {
				return
			}

			akpCli := getTestAkpCli()
			ctx := context.Background()
			ctx = httpctx.SetAuthorizationHeader(ctx, akpCli.Cred.Scheme(), akpCli.Cred.Credential())
			instanceName = fmt.Sprintf("test-cluster-provider-%s", acctest.RandString(8))

			instanceVersion = os.Getenv("AKUITY_ARGOCD_INSTANCE_VERSION")
			if instanceVersion == "" {
				instanceVersion = "v3.1.5-ak.65"
			}

			fmt.Printf("Creating test instance %s with version %s\n", instanceName, instanceVersion)

			// Create instance spec
			instanceStructpb, err := structpb.NewStruct(map[string]any{
				"metadata": map[string]any{
					"name": instanceName,
				},
				"spec": map[string]any{
					"version":     instanceVersion,
					"description": "This is used by the terraform provider to test managing clusters.",
					"instanceSpec": map[string]any{
						"imageUpdaterEnabled":         false,
						"backendIpAllowListEnabled":   false,
						"auditExtensionEnabled":       false,
						"syncHistoryExtensionEnabled": false,
						"assistantExtensionEnabled":   false,
						"appsetPolicy": map[string]any{
							"policy":         "sync",
							"overridePolicy": false,
						},
						"hostAliases": []any{
							map[string]any{
								"ip": "1.2.3.4",
								"hostnames": []any{
									"test-1",
									"test-2",
								},
							},
						},
						"multiClusterK8sDashboardEnabled": false,
						"akuityIntelligenceExtension": map[string]any{
							"enabled":                  true,
							"allowedUsernames":         []any{"admin", "test-user"},
							"allowedGroups":            []any{"admins", "test-group"},
							"aiSupportEngineerEnabled": true,
							"modelVersion":             "",
						},
						"kubeVisionConfig": map[string]any{
							"cveScanConfig": map[string]any{
								"scanEnabled":    true,
								"rescanInterval": "12h",
							},
							"aiConfig": map[string]any{
								"argocdSlackService": "test-slack-service",
								"argocdSlackChannels": []any{
									"test-channel-1",
									"test-channel-2",
								},
								"runbooks": []any{
									map[string]any{
										"name":    "test-incident",
										"content": "Test runbook content for incident response",
										"appliedTo": map[string]any{
											"argocdApplications": []any{"test-app"},
											"k8sNamespaces":      []any{"test-namespace"},
											"clusters":           []any{"test-cluster"},
											"degradedFor":        "5m",
										},
									},
								},
							},
						},
						"appInAnyNamespaceConfig": map[string]any{
							"enabled": false,
						},
						"appsetProgressiveSyncsEnabled": false,
						"appsetPlugins": []any{
							map[string]any{
								"name":           "plugin-test",
								"token":          "random-token",
								"baseUrl":        "http://random-test.xp",
								"requestTimeout": 0,
							},
						},
						"applicationSetExtension": map[string]any{
							"enabled": false,
						},
						"appReconciliationsRateLimiting": map[string]any{
							"bucketRateLimiting": map[string]any{
								"enabled":    false,
								"bucketSize": 500,
								"bucketQps":  50,
							},
							"itemRateLimiting": map[string]any{
								"enabled":         false,
								"failureCooldown": 10000,
								"baseDelay":       1,
								"maxDelay":        1000,
								"backoffFactor":   1.5,
							},
						},
					},
				},
			})
			if err != nil {
				panic(fmt.Sprintf("Failed to create instance struct: %v", err))
			}

			applyReq := &argocdv1.ApplyInstanceRequest{
				OrganizationId: akpCli.OrgId,
				Id:             instanceName,
				IdType:         idv1.Type_NAME,
				Argocd:         instanceStructpb,
			}
			fmt.Printf("Applying instance %s...\n", instanceName)
			_, err = akpCli.Cli.ApplyInstance(ctx, applyReq)
			if err != nil {
				panic(fmt.Sprintf("Failed to create instance: %v", err))
			}
			fmt.Printf("Instance %s created successfully\n", instanceName)

			// Get the created instance to get its ID
			fmt.Printf("Fetching instance %s to get ID...\n", instanceName)
			instanceResponse, err := akpCli.Cli.GetInstance(ctx, &argocdv1.GetInstanceRequest{
				OrganizationId: akpCli.OrgId,
				Id:             instanceName,
				IdType:         idv1.Type_NAME,
			})
			if err != nil {
				panic(fmt.Sprintf("Failed to get created instance: %v", err))
			}
			fmt.Printf("Instance %s has ID: %s\n", instanceName, instanceResponse.Instance.Id)

			getResourceFunc := func(ctx context.Context) (*argocdv1.GetInstanceResponse, error) {
				return akpCli.Cli.GetInstance(ctx, &argocdv1.GetInstanceRequest{
					OrganizationId: akpCli.OrgId,
					Id:             instanceResponse.Instance.Id,
					IdType:         idv1.Type_ID,
				})
			}

			getStatusFunc := func(resp *argocdv1.GetInstanceResponse) healthv1.StatusCode {
				if resp == nil || resp.Instance == nil {
					return healthv1.StatusCode_STATUS_CODE_UNKNOWN
				}
				status := resp.Instance.GetHealthStatus().GetCode()
				// Also log status message if available
				if resp.Instance.GetHealthStatus() != nil && resp.Instance.GetHealthStatus().GetMessage() != "" {
					fmt.Printf("Instance %s health status: %v - %s\n", instanceName, status, resp.Instance.GetHealthStatus().GetMessage())
				}
				return status
			}

			fmt.Printf("Waiting for instance %s to become healthy...\n", instanceName)
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
				// Before panicking, try to fetch the instance one more time to get detailed status
				if finalResp, getErr := akpCli.Cli.GetInstance(ctx, &argocdv1.GetInstanceRequest{
					OrganizationId: akpCli.OrgId,
					Id:             instanceResponse.Instance.Id,
					IdType:         idv1.Type_ID,
				}); getErr == nil {
					fmt.Printf("Final instance state - Health: %v, Message: %s\n",
						finalResp.Instance.GetHealthStatus().GetCode(),
						finalResp.Instance.GetHealthStatus().GetMessage())
				}
				panic(fmt.Sprintf("Test instance did not become healthy: %v", err))
			}

			fmt.Printf("Instance %s is now healthy!\n", instanceName)
			instanceId = instanceResponse.Instance.Id

			// Now that the instance is healthy, add test ArgoCD resources (Application and AppProject)
			// These are required by the data source tests
			appStruct, err := structpb.NewStruct(map[string]any{
				"apiVersion": "argoproj.io/v1alpha1",
				"kind":       "Application",
				"metadata": map[string]any{
					"name":      "app-test",
					"namespace": "argocd",
				},
				"spec": map[string]any{
					"project": "default",
					"source": map[string]any{
						"repoURL":        "https://github.com/argoproj/argocd-example-apps.git",
						"targetRevision": "HEAD",
						"path":           "guestbook",
					},
					"destination": map[string]any{
						"server":    "https://kubernetes.default.svc",
						"namespace": "default",
					},
				},
			})
			if err != nil {
				panic(fmt.Sprintf("Failed to create Application struct: %v", err))
			}

			appProjectStruct, err := structpb.NewStruct(map[string]any{
				"apiVersion": "argoproj.io/v1alpha1",
				"kind":       "AppProject",
				"metadata": map[string]any{
					"name":      "default",
					"namespace": "argocd",
				},
				"spec": map[string]any{
					"sourceRepos": []any{"*"},
					"destinations": []any{
						map[string]any{
							"namespace": "*",
							"server":    "*",
						},
					},
				},
			})
			if err != nil {
				panic(fmt.Sprintf("Failed to create AppProject struct: %v", err))
			}

			applyResourcesReq := &argocdv1.ApplyInstanceRequest{
				OrganizationId: akpCli.OrgId,
				Id:             instanceId,
				IdType:         idv1.Type_ID,
				Applications:   []*structpb.Struct{appStruct},
				AppProjects:    []*structpb.Struct{appProjectStruct},
			}
			fmt.Printf("Adding ArgoCD resources to instance %s...\n", instanceName)
			_, err = akpCli.Cli.ApplyInstance(ctx, applyResourcesReq)
			if err != nil {
				panic(fmt.Sprintf("Failed to add ArgoCD resources to instance: %v", err))
			}
			fmt.Printf("ArgoCD resources added successfully!\n")
		})
	}

	return instanceId
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

	// Check if multi-cluster k8s dashboard feature is enabled
	// If disabled, test should verify proper error handling
	if os.Getenv("MULTI_CLUSTER_K8S_DASHBOARD_FEATURE_ENABLED") != "true" {
		// Test that disabled feature returns proper error
		// TODO: Cluster should be deleted from API if we're trying to _create_ the resource (not when modifying it)
		resource.Test(t, resource.TestCase{
			PreCheck:                 func() { testAccPreCheck(t) },
			ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config:      providerConfig + testAccClusterResourceConfigFeatures(name, getInstanceId()),
					ExpectError: regexp.MustCompile("multi_cluster_k8s_dashboard_enabled feature is not available"),
				},
			},
		})
		return
	}

	// Feature is enabled, test normal functionality
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
	path := tfjsonpath.New("spec")
	data := path.AtMapKey("data")
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
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("akp_cluster.test", plancheck.ResourceActionDestroyBeforeCreate),
						plancheck.ExpectKnownValue("akp_cluster.test", path.AtMapKey("namespace_scoped"), knownvalue.Bool(false)),
						plancheck.ExpectKnownValue("akp_cluster.test", data.AtMapKey("size"), knownvalue.StringExact("small")),
						plancheck.ExpectUnknownValue("akp_cluster.test", data.AtMapKey("auto_agent_size_config")),
						plancheck.ExpectUnknownValue("akp_cluster.test", data.AtMapKey("auto_upgrade_disabled")),
						plancheck.ExpectUnknownValue("akp_cluster.test", data.AtMapKey("kustomization")),
						plancheck.ExpectUnknownValue("akp_cluster.test", data.AtMapKey("multi_cluster_k8s_dashboard_enabled")),
						plancheck.ExpectUnknownValue("akp_cluster.test", data.AtMapKey("redis_tunneling")),
						plancheck.ExpectUnknownValue("akp_cluster.test", data.AtMapKey("target_version")),
					},
				},
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

func TestAccClusterResourceAutoAgentSizeConsistency(t *testing.T) {
	name := fmt.Sprintf("cluster-auto-agent-consistency-%s", acctest.RandString(10))
	path := tfjsonpath.New("spec")
	data := path.AtMapKey("data")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Step 1: Create cluster without auto_agent_size_config
			{
				Config: providerConfig + testAccClusterResourceConfigWithoutAutoAgent(name, getInstanceId()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("akp_cluster.test", "id"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.size", "small"),
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectUnknownValue("akp_cluster.test", data.AtMapKey("auto_agent_size_config")),
					},
				},
			},
			// Step 2: Update to add auto_agent_size_config - should work without inconsistency
			{
				Config: providerConfig + testAccClusterResourceConfigWithAutoAgent(name, getInstanceId()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.size", "auto"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.auto_agent_size_config.application_controller.resource_maximum.cpu", "3"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.auto_agent_size_config.application_controller.resource_maximum.memory", "2Gi"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.auto_agent_size_config.repo_server.replicas_maximum", "3"),
				),
			},
			// Step 3: Revert to explicit size
			{
				Config: providerConfig + testAccClusterResourceConfigWithoutAutoAgent(name, getInstanceId()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.size", "small"),
				),
			},
			// Step 4: update to a invalid size, should be an error
			{
				Config: providerConfig + testAccClusterResourceConfigWithFakeSize(name, getInstanceId()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("akp_cluster.test", "id"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.size", "small"),
				),
				ExpectError: regexp.MustCompile(`Invalid size`),
			},
			// Step 5: Back to small
			{
				Config: providerConfig + testAccClusterResourceConfigWithoutAutoAgent(name, getInstanceId()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.size", "small"),
				),
			},
			// Step 6: Back to auto but with default auto_agent_size_config
			{
				Config: providerConfig + testAccClusterResourceConfigWithAutoAgentDefaultConfig(name, getInstanceId()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.size", "auto"),
				),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{},
				},
			},
		},
	})
}

// TestAccClusterResourceAutoAgentSizeCreateWithConfig tests creating a cluster
// directly with auto_agent_size_config to ensure it works correctly
func TestAccClusterResourceAutoAgentSizeCreateWithConfig(t *testing.T) {
	name := fmt.Sprintf("cluster-auto-agent-create-%s", acctest.RandString(10))

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccClusterResourceConfigWithAutoAgent(name, getInstanceId()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("akp_cluster.test", "id"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.size", "auto"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.auto_agent_size_config.application_controller.resource_maximum.cpu", "3"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.auto_agent_size_config.application_controller.resource_maximum.memory", "2Gi"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.auto_agent_size_config.application_controller.resource_minimum.cpu", "250m"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.auto_agent_size_config.application_controller.resource_minimum.memory", "1Gi"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.auto_agent_size_config.repo_server.replicas_maximum", "3"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.auto_agent_size_config.repo_server.replicas_minimum", "1"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.auto_agent_size_config.repo_server.resource_maximum.cpu", "3"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.auto_agent_size_config.repo_server.resource_maximum.memory", "2.00Gi"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.auto_agent_size_config.repo_server.resource_minimum.cpu", "250m"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.auto_agent_size_config.repo_server.resource_minimum.memory", "256Mi"),
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
}
`, instanceId, name, project)
}

// testAccClusterResourceConfigWithoutAutoAgent creates a cluster config WITHOUT auto_agent_size_config
// This should work properly without showing empty struct values in the state
func testAccClusterResourceConfigWithoutAutoAgent(name, instanceId string) string {
	return fmt.Sprintf(`
resource "akp_cluster" "test" {
  instance_id = %q
  name      = %q
  namespace = "test"
  spec = {
    namespace_scoped = true
    description      = "Auto agent size consistency test cluster"
    data = {
      size = "small"
      auto_upgrade_disabled = false
    }
  }
}
`, instanceId, name)
}

// testAccClusterResourceConfigWithFakeSize creates a cluster config WITH fake size to trigger errors
func testAccClusterResourceConfigWithFakeSize(name, instanceId string) string {
	return fmt.Sprintf(`
resource "akp_cluster" "test" {
  instance_id = %q
  name      = %q
  namespace = "test"
  spec = {
    namespace_scoped = true
    description      = "Auto agent size consistency test cluster"
    data = {
      size = "fake-size"
      auto_upgrade_disabled = false
    }
  }
}
`, instanceId, name)
}

// testAccClusterResourceConfigWithAutoAgent creates a cluster config WITH auto_agent_size_config
// This mirrors the example from auto_agent_size.tf
func testAccClusterResourceConfigWithAutoAgent(name, instanceId string) string {
	return fmt.Sprintf(`
resource "akp_cluster" "test" {
  instance_id = %q
  name      = %q
  namespace = "test"
  spec = {
    namespace_scoped = true
    description      = "Auto agent size consistency test cluster"
    data = {
      size = "auto"
      auto_upgrade_disabled = false
      auto_agent_size_config = {
        application_controller = {
          resource_maximum = {
            cpu    = "3"
            memory = "2Gi"
          },
          resource_minimum = {
            cpu    = "250m",
            memory = "1Gi"
          }
        },
        repo_server = {
          replicas_maximum = 3,
          replicas_minimum = 1,
          resource_maximum = {
            cpu    = "3"
            memory = "2.00Gi"
          },
          resource_minimum = {
            cpu    = "250m",
            memory = "256Mi"
          }
        }
      }
    }
  }
}
`, instanceId, name)
}

// testAccClusterResourceConfigWithAutoAgent creates a cluster config WITH auto_agent_size_config
// This mirrors the example from auto_agent_size.tf
func testAccClusterResourceConfigWithAutoAgentDefaultConfig(name, instanceId string) string {
	return fmt.Sprintf(`
resource "akp_cluster" "test" {
  instance_id = %q
  name      = %q
  namespace = "test"
  spec = {
    namespace_scoped = true
    description      = "Auto agent size consistency test cluster, default values"
    data = {
      size = "auto"
      auto_upgrade_disabled = false
    }
  }
}
`, instanceId, name)
}

func TestAccClusterResourceKubeconfig(t *testing.T) {
	name := fmt.Sprintf("cluster-kubeconfig-%s", acctest.RandString(10))
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:             providerConfig + testAccClusterResourceConfigKubeconfig(name, getInstanceId()),
				ExpectError:        regexp.MustCompile("unable to apply manifests"),
				ExpectNonEmptyPlan: true,
			},
		},
	})

	assert.NoError(t, testCheckClusterCleanedUp(name, instanceId))
}

func testCheckClusterCleanedUp(clusterName, instanceId string) error {
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
	if err != nil && (status.Code(err) == codes.NotFound || status.Code(err) == codes.PermissionDenied) {
		return nil
	}

	return fmt.Errorf("cluster %s should have been automatically cleaned up but still exists in API", clusterName)
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
}
`, instanceId, name)
}

func testAccClusterResourceConfigCustomAgentSizeWithKustomization(name, instanceId string) string {
	return fmt.Sprintf(`
resource "akp_cluster" "test" {
  instance_id = %q
  name      = %q
  namespace = "test"
  spec = {
    namespace_scoped = true
    description      = "Custom agent size with kustomization test"
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
      kustomization = <<EOF
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
patches:
  - patch: |
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: my-custom-app
      spec:
        template:
          spec:
            containers:
            - name: my-app
              resources:
                requests:
                  cpu: 100m
                  memory: 128Mi
    target:
      kind: Deployment
      name: my-custom-app
EOF
    }
  }
}
`, instanceId, name)
}

func testAccClusterResourceConfigCustomAgentSizeWithKustomizationUpdated(name, instanceId string) string {
	return fmt.Sprintf(`
resource "akp_cluster" "test" {
  instance_id = %q
  name      = %q
  namespace = "test"
  spec = {
    namespace_scoped = true
    description      = "Custom agent size with kustomization test - updated"
    data = {
      size = "custom"
      custom_agent_size_config = {
        application_controller = {
          memory = "4Gi"
          cpu    = "2000m"
        }
        repo_server = {
          memory   = "8Gi"
          cpu      = "4000m"
          replicas = 5
        }
      }
      kustomization = <<EOF
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
patches:
  - patch: |
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: my-custom-app
      spec:
        template:
          spec:
            containers:
            - name: my-app
              resources:
                requests:
                  cpu: 200m
                  memory: 256Mi
    target:
      kind: Deployment
      name: my-custom-app
EOF
    }
  }
}
`, instanceId, name)
}

func testAccClusterResourceConfigCustomAgentSizeKustomizationOnly(name, instanceId string) string {
	return fmt.Sprintf(`
resource "akp_cluster" "test" {
  instance_id = %q
  name      = %q
  namespace = "test"
  spec = {
    namespace_scoped = true
    description      = "Custom agent size kustomization only test"
    data = {
      size = "custom"
      custom_agent_size_config = {
        application_controller = {
          memory = "1Gi"
          cpu    = "500m"
        }
        repo_server = {
          memory   = "2Gi"
          cpu      = "1000m"
          replicas = 2
        }
      }
    }
  }
}
`, instanceId, name)
}

func testAccClusterResourceConfigCustomAgentSizeWithComplexKustomization(name, instanceId string) string {
	return fmt.Sprintf(`
resource "akp_cluster" "test" {
  instance_id = %q
  name      = %q
  namespace = "test"
  spec = {
    namespace_scoped = true
    description      = "Custom agent size with complex kustomization test"
    data = {
      size = "custom"
      custom_agent_size_config = {
        application_controller = {
          memory = "3Gi"
          cpu    = "1500m"
        }
        repo_server = {
          memory   = "6Gi"
          cpu      = "3000m"
          replicas = 4
        }
      }
      kustomization = <<EOF
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - base/deployment.yaml
  - base/service.yaml
patches:
  - patch: |
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: external-service
      spec:
        replicas: 2
        template:
          spec:
            containers:
            - name: external-service
              resources:
                requests:
                  cpu: 500m
                  memory: 512Mi
                limits:
                  cpu: 1000m
                  memory: 1Gi
    target:
      kind: Deployment
      name: external-service
  - patch: |
      apiVersion: v1
      kind: ConfigMap
      metadata:
        name: app-config
      data:
        config.yaml: |
          server:
            port: 8080
            host: 0.0.0.0
    target:
      kind: ConfigMap
      name: app-config
replicas:
  - name: my-replica
    count: 2
configMapGenerator:
  - name: env-config
    literals:
    - ENV=production
    - DEBUG=false
EOF
    }
  }
}
`, instanceId, name)
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

func TestAccClusterResourceCustomAgentSizeWithKustomization(t *testing.T) {
	name := fmt.Sprintf("cluster-custom-kustomization-%s", acctest.RandString(10))
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Test 1: Custom agent size with basic user kustomization (no conflicts)
			{
				Config: providerConfig + testAccClusterResourceConfigCustomAgentSizeWithKustomization(name, getInstanceId()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("akp_cluster.test", "id"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.size", "custom"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.custom_agent_size_config.application_controller.memory", "2Gi"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.custom_agent_size_config.application_controller.cpu", "1000m"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.custom_agent_size_config.repo_server.memory", "4Gi"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.custom_agent_size_config.repo_server.cpu", "2000m"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.custom_agent_size_config.repo_server.replicas", "3"),
					// Verify the kustomization includes both user patches and generated patches
					resource.TestCheckResourceAttrSet("akp_cluster.test", "spec.data.kustomization"),
				),
			},
			// Test 2: Update custom agent size config and verify kustomization is updated
			{
				Config: providerConfig + testAccClusterResourceConfigCustomAgentSizeWithKustomizationUpdated(name, getInstanceId()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.custom_agent_size_config.application_controller.memory", "4Gi"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.custom_agent_size_config.application_controller.cpu", "2000m"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.custom_agent_size_config.repo_server.memory", "8Gi"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.custom_agent_size_config.repo_server.cpu", "4000m"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.custom_agent_size_config.repo_server.replicas", "5"),
				),
			},
			// Test 3: Remove custom config and switch to predefined size
			{
				Config: providerConfig + testAccClusterResourceConfig("large", name, "Updated to large", getInstanceId()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.size", "large"),
					resource.TestCheckNoResourceAttr("akp_cluster.test", "spec.data.custom_agent_size_config"),
				),
			},
		},
	})
}

func TestAccClusterResourceCustomAgentSizeKustomizationOnly(t *testing.T) {
	name := fmt.Sprintf("cluster-custom-only-%s", acctest.RandString(10))
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Test: Custom agent size without user kustomization (generated only)
			{
				Config: providerConfig + testAccClusterResourceConfigCustomAgentSizeKustomizationOnly(name, getInstanceId()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("akp_cluster.test", "id"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.size", "custom"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.custom_agent_size_config.application_controller.memory", "1Gi"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.custom_agent_size_config.application_controller.cpu", "500m"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.custom_agent_size_config.repo_server.memory", "2Gi"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.custom_agent_size_config.repo_server.cpu", "1000m"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.custom_agent_size_config.repo_server.replicas", "2"),
					// Verify kustomization is generated automatically
					resource.TestCheckResourceAttrSet("akp_cluster.test", "spec.data.kustomization"),
				),
			},
		},
	})
}

func TestAccClusterResourceCustomAgentSizeTransitions(t *testing.T) {
	name := fmt.Sprintf("cluster-custom-transitions-%s", acctest.RandString(10))
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Start with small size
			{
				Config: providerConfig + testAccClusterResourceConfig("small", name, "Start small", getInstanceId()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.size", "small"),
					resource.TestCheckNoResourceAttr("akp_cluster.test", "spec.data.custom_agent_size_config"),
				),
			},
			// Transition to custom with both kustomization and custom config
			{
				Config: providerConfig + testAccClusterResourceConfigCustomAgentSizeWithComplexKustomization(name, getInstanceId()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.size", "custom"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.custom_agent_size_config.application_controller.memory", "3Gi"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.custom_agent_size_config.repo_server.replicas", "4"),
					resource.TestCheckResourceAttrSet("akp_cluster.test", "spec.data.kustomization"),
				),
			},
			// Transition back to medium size
			{
				Config: providerConfig + testAccClusterResourceConfig("medium", name, "Back to medium", getInstanceId()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.size", "medium"),
					resource.TestCheckNoResourceAttr("akp_cluster.test", "spec.data.custom_agent_size_config"),
				),
			},
		},
	})
}

func TestAccClusterResourceValidationError(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
resource "akp_cluster" "test_validation" {
  instance_id = "invalid-test"
  name        = "test-validation"
  namespace   = "argocd"
  spec = {
    data = {
      size = "small"
      auto_agent_size_config = {
        application_controller = {
          resource_minimum = {
            memory = "256Mi"
            cpu    = "250m"
          }
          resource_maximum = {
            memory = "512Mi"
            cpu    = "500m"
          }
        }
      }
    }
  }
}
`,
				ExpectError: regexp.MustCompile(`auto_agent_size_config cannot be used when size is 'small'`),
			},
			{
				Config: `
resource "akp_cluster" "test_validation" {
  instance_id = "invalid-test"
  name        = "test-validation" 
  namespace   = "argocd"
  spec = {
    data = {
      size = "medium"
      auto_agent_size_config = {
        application_controller = {
          resource_minimum = {
            memory = "256Mi"
            cpu    = "250m"
          }
          resource_maximum = {
            memory = "512Mi"
            cpu    = "500m"
          }
        }
      }
    }
  }
}
`,
				ExpectError: regexp.MustCompile(`auto_agent_size_config cannot be used when size is 'medium'`),
			},
			{
				Config: `
resource "akp_cluster" "test_validation" {
  instance_id = "invalid-test"
  name        = "test-validation"
  namespace   = "argocd"
  spec = {
    data = {
      size = "large"
      auto_agent_size_config = {
        application_controller = {
          resource_minimum = {
            memory = "256Mi"
            cpu    = "250m"
          }
          resource_maximum = {
            memory = "512Mi"
            cpu    = "500m"
          }
        }
      }
    }
  }
}
`,
				ExpectError: regexp.MustCompile(`auto_agent_size_config cannot be used when size is 'large'`),
			},
			{
				Config: `
resource "akp_cluster" "test_validation" {
  instance_id = "invalid-test"
  name        = "test-validation"
  namespace   = "argocd"
  spec = {
    data = {
      size = "custom"
      auto_agent_size_config = {
        application_controller = {
          resource_minimum = {
            memory = "256Mi"
            cpu    = "250m"
          }
          resource_maximum = {
            memory = "512Mi"
            cpu    = "500m"
          }
        }
      }
      custom_agent_size_config = {
        application_controller = {
          memory = "1Gi"
          cpu    = "500m"
        }
      }
    }
  }
}
`,
				ExpectError: regexp.MustCompile(`auto_agent_size_config cannot be used when size is 'custom'`),
			},
			{
				Config: `
resource "akp_cluster" "test_validation" {
  instance_id = "invalid-test"
  name        = "test-validation"
  namespace   = "argocd"
  spec = {
    data = {
      size = "custom"
    }
  }
}
`,
				ExpectError: regexp.MustCompile(`When size is 'custom', custom_agent_size_config must be specified`),
			},
			{
				Config: `
resource "akp_cluster" "test_validation" {
  instance_id = "invalid-test"
  name        = "test-validation"
  namespace   = "argocd"
  spec = {
    data = {
      size = "auto"
      custom_agent_size_config = {
        application_controller = {
          memory = "1Gi"
          cpu    = "500m"
        }
      }
    }
  }
}
`,
				ExpectError: regexp.MustCompile(`custom_agent_size_config cannot be used when size is 'auto'`),
			},
		},
	})
}

func testAccClusterResourceConfigMergeData(name, instanceId string) string {
	return fmt.Sprintf(`
variable "config" {
  type    = map(any)
  default = {}
}

resource "akp_cluster" "test" {
  instance_id = %q
  name        = %q
  namespace   = "test"

  labels = merge({
    "condor.io/cluster-name" = %q
    },
    {
      environment = "test"
      managed-by  = "terraform"
    }
  )

  annotations = merge({
    "argocd.argoproj.io/instance" = %q
    },
    {
      team  = "platform"
      owner = "devops"
    }
  )

  spec = {
    namespace_scoped = false
    description      = "Managed by Terraform, do not edit!"
    data = merge({
      kustomization = <<EOF
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
                  cpu: 500m
                  memory: 1Gi
    target:
      kind: Deployment
      name: argocd-repo-server
EOF
    }, {
      size                  = "small"
      auto_upgrade_disabled = true
    }, var.config)
  }
}
`, instanceId, name, name, "test-instance")
}

func TestAccClusterResourceMergeData(t *testing.T) {
	name := acctest.RandomWithPrefix("test-merge-data")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: providerConfig + testAccClusterResourceConfigMergeData(name, getInstanceId()),
			},
			{
				Config: providerConfig + testAccClusterResourceConfigMergeData(name, getInstanceId()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.size", "small"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.auto_upgrade_disabled", "true"),
				),
			},
		},
	})
}

// TestAccCluster_CustomAgentSizeInconsistency tests for custom agent size configuration inconsistencies
// This test provokes issues in the complex custom agent size logic in types.go lines 226-241
func TestAccCluster_CustomAgentSizeInconsistency(t *testing.T) {
	name := acctest.RandomWithPrefix("test-custom-size-inconsistency")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Start with a regular size
				Config: providerConfig + testAccClusterCustomAgentSizeInconsistencyConfig(name, getInstanceId(), "small", false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.size", "small"),
					resource.TestCheckNoResourceAttr("akp_cluster.test", "spec.data.custom_agent_size_config"),
				),
			},
			{
				// Change to custom size - this should trigger the complex logic in types.go
				Config: providerConfig + testAccClusterCustomAgentSizeInconsistencyConfig(name, getInstanceId(), "custom", true),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						// If there's an inconsistency, this might not be a simple update
						plancheck.ExpectResourceAction("akp_cluster.test", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.size", "custom"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.custom_agent_size_config.application_controller.memory", "2Gi"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.custom_agent_size_config.application_controller.cpu", "1000m"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.custom_agent_size_config.repo_server.memory", "4Gi"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.custom_agent_size_config.repo_server.cpu", "2000m"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.custom_agent_size_config.repo_server.replicas", "3"),
				),
			},
			{
				// Change back to regular size - this tests the reverse logic and potential for inconsistency
				Config: providerConfig + testAccClusterCustomAgentSizeInconsistencyConfig(name, getInstanceId(), "medium", false),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						// This should be an update, but inconsistencies might cause unexpected behavior
						plancheck.ExpectResourceAction("akp_cluster.test", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.data.size", "medium"),
					// custom_agent_size_config should not be present for non-custom sizes
					resource.TestCheckNoResourceAttr("akp_cluster.test", "spec.data.custom_agent_size_config"),
				),
			},
		},
	})
}

// TestAccCluster_NamespaceScopedMissingField tests for namespace_scoped inconsistencies when field is omitted
// This test provokes the issue where namespace_scoped field is not specified, causing inconsistencies
// between plan and API state due to different default value handling
func TestAccCluster_NamespaceScopedMissingField(t *testing.T) {
	name := acctest.RandomWithPrefix("test-ns-missing")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				// Create cluster WITHOUT specifying namespace_scoped - this should work consistently
				Config: providerConfig + testAccClusterNamespaceScopedMissingConfig(name, getInstanceId()),
			},
			{
				// Update the cluster description while still omitting namespace_scoped
				Config: providerConfig + testAccClusterNamespaceScopedMissingUpdatedConfig(name, getInstanceId()),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.description", "Updated description without namespace_scoped"),
				),
			},
			{
				// Now explicitly set namespace_scoped to see if it causes inconsistency
				Config: providerConfig + testAccClusterNamespaceScopedExplicitConfig(name, getInstanceId(), true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.namespace_scoped", "true"),
					resource.TestCheckResourceAttr("akp_cluster.test", "spec.description", "Explicit namespace_scoped test"),
				),
			},
		},
	})
}

// Helper function configs

func testAccClusterCustomAgentSizeInconsistencyConfig(name, instanceId, size string, includeCustomConfig bool) string {
	customConfig := ""
	if includeCustomConfig {
		customConfig = `
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
      }`
	}

	return fmt.Sprintf(`
resource "akp_cluster" "test" {
  instance_id = %q
  name        = %q
  namespace   = "test"
  spec = {
    namespace_scoped = true
    description      = "Custom agent size inconsistency test"
    data = {
      size = %q%s
    }
  }
}
`, instanceId, name, size, customConfig)
}

func testAccClusterNamespaceScopedMissingConfig(name, instanceId string) string {
	return fmt.Sprintf(`
resource "akp_cluster" "test" {
  instance_id = %q
  name        = %q
  namespace   = "test"
  spec = {
    # namespace_scoped field intentionally omitted to test default behavior
    description = "Missing namespace_scoped field test"
    data = {
      size = "small"
    }
  }
}
`, instanceId, name)
}

func testAccClusterNamespaceScopedMissingUpdatedConfig(name, instanceId string) string {
	return fmt.Sprintf(`
resource "akp_cluster" "test" {
  instance_id = %q
  name        = %q
  namespace   = "test"
  spec = {
    # namespace_scoped field still omitted
    description = "Updated description without namespace_scoped"
    data = {
      size = "small"
      auto_upgrade_disabled = true
    }
  }
}
`, instanceId, name)
}

func testAccClusterNamespaceScopedExplicitConfig(name, instanceId string, namespaceScoped bool) string {
	return fmt.Sprintf(`
resource "akp_cluster" "test" {
  instance_id = %q
  name        = %q
  namespace   = "test"
  spec = {
    namespace_scoped = %t
    description      = "Explicit namespace_scoped test"
    data = {
      size = "small"
      auto_upgrade_disabled = true
    }
  }
}
`, instanceId, name, namespaceScoped)
}
