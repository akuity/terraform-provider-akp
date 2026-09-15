package akp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"google.golang.org/genproto/googleapis/api/httpbody"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/yaml"

	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	idv1 "github.com/akuity/api-client-go/pkg/api/gen/types/id/v1"
	healthv1 "github.com/akuity/api-client-go/pkg/api/gen/types/status/health/v1"
	reconv1 "github.com/akuity/api-client-go/pkg/api/gen/types/status/reconciliation/v1"
	"github.com/akuity/terraform-provider-akp/akp/kube"
	"github.com/akuity/terraform-provider-akp/akp/types"
)

func NewAkpClusterResource() resource.Resource {
	return &GenericResource[types.Cluster]{
		TypeNameSuffix:  "cluster",
		SchemaFunc:      clusterSchema,
		CreateFunc:      clusterCreate,
		ReadFunc:        clusterRead,
		UpdateFunc:      clusterUpdate,
		DeleteFunc:      clusterDelete,
		ImportStateFunc: importSplitID("instance_id", "name"),
		Validators: []resource.ConfigValidator{
			// auto_agent_size_config and custom_agent_size_config are mutually exclusive
			resourcevalidator.Conflicting(
				path.MatchRoot("spec").AtName("data").AtName("auto_agent_size_config"),
				path.MatchRoot("spec").AtName("data").AtName("custom_agent_size_config"),
			),
			clusterConfigValidator{},
			// Use custom validator to handle size-specific logic
			&sizeConfigValidator{},
		},
	}
}

func clusterCreate(ctx context.Context, cli *AkpCli, diags *diag.Diagnostics, plan *types.Cluster) (*types.Cluster, error) {
	result, err := clusterUpsert(ctx, cli, diags, plan, true)
	if err != nil {
		diags.AddError("Client Error", err.Error())
		tflog.Warn(ctx, fmt.Sprintf("refreshClusterState failed during create, cleaning up cluster %s", plan.Name.ValueString()))
		cleanupErr := deleteCluster(ctx, cli, plan, plan.RemoveAgentResourcesOnDestroy.ValueBool(), true)
		if cleanupErr != nil {
			tflog.Error(ctx, fmt.Sprintf("Failed to clean up dangling cluster %s: %v", plan.Name.ValueString(), cleanupErr))
		}
		tflog.Info(ctx, fmt.Sprintf("Successfully cleaned up dangling cluster %s", plan.Name.ValueString()))
		return nil, nil // Return nil error - we already added the diagnostic
	}
	return result, nil
}

func clusterRead(ctx context.Context, cli *AkpCli, diags *diag.Diagnostics, data *types.Cluster) error {
	if data.Spec == nil {
		ctx = types.WithReadContext(ctx)
	}
	return refreshClusterState(ctx, diags, cli.Cli, data, cli.OrgId, data)
}

func clusterUpdate(ctx context.Context, cli *AkpCli, diags *diag.Diagnostics, plan *types.Cluster) (*types.Cluster, error) {
	result, err := clusterUpsert(ctx, cli, diags, plan, false)
	if err != nil {
		diags.AddError("Client Error", err.Error())
		return result, nil // Return nil error - we already added the diagnostic. Return result to commit partial state.
	}
	return result, nil
}

func clusterDelete(ctx context.Context, cli *AkpCli, _ *diag.Diagnostics, plan *types.Cluster) error {
	return deleteCluster(ctx, cli, plan, plan.RemoveAgentResourcesOnDestroy.ValueBool(), false)
}

func clusterUpsert(ctx context.Context, cli *AkpCli, diagnostics *diag.Diagnostics, plan *types.Cluster, isCreate bool) (*types.Cluster, error) {
	validateClusterConfig(diagnostics, plan)
	if diagnostics.HasError() {
		return nil, nil
	}

	var akiSanitized bool
	if plan != nil && plan.Spec != nil && !plan.Spec.Data.MultiClusterK8SDashboardEnabled.IsNull() && !plan.Spec.Data.MultiClusterK8SDashboardEnabled.IsUnknown() && plan.Spec.Data.MultiClusterK8SDashboardEnabled.ValueBool() {
		instReq := &argocdv1.GetInstanceRequest{
			OrganizationId: cli.OrgId,
			Id:             plan.InstanceID.ValueString(),
			IdType:         idv1.Type_ID,
		}
		instResp, err := retryWithBackoff(ctx, func(ctx context.Context) (*argocdv1.GetInstanceResponse, error) {
			return cli.Cli.GetInstance(ctx, instReq)
		}, "GetInstance")
		if err != nil {
			return nil, fmt.Errorf("unable to get Argo CD instance for capability check: %w", err)
		}

		instSpec := instResp.GetInstance().GetSpec()
		enabled := instSpec != nil && instSpec.MultiClusterK8SDashboardEnabled

		if !enabled {
			akiSanitized = true
			plan.Spec.Data.MultiClusterK8SDashboardEnabled = tftypes.BoolValue(false)
		}
	}

	apiReq := buildClusterApplyRequest(ctx, diagnostics, plan, cli.OrgId)
	if diagnostics.HasError() {
		return nil, nil
	}
	result, err := applyCluster(ctx, cli, plan, apiReq, isCreate, cli.Cli.ApplyInstance, clusterUpsertKubeConfig, clusterWaitForReconciliation, clusterWaitForHealth)
	if result != nil && akiSanitized {
		plan.Spec.Data.MultiClusterK8SDashboardEnabled = tftypes.BoolValue(true)
	}
	if err == nil {
		if syncErr := syncClusterMaintenanceMode(ctx, cli, plan); syncErr != nil {
			err = syncErr
		}
	}
	// Always refresh cluster state to ensure we have consistent state, even if kubeconfig application failed
	if result != nil {
		refreshErr := refreshClusterState(ctx, diagnostics, cli.Cli, result, cli.OrgId, plan)
		if refreshErr != nil && err == nil {
			// If we didn't have an error before but refresh failed, return the refresh error
			return result, refreshErr
		}
	}
	return result, err
}

func syncClusterMaintenanceMode(ctx context.Context, cli *AkpCli, plan *types.Cluster) error {
	if cli == nil || cli.Cli == nil || cli.OrgCli == nil || plan == nil || plan.Spec == nil {
		return nil
	}

	maintenanceModeConfigured := !plan.Spec.Data.MaintenanceMode.IsNull() && !plan.Spec.Data.MaintenanceMode.IsUnknown()
	maintenanceModeExpiryConfigured := isKnownNonEmptyString(plan.Spec.Data.MaintenanceModeExpiry)
	if !maintenanceModeConfigured && !maintenanceModeExpiryConfigured {
		return nil
	}

	workspaceID, err := resolveClusterWorkspaceID(ctx, cli, plan.InstanceID.ValueString())
	if err != nil {
		return fmt.Errorf("unable to resolve cluster workspace: %w", err)
	}

	req := &argocdv1.SetClusterMaintenanceModeRequest{
		OrganizationId:  cli.OrgId,
		InstanceId:      plan.InstanceID.ValueString(),
		WorkspaceId:     workspaceID,
		ClusterNames:    []string{plan.Name.ValueString()},
		MaintenanceMode: plan.Spec.Data.MaintenanceMode.ValueBool(),
	}
	if maintenanceModeExpiryConfigured {
		expiry, err := time.Parse(time.RFC3339, plan.Spec.Data.MaintenanceModeExpiry.ValueString())
		if err != nil {
			return fmt.Errorf("unable to parse maintenance_mode_expiry: %w", err)
		}
		req.Expiry = timestamppb.New(expiry)
	}

	_, err = retryWithBackoff(ctx, func(ctx context.Context) (*argocdv1.SetClusterMaintenanceModeResponse, error) {
		return cli.Cli.SetClusterMaintenanceMode(ctx, req)
	}, "SetClusterMaintenanceMode")
	if err != nil {
		return fmt.Errorf("unable to set cluster maintenance mode: %w", err)
	}
	return nil
}

func resolveClusterWorkspaceID(ctx context.Context, cli *AkpCli, instanceID string) (string, error) {
	if cli == nil || cli.Cli == nil || cli.OrgCli == nil {
		return "", fmt.Errorf("client is not configured")
	}

	resp, err := retryWithBackoff(ctx, func(ctx context.Context) (*argocdv1.GetInstanceResponse, error) {
		return cli.Cli.GetInstance(ctx, &argocdv1.GetInstanceRequest{
			OrganizationId: cli.OrgId,
			Id:             instanceID,
			IdType:         idv1.Type_ID,
		})
	}, "GetInstance")
	if err != nil {
		return "", err
	}

	if workspaceID := resp.GetInstance().GetWorkspaceId(); workspaceID != "" {
		return workspaceID, nil
	}

	workspace, err := getWorkspace(ctx, cli.OrgCli, cli.OrgId, "")
	if err != nil {
		return "", err
	}
	return workspace.GetId(), nil
}

func applyCluster(
	ctx context.Context,
	cli *AkpCli,
	plan *types.Cluster,
	apiReq *argocdv1.ApplyInstanceRequest,
	isCreate bool,
	applyInstance func(context.Context, *argocdv1.ApplyInstanceRequest) (*argocdv1.ApplyInstanceResponse, error),
	upsertKubeConfig func(ctx context.Context, cli *AkpCli, plan *types.Cluster) error,
	waitForReconciliation func(ctx context.Context, cli *AkpCli, plan *types.Cluster) error,
	waitForHealth func(ctx context.Context, cli *AkpCli, plan *types.Cluster) error,
) (*types.Cluster, error) {
	kubeconfig := plan.Kubeconfig
	plan.Kubeconfig = nil
	// A failed create must not leave a dangling cluster behind in the API.
	cleanupOnCreateFailure := func(err error) error {
		if !isCreate {
			return err
		}
		tflog.Warn(ctx, fmt.Sprintf("create failed, cleaning up cluster %s: %v", plan.Name.ValueString(), err))
		if cleanupErr := deleteCluster(ctx, cli, plan, plan.RemoveAgentResourcesOnDestroy.ValueBool(), true); cleanupErr != nil {
			return fmt.Errorf("%s (and failed to clean up cluster: %s)", err, cleanupErr)
		}
		return err
	}
	tflog.Debug(ctx, fmt.Sprintf("Apply cluster request: %s", apiReq))
	_, err := retryWithBackoff(ctx, func(ctx context.Context) (*argocdv1.ApplyInstanceResponse, error) {
		return applyInstance(ctx, apiReq)
	}, "ApplyInstance")
	if err != nil {
		return nil, cleanupOnCreateFailure(fmt.Errorf("unable to create Argo CD instance: %s", err))
	}

	if err := waitForReconciliation(ctx, cli, plan); err != nil {
		return nil, cleanupOnCreateFailure(fmt.Errorf("cluster reconciliation failed: %w", err))
	}

	if kubeconfig != nil {
		plan.Kubeconfig = kubeconfig
		shouldApply := isCreate || plan.ReapplyManifestsOnUpdate.ValueBool()
		if shouldApply {
			if err := upsertKubeConfig(ctx, cli, plan); err != nil {
				err = fmt.Errorf("unable to apply manifests: %s", err)
				if isCreate {
					return nil, cleanupOnCreateFailure(err)
				}
				plan.Kubeconfig = nil
				return plan, err
			}
		} else {
			if err := waitForHealth(ctx, cli, plan); err != nil {
				return plan, fmt.Errorf("cluster health check failed: %w", err)
			}
		}
	} else {
		agentWillAutoUpdate := !isCreate && plan.Spec != nil && !plan.Spec.Data.AutoUpgradeDisabled.ValueBool()
		if agentWillAutoUpdate {
			if err := waitForHealth(ctx, cli, plan); err != nil {
				return plan, fmt.Errorf("cluster health check failed: %w", err)
			}
		}
	}

	return plan, nil
}

func clusterUpsertKubeConfig(ctx context.Context, cli *AkpCli, plan *types.Cluster) error {
	// Apply agent manifests to clusters if the kubeconfig is specified for cluster.
	kubeconfig, err := getKubeconfig(ctx, plan.Kubeconfig)
	if err != nil {
		return err
	}

	// Apply the manifests
	if kubeconfig != nil {
		manifests, err := getManifests(ctx, cli.Cli, cli.OrgId, plan)
		if err != nil {
			return err
		}

		err = applyManifests(ctx, manifests, kubeconfig)
		if err != nil {
			return err
		}
		return waitClusterHealthStatus(ctx, cli.Cli, cli.OrgId, plan, plan.EnsureHealthy.ValueBool())
	}
	return nil
}

func clusterWaitForReconciliation(ctx context.Context, cli *AkpCli, plan *types.Cluster) error {
	return waitForClusterReconciliation(ctx, cli.Cli, cli.OrgId, plan)
}

func clusterWaitForHealth(ctx context.Context, cli *AkpCli, plan *types.Cluster) error {
	return waitClusterHealthStatus(ctx, cli.Cli, cli.OrgId, plan, plan.EnsureHealthy.ValueBool())
}

func refreshClusterState(ctx context.Context, diagnostics *diag.Diagnostics, client argocdv1.ArgoCDServiceGatewayClient, cluster *types.Cluster,
	orgID string, plan *types.Cluster,
) error {
	clusterReq := &argocdv1.GetInstanceClusterRequest{
		OrganizationId: orgID,
		InstanceId:     cluster.InstanceID.ValueString(),
		Id:             cluster.Name.ValueString(),
		IdType:         idv1.Type_NAME,
	}

	tflog.Debug(ctx, fmt.Sprintf("Get cluster request: %s", clusterReq))
	clusterResp, err := retryWithBackoff(ctx, func(ctx context.Context) (*argocdv1.GetInstanceClusterResponse, error) {
		return client.GetInstanceCluster(ctx, clusterReq)
	}, "GetInstanceCluster")
	if err != nil {
		err = normalizeMissingClusterReadError(ctx, client, orgID, cluster.InstanceID.ValueString(), cluster.Name.ValueString(), err)
		return fmt.Errorf("unable to read Argo CD cluster: %w", err)
	}
	tflog.Debug(ctx, fmt.Sprintf("Get cluster response: %s", clusterResp))

	apiCluster := clusterResp.GetCluster()

	if plan != nil && plan.Spec != nil && (!plan.Spec.Data.MultiClusterK8SDashboardEnabled.IsNull() || !plan.Spec.Data.MultiClusterK8SDashboardEnabled.IsUnknown()) && plan.Spec.Data.MultiClusterK8SDashboardEnabled.ValueBool() && !apiCluster.GetData().GetMultiClusterK8SDashboardEnabled() {
		diagnostics.AddWarning("multi_cluster_k8s_dashboard_enabled ignored", "The Akuity Intelligence feature cannot be set, it's possible that it's not enabled for your Argo CD instance. Please enable the feature on instance level first before enabling it on cluster level.")
		if apiCluster != nil && apiCluster.Data != nil {
			apiCluster.Data.MultiClusterK8SDashboardEnabled = plan.Spec.Data.MultiClusterK8SDashboardEnabled.ValueBoolPointer()
		}
	}

	cluster.Update(ctx, diagnostics, apiCluster, plan)
	return nil
}

func normalizeMissingClusterReadError(
	ctx context.Context,
	client argocdv1.ArgoCDServiceGatewayClient,
	orgID string,
	instanceID string,
	clusterName string,
	err error,
) error {
	if status.Code(err) != codes.PermissionDenied {
		return err
	}

	_, instanceErr := retryWithBackoff(ctx, func(ctx context.Context) (*argocdv1.GetInstanceResponse, error) {
		return client.GetInstance(ctx, &argocdv1.GetInstanceRequest{
			OrganizationId: orgID,
			Id:             instanceID,
			IdType:         idv1.Type_ID,
		})
	}, "GetInstance")
	if instanceErr == nil {
		return status.Errorf(codes.NotFound, "cluster %q not found", clusterName)
	}

	return err
}

func buildClusterApplyRequest(ctx context.Context, diagnostics *diag.Diagnostics, cluster *types.Cluster, orgId string) *argocdv1.ApplyInstanceRequest {
	applyReq := &argocdv1.ApplyInstanceRequest{
		OrganizationId: orgId,
		IdType:         idv1.Type_ID,
		Id:             cluster.InstanceID.ValueString(),
		Clusters:       buildClusters(ctx, diagnostics, cluster),
	}
	return applyReq
}

func buildClusters(ctx context.Context, diagnostics *diag.Diagnostics, cluster *types.Cluster) []*structpb.Struct {
	var labels map[string]string
	var annotations map[string]string
	diagnostics.Append(cluster.Labels.ElementsAs(ctx, &labels, true)...)
	diagnostics.Append(cluster.Annotations.ElementsAs(ctx, &annotations, true)...)

	// Validate directClusterSpec if present
	if cluster.Spec.Data.DirectClusterSpec != nil {
		clusterType := cluster.Spec.Data.DirectClusterSpec.ClusterType.ValueString()
		if clusterType == "" || clusterType != types.DirectClusterTypeString[argocdv1.DirectClusterType_DIRECT_CLUSTER_TYPE_KARGO] {
			diagnostics.AddError("unsupported cluster type", fmt.Sprintf("cluster_type %s is not supported, supported cluster_type: `kargo`", cluster.Spec.Data.DirectClusterSpec.ClusterType.String()))
			return nil
		}
	}

	rawMap := types.TFToMapWithOverrides(cluster.Spec, types.OverridesMap, types.RenamesMap)
	if rawMap == nil {
		diagnostics.AddError("Client Error", "Unable to convert cluster spec to map")
		return nil
	}
	pruneNormalizedEmptyFields(rawMap, "maintenanceModeExpiry")

	clusterSize := cluster.Spec.Data.Size.ValueString()
	var kustomizationStr string
	if clusterSize == "custom" {
		var err error
		kustomizationStr, err = types.GenerateExpectedKustomization(cluster.Spec.Data.CustomAgentSizeConfig, cluster.Spec.Data.Kustomization.ValueString())
		if err != nil {
			diagnostics.AddError("failed to generate expected kustomization", err.Error())
			return nil
		}
	} else {
		kustomizationStr = cluster.Spec.Data.Kustomization.ValueString()
	}

	if kustomizationStr != "" {
		jsonBytes, err := yaml.YAMLToJSON([]byte(kustomizationStr))
		if err != nil {
			diagnostics.AddError("failed to convert kustomization YAML to JSON", err.Error())
			return nil
		}
		var kustomizationObj map[string]any
		if err := json.Unmarshal(jsonBytes, &kustomizationObj); err != nil {
			diagnostics.AddError("failed to unmarshal kustomization JSON", err.Error())
			return nil
		}
		if dataMap, ok := rawMap["data"].(map[string]any); ok {
			dataMap["kustomization"] = kustomizationObj
		}
	} else if !cluster.Spec.Data.Kustomization.IsNull() && !cluster.Spec.Data.Kustomization.IsUnknown() {
		if dataMap, ok := rawMap["data"].(map[string]any); ok {
			dataMap["kustomization"] = map[string]any{}
		}
	}

	metadata := map[string]any{
		"name":      cluster.Name.ValueString(),
		"namespace": cluster.Namespace.ValueString(),
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
		"kind":       "Cluster",
		"apiVersion": "argocd.akuity.io/v1alpha1",
		"metadata":   metadata,
		"spec":       rawMap,
	}

	s, err := structpb.NewStruct(rawMap)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Cluster. %s", err))
		return nil
	}
	return []*structpb.Struct{s}
}

func getKubeconfig(ctx context.Context, kubeConfig *types.Kubeconfig) (*rest.Config, error) {
	if kubeConfig == nil {
		return nil, nil
	}
	kcfg, err := kube.InitializeConfiguration(ctx, kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("cannot initialize the kubernetes client, please check the kubernetes configuration: %w", err)
	}
	return kcfg, nil
}

func getManifests(ctx context.Context, client argocdv1.ArgoCDServiceGatewayClient, orgId string, cluster *types.Cluster) (string, error) {
	clusterReq := &argocdv1.GetInstanceClusterRequest{
		OrganizationId: orgId,
		InstanceId:     cluster.InstanceID.ValueString(),
		Id:             cluster.Name.ValueString(),
		IdType:         idv1.Type_NAME,
	}
	clusterResp, err := retryWithBackoff(ctx, func(ctx context.Context) (*argocdv1.GetInstanceClusterResponse, error) {
		return client.GetInstanceCluster(ctx, clusterReq)
	}, "GetInstanceCluster")
	if err != nil {
		return "", fmt.Errorf("unable to read instance cluster: %w", err)
	}
	c, err := waitClusterReconStatus(ctx, client, clusterResp.GetCluster(), orgId, cluster.InstanceID.ValueString())
	if err != nil {
		return "", fmt.Errorf("unable to check cluster reconciliation status: %w", err)
	}
	apiReq := &argocdv1.GetInstanceClusterManifestsRequest{
		OrganizationId: orgId,
		InstanceId:     cluster.InstanceID.ValueString(),
		Id:             c.Id,
	}
	resChan, errChan, err := client.GetInstanceClusterManifests(ctx, apiReq)
	if err != nil {
		return "", fmt.Errorf("unable to download manifests: %w", err)
	}
	res, err := readStream(resChan, errChan)
	if err != nil {
		return "", fmt.Errorf("unable to parse manifests: %w", err)
	}

	return string(res), nil
}

func applyManifests(ctx context.Context, manifests string, cfg *rest.Config) error {
	client, err := kube.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}
	resources, err := kube.SplitYAML([]byte(manifests))
	if err != nil {
		return fmt.Errorf("failed to parse manifests: %w", err)
	}
	for _, un := range resources {
		if err := client.Apply(ctx, &un); err != nil {
			return fmt.Errorf("failed to apply manifest: %w", err)
		}
	}
	return nil
}

func deleteManifests(ctx context.Context, manifests string, cfg *rest.Config) error {
	client, err := kube.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}
	resources, err := kube.SplitYAML([]byte(manifests))
	if err != nil {
		return fmt.Errorf("failed to parse manifests: %w", err)
	}
	tflog.Info(ctx, fmt.Sprintf("%d resources to delete", len(resources)))
	// Delete the resources in reverse order
	for _, v := range slices.Backward(resources) {
		if err := client.Delete(ctx, &v); err != nil {
			return fmt.Errorf("failed to delete manifest: %s: %w", v, err)
		}
	}
	return nil
}

func waitClusterHealthStatus(ctx context.Context, client argocdv1.ArgoCDServiceGatewayClient, orgID string, c *types.Cluster, ensureHealthy bool) error {
	const (
		healthStatusTimeout      = 10 * time.Minute
		healthStatusPollInterval = 5 * time.Second
	)

	targetStatuses := []healthv1.StatusCode{
		healthv1.StatusCode_STATUS_CODE_HEALTHY,
	}
	if !ensureHealthy {
		return nil
	}

	var lastHealthMessage string
	if err := waitForStatus(
		ctx,
		func(ctx context.Context) (*argocdv1.Cluster, error) {
			resp, err := retryWithBackoff(ctx, func(ctx context.Context) (*argocdv1.GetInstanceClusterResponse, error) {
				return client.GetInstanceCluster(ctx, &argocdv1.GetInstanceClusterRequest{
					OrganizationId: orgID,
					InstanceId:     c.InstanceID.ValueString(),
					Id:             c.Name.ValueString(),
					IdType:         idv1.Type_NAME,
				})
			}, "GetInstanceCluster")
			if err != nil {
				return nil, err
			}
			return resp.GetCluster(), nil
		},
		func(cluster *argocdv1.Cluster) healthv1.StatusCode {
			healthStatus := cluster.GetHealthStatus()
			if msg := healthStatus.GetMessage(); msg != "" {
				lastHealthMessage = msg
			}
			return healthStatus.GetCode()
		},
		targetStatuses,
		healthStatusPollInterval,
		healthStatusTimeout,
		c.Name.ValueString(),
		"health",
	); err != nil {
		var errMsg strings.Builder
		errMsg.WriteString(err.Error())

		if lastHealthMessage != "" {
			fmt.Fprintf(&errMsg, "\n\nHealth status message: %s", lastHealthMessage)
		}

		errMsg.WriteString("\n\nTroubleshooting steps:")
		errMsg.WriteString("\n  1. Check the cluster health in the Akuity Console")
		errMsg.WriteString("\n  2. Verify the Akuity agent is running in the cluster (for example in akuity namespace):")
		errMsg.WriteString("\n       kubectl get pods -n akuity")
		errMsg.WriteString("\n  3. Check agent logs for errors:")
		errMsg.WriteString("\n       kubectl logs -n akuity -l app.kubernetes.io/name=akuity-agent --tail=100")
		errMsg.WriteString("\n  4. Ensure the cluster has network connectivity to the Akuity Platform")

		return errors.New(errMsg.String())
	}

	return nil
}

func waitClusterReconStatus(ctx context.Context, client argocdv1.ArgoCDServiceGatewayClient, cluster *argocdv1.Cluster, orgId, instanceId string) (*argocdv1.Cluster, error) {
	const (
		reconStatusTimeout      = 10 * time.Minute
		reconStatusPollInterval = 5 * time.Second
	)

	targetStatuses := []reconv1.StatusCode{
		reconv1.StatusCode_STATUS_CODE_SUCCESSFUL,
		reconv1.StatusCode_STATUS_CODE_FAILED,
	}

	var finalCluster *argocdv1.Cluster

	err := waitForStatus(
		ctx,
		func(ctx context.Context) (*argocdv1.Cluster, error) {
			resp, err := retryWithBackoff(ctx, func(ctx context.Context) (*argocdv1.GetInstanceClusterResponse, error) {
				return client.GetInstanceCluster(ctx, &argocdv1.GetInstanceClusterRequest{
					OrganizationId: orgId,
					InstanceId:     instanceId,
					Id:             cluster.Id,
					IdType:         idv1.Type_ID,
				})
			}, "GetInstanceCluster")
			if err != nil {
				return nil, err
			}
			finalCluster = resp.GetCluster()
			return finalCluster, nil
		},
		func(cluster *argocdv1.Cluster) reconv1.StatusCode {
			return cluster.GetReconciliationStatus().GetCode()
		},
		targetStatuses,
		reconStatusPollInterval,
		reconStatusTimeout,
		cluster.Name,
		"reconciliation",
	)
	if err != nil {
		return nil, err
	}

	return finalCluster, nil
}

func waitForClusterReconciliation(ctx context.Context, client argocdv1.ArgoCDServiceGatewayClient, orgID string, plan *types.Cluster) error {
	clusterReq := &argocdv1.GetInstanceClusterRequest{
		OrganizationId: orgID,
		InstanceId:     plan.InstanceID.ValueString(),
		Id:             plan.Name.ValueString(),
		IdType:         idv1.Type_NAME,
	}

	clusterResp, err := retryWithBackoff(ctx, func(ctx context.Context) (*argocdv1.GetInstanceClusterResponse, error) {
		return client.GetInstanceCluster(ctx, clusterReq)
	}, "GetInstanceCluster")
	if err != nil {
		return fmt.Errorf("unable to get cluster for reconciliation check: %w", err)
	}

	finalCluster, err := waitClusterReconStatus(ctx, client, clusterResp.GetCluster(), orgID, plan.InstanceID.ValueString())
	if err != nil {
		return fmt.Errorf("unable to wait for cluster reconciliation: %w", err)
	}

	if finalCluster.GetReconciliationStatus().GetCode() == reconv1.StatusCode_STATUS_CODE_FAILED {
		msg := finalCluster.GetReconciliationStatus().GetMessage()
		if msg == "" {
			msg = "cluster reconciliation failed"
		}
		return errors.New(msg)
	}

	return nil
}

func readStream(resChan <-chan *httpbody.HttpBody, errChan <-chan error) ([]byte, error) {
	var data []byte
	for resChan != nil && errChan != nil {
		select {
		case dataChunk, ok := <-resChan:
			if !ok {
				resChan = nil
				continue
			} else {
				data = append(data, dataChunk.Data...)
			}
		case err, ok := <-errChan:
			if !ok {
				errChan = nil
				continue
			} else {
				return nil, err
			}
		}
	}
	return data, nil
}

// deleteCluster handles the deletion of a cluster and optionally its manifests.
// If getIdByName is true, it will lookup the cluster ID by name before deletion.
func deleteCluster(ctx context.Context, cli *AkpCli, plan *types.Cluster, includeManifests, getIdByName bool) error {
	var clusterID string

	if getIdByName {
		existingClusterReq := &argocdv1.GetInstanceClusterRequest{
			OrganizationId: cli.OrgId,
			InstanceId:     plan.InstanceID.ValueString(),
			Id:             plan.Name.ValueString(),
			IdType:         idv1.Type_NAME,
		}

		existingResp, err := retryWithBackoff(ctx, func(ctx context.Context) (*argocdv1.GetInstanceClusterResponse, error) {
			return cli.Cli.GetInstanceCluster(ctx, existingClusterReq)
		}, "GetInstanceCluster")
		if err != nil {
			if isGoneErr(err) {
				// Cluster not found, nothing to delete
				return nil
			}
			// Unexpected error occurred while looking up cluster ID
			return fmt.Errorf("unable to lookup cluster ID by name: %s", err)
		}

		// Found cluster, use its ID for deletion
		clusterID = existingResp.GetCluster().Id
		tflog.Info(ctx, fmt.Sprintf("Found existing cluster %s during create, deleting it first", plan.Name.ValueString()))
	} else {
		// Use the ID from the plan (delete when used with `terraform destroy`)
		clusterID = plan.ID.ValueString()
		if clusterID == "" {
			return nil // Nothing to delete
		}
	}

	// Delete the manifests if requested and kubeconfig is available
	if includeManifests {
		kubeconfig, err := getKubeconfig(ctx, plan.Kubeconfig)
		if err != nil {
			return fmt.Errorf("failed to get kubeconfig: %s", err)
		}

		if kubeconfig != nil {
			manifests, err := getManifests(ctx, cli.Cli, cli.OrgId, plan)
			if err != nil {
				return fmt.Errorf("failed to get manifests: %s", err)
			}

			err = deleteManifests(ctx, manifests, kubeconfig)
			if err != nil {
				tflog.Error(ctx, fmt.Sprintf("failed to delete manifests while deleting cluster: %s", err))
			}
		}
	}

	// Delete the cluster from the API
	apiReq := &argocdv1.DeleteInstanceClusterRequest{
		OrganizationId: cli.OrgId,
		InstanceId:     plan.InstanceID.ValueString(),
		Id:             clusterID,
	}

	_, err := retryWithBackoff(ctx, func(ctx context.Context) (*argocdv1.DeleteInstanceClusterResponse, error) {
		return cli.Cli.DeleteInstanceCluster(ctx, apiReq)
	}, "DeleteInstanceCluster")
	if err != nil && !isGoneErr(err) {
		return fmt.Errorf("unable to delete Akuity cluster: %s", err)
	}

	return waitForClusterDeletion(ctx, cli, plan.InstanceID.ValueString(), clusterID)
}

// waitForClusterDeletion polls the API until the cluster is gone.
func waitForClusterDeletion(ctx context.Context, cli *AkpCli, instanceID, clusterID string) error {
	const maxWait = 10 * time.Minute
	start := time.Now()
	err := wait.PollUntilContextTimeout(ctx, 5*time.Second, maxWait, true, func(ctx context.Context) (bool, error) {
		resp, err := cli.Cli.GetInstanceCluster(ctx, &argocdv1.GetInstanceClusterRequest{
			OrganizationId: cli.OrgId,
			InstanceId:     instanceID,
			Id:             clusterID,
			IdType:         idv1.Type_ID,
		})
		if isGoneErr(err) {
			return true, nil
		}
		if err != nil {
			tflog.Warn(ctx, fmt.Sprintf("Error checking cluster deletion status: %v", err))
			return false, nil
		}
		tflog.Info(ctx, fmt.Sprintf("Cluster %s still exists after %v - reconciliation: %s, health: %s",
			clusterID, time.Since(start), resp.GetCluster().GetReconciliationStatus(), resp.GetCluster().GetHealthStatus()))
		return false, nil
	})
	if wait.Interrupted(err) && ctx.Err() == nil {
		return fmt.Errorf("cluster deletion did not complete within %v", maxWait)
	}
	return err
}

type clusterConfigValidator struct{}

func (v clusterConfigValidator) Description(context.Context) string {
	return "Validates cluster field combinations that the control plane normalizes or rejects"
}

func (v clusterConfigValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v clusterConfigValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	data, ok := configDataAttributes(ctx, req, resp)
	if !ok {
		return
	}
	validateMaintenanceModeExpiry(
		&resp.Diagnostics,
		path.Root("spec").AtName("data").AtName("maintenance_mode_expiry"),
		data["maintenance_mode"].(tftypes.Bool),
		data["maintenance_mode_expiry"].(tftypes.String),
	)
}

// configDataAttributes returns the configured spec.data attributes, or false
// when the block is absent or wholly unknown.
func configDataAttributes(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) (map[string]attr.Value, bool) {
	var data tftypes.Object
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("spec").AtName("data"), &data)...)
	if resp.Diagnostics.HasError() || data.IsNull() || data.IsUnknown() {
		return nil, false
	}
	return data.Attributes(), true
}

func validateClusterConfig(diagnostics *diag.Diagnostics, plan *types.Cluster) {
	if diagnostics == nil || plan == nil || plan.Spec == nil {
		return
	}

	validateMaintenanceModeExpiry(
		diagnostics,
		path.Root("spec").AtName("data").AtName("maintenance_mode_expiry"),
		plan.Spec.Data.MaintenanceMode,
		plan.Spec.Data.MaintenanceModeExpiry,
	)
}

func validateMaintenanceModeExpiry(
	diagnostics *diag.Diagnostics,
	attrPath path.Path,
	maintenanceMode tftypes.Bool,
	maintenanceModeExpiry tftypes.String,
) {
	if diagnostics == nil {
		return
	}

	maintenanceModeExpiryConfigured := isKnownNonEmptyString(maintenanceModeExpiry)
	maintenanceModeMissingOrDisabled := maintenanceMode.IsNull() || (!maintenanceMode.IsUnknown() && !maintenanceMode.ValueBool())
	if maintenanceModeExpiryConfigured && maintenanceModeMissingOrDisabled {
		diagnostics.AddAttributeError(
			attrPath,
			"Invalid maintenance_mode_expiry",
			"maintenance_mode_expiry requires maintenance_mode = true. The control plane clears the expiry when maintenance mode is disabled.",
		)
	}
}

// sizeConfigValidator validates the relationship between size and size configuration attributes
type sizeConfigValidator struct{}

func (v sizeConfigValidator) Description(ctx context.Context) string {
	return "Validates that size configuration attributes are only used with appropriate size values"
}

func (v sizeConfigValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v sizeConfigValidator) ValidateResource(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	data, ok := configDataAttributes(ctx, req, resp)
	if !ok {
		return
	}
	size := data["size"].(tftypes.String)
	autoConfig := data["auto_agent_size_config"].(tftypes.Object)
	customConfig := data["custom_agent_size_config"].(tftypes.Object)
	if size.IsNull() || size.IsUnknown() || autoConfig.IsUnknown() || customConfig.IsUnknown() {
		return
	}
	sizeValue := size.ValueString()
	hasAutoConfig := !autoConfig.IsNull()
	hasCustomConfig := !customConfig.IsNull()

	switch sizeValue {
	case "auto":
		// auto_agent_size_config is optional when size is "auto" - API provides defaults if not specified
		if hasCustomConfig {
			resp.Diagnostics.AddAttributeError(
				path.Root("spec").AtName("data").AtName("custom_agent_size_config"),
				"Invalid custom_agent_size_config",
				"custom_agent_size_config cannot be used when size is 'auto'",
			)
		}
	case "custom":
		if !hasCustomConfig {
			resp.Diagnostics.AddAttributeError(
				path.Root("spec").AtName("data").AtName("custom_agent_size_config"),
				"Missing custom_agent_size_config",
				"When size is 'custom', custom_agent_size_config must be specified",
			)
		}
		if hasAutoConfig {
			resp.Diagnostics.AddAttributeError(
				path.Root("spec").AtName("data").AtName("auto_agent_size_config"),
				"Invalid auto_agent_size_config",
				"auto_agent_size_config cannot be used when size is 'custom'",
			)
		}
	case "small", "medium", "large":
		if hasAutoConfig {
			resp.Diagnostics.AddAttributeError(
				path.Root("spec").AtName("data").AtName("auto_agent_size_config"),
				"Invalid auto_agent_size_config",
				fmt.Sprintf("auto_agent_size_config cannot be used when size is '%s'", sizeValue),
			)
		}
		if hasCustomConfig {
			resp.Diagnostics.AddAttributeError(
				path.Root("spec").AtName("data").AtName("custom_agent_size_config"),
				"Invalid custom_agent_size_config",
				fmt.Sprintf("custom_agent_size_config cannot be used when size is '%s'", sizeValue),
			)
		}
	default:
		resp.Diagnostics.AddAttributeError(
			path.Root("spec").AtName("data").AtName("size"),
			"Invalid size",
			"size must be one of 'auto', 'small', 'medium', 'large', or 'custom'",
		)
	}
}
