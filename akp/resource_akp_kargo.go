package akp

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/structpb"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	kargov1 "github.com/akuity/api-client-go/pkg/api/gen/kargo/v1"
	orgcv1 "github.com/akuity/api-client-go/pkg/api/gen/organization/v1"
	idv1 "github.com/akuity/api-client-go/pkg/api/gen/types/id/v1"
	healthv1 "github.com/akuity/api-client-go/pkg/api/gen/types/status/health/v1"
	reconv1 "github.com/akuity/api-client-go/pkg/api/gen/types/status/reconciliation/v1"
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
	}
}

func kargoInstanceCreateOrUpdate(ctx context.Context, cli *AkpCli, diags *diag.Diagnostics, plan *types.KargoInstance) (*types.KargoInstance, error) {
	applied, err := kargoInstanceUpsert(ctx, cli, diags, plan)
	if applied {
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

func kargoInstanceUpsert(ctx context.Context, cli *AkpCli, diagnostics *diag.Diagnostics, plan *types.KargoInstance) (applied bool, err error) {
	lc := &ResourceLifecycle[types.KargoInstance, *kargov1.GetKargoInstanceResponse, healthv1.StatusCode]{
		Apply: func(ctx context.Context, diagnostics *diag.Diagnostics, plan *types.KargoInstance) error {
			if err := validateKargoInstanceAIFeatures(ctx, plan); err != nil {
				return err
			}

			workspace, err := getWorkspace(ctx, cli.OrgCli, cli.OrgId, plan.Workspace.ValueString())
			if err != nil {
				diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get workspace. %s", err))
				return errors.New("Unable to get workspace")
			}

			apiReq := buildKargoApplyRequest(ctx, diagnostics, cli.KargoCli, plan, cli.OrgId, workspace.GetId())
			if diagnostics.HasError() {
				return errors.New("Unable to build Kargo instance request")
			}
			tflog.Debug(ctx, fmt.Sprintf("Apply instance request: %s", apiReq))

			_, err = retryWithBackoff(ctx, func(ctx context.Context) (*kargov1.ApplyKargoInstanceResponse, error) {
				return cli.KargoCli.ApplyKargoInstance(ctx, apiReq)
			}, "ApplyKargoInstance")
			if err != nil {
				return errors.Wrap(err, "Unable to upsert Kargo instance")
			}

			if plan.Workspace.ValueString() == "" {
				plan.Workspace = tftypes.StringValue(workspace.GetName())
			}
			return nil
		},
		Get: func(ctx context.Context, plan *types.KargoInstance) (*kargov1.GetKargoInstanceResponse, error) {
			return retryWithBackoff(ctx, func(ctx context.Context) (*kargov1.GetKargoInstanceResponse, error) {
				return cli.KargoCli.GetKargoInstance(ctx, &kargov1.GetKargoInstanceRequest{
					OrganizationId: cli.OrgId,
					Name:           plan.Name.ValueString(),
					WorkspaceId:    plan.Workspace.ValueString(),
				})
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
		Timeout:      5 * time.Minute,
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

	agentMaps := buildAgentMaps(ctx, client, id, orgID, idType)

	applyReq := &kargov1.ApplyKargoInstanceRequest{
		OrganizationId: orgID,
		Id:             id,
		IdType:         idType,
		WorkspaceId:    workspaceID,
		Kargo:          buildKargo(ctx, diagnostics, kargo, agentMaps),
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

var kargoResourceGroups = map[string]struct {
	appendFunc resourceGroupAppender[*kargov1.ApplyKargoInstanceRequest]
}{
	"Project": {
		appendFunc: func(req *kargov1.ApplyKargoInstanceRequest, item *structpb.Struct) {
			req.Projects = append(req.Projects, item)
		},
	},
	"Warehouse": {
		appendFunc: func(req *kargov1.ApplyKargoInstanceRequest, item *structpb.Struct) {
			req.Warehouses = append(req.Warehouses, item)
		},
	},
	"Stage": {
		appendFunc: func(req *kargov1.ApplyKargoInstanceRequest, item *structpb.Struct) {
			req.Stages = append(req.Stages, item)
		},
	},
	"AnalysisTemplate": {
		appendFunc: func(req *kargov1.ApplyKargoInstanceRequest, item *structpb.Struct) {
			req.AnalysisTemplates = append(req.AnalysisTemplates, item)
		},
	},
	"Secret": {
		appendFunc: func(req *kargov1.ApplyKargoInstanceRequest, item *structpb.Struct) {
			req.RepoCredentials = append(req.RepoCredentials, item)
		},
	},
	"PromotionTask": {
		appendFunc: func(req *kargov1.ApplyKargoInstanceRequest, item *structpb.Struct) {
			req.PromotionTasks = append(req.PromotionTasks, item)
		},
	},
	"ClusterPromotionTask": {
		appendFunc: func(req *kargov1.ApplyKargoInstanceRequest, item *structpb.Struct) {
			req.ClusterPromotionTasks = append(req.ClusterPromotionTasks, item)
		},
	},
	"ServiceAccount": {
		appendFunc: func(req *kargov1.ApplyKargoInstanceRequest, item *structpb.Struct) {
			req.ServiceAccounts = append(req.ServiceAccounts, item)
		},
	},
	"Role": {
		appendFunc: func(req *kargov1.ApplyKargoInstanceRequest, item *structpb.Struct) {
			req.Roles = append(req.Roles, item)
		},
	},
	"RoleBinding": {
		appendFunc: func(req *kargov1.ApplyKargoInstanceRequest, item *structpb.Struct) {
			req.RoleBindings = append(req.RoleBindings, item)
		},
	},
	"ConfigMap": {
		appendFunc: func(req *kargov1.ApplyKargoInstanceRequest, item *structpb.Struct) {
			req.Configmaps = append(req.Configmaps, item)
		},
	},
	"ProjectConfig": {
		appendFunc: func(req *kargov1.ApplyKargoInstanceRequest, item *structpb.Struct) {
			req.ProjectConfigs = append(req.ProjectConfigs, item)
		},
	},
	"MessageChannel": {
		appendFunc: func(req *kargov1.ApplyKargoInstanceRequest, item *structpb.Struct) {
			req.MessageChannels = append(req.MessageChannels, item)
		},
	},
	"ClusterMessageChannel": {
		appendFunc: func(req *kargov1.ApplyKargoInstanceRequest, item *structpb.Struct) {
			req.ClusterMessageChannels = append(req.ClusterMessageChannels, item)
		},
	},
	"EventRouter": {
		appendFunc: func(req *kargov1.ApplyKargoInstanceRequest, item *structpb.Struct) {
			req.EventRouters = append(req.EventRouters, item)
		},
	},
}

var kargoSupportedGroupKinds = map[schema.GroupKind]func(*unstructured.Unstructured) error{
	{Group: "kargo.akuity.io", Kind: "Project"}:                  nil,
	{Group: "kargo.akuity.io", Kind: "ProjectConfig"}:            nil,
	{Group: "kargo.akuity.io", Kind: "Warehouse"}:                nil,
	{Group: "kargo.akuity.io", Kind: "Stage"}:                    nil,
	{Group: "kargo.akuity.io", Kind: "PromotionTask"}:            nil,
	{Group: "kargo.akuity.io", Kind: "ClusterPromotionTask"}:     nil,
	{Group: "ee.kargo.akuity.io", Kind: "MessageChannel"}:        nil,
	{Group: "ee.kargo.akuity.io", Kind: "ClusterMessageChannel"}: nil,
	{Group: "ee.kargo.akuity.io", Kind: "EventRouter"}:           nil,
	{Group: "argoproj.io", Kind: "AnalysisTemplate"}:             nil,
	{Group: "rbac.authorization.k8s.io", Kind: "Role"}:           nil,
	{Group: "rbac.authorization.k8s.io", Kind: "RoleBinding"}:    nil,
	{Group: "", Kind: "ServiceAccount"}:                          nil,
	{Group: "", Kind: "ConfigMap"}:                               nil,
	{Group: "", Kind: "Secret"}: func(un *unstructured.Unstructured) error {
		if v, ok := un.GetLabels()["kargo.akuity.io/cred-type"]; !ok || v == "" {
			return errors.New("secret must have a kargo.akuity.io/cred-type label")
		}
		return nil
	},
}

func isKargoResourceValid(un *unstructured.Unstructured) error {
	if un == nil {
		return errors.New("unstructured is nil")
	}
	if un.GetName() == "" {
		return errors.New("name is required")
	}
	gk := schema.FromAPIVersionAndKind(un.GetAPIVersion(), un.GetKind()).GroupKind()
	validator, ok := kargoSupportedGroupKinds[gk]
	if !ok {
		return errors.New("unsupported kind")
	}
	if validator != nil {
		if err := validator(un); err != nil {
			return err
		}
	}
	return nil
}

func buildKargo(_ context.Context, diagnostics *diag.Diagnostics, kargo *types.KargoInstance, agentMaps *types.AgentMaps) *structpb.Struct {
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
	req := &kargov1.GetKargoInstanceRequest{
		OrganizationId: orgID,
		Name:           kargo.Name.ValueString(),
	}
	tflog.Debug(ctx, fmt.Sprintf("Get Kargo instance request: %s", req))
	resp, err := retryWithBackoff(ctx, func(ctx context.Context) (*kargov1.GetKargoInstanceResponse, error) {
		return cli.KargoCli.GetKargoInstance(ctx, req)
	}, "GetKargoInstance")
	if err != nil {
		return errors.Wrap(err, "Unable to read Kargo instance")
	}
	tflog.Debug(ctx, fmt.Sprintf("Get Kargo instance response: %s", resp))
	kargo.ID = tftypes.StringValue(resp.Instance.Id)
	if kargo.Workspace.IsNull() || kargo.Workspace.ValueString() == "" {
		workspace, wErr := getWorkspaceByID(ctx, cli.OrgCli, orgID, resp.Instance.WorkspaceId)
		if wErr == nil && workspace != nil {
			kargo.Workspace = tftypes.StringValue(workspace.GetName())
		}
	}

	agentMaps := buildAgentMaps(ctx, cli.KargoCli, kargo.ID.ValueString(), orgID, idv1.Type_ID)

	exportReq := &kargov1.ExportKargoInstanceRequest{
		OrganizationId: orgID,
		Id:             kargo.ID.ValueString(),
		WorkspaceId:    resp.Instance.WorkspaceId,
	}
	tflog.Debug(ctx, fmt.Sprintf("Export Kargo instance request: %s", exportReq))
	exportResp, err := retryWithBackoff(ctx, func(ctx context.Context) (*kargov1.ExportKargoInstanceResponse, error) {
		return cli.KargoCli.ExportKargoInstance(ctx, exportReq)
	}, "ExportKargoInstance")
	if err != nil {
		return errors.Wrap(err, "Unable to export Kargo instance")
	}
	tflog.Debug(ctx, fmt.Sprintf("Export Kargo instance response: %s", exportResp))
	return kargo.Update(ctx, diagnostics, exportResp, agentMaps, isDataSource)
}

func buildAgentMaps(ctx context.Context, client kargov1.KargoServiceGatewayClient, instanceID, orgID string, idType idv1.Type) *types.AgentMaps {
	if instanceID == "" || orgID == "" {
		return nil
	}
	if idType == idv1.Type_NAME {
		instance, err := retryWithBackoff(ctx, func(ctx context.Context) (*kargov1.GetKargoInstanceResponse, error) {
			return client.GetKargoInstance(ctx, &kargov1.GetKargoInstanceRequest{
				OrganizationId: orgID,
				Name:           instanceID,
			})
		}, "GetKargoInstance")
		if err != nil {
			tflog.Warn(ctx, fmt.Sprintf("Unable to get Kargo instance: %s", err))
			return nil
		}
		instanceID = instance.GetInstance().GetId()
	}
	agentsResp, err := retryWithBackoff(ctx, func(ctx context.Context) (*kargov1.ListKargoInstanceAgentsResponse, error) {
		return client.ListKargoInstanceAgents(ctx, &kargov1.ListKargoInstanceAgentsRequest{
			OrganizationId: orgID,
			InstanceId:     instanceID,
		})
	}, "ListKargoInstanceAgents")
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Unable to list Kargo agents for name<->ID mapping: %s", err))
		return nil
	}

	agentMaps := &types.AgentMaps{
		NameToID: make(map[string]string),
		IDToName: make(map[string]string),
	}

	for _, agent := range agentsResp.GetAgents() {
		name := agent.GetName()
		id := agent.GetId()
		if name != "" && id != "" {
			agentMaps.NameToID[name] = id
			agentMaps.IDToName[id] = name
		}
	}

	return agentMaps
}

func getWorkspace(ctx context.Context, orgc orgcv1.OrganizationServiceGatewayClient, orgid, name string) (*orgcv1.Workspace, error) {
	workspaces, err := retryWithBackoff(ctx, func(ctx context.Context) (*orgcv1.ListWorkspacesResponse, error) {
		return orgc.ListWorkspaces(ctx, &orgcv1.ListWorkspacesRequest{
			OrganizationId: orgid,
		})
	}, "ListWorkspaces")
	if err != nil {
		return nil, errors.Wrap(err, "unable to read org workspaces")
	}
	for _, w := range workspaces.GetWorkspaces() {
		if name == "" && w.IsDefault {
			return w, nil
		}
		if w.Name == name {
			return w, nil
		}
	}

	return nil, fmt.Errorf("workspace %s not found", name)
}

func getWorkspaceByID(ctx context.Context, orgc orgcv1.OrganizationServiceGatewayClient, orgid, id string) (*orgcv1.Workspace, error) {
	workspaces, err := retryWithBackoff(ctx, func(ctx context.Context) (*orgcv1.ListWorkspacesResponse, error) {
		return orgc.ListWorkspaces(ctx, &orgcv1.ListWorkspacesRequest{
			OrganizationId: orgid,
		})
	}, "ListWorkspaces")
	if err != nil {
		return nil, errors.Wrap(err, "unable to read org workspaces")
	}
	for _, w := range workspaces.GetWorkspaces() {
		if w.Id == id {
			return w, nil
		}
	}
	return nil, fmt.Errorf("workspace with id %s not found", id)
}
