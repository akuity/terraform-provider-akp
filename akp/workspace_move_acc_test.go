//go:build !unit

package akp

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	httpctx "github.com/akuity/grpc-gateway-client/pkg/http/context"
	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	kargov1 "github.com/akuity/api-client-go/pkg/api/gen/kargo/v1"
	idv1 "github.com/akuity/api-client-go/pkg/api/gen/types/id/v1"

	_ "github.com/lib/pq"
)

const (
	workspaceMoveArgoCDResourceName = "akp_instance.workspace_move"
	workspaceMoveArgoCDDataName     = "data.akp_instance.workspace_move"
	workspaceMoveKargoResourceName  = "akp_kargo_instance.workspace_move"
	workspaceMoveKargoDataName      = "data.akp_kargo_instance.workspace_move"
)

func runWorkspaceMoveArgoCD(t *testing.T) {
	version := os.Getenv("AKUITY_ARGOCD_INSTANCE_VERSION")

	name := acctest.RandomWithPrefix("workspace-move-argocd")
	renamed := name + "-renamed"
	workspaceName := acctest.RandomWithPrefix("workspace-move-argocd")
	stableID := ""
	defaultConfig := providerConfig + testAccWorkspaceMoveArgoCDConfig(name, workspaceName, "default", version)
	namedConfig := providerConfig + testAccWorkspaceMoveArgoCDConfig(name, workspaceName, workspaceName, version)
	renamedDefaultConfig := providerConfig + testAccWorkspaceMoveArgoCDConfig(renamed, workspaceName, "", version)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			if version == "" {
				t.Fatal("AKUITY_ARGOCD_INSTANCE_VERSION must be set for the workspace move acceptance test")
			}
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: namedConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrWith(workspaceMoveArgoCDResourceName, "id", testAccCheckStableID(&stableID)),
					resource.TestCheckResourceAttr(workspaceMoveArgoCDResourceName, "name", name),
					resource.TestCheckResourceAttr(workspaceMoveArgoCDResourceName, "workspace", workspaceName),
					resource.TestCheckResourceAttr(workspaceMoveArgoCDDataName, "workspace", workspaceName),
					resource.TestCheckResourceAttrPair(workspaceMoveArgoCDResourceName, "id", workspaceMoveArgoCDDataName, "id"),
					testAccCheckArgoCDWorkspace(workspaceMoveArgoCDResourceName, workspaceName),
				),
			},
			{
				Config: namedConfig,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				Config: defaultConfig,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(workspaceMoveArgoCDResourceName, plancheck.ResourceActionUpdate),
						expectManagedPlanActions{creates: 0, updates: 1, destroys: 0},
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrWith(workspaceMoveArgoCDResourceName, "id", testAccCheckStableID(&stableID)),
					resource.TestCheckResourceAttr(workspaceMoveArgoCDResourceName, "name", name),
					resource.TestCheckResourceAttr(workspaceMoveArgoCDResourceName, "workspace", "default"),
					resource.TestCheckResourceAttr(workspaceMoveArgoCDDataName, "workspace", "default"),
					resource.TestCheckResourceAttrPair(workspaceMoveArgoCDResourceName, "id", workspaceMoveArgoCDDataName, "id"),
					testAccCheckArgoCDWorkspace(workspaceMoveArgoCDResourceName, "default"),
				),
			},
			{
				Config: defaultConfig,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				Config: namedConfig,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(workspaceMoveArgoCDResourceName, plancheck.ResourceActionUpdate),
						expectManagedPlanActions{creates: 0, updates: 1, destroys: 0},
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrWith(workspaceMoveArgoCDResourceName, "id", testAccCheckStableID(&stableID)),
					resource.TestCheckResourceAttr(workspaceMoveArgoCDResourceName, "name", name),
					resource.TestCheckResourceAttr(workspaceMoveArgoCDResourceName, "workspace", workspaceName),
					resource.TestCheckResourceAttr(workspaceMoveArgoCDDataName, "workspace", workspaceName),
					resource.TestCheckResourceAttrPair(workspaceMoveArgoCDResourceName, "id", workspaceMoveArgoCDDataName, "id"),
					testAccCheckArgoCDWorkspace(workspaceMoveArgoCDResourceName, workspaceName),
				),
			},
			{
				Config: namedConfig,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				Config: renamedDefaultConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrWith(workspaceMoveArgoCDResourceName, "id", testAccCheckStableID(&stableID)),
					resource.TestCheckResourceAttr(workspaceMoveArgoCDResourceName, "name", renamed),
					resource.TestCheckResourceAttr(workspaceMoveArgoCDResourceName, "workspace", ""),
					resource.TestCheckResourceAttr(workspaceMoveArgoCDDataName, "name", renamed),
					resource.TestCheckResourceAttr(workspaceMoveArgoCDDataName, "workspace", "default"),
					resource.TestCheckResourceAttrPair(workspaceMoveArgoCDResourceName, "id", workspaceMoveArgoCDDataName, "id"),
					testAccCheckArgoCDWorkspace(workspaceMoveArgoCDResourceName, "default"),
				),
			},
			{
				Config: renamedDefaultConfig,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				PreConfig: func() {
					if err := testAccMoveArgoCDInstanceWorkspace(renamed, workspaceName); err != nil {
						t.Fatalf("move Argo CD instance out of band: %v", err)
					}
				},
				Config:             renamedDefaultConfig,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			{
				Config: renamedDefaultConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrWith(workspaceMoveArgoCDResourceName, "id", testAccCheckStableID(&stableID)),
					resource.TestCheckResourceAttr(workspaceMoveArgoCDResourceName, "workspace", ""),
					resource.TestCheckResourceAttr(workspaceMoveArgoCDDataName, "workspace", "default"),
					resource.TestCheckResourceAttrPair(workspaceMoveArgoCDResourceName, "id", workspaceMoveArgoCDDataName, "id"),
					testAccCheckArgoCDWorkspace(workspaceMoveArgoCDResourceName, "default"),
				),
			},
		},
	})
}

func runWorkspaceMoveKargo(t *testing.T) {
	version := os.Getenv("AKUITY_KARGO_VERSION")

	name := acctest.RandomWithPrefix("workspace-move-kargo")
	renamed := name + "-renamed"
	workspaceName := acctest.RandomWithPrefix("workspace-move-kargo")
	stableID := ""
	defaultConfig := providerConfig + testAccWorkspaceMoveKargoConfig(name, workspaceName, "default", version, false)
	namedConfig := providerConfig + testAccWorkspaceMoveKargoConfig(name, workspaceName, workspaceName, version, false)
	invalidDefaultConfig := providerConfig + testAccWorkspaceMoveKargoConfig(name, workspaceName, "default", version, true)
	renamedDefaultConfig := providerConfig + testAccWorkspaceMoveKargoConfig(renamed, workspaceName, "", version, false)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			if version == "" {
				t.Fatal("AKUITY_KARGO_VERSION must be set for the workspace move acceptance test")
			}
		},
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: namedConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrWith(workspaceMoveKargoResourceName, "id", testAccCheckStableID(&stableID)),
					resource.TestCheckResourceAttr(workspaceMoveKargoResourceName, "name", name),
					resource.TestCheckResourceAttr(workspaceMoveKargoResourceName, "workspace", workspaceName),
					resource.TestCheckResourceAttr(workspaceMoveKargoDataName, "workspace", workspaceName),
					resource.TestCheckResourceAttrPair(workspaceMoveKargoResourceName, "id", workspaceMoveKargoDataName, "id"),
					testAccCheckKargoWorkspace(workspaceMoveKargoResourceName, workspaceName),
				),
			},
			{
				Config: namedConfig,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				Config:      invalidDefaultConfig,
				ExpectError: regexp.MustCompile("subdomain and fqdn cannot be set at the same time"),
			},
			{
				PreConfig: func() {
					if err := testAccAssertKargoInstanceWorkspace(name, workspaceName); err != nil {
						t.Fatalf("invalid Kargo config moved the instance before validation: %v", err)
					}
				},
				Config: defaultConfig,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(workspaceMoveKargoResourceName, plancheck.ResourceActionUpdate),
						expectManagedPlanActions{creates: 0, updates: 1, destroys: 0},
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrWith(workspaceMoveKargoResourceName, "id", testAccCheckStableID(&stableID)),
					resource.TestCheckResourceAttr(workspaceMoveKargoResourceName, "name", name),
					resource.TestCheckResourceAttr(workspaceMoveKargoResourceName, "workspace", "default"),
					resource.TestCheckResourceAttr(workspaceMoveKargoDataName, "workspace", "default"),
					resource.TestCheckResourceAttrPair(workspaceMoveKargoResourceName, "id", workspaceMoveKargoDataName, "id"),
					testAccCheckKargoWorkspace(workspaceMoveKargoResourceName, "default"),
				),
			},
			{
				Config: defaultConfig,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				Config: namedConfig,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction(workspaceMoveKargoResourceName, plancheck.ResourceActionUpdate),
						expectManagedPlanActions{creates: 0, updates: 1, destroys: 0},
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrWith(workspaceMoveKargoResourceName, "id", testAccCheckStableID(&stableID)),
					resource.TestCheckResourceAttr(workspaceMoveKargoResourceName, "name", name),
					resource.TestCheckResourceAttr(workspaceMoveKargoResourceName, "workspace", workspaceName),
					resource.TestCheckResourceAttr(workspaceMoveKargoDataName, "workspace", workspaceName),
					resource.TestCheckResourceAttrPair(workspaceMoveKargoResourceName, "id", workspaceMoveKargoDataName, "id"),
					testAccCheckKargoWorkspace(workspaceMoveKargoResourceName, workspaceName),
				),
			},
			{
				Config: namedConfig,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				Config: renamedDefaultConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrWith(workspaceMoveKargoResourceName, "id", testAccCheckStableID(&stableID)),
					resource.TestCheckResourceAttr(workspaceMoveKargoResourceName, "name", renamed),
					resource.TestCheckResourceAttr(workspaceMoveKargoResourceName, "workspace", ""),
					resource.TestCheckResourceAttr(workspaceMoveKargoDataName, "name", renamed),
					resource.TestCheckResourceAttr(workspaceMoveKargoDataName, "workspace", "default"),
					resource.TestCheckResourceAttrPair(workspaceMoveKargoResourceName, "id", workspaceMoveKargoDataName, "id"),
					testAccCheckKargoWorkspace(workspaceMoveKargoResourceName, "default"),
				),
			},
			{
				Config: renamedDefaultConfig,
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{plancheck.ExpectEmptyPlan()},
				},
			},
			{
				PreConfig: func() {
					if err := testAccMoveKargoInstanceWorkspace(renamed, workspaceName); err != nil {
						t.Fatalf("move Kargo instance out of band: %v", err)
					}
				},
				Config:             renamedDefaultConfig,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			{
				Config: renamedDefaultConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrWith(workspaceMoveKargoResourceName, "id", testAccCheckStableID(&stableID)),
					resource.TestCheckResourceAttr(workspaceMoveKargoResourceName, "workspace", ""),
					resource.TestCheckResourceAttr(workspaceMoveKargoDataName, "workspace", "default"),
					resource.TestCheckResourceAttrPair(workspaceMoveKargoResourceName, "id", workspaceMoveKargoDataName, "id"),
					testAccCheckKargoWorkspace(workspaceMoveKargoResourceName, "default"),
				),
			},
		},
	})
}

func testAccWorkspaceMoveArgoCDConfig(name, workspaceName, workspace, version string) string {
	return fmt.Sprintf(`
resource "akp_workspace" "workspace_move" {
  name        = %q
  description = "Workspace move acceptance target"
}

resource "akp_instance" "workspace_move" {
  name       = %q
  workspace  = %q
  depends_on = [akp_workspace.workspace_move]

  argocd = {
    spec = {
      version     = %q
      description = "Workspace move acceptance instance"
      instance_spec = {
        declarative_management_enabled         = true
        multi_cluster_k8s_dashboard_enabled    = true
      }
    }
  }

  argocd_cm = {
    "exec.enabled"                   = false
    "ga.anonymizeusers"              = false
    "helm.enabled"                   = true
    "kustomize.enabled"              = true
    "server.rbac.log.enforce.enable" = false
    "statusbadge.enabled"            = false
    "ui.bannerpermanent"             = false
    "users.anonymous.enabled"        = false

    "kustomize.buildOptions" = "--enable-helm"
    "accounts.admin"         = "login"
  }
}

data "akp_instance" "workspace_move" {
  name = akp_instance.workspace_move.name
}
`, workspaceName, name, workspace, version)
}

func testAccWorkspaceMoveKargoConfig(name, workspaceName, workspace, version string, invalidHostnames bool) string {
	hostnames := ""
	if invalidHostnames {
		hostnames = `
      fqdn      = "workspace-move.invalid.example.com"
      subdomain = "workspace-move-invalid"`
	}

	return fmt.Sprintf(`
resource "akp_workspace" "workspace_move" {
  name        = %q
  description = "Workspace move acceptance target"
}

resource "akp_kargo_instance" "workspace_move" {
  name       = %q
  workspace  = %q
  depends_on = [akp_workspace.workspace_move]

  kargo = {
    spec = {
      version     = %q
      description = "Workspace move acceptance instance"%s
      kargo_instance_spec = {
        backend_ip_allow_list_enabled = false
        promo_controller_enabled      = true
      }
    }
  }
}

data "akp_kargo_instance" "workspace_move" {
  name = akp_kargo_instance.workspace_move.name
}
`, workspaceName, name, workspace, version, hostnames)
}

func testAccMoveArgoCDInstanceWorkspace(instanceName, targetWorkspaceName string) error {
	cli := getTestAkpCli()
	if cli == nil {
		return fmt.Errorf("could not get test client")
	}
	ctx := httpctx.SetAuthorizationHeader(context.Background(), cli.Cred.Scheme(), cli.Cred.Credential())
	instanceResp, err := cli.Cli.GetInstance(ctx, &argocdv1.GetInstanceRequest{
		OrganizationId: cli.OrgId,
		IdType:         idv1.Type_NAME,
		Id:             instanceName,
	})
	if err != nil {
		return fmt.Errorf("get Argo CD instance %q: %w", instanceName, err)
	}
	target, err := getWorkspace(ctx, cli.OrgCli, cli.OrgId, targetWorkspaceName)
	if err != nil {
		return fmt.Errorf("get target workspace %q: %w", targetWorkspaceName, err)
	}
	instance := instanceResp.GetInstance()
	if instance.GetWorkspaceId() == target.GetId() {
		return nil
	}
	_, err = cli.Cli.UpdateInstanceWorkspace(ctx, &argocdv1.UpdateInstanceWorkspaceRequest{
		OrganizationId: cli.OrgId,
		Id:             instance.GetId(),
		WorkspaceId:    instance.GetWorkspaceId(),
		NewWorkspaceId: target.GetId(),
	})
	if err != nil {
		return fmt.Errorf("move Argo CD instance %q to workspace %q: %w", instanceName, targetWorkspaceName, err)
	}
	return nil
}

func testAccMoveKargoInstanceWorkspace(instanceName, targetWorkspaceName string) error {
	cli := getTestAkpCli()
	if cli == nil {
		return fmt.Errorf("could not get test client")
	}
	ctx := httpctx.SetAuthorizationHeader(context.Background(), cli.Cred.Scheme(), cli.Cred.Credential())
	instanceResp, err := cli.KargoCli.GetKargoInstance(ctx, &kargov1.GetKargoInstanceRequest{
		OrganizationId: cli.OrgId,
		Name:           instanceName,
	})
	if err != nil {
		return fmt.Errorf("get Kargo instance %q: %w", instanceName, err)
	}
	target, err := getWorkspace(ctx, cli.OrgCli, cli.OrgId, targetWorkspaceName)
	if err != nil {
		return fmt.Errorf("get target workspace %q: %w", targetWorkspaceName, err)
	}
	instance := instanceResp.GetInstance()
	if instance.GetWorkspaceId() == target.GetId() {
		return nil
	}
	_, err = cli.KargoCli.UpdateKargoInstanceWorkspace(ctx, &kargov1.UpdateKargoInstanceWorkspaceRequest{
		OrganizationId: cli.OrgId,
		Id:             instance.GetId(),
		WorkspaceId:    instance.GetWorkspaceId(),
		NewWorkspaceId: target.GetId(),
	})
	if err != nil {
		return fmt.Errorf("move Kargo instance %q to workspace %q: %w", instanceName, targetWorkspaceName, err)
	}
	return nil
}

func testAccAssertKargoInstanceWorkspace(instanceName, expectedWorkspaceName string) error {
	cli := getTestAkpCli()
	if cli == nil {
		return fmt.Errorf("could not get test client")
	}
	ctx := httpctx.SetAuthorizationHeader(context.Background(), cli.Cred.Scheme(), cli.Cred.Credential())
	instanceResp, err := cli.KargoCli.GetKargoInstance(ctx, &kargov1.GetKargoInstanceRequest{
		OrganizationId: cli.OrgId,
		Name:           instanceName,
	})
	if err != nil {
		return fmt.Errorf("get Kargo instance %q: %w", instanceName, err)
	}
	expected, err := getWorkspace(ctx, cli.OrgCli, cli.OrgId, expectedWorkspaceName)
	if err != nil {
		return fmt.Errorf("get expected workspace %q: %w", expectedWorkspaceName, err)
	}
	if actual := instanceResp.GetInstance().GetWorkspaceId(); actual != expected.GetId() {
		return fmt.Errorf("Kargo instance %q workspace = %q, want %q", instanceName, actual, expected.GetId())
	}
	return nil
}

func testAccCheckStableID(stableID *string) resource.CheckResourceAttrWithFunc {
	return func(actual string) error {
		if actual == "" {
			return fmt.Errorf("instance ID is empty")
		}
		if *stableID == "" {
			*stableID = actual
			return nil
		}
		if actual != *stableID {
			return fmt.Errorf("instance ID changed from %q to %q", *stableID, actual)
		}
		return nil
	}
}

type expectManagedPlanActions struct {
	creates  int
	updates  int
	destroys int
}

func (e expectManagedPlanActions) CheckPlan(_ context.Context, req plancheck.CheckPlanRequest, resp *plancheck.CheckPlanResponse) {
	var creates, updates, destroys int
	for _, change := range req.Plan.ResourceChanges {
		if string(change.Mode) != "managed" || change.Change == nil || change.Change.Actions.NoOp() {
			continue
		}
		switch actions := change.Change.Actions; {
		case actions.Create():
			creates++
		case actions.Update():
			updates++
		case actions.Delete():
			destroys++
		case actions.Replace():
			creates++
			destroys++
		default:
			resp.Error = fmt.Errorf("unexpected managed resource action for %s: %v", change.Address, actions)
			return
		}
	}
	if creates != e.creates || updates != e.updates || destroys != e.destroys {
		resp.Error = fmt.Errorf(
			"managed plan actions = %d create, %d update, %d destroy; want %d create, %d update, %d destroy",
			creates, updates, destroys, e.creates, e.updates, e.destroys,
		)
	}
}

func testAccCheckArgoCDWorkspace(resourceName, expectedWorkspaceName string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		instanceID, err := testAccResourceID(state, resourceName)
		if err != nil {
			return err
		}
		cli := getTestAkpCli()
		if cli == nil {
			return fmt.Errorf("could not get test client")
		}
		ctx := httpctx.SetAuthorizationHeader(context.Background(), cli.Cred.Scheme(), cli.Cred.Credential())
		expectedWorkspace, err := getWorkspace(ctx, cli.OrgCli, cli.OrgId, expectedWorkspaceName)
		if err != nil {
			return fmt.Errorf("get expected workspace %q: %w", expectedWorkspaceName, err)
		}
		resp, err := cli.Cli.GetInstance(ctx, &argocdv1.GetInstanceRequest{
			OrganizationId: cli.OrgId,
			IdType:         idv1.Type_ID,
			Id:             instanceID,
		})
		if err != nil {
			return fmt.Errorf("get Argo CD instance %q by ID: %w", instanceID, err)
		}
		instance := resp.GetInstance()
		if instance == nil {
			return fmt.Errorf("Argo CD instance response did not include an instance")
		}
		if instance.GetId() != instanceID {
			return fmt.Errorf("Argo CD API returned instance ID %q, want %q", instance.GetId(), instanceID)
		}
		if instance.GetWorkspaceId() != expectedWorkspace.GetId() {
			return fmt.Errorf(
				"Argo CD API workspace ID = %q, want %q (%s)",
				instance.GetWorkspaceId(), expectedWorkspace.GetId(), expectedWorkspace.GetName(),
			)
		}
		return testAccCheckDatabaseWorkspace(ctx, "argocd", instanceID, cli.OrgId, expectedWorkspace.GetId())
	}
}

func testAccCheckKargoWorkspace(resourceName, expectedWorkspaceName string) resource.TestCheckFunc {
	return func(state *terraform.State) error {
		instanceID, err := testAccResourceID(state, resourceName)
		if err != nil {
			return err
		}
		cli := getTestAkpCli()
		if cli == nil {
			return fmt.Errorf("could not get test client")
		}
		ctx := httpctx.SetAuthorizationHeader(context.Background(), cli.Cred.Scheme(), cli.Cred.Credential())
		expectedWorkspace, err := getWorkspace(ctx, cli.OrgCli, cli.OrgId, expectedWorkspaceName)
		if err != nil {
			return fmt.Errorf("get expected workspace %q: %w", expectedWorkspaceName, err)
		}
		resp, err := getKargoInstanceByIdentity(ctx, cli.KargoCli, cli.OrgId, instanceID, "")
		if err != nil {
			return fmt.Errorf("get Kargo instance %q by ID: %w", instanceID, err)
		}
		instance := resp.GetInstance()
		if instance == nil {
			return fmt.Errorf("Kargo instance response did not include an instance")
		}
		if instance.GetId() != instanceID {
			return fmt.Errorf("Kargo API returned instance ID %q, want %q", instance.GetId(), instanceID)
		}
		if instance.GetWorkspaceId() != expectedWorkspace.GetId() {
			return fmt.Errorf(
				"Kargo API workspace ID = %q, want %q (%s)",
				instance.GetWorkspaceId(), expectedWorkspace.GetId(), expectedWorkspace.GetName(),
			)
		}
		return testAccCheckDatabaseWorkspace(ctx, "kargo", instanceID, cli.OrgId, expectedWorkspace.GetId())
	}
}

func testAccResourceID(state *terraform.State, resourceName string) (string, error) {
	resourceState, ok := state.RootModule().Resources[resourceName]
	if !ok {
		return "", fmt.Errorf("resource %s not found in Terraform state", resourceName)
	}
	if resourceState.Primary.ID == "" {
		return "", fmt.Errorf("resource %s has an empty ID", resourceName)
	}
	return resourceState.Primary.ID, nil
}

func testAccCheckDatabaseWorkspace(ctx context.Context, instanceType, instanceID, organizationID, expectedWorkspaceID string) error {
	// Cloud acceptance environments do not expose PostgreSQL. Local runs opt in
	// so the same checks prove the API response and stored workspace agree.
	dsn := os.Getenv("AKUITY_ACC_POSTGRES_DSN")
	if dsn == "" {
		return nil
	}

	query := ""
	switch instanceType {
	case "argocd":
		query = `SELECT workspace_id FROM public.argo_cd_instance WHERE id = $1 AND organization_owner = $2`
	case "kargo":
		query = `SELECT workspace_id FROM public.kargo_instance WHERE id = $1 AND organization_owner = $2`
	default:
		return fmt.Errorf("unsupported instance type %q", instanceType)
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("open local acceptance database: %w", err)
	}
	defer db.Close()

	queryCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	var actualWorkspaceID sql.NullString
	if err := db.QueryRowContext(queryCtx, query, instanceID, organizationID).Scan(&actualWorkspaceID); err != nil {
		return fmt.Errorf("query %s instance %q workspace from database: %w", instanceType, instanceID, err)
	}
	if !actualWorkspaceID.Valid {
		return fmt.Errorf("%s instance %q has NULL workspace_id in database", instanceType, instanceID)
	}
	if actualWorkspaceID.String != expectedWorkspaceID {
		return fmt.Errorf(
			"%s database workspace_id = %q, want %q",
			instanceType, actualWorkspaceID.String, expectedWorkspaceID,
		)
	}
	return nil
}
