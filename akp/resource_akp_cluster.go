package akp

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/pkg/errors"
	"golang.org/x/exp/slices"
	"google.golang.org/genproto/googleapis/api/httpbody"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"k8s.io/client-go/rest"

	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	idv1 "github.com/akuity/api-client-go/pkg/api/gen/types/id/v1"
	healthv1 "github.com/akuity/api-client-go/pkg/api/gen/types/status/health/v1"
	reconv1 "github.com/akuity/api-client-go/pkg/api/gen/types/status/reconciliation/v1"
	httpctx "github.com/akuity/grpc-gateway-client/pkg/http/context"
	"github.com/akuity/terraform-provider-akp/akp/kube"
	"github.com/akuity/terraform-provider-akp/akp/marshal"
	"github.com/akuity/terraform-provider-akp/akp/types"
)

// Ensure provider defined types fully satisfy framework interfaces
var (
	_ resource.Resource                     = &AkpClusterResource{}
	_ resource.ResourceWithImportState      = &AkpClusterResource{}
	_ resource.ResourceWithConfigValidators = &AkpClusterResource{}
)

func NewAkpClusterResource() resource.Resource {
	return &AkpClusterResource{}
}

// AkpClusterResource defines the resource implementation.
type AkpClusterResource struct {
	akpCli *AkpCli
}

func (r *AkpClusterResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cluster"
}

func (r *AkpClusterResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	akpCli, ok := req.ProviderData.(*AkpCli)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *AkpCli, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}
	r.akpCli = akpCli
}

func (r *AkpClusterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Debug(ctx, "Creating a Cluster")
	var plan types.Cluster

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.upsert(ctx, &resp.Diagnostics, &plan, true)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		tflog.Warn(ctx, fmt.Sprintf("refreshClusterState failed during create, cleaning up cluster %s", plan.Name.ValueString()))
		cleanupErr := r.deleteCluster(ctx, &plan, plan.RemoveAgentResourcesOnDestroy.ValueBool(), true)
		if cleanupErr != nil {
			tflog.Error(ctx, fmt.Sprintf("Failed to clean up dangling cluster %s: %v", plan.Name.ValueString(), cleanupErr))
		}
		tflog.Info(ctx, fmt.Sprintf("Successfully cleaned up dangling cluster %s", plan.Name.ValueString()))
	} else if result != nil {
		resp.Diagnostics.Append(resp.State.Set(ctx, &result)...)
	}
}

func (r *AkpClusterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Debug(ctx, "Reading a Cluster")
	var data types.Cluster
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = httpctx.SetAuthorizationHeader(ctx, r.akpCli.Cred.Scheme(), r.akpCli.Cred.Credential())
	err := refreshClusterState(ctx, &resp.Diagnostics, r.akpCli.Cli, &data, r.akpCli.OrgId, &data)
	if err != nil {
		handleReadResourceError(ctx, resp, err)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AkpClusterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Debug(ctx, "Updating a Cluster")
	var plan types.Cluster

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.upsert(ctx, &resp.Diagnostics, &plan, false)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
	}
	// In this case we commit state regardless whether there's an error or not. This is because there can be partial
	// state (e.g. a cluster could be created in AKP but the manifests failed to be applied).
	if result != nil {
		resp.Diagnostics.Append(resp.State.Set(ctx, &result)...)
	}
}

func (r *AkpClusterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Debug(ctx, "Deleting a Cluster")
	var plan types.Cluster
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.deleteCluster(ctx, &plan, plan.RemoveAgentResourcesOnDestroy.ValueBool(), false)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
}

func (r *AkpClusterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	idParts := strings.Split(req.ID, "/")
	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: instance_id/name. Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("instance_id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), idParts[1])...)
}

func (r *AkpClusterResource) ConfigValidators(ctx context.Context) []resource.ConfigValidator {
	return []resource.ConfigValidator{
		// auto_agent_size_config and custom_agent_size_config are mutually exclusive
		resourcevalidator.Conflicting(
			path.MatchRoot("spec").AtName("data").AtName("auto_agent_size_config"),
			path.MatchRoot("spec").AtName("data").AtName("custom_agent_size_config"),
		),
		// Use custom validator to handle size-specific logic
		&sizeConfigValidator{},
	}
}

func (r *AkpClusterResource) upsert(ctx context.Context, diagnostics *diag.Diagnostics, plan *types.Cluster, isCreate bool) (*types.Cluster, error) {
	ctx = httpctx.SetAuthorizationHeader(ctx, r.akpCli.Cred.Scheme(), r.akpCli.Cred.Credential())

	var akiSanitized bool
	if plan != nil && plan.Spec != nil && !plan.Spec.Data.MultiClusterK8SDashboardEnabled.IsNull() && !plan.Spec.Data.MultiClusterK8SDashboardEnabled.IsUnknown() && plan.Spec.Data.MultiClusterK8SDashboardEnabled.ValueBool() {
		instReq := &argocdv1.GetInstanceRequest{
			OrganizationId: r.akpCli.OrgId,
			Id:             plan.InstanceID.ValueString(),
			IdType:         idv1.Type_ID,
		}
		instResp, err := retryWithBackoff(ctx, func(ctx context.Context) (*argocdv1.GetInstanceResponse, error) {
			return r.akpCli.Cli.GetInstance(ctx, instReq)
		}, "GetInstance")
		if err != nil {
			return nil, errors.Wrap(err, "Unable to get Argo CD instance for capability check")
		}

		instSpec := instResp.GetInstance().GetSpec()
		enabled := instSpec != nil && instSpec.MultiClusterK8SDashboardEnabled

		if !enabled {
			akiSanitized = true
			plan.Spec.Data.MultiClusterK8SDashboardEnabled = tftypes.BoolValue(false)
		}
	}

	apiReq := buildClusterApplyRequest(ctx, diagnostics, plan, r.akpCli.OrgId)
	if diagnostics.HasError() {
		return nil, nil
	}
	result, err := r.applyInstance(ctx, plan, apiReq, isCreate, r.akpCli.Cli.ApplyInstance, r.upsertKubeConfig)
	// Always refresh cluster state to ensure we have consistent state, even if kubeconfig application failed
	if result != nil {
		if akiSanitized {
			plan.Spec.Data.MultiClusterK8SDashboardEnabled = tftypes.BoolValue(true)
		}
		refreshErr := refreshClusterState(ctx, diagnostics, r.akpCli.Cli, result, r.akpCli.OrgId, plan)
		if refreshErr != nil && err == nil {
			// If we didn't have an error before but refresh failed, return the refresh error
			return result, refreshErr
		}
	}
	return result, err
}

func (r *AkpClusterResource) applyInstance(ctx context.Context, plan *types.Cluster, apiReq *argocdv1.ApplyInstanceRequest, isCreate bool, applyInstance func(context.Context, *argocdv1.ApplyInstanceRequest) (*argocdv1.ApplyInstanceResponse, error), upsertKubeConfig func(ctx context.Context, plan *types.Cluster) error) (*types.Cluster, error) {
	kubeconfig := plan.Kubeconfig
	plan.Kubeconfig = nil
	tflog.Debug(ctx, fmt.Sprintf("Apply cluster request: %s", apiReq))
	_, err := retryWithBackoff(ctx, func(ctx context.Context) (*argocdv1.ApplyInstanceResponse, error) {
		return applyInstance(ctx, apiReq)
	}, "ApplyInstance")
	if err != nil {
		// If this is a create operation and kubeconfig application fails,
		// clean up the dangling cluster from the API
		if isCreate {
			tflog.Warn(ctx, fmt.Sprintf("applyInstance failed during create, cleaning up cluster %s", plan.Name.ValueString()))
			cleanupErr := r.deleteCluster(ctx, plan, plan.RemoveAgentResourcesOnDestroy.ValueBool(), true)
			if cleanupErr != nil {
				tflog.Error(ctx, fmt.Sprintf("Failed to clean up dangling cluster %s: %v", plan.Name.ValueString(), cleanupErr))
				return nil, fmt.Errorf("unable to create Argo CD instance %s (and failed to clean up cluster: %s)", err, cleanupErr)
			}
			tflog.Info(ctx, fmt.Sprintf("Successfully cleaned up dangling cluster %s", plan.Name.ValueString()))
		}
		return nil, fmt.Errorf("unable to create Argo CD instance: %s", err)
	}

	if kubeconfig != nil {
		plan.Kubeconfig = kubeconfig
		shouldApply := isCreate || plan.ReapplyManifestsOnUpdate.ValueBool()
		if shouldApply {
			err = upsertKubeConfig(ctx, plan)
			if err != nil {
				// If this is a create operation and kubeconfig application fails,
				// clean up the dangling cluster from the API
				if isCreate {
					tflog.Warn(ctx, fmt.Sprintf("Kubeconfig application failed during create, cleaning up cluster %s", plan.Name.ValueString()))
					cleanupErr := r.deleteCluster(ctx, plan, plan.RemoveAgentResourcesOnDestroy.ValueBool(), true)
					if cleanupErr != nil {
						tflog.Error(ctx, fmt.Sprintf("Failed to clean up dangling cluster %s: %v", plan.Name.ValueString(), cleanupErr))
						return nil, fmt.Errorf("unable to apply manifests: %s (and failed to clean up cluster: %s)", err, cleanupErr)
					}
					tflog.Info(ctx, fmt.Sprintf("Successfully cleaned up dangling cluster %s", plan.Name.ValueString()))
					return nil, fmt.Errorf("unable to apply manifests: %s", err)
				}
				// For updates, just ensure kubeconfig won't be committed to state by setting it to nil
				plan.Kubeconfig = nil
				return plan, fmt.Errorf("unable to apply manifests: %s", err)
			}
		}
	}

	return plan, nil
}

func (r *AkpClusterResource) upsertKubeConfig(ctx context.Context, plan *types.Cluster) error {
	// Apply agent manifests to clusters if the kubeconfig is specified for cluster.
	kubeconfig, err := getKubeconfig(ctx, plan.Kubeconfig)
	if err != nil {
		return err
	}

	// Apply the manifests
	if kubeconfig != nil {
		manifests, err := getManifests(ctx, r.akpCli.Cli, r.akpCli.OrgId, plan)
		if err != nil {
			return err
		}

		err = applyManifests(ctx, manifests, kubeconfig)
		if err != nil {
			return err
		}
		return waitClusterHealthStatus(ctx, r.akpCli.Cli, r.akpCli.OrgId, plan)
	}
	return nil
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
		return errors.Wrap(err, "Unable to read Argo CD cluster")
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
	var cs []*structpb.Struct
	apiCluster := cluster.ToClusterAPIModel(ctx, diagnostics)
	s, err := marshal.ApiModelToPBStruct(apiCluster)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Cluster. %s", err))
		return nil
	}
	cs = append(cs, s)
	return cs
}

func getKubeconfig(ctx context.Context, kubeConfig *types.Kubeconfig) (*rest.Config, error) {
	if kubeConfig == nil {
		return nil, nil
	}
	kcfg, err := kube.InitializeConfiguration(ctx, kubeConfig)
	if err != nil {
		return nil, errors.Wrap(err, "Cannot initialize Kubectl. Please check kubernetes configuration")
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
		return "", errors.Wrap(err, "Unable to read instance cluster")
	}
	c, err := waitClusterReconStatus(ctx, client, clusterResp.GetCluster(), orgId, cluster.InstanceID.ValueString())
	if err != nil {
		return "", errors.Wrap(err, "Unable to check cluster reconciliation status")
	}
	apiReq := &argocdv1.GetInstanceClusterManifestsRequest{
		OrganizationId: orgId,
		InstanceId:     cluster.InstanceID.ValueString(),
		Id:             c.Id,
	}
	resChan, errChan, err := client.GetInstanceClusterManifests(ctx, apiReq)
	if err != nil {
		return "", errors.Wrap(err, "Unable to download manifests")
	}
	res, err := readStream(resChan, errChan)
	if err != nil {
		return "", errors.Wrap(err, "Unable to parse manifests")
	}

	return string(res), nil
}

func applyManifests(ctx context.Context, manifests string, cfg *rest.Config) error {
	kubectl, err := kube.NewKubectl(cfg)
	if err != nil {
		return errors.Wrap(err, "Failed to create Kubectl")
	}
	resources, err := kube.SplitYAML([]byte(manifests))
	if err != nil {
		return errors.Wrap(err, "Failed to parse manifests")
	}

	for _, un := range resources {
		msg, err := kubectl.ApplyResource(ctx, &un, kube.ApplyOpts{})
		if err != nil {
			return errors.Wrap(err, "failed to apply manifest")
		}
		tflog.Debug(ctx, msg)
	}
	return nil
}

func deleteManifests(ctx context.Context, manifests string, cfg *rest.Config) error {
	kubectl, err := kube.NewKubectl(cfg)
	if err != nil {
		return errors.Wrap(err, "failed to create kubectl")
	}
	resources, err := kube.SplitYAML([]byte(manifests))
	tflog.Info(ctx, fmt.Sprintf("%d resources to delete", len(resources)))
	if err != nil {
		return errors.Wrap(err, "failed to parse manifests")
	}

	// Delete the resources in reverse order
	for i := len(resources) - 1; i >= 0; i-- {
		msg, err := kubectl.DeleteResource(ctx, &resources[i], kube.DeleteOpts{
			IgnoreNotFound:  true,
			WaitForDeletion: true,
			Force:           false,
		})
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("failed to delete manifest: %s", resources[i]))
		}
		tflog.Debug(ctx, msg)
	}
	return nil
}

func waitClusterHealthStatus(ctx context.Context, client argocdv1.ArgoCDServiceGatewayClient, orgID string, c *types.Cluster) error {
	cluster := &argocdv1.Cluster{}
	healthStatus := cluster.GetHealthStatus()
	breakStatusesHealth := []healthv1.StatusCode{healthv1.StatusCode_STATUS_CODE_HEALTHY, healthv1.StatusCode_STATUS_CODE_DEGRADED}

	for !slices.Contains(breakStatusesHealth, healthStatus.GetCode()) {
		time.Sleep(1 * time.Second)
		apiResp, err := retryWithBackoff(ctx, func(ctx context.Context) (*argocdv1.GetInstanceClusterResponse, error) {
			return client.GetInstanceCluster(ctx, &argocdv1.GetInstanceClusterRequest{
				OrganizationId: orgID,
				InstanceId:     c.InstanceID.ValueString(),
				Id:             c.Name.ValueString(),
				IdType:         idv1.Type_NAME,
			})
		}, "GetInstanceCluster")
		if err != nil {
			return err
		}
		cluster = apiResp.GetCluster()
		healthStatus = cluster.GetHealthStatus()
		tflog.Debug(ctx, fmt.Sprintf("Cluster health status: %s", healthStatus.String()))
	}
	return nil
}

func waitClusterReconStatus(ctx context.Context, client argocdv1.ArgoCDServiceGatewayClient, cluster *argocdv1.Cluster, orgId, instanceId string) (*argocdv1.Cluster, error) {
	reconStatus := cluster.GetReconciliationStatus()
	breakStatusesRecon := []reconv1.StatusCode{reconv1.StatusCode_STATUS_CODE_SUCCESSFUL, reconv1.StatusCode_STATUS_CODE_FAILED}

	for !slices.Contains(breakStatusesRecon, reconStatus.GetCode()) {
		time.Sleep(1 * time.Second)
		apiResp, err := retryWithBackoff(ctx, func(ctx context.Context) (*argocdv1.GetInstanceClusterResponse, error) {
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
		cluster = apiResp.GetCluster()
		reconStatus = cluster.GetReconciliationStatus()
		tflog.Debug(ctx, fmt.Sprintf("Cluster recon status: %s", reconStatus.String()))
	}
	return cluster, nil
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
func (r *AkpClusterResource) deleteCluster(ctx context.Context, plan *types.Cluster, includeManifests, getIdByName bool) error {
	ctx = httpctx.SetAuthorizationHeader(ctx, r.akpCli.Cred.Scheme(), r.akpCli.Cred.Credential())

	var clusterID string

	if getIdByName {
		existingClusterReq := &argocdv1.GetInstanceClusterRequest{
			OrganizationId: r.akpCli.OrgId,
			InstanceId:     plan.InstanceID.ValueString(),
			Id:             plan.Name.ValueString(),
			IdType:         idv1.Type_NAME,
		}

		existingResp, err := retryWithBackoff(ctx, func(ctx context.Context) (*argocdv1.GetInstanceClusterResponse, error) {
			return r.akpCli.Cli.GetInstanceCluster(ctx, existingClusterReq)
		}, "GetInstanceCluster")
		if err != nil {
			if status.Code(err) == codes.NotFound || status.Code(err) == codes.PermissionDenied {
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
			manifests, err := getManifests(ctx, r.akpCli.Cli, r.akpCli.OrgId, plan)
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
		OrganizationId: r.akpCli.OrgId,
		InstanceId:     plan.InstanceID.ValueString(),
		Id:             clusterID,
	}

	_, err := retryWithBackoff(ctx, func(ctx context.Context) (*argocdv1.DeleteInstanceClusterResponse, error) {
		return r.akpCli.Cli.DeleteInstanceCluster(ctx, apiReq)
	}, "DeleteInstanceCluster")
	if err != nil && (status.Code(err) != codes.NotFound && status.Code(err) != codes.PermissionDenied) {
		return fmt.Errorf("unable to delete Akuity cluster: %s", err)
	}

	// Wait for the cluster to actually be deleted with exponential backoff
	return r.waitForClusterDeletion(ctx, plan.InstanceID.ValueString(), clusterID)
}

// waitForClusterDeletion polls the API to verify the cluster is actually deleted,
// using exponential backoff with a maximum wait time of 1 minute.
func (r *AkpClusterResource) waitForClusterDeletion(ctx context.Context, instanceID, clusterID string) error {
	const (
		initialDelay  = 500 * time.Millisecond
		maxDelay      = 8 * time.Second
		maxWait       = 5 * time.Minute
		backoffFactor = 2.0
	)

	delay := initialDelay
	start := time.Now()

	for {
		// Check if we've exceeded the maximum wait time
		if time.Since(start) > maxWait {
			return fmt.Errorf("cluster deletion did not complete within %v", maxWait)
		}

		// Try to get the cluster - if it's gone, we're done
		getReq := &argocdv1.GetInstanceClusterRequest{
			OrganizationId: r.akpCli.OrgId,
			InstanceId:     instanceID,
			Id:             clusterID,
			IdType:         idv1.Type_ID,
		}

		_, err := retryWithBackoff(ctx, func(ctx context.Context) (*argocdv1.GetInstanceClusterResponse, error) {
			return r.akpCli.Cli.GetInstanceCluster(ctx, getReq)
		}, "GetInstanceCluster")
		if err != nil {
			if status.Code(err) == codes.NotFound || status.Code(err) == codes.PermissionDenied {
				// Cluster is gone, deletion successful
				tflog.Debug(ctx, fmt.Sprintf("Cluster %s successfully deleted after %v", clusterID, time.Since(start)))
				return nil
			}
			// Some other error occurred, but continue polling
			tflog.Warn(ctx, fmt.Sprintf("Error checking cluster deletion status: %v", err))
		}

		// Cluster still exists, wait before retrying
		tflog.Info(ctx, fmt.Sprintf("Cluster %s still exists, waiting %v before next check", clusterID, delay))
		time.Sleep(delay)

		// Exponential backoff with cap
		delay = time.Duration(float64(delay) * backoffFactor)
		if delay > maxDelay {
			delay = maxDelay
		}
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
	dataPath := path.Root("spec").AtName("data")

	var data tftypes.Object
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, dataPath, &data)...)
	if resp.Diagnostics.HasError() || data.IsNull() || data.IsUnknown() {
		return
	}

	var size tftypes.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, dataPath.AtName("size"), &size)...)
	if resp.Diagnostics.HasError() || size.IsNull() || size.IsUnknown() {
		return
	}
	sizeValue := size.ValueString()

	var autoConfig tftypes.Object
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, dataPath.AtName("auto_agent_size_config"), &autoConfig)...)
	if resp.Diagnostics.HasError() || autoConfig.IsUnknown() {
		return
	}
	hasAutoConfig := !autoConfig.IsNull()

	customConfigPath := dataPath.AtName("custom_agent_size_config")
	var customConfig tftypes.Object
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, customConfigPath, &customConfig)...)
	if resp.Diagnostics.HasError() || customConfig.IsUnknown() {
		return
	}
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
