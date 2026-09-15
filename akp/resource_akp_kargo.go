package akp

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	kargov1 "github.com/akuity/api-client-go/pkg/api/gen/kargo/v1"
	orgcv1 "github.com/akuity/api-client-go/pkg/api/gen/organization/v1"
	idv1 "github.com/akuity/api-client-go/pkg/api/gen/types/id/v1"
	healthv1 "github.com/akuity/api-client-go/pkg/api/gen/types/status/health/v1"
	reconv1 "github.com/akuity/api-client-go/pkg/api/gen/types/status/reconciliation/v1"
	"github.com/akuity/api-client-go/pkg/api/kargoexport"
	"github.com/akuity/terraform-provider-akp/akp/types"
)

func NewAkpKargoInstanceResource() resource.Resource {
	return &GenericResource[types.KargoInstance]{
		TypeNameSuffix: "kargo_instance",
		SchemaFunc:     kargoInstanceSchema,
		CreateFunc:     kargoInstanceCreateOrUpdate,
		ReadFunc:       kargoInstanceRead,
		UpdateFunc:     kargoInstanceCreateOrUpdate,
		DeleteFunc:     kargoInstanceDelete,
		ImportStateFunc: func(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
			resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
		},
		Validators: []resource.ConfigValidator{
			agentSizeDefaultValidator{
				defaultsPath: path.Root("kargo").AtName("spec").
					AtName("kargo_instance_spec").AtName("agent_customization_defaults"),
			},
		},
	}
}

func kargoInstanceCreateOrUpdate(ctx context.Context, cli *AkpCli, diags *diag.Diagnostics, plan *types.KargoInstance) (*types.KargoInstance, error) {
	stateCanBeCommitted, err := kargoInstanceUpsert(ctx, cli, diags, plan)
	if stateCanBeCommitted {
		return plan, err
	}
	return nil, err
}

func kargoInstanceRead(ctx context.Context, cli *AkpCli, diags *diag.Diagnostics, data *types.KargoInstance) error {
	if data.Kargo == nil || data.ID.IsNull() || data.ID.ValueString() == "" {
		ctx = types.WithReadContext(ctx)
	}
	return refreshKargoState(ctx, diags, cli, data, cli.OrgId, false)
}

func kargoInstanceDelete(ctx context.Context, cli *AkpCli, _ *diag.Diagnostics, state *types.KargoInstance) error {
	err := deleteWithCooldown(ctx, func(ctx context.Context) (*kargov1.DeleteInstanceResponse, error) {
		return cli.KargoCli.DeleteInstance(ctx, &kargov1.DeleteInstanceRequest{
			Id:             state.ID.ValueString(),
			OrganizationId: cli.OrgId,
		})
	}, "DeleteInstance", 2*time.Second)
	if err != nil {
		return fmt.Errorf("unable to delete Kargo instance, got error: %s", err)
	}
	return nil
}

func validateKargoInstanceAIFeatures(ctx context.Context, plan *types.KargoInstance) error {
	if plan.Kargo == nil || plan.Kargo.Spec.KargoInstanceSpec.AkuityIntelligence == nil {
		return nil
	}
	aiExt := plan.Kargo.Spec.KargoInstanceSpec.AkuityIntelligence
	if aiExt.Enabled.IsNull() || aiExt.Enabled.IsUnknown() {
		return nil
	}

	if !aiExt.Enabled.ValueBool() {
		if aiExt.AiSupportEngineerEnabled.ValueBool() ||
			aiExt.ModelVersion.ValueString() != "" ||
			len(aiExt.AllowedUsernames) > 0 ||
			len(aiExt.AllowedGroups) > 0 {
			return fmt.Errorf("AI configs are specified but AI Intelligence is disabled")
		}
	} else {
		if len(aiExt.AllowedUsernames) == 0 && len(aiExt.AllowedGroups) == 0 {
			tflog.Warn(ctx, "AI Intelligence is enabled but no allowed usernames or groups are specified")
		}
	}
	return nil
}

func kargoInstanceUpsert(ctx context.Context, cli *AkpCli, diagnostics *diag.Diagnostics, plan *types.KargoInstance) (stateCanBeCommitted bool, err error) {
	lc := &ResourceLifecycle[types.KargoInstance, *kargov1.GetKargoInstanceResponse, healthv1.StatusCode]{
		Apply: func(ctx context.Context, diagnostics *diag.Diagnostics, plan *types.KargoInstance) (bool, error) {
			if err := validateKargoInstanceAIFeatures(ctx, plan); err != nil {
				return false, err
			}

			workspace, err := getWorkspace(ctx, cli.OrgCli, cli.OrgId, plan.Workspace.ValueString())
			if err != nil {
				diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get workspace. %s", err))
				return false, errors.New("unable to get workspace")
			}

			apiReq := buildKargoApplyRequest(ctx, diagnostics, cli.KargoCli, plan, cli.OrgId, workspace.GetId())
			if diagnostics.HasError() {
				return false, errors.New("unable to build Kargo instance request")
			}

			// ApplyKargoInstance only honors workspace_id when creating an
			// instance; moving an existing one requires the dedicated workspace
			// API, and must happen before the apply so the request's
			// workspace_id matches the instance's actual workspace.
			_, err = ensureKargoInstanceWorkspace(ctx, cli.KargoCli, cli.OrgId, plan.ID.ValueString(), plan.Name.ValueString(), workspace)
			if err != nil {
				// The instance configuration has not been applied yet. Preserve
				// prior state so a later plan observes and repairs any move drift.
				return false, err
			}

			tflog.Debug(ctx, fmt.Sprintf("Apply instance request: %s", apiReq))

			_, err = retryWithBackoff(ctx, func(ctx context.Context) (*kargov1.ApplyKargoInstanceResponse, error) {
				return cli.KargoCli.ApplyKargoInstance(ctx, apiReq)
			}, "ApplyKargoInstance")
			if err != nil {
				// Export cannot reconstruct write-only fields such as secrets. Do
				// not commit the planned state unless the full apply succeeded.
				return false, fmt.Errorf("unable to upsert Kargo instance: %w", err)
			}

			plan.Workspace = workspaceStateValue(plan.Workspace, workspace)
			return true, nil
		},
		Get: func(ctx context.Context, plan *types.KargoInstance) (*kargov1.GetKargoInstanceResponse, error) {
			return retryWithBackoff(ctx, func(ctx context.Context) (*kargov1.GetKargoInstanceResponse, error) {
				return getKargoInstanceByIdentity(ctx, cli.KargoCli, cli.OrgId, plan.ID.ValueString(), plan.Name.ValueString())
			}, "GetKargoInstance")
		},
		GetStatus: func(resp *kargov1.GetKargoInstanceResponse) healthv1.StatusCode {
			if resp == nil || resp.Instance == nil {
				return healthv1.StatusCode_STATUS_CODE_UNKNOWN
			}
			return resp.Instance.GetHealthStatus().GetCode()
		},
		GetGeneration: func(resp *kargov1.GetKargoInstanceResponse) uint32 {
			if resp == nil || resp.Instance == nil {
				return 0
			}
			return resp.Instance.GetGeneration()
		},
		GetReconciliationDone: func(resp *kargov1.GetKargoInstanceResponse) bool {
			if resp == nil || resp.Instance == nil {
				return false
			}
			code := resp.Instance.GetReconciliationStatus().GetCode()
			return code == reconv1.StatusCode_STATUS_CODE_SUCCESSFUL
		},
		GetReconciliationFailed: func(resp *kargov1.GetKargoInstanceResponse) bool {
			if resp == nil || resp.Instance == nil {
				return false
			}
			return resp.Instance.GetReconciliationStatus().GetCode() == reconv1.StatusCode_STATUS_CODE_FAILED
		},
		TargetStatuses: []healthv1.StatusCode{healthv1.StatusCode_STATUS_CODE_HEALTHY},
		Refresh: func(ctx context.Context, diagnostics *diag.Diagnostics, plan *types.KargoInstance) error {
			return refreshKargoState(ctx, diagnostics, cli, plan, cli.OrgId, false)
		},
		ResourceName: func(plan *types.KargoInstance) string {
			return fmt.Sprintf("Instance %s", plan.Name.ValueString())
		},
		StatusName:   "health",
		PollInterval: 10 * time.Second,
		Timeout:      10 * time.Minute,
	}

	return lc.Upsert(ctx, diagnostics, plan)
}

func buildKargoApplyRequest(ctx context.Context, diagnostics *diag.Diagnostics, client kargov1.KargoServiceGatewayClient, kargo *types.KargoInstance, orgID, workspaceID string) *kargov1.ApplyKargoInstanceRequest {
	idType := idv1.Type_NAME
	id := kargo.Name.ValueString()

	if !kargo.ID.IsNull() && kargo.ID.ValueString() != "" {
		idType = idv1.Type_ID
		id = kargo.ID.ValueString()
	}

	applyReq := &kargov1.ApplyKargoInstanceRequest{
		OrganizationId: orgID,
		Id:             id,
		IdType:         idType,
		WorkspaceId:    workspaceID,
		Kargo:          buildKargo(ctx, diagnostics, kargo),
		KargoConfigmap: buildConfigMap(ctx, diagnostics, kargo.KargoConfigMap, "kargo-cm"),
		KargoSecret:    buildSecret(ctx, diagnostics, kargo.KargoSecret, "kargo-secret", nil),
	}

	if !kargo.KargoResources.IsUnknown() {
		processResources(
			ctx,
			diagnostics,
			kargo.KargoResources,
			kargoResourceGroups,
			isKargoResourceValid,
			applyReq,
			"Kargo",
		)
	}

	return applyReq
}

type kargoApplyReq = *kargov1.ApplyKargoInstanceRequest

var kargoResourceGroups = map[string]resourceGroup[kargoApplyReq]{
	"Project":               {"kargo.akuity.io", func(r kargoApplyReq) *[]*structpb.Struct { return &r.Projects }},
	"ProjectConfig":         {"kargo.akuity.io", func(r kargoApplyReq) *[]*structpb.Struct { return &r.ProjectConfigs }},
	"ClusterConfig":         {"kargo.akuity.io", func(r kargoApplyReq) *[]*structpb.Struct { return &r.ClusterConfigs }},
	"Warehouse":             {"kargo.akuity.io", func(r kargoApplyReq) *[]*structpb.Struct { return &r.Warehouses }},
	"Stage":                 {"kargo.akuity.io", func(r kargoApplyReq) *[]*structpb.Struct { return &r.Stages }},
	"PromotionTask":         {"kargo.akuity.io", func(r kargoApplyReq) *[]*structpb.Struct { return &r.PromotionTasks }},
	"ClusterPromotionTask":  {"kargo.akuity.io", func(r kargoApplyReq) *[]*structpb.Struct { return &r.ClusterPromotionTasks }},
	"MessageChannel":        {"ee.kargo.akuity.io", func(r kargoApplyReq) *[]*structpb.Struct { return &r.MessageChannels }},
	"ClusterMessageChannel": {"ee.kargo.akuity.io", func(r kargoApplyReq) *[]*structpb.Struct { return &r.ClusterMessageChannels }},
	"EventRouter":           {"ee.kargo.akuity.io", func(r kargoApplyReq) *[]*structpb.Struct { return &r.EventRouters }},
	"CustomPromotionStep":   {"ee.kargo.akuity.io", func(r kargoApplyReq) *[]*structpb.Struct { return &r.CustomPromotionSteps }},
	"AnalysisTemplate":      {"argoproj.io", func(r kargoApplyReq) *[]*structpb.Struct { return &r.AnalysisTemplates }},
	"Role":                  {"rbac.authorization.k8s.io", func(r kargoApplyReq) *[]*structpb.Struct { return &r.Roles }},
	"RoleBinding":           {"rbac.authorization.k8s.io", func(r kargoApplyReq) *[]*structpb.Struct { return &r.RoleBindings }},
	"ServiceAccount":        {"", func(r kargoApplyReq) *[]*structpb.Struct { return &r.ServiceAccounts }},
	"ConfigMap":             {"", func(r kargoApplyReq) *[]*structpb.Struct { return &r.Configmaps }},
	"Secret":                {"", func(r kargoApplyReq) *[]*structpb.Struct { return &r.RepoCredentials }},
}

func isKargoResourceValid(un *unstructured.Unstructured) error {
	if err := validateResource(un, kargoResourceGroups); err != nil {
		return err
	}
	if un.GetKind() == "Secret" && un.GetLabels()["kargo.akuity.io/cred-type"] == "" {
		return errors.New("secret must have a kargo.akuity.io/cred-type label")
	}
	return nil
}

func buildKargo(_ context.Context, diagnostics *diag.Diagnostics, kargo *types.KargoInstance) *structpb.Struct {
	subdomain := kargo.Kargo.Spec.Subdomain.ValueString()
	fqdn := kargo.Kargo.Spec.Fqdn.ValueString()
	if subdomain != "" && fqdn != "" {
		diagnostics.AddError("subdomain and fqdn cannot be set at the same time", "subdomain and fqdn are mutually exclusive")
		return nil
	}

	rawMap := types.TFToMapWithOverrides(kargo.Kargo, types.KargoOverridesMap, types.KargoRenamesMap)
	if rawMap == nil {
		diagnostics.AddError("Client Error", "Unable to convert Kargo instance to map")
		return nil
	}

	rawMap["metadata"] = map[string]any{
		"name": kargo.Name.ValueString(),
	}

	if spec, ok := rawMap["spec"].(map[string]any); ok {
		kargoInstanceSpec, sok := spec["kargoInstanceSpec"].(map[string]any)
		if sok {
			delete(kargoInstanceSpec, "defaultShardAgent")
		}

		_, fok := spec["fqdn"].(string)
		if !fok {
			spec["fqdn"] = ""
		}
	}

	s, err := structpb.NewStruct(rawMap)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Kargo instance struct. %s", err))
		return nil
	}
	return s
}

func refreshKargoState(ctx context.Context, diagnostics *diag.Diagnostics, cli *AkpCli, kargo *types.KargoInstance, orgID string, isDataSource bool) error {
	tflog.Debug(ctx, fmt.Sprintf("Get Kargo instance by identity: id=%q name=%q", kargo.ID.ValueString(), kargo.Name.ValueString()))
	resp, err := retryWithBackoff(ctx, func(ctx context.Context) (*kargov1.GetKargoInstanceResponse, error) {
		return getKargoInstanceByIdentity(ctx, cli.KargoCli, orgID, kargo.ID.ValueString(), kargo.Name.ValueString())
	}, "GetKargoInstance")
	if err != nil {
		return fmt.Errorf("unable to read Kargo instance: %w", err)
	}
	tflog.Debug(ctx, fmt.Sprintf("Get Kargo instance response: %s", resp))
	kargo.ID = tftypes.StringValue(resp.Instance.Id)
	kargo.Name = tftypes.StringValue(resp.Instance.Name)
	// Always resolve the workspace from the API so out-of-band moves (e.g. via
	// the UI) surface as drift instead of being masked by the stored state.
	if workspaceID := resp.Instance.GetWorkspaceId(); workspaceID != "" {
		workspace, workspaceErr := getWorkspaceByID(ctx, cli.OrgCli, orgID, workspaceID)
		if workspaceErr == nil && workspace != nil {
			kargo.Workspace = workspaceStateValue(kargo.Workspace, workspace)
		}
	}

	exportReq := &kargov1.ExportKargoInstanceRequest{
		OrganizationId: orgID,
		Id:             kargo.ID.ValueString(),
		WorkspaceId:    resp.Instance.WorkspaceId,
	}
	tflog.Debug(ctx, fmt.Sprintf("Export Kargo instance request: %s", exportReq))
	exportResp, err := retryWithBackoff(ctx, func(ctx context.Context) (*kargov1.ExportKargoInstanceResponse, error) {
		return kargoexport.ExportKargoInstance(ctx, cli.KargoCli, exportReq)
	}, "ExportKargoInstance")
	if err != nil {
		return fmt.Errorf("unable to export Kargo instance: %w", err)
	}
	tflog.Debug(ctx, fmt.Sprintf("Export Kargo instance response: %s", exportResp))
	return kargo.Update(ctx, diagnostics, exportResp, isDataSource)
}

func getWorkspace(ctx context.Context, orgc orgcv1.OrganizationServiceGatewayClient, orgid, name string) (*orgcv1.Workspace, error) {
	workspaces, err := retryWithBackoff(ctx, func(ctx context.Context) (*orgcv1.ListWorkspacesResponse, error) {
		return orgc.ListWorkspaces(ctx, &orgcv1.ListWorkspacesRequest{
			OrganizationId: orgid,
		})
	}, "ListWorkspaces")
	if err != nil {
		return nil, fmt.Errorf("unable to read org workspaces: %w", err)
	}
	for _, w := range workspaces.GetWorkspaces() {
		if name == "" && w.IsDefault {
			return w, nil
		}
		if w.Name == name {
			return w, nil
		}
	}

	// Surface this as a gRPC NotFound so callers can isGoneErr/status.Code
	// against it the same as they would a server-side 404.
	return nil, status.Errorf(codes.NotFound, "workspace %s not found", name)
}

func getKargoInstanceByID(ctx context.Context, cli *AkpCli, instanceID string) (*kargov1.KargoInstance, error) {
	resp, err := retryWithBackoff(ctx, func(ctx context.Context) (*kargov1.GetKargoInstanceResponse, error) {
		return getKargoInstanceByIdentity(ctx, cli.KargoCli, cli.OrgId, instanceID, "")
	}, "GetKargoInstance")
	if err != nil {
		return nil, err
	}
	return resp.GetInstance(), nil
}

func getWorkspaceByID(ctx context.Context, orgc orgcv1.OrganizationServiceGatewayClient, orgid, id string) (*orgcv1.Workspace, error) {
	workspaces, err := retryWithBackoff(ctx, func(ctx context.Context) (*orgcv1.ListWorkspacesResponse, error) {
		return orgc.ListWorkspaces(ctx, &orgcv1.ListWorkspacesRequest{
			OrganizationId: orgid,
		})
	}, "ListWorkspaces")
	if err != nil {
		return nil, fmt.Errorf("unable to read org workspaces: %w", err)
	}
	for _, w := range workspaces.GetWorkspaces() {
		if w.Id == id {
			return w, nil
		}
	}
	return nil, status.Errorf(codes.NotFound, "workspace with id %s not found", id)
}
