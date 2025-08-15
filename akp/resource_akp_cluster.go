package akp

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-log/tflog"
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
var _ resource.Resource = &AkpClusterResource{}
var _ resource.ResourceWithImportState = &AkpClusterResource{}

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
	}
	// In this case we commit state regardless whether there's an error or not. This is because there can be partial
	// state (e.g. a cluster could be created in AKP but the manifests failed to be applied).
	if result != nil {
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
	err := refreshClusterState(ctx, &resp.Diagnostics, r.akpCli.Cli, &data, r.akpCli.OrgId, &resp.State, &data)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
	} else {
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	}
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

	ctx = httpctx.SetAuthorizationHeader(ctx, r.akpCli.Cred.Scheme(), r.akpCli.Cred.Credential())
	kubeconfig, err := getKubeconfig(ctx, plan.Kubeconfig)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	// Delete the manifests
	if kubeconfig != nil && plan.RemoveAgentResourcesOnDestroy.ValueBool() {
		manifests, err := getManifests(ctx, r.akpCli.Cli, r.akpCli.OrgId, &plan)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", err.Error())
			return
		}

		err = deleteManifests(ctx, manifests, kubeconfig)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", err.Error())
			return
		}
	}
	apiReq := &argocdv1.DeleteInstanceClusterRequest{
		OrganizationId: r.akpCli.OrgId,
		InstanceId:     plan.InstanceID.ValueString(),
		Id:             plan.ID.ValueString(),
	}
	_, err = r.akpCli.Cli.DeleteInstanceCluster(ctx, apiReq)
	if err != nil && status.Code(err) != codes.NotFound {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete Akuity cluster. %s", err))
		return
	}
	// Give it some time to remove the cluster. This is useful when the terraform provider is performing a replace operation, to give it enough time to destroy the previous cluster.
	time.Sleep(2 * time.Second)
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

func (r *AkpClusterResource) upsert(ctx context.Context, diagnostics *diag.Diagnostics, plan *types.Cluster, isCreate bool) (*types.Cluster, error) {
	ctx = httpctx.SetAuthorizationHeader(ctx, r.akpCli.Cred.Scheme(), r.akpCli.Cred.Credential())
	apiReq := buildClusterApplyRequest(ctx, diagnostics, plan, r.akpCli.OrgId)
	if diagnostics.HasError() {
		return nil, nil
	}
	result, err := r.applyInstance(ctx, plan, apiReq, isCreate, r.akpCli.Cli.ApplyInstance, r.upsertKubeConfig)
	if err != nil {
		return result, err
	}
	return result, refreshClusterState(ctx, diagnostics, r.akpCli.Cli, result, r.akpCli.OrgId, nil, plan)
}

func (r *AkpClusterResource) applyInstance(ctx context.Context, plan *types.Cluster, apiReq *argocdv1.ApplyInstanceRequest, isCreate bool, applyInstance func(context.Context, *argocdv1.ApplyInstanceRequest) (*argocdv1.ApplyInstanceResponse, error), upsertKubeConfig func(ctx context.Context, plan *types.Cluster) error) (*types.Cluster, error) {
	kubeconfig := plan.Kubeconfig
	plan.Kubeconfig = nil
	tflog.Debug(ctx, fmt.Sprintf("Apply cluster request: %s", apiReq))
	_, err := applyInstance(ctx, apiReq)
	if err != nil {
		return nil, fmt.Errorf("unable to create Argo CD instance: %s", err)
	}

	if kubeconfig != nil {
		plan.Kubeconfig = kubeconfig
		shouldApply := isCreate || plan.ReapplyManifestsOnUpdate.ValueBool()
		if shouldApply {
			err = upsertKubeConfig(ctx, plan)
			if err != nil {
				// Ensure kubeconfig won't be committed to state by setting it to nil
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
	orgID string, state *tfsdk.State, plan *types.Cluster) error {
	clusterReq := &argocdv1.GetInstanceClusterRequest{
		OrganizationId: orgID,
		InstanceId:     cluster.InstanceID.ValueString(),
		Id:             cluster.Name.ValueString(),
		IdType:         idv1.Type_NAME,
	}

	tflog.Debug(ctx, fmt.Sprintf("Get cluster request: %s", clusterReq))
	clusterResp, err := client.GetInstanceCluster(ctx, clusterReq)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			state.RemoveResource(ctx)
		} else {
			return errors.Wrap(err, "Unable to read Argo CD cluster")
		}
	}
	tflog.Debug(ctx, fmt.Sprintf("Get cluster response: %s", clusterResp))
	cluster.Update(ctx, diagnostics, clusterResp.GetCluster(), plan)
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
	clusterResp, err := client.GetInstanceCluster(ctx, clusterReq)
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
			return errors.Wrap(err, fmt.Sprintf("failed to apply manifest"))
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
		apiResp, err := client.GetInstanceCluster(ctx, &argocdv1.GetInstanceClusterRequest{
			OrganizationId: orgID,
			InstanceId:     c.InstanceID.ValueString(),
			Id:             c.Name.ValueString(),
			IdType:         idv1.Type_NAME,
		})
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
		apiResp, err := client.GetInstanceCluster(ctx, &argocdv1.GetInstanceClusterRequest{
			OrganizationId: orgId,
			InstanceId:     instanceId,
			Id:             cluster.Id,
			IdType:         idv1.Type_ID,
		})
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
