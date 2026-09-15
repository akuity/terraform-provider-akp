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
	"google.golang.org/protobuf/types/known/timestamppb"

	kargov1 "github.com/akuity/api-client-go/pkg/api/gen/kargo/v1"
	healthv1 "github.com/akuity/api-client-go/pkg/api/gen/types/status/health/v1"
	reconv1 "github.com/akuity/api-client-go/pkg/api/gen/types/status/reconciliation/v1"
	"github.com/akuity/terraform-provider-akp/akp/types"
)

func NewAkpKargoAgentResource() resource.Resource {
	return &GenericResource[types.KargoAgent]{
		TypeNameSuffix:  "kargo_agent",
		SchemaFunc:      kargoAgentSchema,
		CreateFunc:      kargoAgentCreate,
		ReadFunc:        kargoAgentRead,
		UpdateFunc:      kargoAgentUpdate,
		DeleteFunc:      kargoAgentDelete,
		ImportStateFunc: importSplitID("instance_id", "name"),
		Validators:      []resource.ConfigValidator{kargoAgentConfigValidator{}},
	}
}

func kargoAgentCreate(ctx context.Context, cli *AkpCli, diags *diag.Diagnostics, plan *types.KargoAgent) (*types.KargoAgent, error) {
	return kargoAgentUpsert(ctx, cli, diags, plan, true)
}

func kargoAgentRead(ctx context.Context, cli *AkpCli, diags *diag.Diagnostics, data *types.KargoAgent) error {
	if data.Spec == nil {
		ctx = types.WithReadContext(ctx)
	}
	return refreshKargoAgentState(ctx, diags, cli, data, data)
}

func kargoAgentUpdate(ctx context.Context, cli *AkpCli, diags *diag.Diagnostics, plan *types.KargoAgent) (*types.KargoAgent, error) {
	return kargoAgentUpsert(ctx, cli, diags, plan, false)
}

func kargoAgentDelete(ctx context.Context, cli *AkpCli, _ *diag.Diagnostics, plan *types.KargoAgent) error {
	kubeconfig, err := getKubeconfig(ctx, plan.Kubeconfig)
	if err != nil {
		return fmt.Errorf("unable to get kubeconfig: %s", err)
	}

	workspaceID, _ := resolveKargoAgentWorkspace(ctx, cli, plan)

	if err := checkAgentIsNotDefaultShard(ctx, cli, plan); err != nil {
		return err
	}

	// Delete the manifests
	if kubeconfig != nil && plan.RemoveAgentResourcesOnDestroy.ValueBool() {
		manifests, _, err := getKargoManifests(ctx, cli.KargoCli, cli.OrgId, plan)
		if err != nil {
			return fmt.Errorf("unable to get kargo manifests: %s", err)
		}

		err = deleteManifests(ctx, manifests, kubeconfig)
		if err != nil {
			return fmt.Errorf("unable to delete manifests: %s", err)
		}
	}

	apiReq := &kargov1.DeleteInstanceAgentRequest{
		OrganizationId: cli.OrgId,
		InstanceId:     plan.InstanceID.ValueString(),
		Id:             plan.ID.ValueString(),
		WorkspaceId:    workspaceID,
	}

	err = deleteWithCooldown(ctx, func(ctx context.Context) (*kargov1.DeleteInstanceAgentResponse, error) {
		resp, err := cli.KargoCli.DeleteInstanceAgent(ctx, apiReq)
		if isGoneErr(err) {
			return resp, nil
		}
		return resp, err
	}, "DeleteInstanceAgent", 5*time.Second)
	if err != nil {
		return fmt.Errorf("unable to delete Kargo agent. %s", err)
	}
	return nil
}

func kargoAgentUpsert(ctx context.Context, cli *AkpCli, diagnostics *diag.Diagnostics, plan *types.KargoAgent, isCreate bool) (*types.KargoAgent, error) {
	validateKargoAgentConfig(diagnostics, plan)
	if diagnostics.HasError() {
		return nil, nil
	}

	workspace, err := getWorkspace(ctx, cli.OrgCli, cli.OrgId, plan.Workspace.ValueString())
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get workspace. %s", err))
		return nil, errors.New("unable to get workspace")
	}
	apiReq := buildKargoAgentApplyRequest(ctx, diagnostics, plan, cli.OrgId, workspace.Id)
	if diagnostics.HasError() {
		return nil, nil
	}
	result, err := applyKargoAgent(ctx, cli, plan, apiReq, isCreate)
	if err != nil {
		return result, err
	}

	if err := syncKargoAgentMaintenanceMode(ctx, cli, workspace.Id, plan); err != nil {
		return result, err
	}

	if plan.Workspace.ValueString() == "" {
		plan.Workspace = tftypes.StringValue(workspace.GetName())
	}
	if err := autoSetDefaultShardAgent(ctx, cli, result); err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Failed to auto-set defaultShardAgent: %s", err))
	}

	return result, refreshKargoAgentState(ctx, diagnostics, cli, result, plan)
}

func applyKargoAgent(ctx context.Context, cli *AkpCli, plan *types.KargoAgent, apiReq *kargov1.ApplyKargoInstanceRequest, isCreate bool) (*types.KargoAgent, error) {
	kubeconfig := plan.Kubeconfig
	plan.Kubeconfig = nil
	tflog.Debug(ctx, fmt.Sprintf("Apply Kargo agent request: %s", apiReq))

	_, err := retryWithBackoff(ctx, func(ctx context.Context) (*kargov1.ApplyKargoInstanceResponse, error) {
		return cli.KargoCli.ApplyKargoInstance(ctx, apiReq)
	}, "ApplyKargoInstance")
	if err != nil {
		return nil, fmt.Errorf("unable to create Kargo agent: %s", err)
	}

	if kubeconfig != nil {
		plan.Kubeconfig = kubeconfig
		shouldApply := isCreate || plan.ReapplyManifestsOnUpdate.ValueBool()
		if shouldApply {
			err = kargoAgentUpsertKubeConfig(ctx, cli, plan)
			if err != nil {
				// Ensure kubeconfig won't be committed to state by setting it to nil
				plan.Kubeconfig = nil
				return plan, fmt.Errorf("unable to apply kargo manifests: %s", err)
			}
		}
	}

	return plan, nil
}

func syncKargoAgentMaintenanceMode(ctx context.Context, cli *AkpCli, workspaceID string, plan *types.KargoAgent) error {
	if cli == nil || cli.KargoCli == nil || plan == nil || plan.Spec == nil {
		return nil
	}

	maintenanceModeConfigured := !plan.Spec.Data.MaintenanceMode.IsNull() && !plan.Spec.Data.MaintenanceMode.IsUnknown()
	maintenanceModeExpiryConfigured := isKnownNonEmptyString(plan.Spec.Data.MaintenanceModeExpiry)
	if !maintenanceModeConfigured && !maintenanceModeExpiryConfigured {
		return nil
	}

	req := &kargov1.SetAgentMaintenanceModeRequest{
		OrganizationId:  cli.OrgId,
		InstanceId:      plan.InstanceID.ValueString(),
		WorkspaceId:     workspaceID,
		AgentNames:      []string{plan.Name.ValueString()},
		MaintenanceMode: plan.Spec.Data.MaintenanceMode.ValueBool(),
	}
	if maintenanceModeExpiryConfigured {
		expiry, err := time.Parse(time.RFC3339, plan.Spec.Data.MaintenanceModeExpiry.ValueString())
		if err != nil {
			return fmt.Errorf("unable to parse maintenance_mode_expiry: %w", err)
		}
		req.Expiry = timestamppb.New(expiry)
	}

	_, err := retryWithBackoff(ctx, func(ctx context.Context) (*kargov1.SetAgentMaintenanceModeResponse, error) {
		return cli.KargoCli.SetAgentMaintenanceMode(ctx, req)
	}, "SetAgentMaintenanceMode")
	if err != nil {
		return fmt.Errorf("unable to set Kargo agent maintenance mode: %w", err)
	}
	return nil
}

func kargoAgentUpsertKubeConfig(ctx context.Context, cli *AkpCli, plan *types.KargoAgent) error {
	// Apply agent manifests to clusters if the kubeconfig is specified for cluster.
	kubeconfig, err := getKubeconfig(ctx, plan.Kubeconfig)
	if err != nil {
		return err
	}

	// Apply the manifests
	if kubeconfig != nil {
		manifests, id, err := getKargoManifests(ctx, cli.KargoCli, cli.OrgId, plan)
		if err != nil {
			return err
		}
		if plan.ID.ValueString() == "" {
			plan.ID = tftypes.StringValue(id)
		}

		err = applyManifests(ctx, manifests, kubeconfig)
		if err != nil {
			return err
		}
		return waitKargoAgentHealthStatus(ctx, cli.KargoCli, cli.OrgId, plan)
	}
	return nil
}

func refreshKargoAgentState(ctx context.Context, diagnostics *diag.Diagnostics, cli *AkpCli, kargoAgent, plan *types.KargoAgent) error {
	source := kargoAgent
	if preferredKargoAgentWorkspaceName(source) == "" && plan != nil {
		source = plan
	}
	workspaceID, workspaceName := resolveKargoAgentWorkspace(ctx, cli, source)
	agentID := kargoAgent.ID.ValueString()

	if agentID == "" {
		agent, err := findKargoAgentByName(ctx, cli.KargoCli, cli.OrgId, kargoAgent.InstanceID.ValueString(), workspaceID, kargoAgent.Name.ValueString())
		if err != nil {
			return err
		}
		agentID = agent.GetId()
	}

	resp, err := retryWithBackoff(ctx, func(ctx context.Context) (*kargov1.GetKargoInstanceAgentResponse, error) {
		return cli.KargoCli.GetKargoInstanceAgent(ctx, &kargov1.GetKargoInstanceAgentRequest{
			OrganizationId: cli.OrgId,
			InstanceId:     kargoAgent.InstanceID.ValueString(),
			Id:             agentID,
			WorkspaceId:    workspaceID,
		})
	}, "GetKargoInstanceAgent")
	if err != nil {
		return fmt.Errorf("unable to read Kargo agent: %w", err)
	}
	if resp.GetAgent() == nil {
		return status.Error(codes.NotFound, "Kargo agent not found")
	}

	tflog.Debug(ctx, fmt.Sprintf("current kargo agent: %s", resp.GetAgent()))
	kargoAgent.Update(ctx, diagnostics, resp.GetAgent(), plan)
	hydrateKargoAgentFieldsFromGet(kargoAgent, resp.GetAgent())
	types.NormalizeKargoAgentReadStateForRefresh(kargoAgent)
	if (kargoAgent.Workspace.IsNull() || kargoAgent.Workspace.ValueString() == "") && workspaceName != "" {
		kargoAgent.Workspace = tftypes.StringValue(workspaceName)
	}
	return nil
}

// resolveKargoAgentWorkspace picks the workspace to use for Kargo agent calls.
// It prefers the workspace name already stored on the agent (from state or
// plan) and resolves it by name, falling back to the instance's own workspace
// only when the name is absent — typically during `terraform import`, which
// seeds the resource with instance_id/name only.
func resolveKargoAgentWorkspace(ctx context.Context, cli *AkpCli, kargoAgent *types.KargoAgent) (string, string) {
	if cli == nil || kargoAgent == nil {
		return "", ""
	}

	if name := preferredKargoAgentWorkspaceName(kargoAgent); name != "" && cli.OrgCli != nil {
		ws, err := getWorkspace(ctx, cli.OrgCli, cli.OrgId, name)
		if err == nil && ws != nil {
			return ws.GetId(), ws.GetName()
		}
		tflog.Warn(ctx, fmt.Sprintf("Unable to resolve Kargo agent workspace %q by name, falling back to instance scan: %s", name, err))
	}

	return resolveKargoInstanceWorkspace(ctx, cli, kargoAgent.InstanceID.ValueString())
}

// preferredKargoAgentWorkspaceName returns the workspace name recorded on the
// agent, treating Null/Unknown/empty as absent.
func preferredKargoAgentWorkspaceName(kargoAgent *types.KargoAgent) string {
	if kargoAgent == nil {
		return ""
	}
	if kargoAgent.Workspace.IsNull() || kargoAgent.Workspace.IsUnknown() {
		return ""
	}
	return kargoAgent.Workspace.ValueString()
}

func resolveKargoInstanceWorkspace(ctx context.Context, cli *AkpCli, instanceID string) (string, string) {
	if cli == nil || cli.OrgCli == nil || cli.KargoCli == nil || instanceID == "" {
		return "", ""
	}
	instance, err := getKargoInstanceByID(ctx, cli, instanceID)
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Unable to resolve Kargo instance workspace: %s", err))
		return "", ""
	}
	if instance.GetWorkspaceId() == "" {
		return "", ""
	}
	workspace, err := getWorkspaceByID(ctx, cli.OrgCli, cli.OrgId, instance.GetWorkspaceId())
	if err != nil {
		tflog.Warn(ctx, fmt.Sprintf("Unable to resolve Kargo instance workspace: %s", err))
		return "", ""
	}
	return workspace.GetId(), workspace.GetName()
}

// findKargoAgentByName returns the instance's agent called name, or a NotFound status.
func findKargoAgentByName(ctx context.Context, client kargov1.KargoServiceGatewayClient, orgID, instanceID, workspaceID, name string) (*kargov1.KargoAgent, error) {
	agents, err := retryWithBackoff(ctx, func(ctx context.Context) (*kargov1.ListKargoInstanceAgentsResponse, error) {
		return client.ListKargoInstanceAgents(ctx, &kargov1.ListKargoInstanceAgentsRequest{
			OrganizationId: orgID,
			InstanceId:     instanceID,
			WorkspaceId:    workspaceID,
		})
	}, "ListKargoInstanceAgents")
	if err != nil {
		return nil, fmt.Errorf("unable to list Kargo agents: %w", err)
	}
	for _, a := range agents.GetAgents() {
		if a.GetName() == name {
			return a, nil
		}
	}
	return nil, status.Errorf(codes.NotFound, "Kargo agent %q not found", name)
}

// hydrateKargoAgentFieldsFromGet fills fields that Update leaves empty (e.g. on
// import, where there is no plan to take them from) using the agent already
// returned by GetKargoInstanceAgent, so the read path does not need any extra
// API calls. Fields the user has set are never overwritten.
func hydrateKargoAgentFieldsFromGet(kargoAgent *types.KargoAgent, agent *kargov1.KargoAgent) {
	if kargoAgent == nil || kargoAgent.Spec == nil || agent == nil {
		return
	}

	data := agent.GetData()
	if kargoAgent.Spec.Data.ArgocdNamespace.ValueString() == "" && data.GetArgocdNamespace() != "" {
		kargoAgent.Spec.Data.ArgocdNamespace = tftypes.StringValue(data.GetArgocdNamespace())
	}

	if kargoAgent.Spec.Data.MaintenanceModeExpiry.ValueString() == "" && data.GetMaintenanceModeExpiry() != nil {
		// Match the RFC3339 seconds-precision format the field uses everywhere else.
		kargoAgent.Spec.Data.MaintenanceModeExpiry = tftypes.StringValue(
			data.GetMaintenanceModeExpiry().AsTime().Format(time.RFC3339))
	}
}

func buildKargoAgentApplyRequest(ctx context.Context, diagnostics *diag.Diagnostics, kargoAgent *types.KargoAgent, orgId, workspaceId string) *kargov1.ApplyKargoInstanceRequest {
	applyReq := &kargov1.ApplyKargoInstanceRequest{
		OrganizationId: orgId,
		Id:             kargoAgent.InstanceID.ValueString(),
		WorkspaceId:    workspaceId,
		Agents:         buildKargoAgents(ctx, diagnostics, kargoAgent),
	}
	return applyReq
}

func buildKargoAgents(ctx context.Context, diagnostics *diag.Diagnostics, kargoAgent *types.KargoAgent) []*structpb.Struct {
	var labels map[string]string
	var annotations map[string]string
	diagnostics.Append(kargoAgent.Labels.ElementsAs(ctx, &labels, true)...)
	diagnostics.Append(kargoAgent.Annotations.ElementsAs(ctx, &annotations, true)...)

	rawMap := types.TFToMapWithOverrides(kargoAgent.Spec, types.KargoOverridesMap, types.KargoRenamesMap)
	if rawMap == nil {
		diagnostics.AddError("Client Error", "Unable to convert Kargo agent to map")
		return nil
	}
	pruneNormalizedEmptyKargoAgentFields(rawMap)

	metadata := map[string]any{
		"name":      kargoAgent.Name.ValueString(),
		"namespace": kargoAgent.Namespace.ValueString(),
	}
	if len(labels) > 0 {
		labelsAny := make(map[string]any, len(labels))
		for k, v := range labels {
			labelsAny[k] = v
		}
		metadata["labels"] = labelsAny
	}
	if len(annotations) > 0 {
		annotationsAny := make(map[string]any, len(annotations))
		for k, v := range annotations {
			annotationsAny[k] = v
		}
		metadata["annotations"] = annotationsAny
	}
	rawMap = map[string]any{
		"kind":       "KargoAgent",
		"apiVersion": "kargo.akuity.io/v1alpha1",
		"metadata":   metadata,
		"spec":       rawMap,
	}

	s, err := structpb.NewStruct(rawMap)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Kargo agent. %s", err))
		return nil
	}
	return []*structpb.Struct{s}
}

func pruneNormalizedEmptyKargoAgentFields(rawMap map[string]any) {
	pruneNormalizedEmptyFields(rawMap, "argocdNamespace", "maintenanceModeExpiry")

	// The control plane rejects `size` for Akuity-managed agents because the
	// size is owned by AIMS. Drop it so reads from state (which may carry a
	// server-computed size) do not leak back into apply payloads.
	dataMap, _ := rawMap["data"].(map[string]any)
	if akuityManaged, _ := dataMap["akuityManaged"].(bool); akuityManaged {
		delete(dataMap, "size")
	}
}

type kargoAgentConfigValidator struct{}

func (v kargoAgentConfigValidator) Description(context.Context) string {
	return "Validates Kargo agent field combinations that the control plane normalizes or rejects"
}

func (v kargoAgentConfigValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v kargoAgentConfigValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	data, ok := configDataAttributes(ctx, req, resp)
	if !ok {
		return
	}
	validateKargoAgentConfigValues(
		&resp.Diagnostics,
		path.Root("spec").AtName("data"),
		data["argocd_namespace"].(tftypes.String),
		data["remote_argocd"].(tftypes.String),
		data["akuity_managed"].(tftypes.Bool),
		data["maintenance_mode"].(tftypes.Bool),
		data["maintenance_mode_expiry"].(tftypes.String),
		data["size"].(tftypes.String),
	)
}

func validateKargoAgentConfig(diagnostics *diag.Diagnostics, plan *types.KargoAgent) {
	if diagnostics == nil || plan == nil || plan.Spec == nil {
		return
	}

	validateKargoAgentConfigValues(
		diagnostics,
		path.Root("spec").AtName("data"),
		plan.Spec.Data.ArgocdNamespace,
		plan.Spec.Data.RemoteArgocd,
		plan.Spec.Data.AkuityManaged,
		plan.Spec.Data.MaintenanceMode,
		plan.Spec.Data.MaintenanceModeExpiry,
		plan.Spec.Data.Size,
	)
}

func validateKargoAgentConfigValues(
	diagnostics *diag.Diagnostics,
	dataPath path.Path,
	argocdNamespace tftypes.String,
	remoteArgocd tftypes.String,
	akuityManaged tftypes.Bool,
	maintenanceMode tftypes.Bool,
	maintenanceModeExpiry tftypes.String,
	size tftypes.String,
) {
	if diagnostics == nil {
		return
	}

	argocdNamespaceConfigured := isKnownNonEmptyString(argocdNamespace)
	remoteArgocdConfigured := isKnownNonEmptyString(remoteArgocd)
	akuityManagedEnabled := isKnownTrueBool(akuityManaged)
	maintenanceModeMissingOrDisabled := maintenanceMode.IsNull() || (!maintenanceMode.IsUnknown() && !maintenanceMode.ValueBool())
	maintenanceModeExpiryConfigured := isKnownNonEmptyString(maintenanceModeExpiry)
	sizeConfigured := isKnownNonEmptyString(size)

	if argocdNamespaceConfigured && (remoteArgocdConfigured || akuityManagedEnabled) {
		diagnostics.AddAttributeError(
			dataPath.AtName("argocd_namespace"),
			"Invalid argocd_namespace",
			"argocd_namespace can only be configured for self-managed Kargo agents without remote_argocd or akuity_managed enabled.",
		)
	}

	if maintenanceModeExpiryConfigured && maintenanceModeMissingOrDisabled {
		diagnostics.AddAttributeError(
			dataPath.AtName("maintenance_mode_expiry"),
			"Invalid maintenance_mode_expiry",
			"maintenance_mode_expiry requires maintenance_mode = true. The control plane clears the expiry when maintenance mode is disabled.",
		)
	}

	if sizeConfigured && akuityManagedEnabled {
		diagnostics.AddAttributeError(
			dataPath.AtName("size"),
			"Invalid size",
			"size must be omitted when akuity_managed = true. The size of an Akuity-managed Kargo agent is managed by Akuity; change it through the Akuity UI or the AIMS API instead.",
		)
	}
}

func isKnownNonEmptyString(value tftypes.String) bool {
	return !value.IsNull() && !value.IsUnknown() && value.ValueString() != ""
}

func isKnownTrueBool(value tftypes.Bool) bool {
	return !value.IsNull() && !value.IsUnknown() && value.ValueBool()
}

func getKargoManifests(ctx context.Context, client kargov1.KargoServiceGatewayClient, orgId string, kargoAgent *types.KargoAgent) (string, string, error) {
	agent, err := findKargoAgentByName(ctx, client, orgId, kargoAgent.InstanceID.ValueString(), "", kargoAgent.Name.ValueString())
	if err != nil {
		return "", "", err
	}

	k, err := waitKargoAgentReconStatus(ctx, client, agent, orgId, kargoAgent.InstanceID.ValueString())
	if err != nil {
		return "", "", fmt.Errorf("unable to check kargo agent health status: %w", err)
	}
	apiReq := &kargov1.GetKargoInstanceAgentManifestsRequest{
		OrganizationId: orgId,
		InstanceId:     kargoAgent.InstanceID.ValueString(),
		Id:             k.Id,
	}
	resChan, errChan, err := client.GetKargoInstanceAgentManifests(ctx, apiReq)
	if err != nil {
		return "", "", fmt.Errorf("unable to download manifests: %w", err)
	}
	res, err := readStream(resChan, errChan)
	if err != nil {
		return "", "", fmt.Errorf("unable to parse manifests: %w", err)
	}

	return string(res), k.Id, nil
}

func waitKargoAgentHealthStatus(ctx context.Context, client kargov1.KargoServiceGatewayClient, orgID string, c *types.KargoAgent) error {
	getResourceFunc := func(ctx context.Context) (*kargov1.GetKargoInstanceAgentResponse, error) {
		return retryWithBackoff(ctx, func(ctx context.Context) (*kargov1.GetKargoInstanceAgentResponse, error) {
			return client.GetKargoInstanceAgent(ctx, &kargov1.GetKargoInstanceAgentRequest{
				OrganizationId: orgID,
				InstanceId:     c.InstanceID.ValueString(),
				Id:             c.ID.ValueString(),
			})
		}, "GetKargoInstanceAgent")
	}

	getStatusFunc := func(resp *kargov1.GetKargoInstanceAgentResponse) healthv1.StatusCode {
		if resp == nil || resp.Agent == nil {
			return healthv1.StatusCode_STATUS_CODE_UNKNOWN
		}
		return resp.Agent.GetHealthStatus().GetCode()
	}

	return waitForStatus(
		ctx,
		getResourceFunc,
		getStatusFunc,
		[]healthv1.StatusCode{healthv1.StatusCode_STATUS_CODE_HEALTHY, healthv1.StatusCode_STATUS_CODE_DEGRADED},
		5*time.Second,
		5*time.Minute,
		fmt.Sprintf("KargoAgent %s", c.Name.ValueString()),
		"health",
	)
}

func waitKargoAgentReconStatus(ctx context.Context, client kargov1.KargoServiceGatewayClient, kargoAgent *kargov1.KargoAgent, orgId, instanceId string) (*kargov1.KargoAgent, error) {
	// Capture the last seen agent so the caller can use it after wait completes.
	var lastAgent *kargov1.KargoAgent

	getResourceFunc := func(ctx context.Context) (*kargov1.GetKargoInstanceAgentResponse, error) {
		resp, err := retryWithBackoff(ctx, func(ctx context.Context) (*kargov1.GetKargoInstanceAgentResponse, error) {
			return client.GetKargoInstanceAgent(ctx, &kargov1.GetKargoInstanceAgentRequest{
				OrganizationId: orgId,
				InstanceId:     instanceId,
				Id:             kargoAgent.Id,
			})
		}, "GetKargoInstanceAgent")
		if err == nil && resp != nil {
			lastAgent = resp.GetAgent()
		}
		return resp, err
	}

	getStatusFunc := func(resp *kargov1.GetKargoInstanceAgentResponse) reconv1.StatusCode {
		if resp == nil || resp.Agent == nil {
			return reconv1.StatusCode_STATUS_CODE_UNSPECIFIED
		}
		return resp.Agent.GetReconciliationStatus().GetCode()
	}

	err := waitForStatus(
		ctx,
		getResourceFunc,
		getStatusFunc,
		[]reconv1.StatusCode{reconv1.StatusCode_STATUS_CODE_SUCCESSFUL, reconv1.StatusCode_STATUS_CODE_FAILED},
		5*time.Second,
		5*time.Minute,
		fmt.Sprintf("KargoAgent %s", kargoAgent.GetName()),
		"reconciliation",
	)
	if err != nil {
		return nil, err
	}
	if lastAgent != nil {
		return lastAgent, nil
	}
	return kargoAgent, nil
}

func autoSetDefaultShardAgent(ctx context.Context, cli *AkpCli, agent *types.KargoAgent) error {
	instance, err := getKargoInstanceByID(ctx, cli, agent.InstanceID.ValueString())
	if err != nil {
		return err
	}
	if instance.GetSpec().GetDefaultShardAgent() != "" {
		return nil
	}
	kargoAgent, err := findKargoAgentByName(ctx, cli.KargoCli, cli.OrgId, agent.InstanceID.ValueString(), "", agent.Name.ValueString())
	if err != nil {
		return err
	}
	if err := setDefaultShardAgent(ctx, cli, agent.InstanceID.ValueString(), kargoAgent.GetId()); err != nil {
		return err
	}
	tflog.Info(ctx, fmt.Sprintf("Auto-set defaultShardAgent to '%s'", agent.Name.ValueString()))
	return nil
}

func checkAgentIsNotDefaultShard(ctx context.Context, cli *AkpCli, agent *types.KargoAgent) error {
	instance, err := getKargoInstanceByID(ctx, cli, agent.InstanceID.ValueString())
	if status.Code(err) == codes.NotFound {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to get Kargo instance: %w", err)
	}
	if instance.GetSpec().GetDefaultShardAgent() == agent.ID.ValueString() {
		return fmt.Errorf("unable to delete Kargo agent: cannot delete default shard agent, change default shard before deleting")
	}
	return nil
}
