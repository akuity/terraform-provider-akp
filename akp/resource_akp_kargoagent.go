package akp

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/pkg/errors"
	"golang.org/x/exp/slices"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"

	kargov1 "github.com/akuity/api-client-go/pkg/api/gen/kargo/v1"
	healthv1 "github.com/akuity/api-client-go/pkg/api/gen/types/status/health/v1"
	reconv1 "github.com/akuity/api-client-go/pkg/api/gen/types/status/reconciliation/v1"
	httpctx "github.com/akuity/grpc-gateway-client/pkg/http/context"
	"github.com/akuity/terraform-provider-akp/akp/marshal"
	"github.com/akuity/terraform-provider-akp/akp/types"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ resource.Resource = &AkpKargoAgentResource{}
var _ resource.ResourceWithImportState = &AkpKargoAgentResource{}

func NewKargoAgentResource() resource.Resource {
	return &AkpKargoAgentResource{}
}

type AkpKargoAgentResource struct {
	akpCli *AkpCli
}

func (r *AkpKargoAgentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kargo_agent"
}

func (r *AkpKargoAgentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *AkpKargoAgentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Debug(ctx, "Creating a Kargo Agent")
	var plan types.KargoAgent

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

func (r *AkpKargoAgentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Debug(ctx, "Reading a Kargo Agent")
	var data types.KargoAgent
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = httpctx.SetAuthorizationHeader(ctx, r.akpCli.Cred.Scheme(), r.akpCli.Cred.Credential())
	err := refreshKargoAgentState(ctx, &resp.Diagnostics, r.akpCli.KargoCli, &data, r.akpCli.OrgId, &resp.State, &data)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
	} else {
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	}
}

func (r *AkpKargoAgentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Debug(ctx, "Updating a Kargo Agent")
	var plan types.KargoAgent

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

func (r *AkpKargoAgentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Debug(ctx, "Deleting a Kargo Agent")
	var plan types.KargoAgent
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	ctx = httpctx.SetAuthorizationHeader(ctx, r.akpCli.Cred.Scheme(), r.akpCli.Cred.Credential())
	kubeconfig, err := getKubeconfig(plan.Kubeconfig)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	// Delete the manifests
	if kubeconfig != nil && plan.RemoveAgentResourcesOnDestroy.ValueBool() {
		manifests, err := getKargoManifests(ctx, r.akpCli.KargoCli, r.akpCli.OrgId, &plan)
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
	apiReq := &kargov1.DeleteInstanceAgentRequest{
		OrganizationId: r.akpCli.OrgId,
		InstanceId:     plan.InstanceID.ValueString(),
		Id:             plan.ID.ValueString(),
		WorkspaceId:    plan.WorkspaceID.ValueString(),
	}
	_, err = r.akpCli.KargoCli.DeleteInstanceAgent(ctx, apiReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete Kargo agent. %s", err))
		return
	}
	// Give it some time to remove the cluster. This is useful when the terraform provider is performing a replace operation, to give it enough time to destroy the previous cluster.
	time.Sleep(2 * time.Second)
}

func (r *AkpKargoAgentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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

func (r *AkpKargoAgentResource) upsert(ctx context.Context, diagnostics *diag.Diagnostics, plan *types.KargoAgent, isCreate bool) (*types.KargoAgent, error) {
	ctx = httpctx.SetAuthorizationHeader(ctx, r.akpCli.Cred.Scheme(), r.akpCli.Cred.Credential())
	apiReq := buildKargoAgentApplyRequest(ctx, diagnostics, plan, r.akpCli.OrgId)
	if diagnostics.HasError() {
		return nil, nil
	}
	result, err := r.applyKargoInstance(ctx, plan, apiReq, isCreate, r.akpCli.KargoCli.ApplyKargoInstance, r.upsertKubeConfig)
	if err != nil {
		return result, err
	}
	return result, refreshKargoAgentState(ctx, diagnostics, r.akpCli.KargoCli, result, r.akpCli.OrgId, nil, plan)
}

func (r *AkpKargoAgentResource) applyKargoInstance(ctx context.Context, plan *types.KargoAgent, apiReq *kargov1.ApplyKargoInstanceRequest, isCreate bool, applyKargoInstance func(context.Context, *kargov1.ApplyKargoInstanceRequest) (*kargov1.ApplyKargoInstanceResponse, error), upsertKubeConfig func(ctx context.Context, plan *types.KargoAgent, isCreate bool) error) (*types.KargoAgent, error) {
	kubeconfig := plan.Kubeconfig
	plan.Kubeconfig = nil
	tflog.Debug(ctx, fmt.Sprintf("Apply Kargo agent request: %s", apiReq))
	_, err := applyKargoInstance(ctx, apiReq)
	if err != nil {
		return nil, fmt.Errorf("unable to create Kargo agent: %s", err)
	}

	if kubeconfig != nil {
		plan.Kubeconfig = kubeconfig
		err = upsertKubeConfig(ctx, plan, isCreate)
		if err != nil {
			// Ensure kubeconfig won't be committed to state by setting it to nil
			plan.Kubeconfig = nil
			return plan, fmt.Errorf("unable to apply manifests: %s", err)
		}
	}

	return plan, nil
}

func (r *AkpKargoAgentResource) upsertKubeConfig(ctx context.Context, plan *types.KargoAgent, isCreate bool) error {
	// Apply agent manifests to clusters if the kubeconfig is specified for cluster.
	kubeconfig, err := getKubeconfig(plan.Kubeconfig)
	if err != nil {
		return err
	}

	// Apply the manifests
	if kubeconfig != nil && isCreate {
		manifests, err := getKargoManifests(ctx, r.akpCli.KargoCli, r.akpCli.OrgId, plan)
		if err != nil {
			return err
		}

		err = applyManifests(ctx, manifests, kubeconfig)
		if err != nil {
			return err
		}
		return waitKargoAgentHealthStatus(ctx, r.akpCli.KargoCli, r.akpCli.OrgId, plan)
	}
	return nil
}

func refreshKargoAgentState(ctx context.Context, diagnostics *diag.Diagnostics, client kargov1.KargoServiceGatewayClient, kargoAgent *types.KargoAgent,
	orgID string, state *tfsdk.State, plan *types.KargoAgent) error {
	kargoAgentReq := &kargov1.GetKargoInstanceAgentRequest{
		OrganizationId: orgID,
		InstanceId:     kargoAgent.InstanceID.ValueString(),
		Id:             kargoAgent.ID.ValueString(),
		WorkspaceId:    kargoAgent.WorkspaceID.ValueString(),
	}

	tflog.Debug(ctx, fmt.Sprintf("Get kargo agent request: %s", kargoAgentReq))
	kargoAgentResp, err := client.GetKargoInstanceAgent(ctx, kargoAgentReq)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			state.RemoveResource(ctx)
			return nil
		}
		return errors.Wrap(err, "Unable to read Kargo agent")
	}
	tflog.Debug(ctx, fmt.Sprintf("Get kargo agent response: %s", kargoAgentResp))
	kargoAgent.Update(ctx, diagnostics, kargoAgentResp.GetAgent(), plan)
	return nil
}

func buildKargoAgentApplyRequest(ctx context.Context, diagnostics *diag.Diagnostics, kargoAgent *types.KargoAgent, orgId string) *kargov1.ApplyKargoInstanceRequest {
	applyReq := &kargov1.ApplyKargoInstanceRequest{
		OrganizationId: orgId,
		Id:             kargoAgent.InstanceID.ValueString(),
		WorkspaceId:    kargoAgent.WorkspaceID.ValueString(),
		Agents:         buildKargoAgents(ctx, diagnostics, kargoAgent),
	}
	return applyReq
}

func buildKargoAgents(ctx context.Context, diagnostics *diag.Diagnostics, kargoAgent *types.KargoAgent) []*structpb.Struct {
	var cs []*structpb.Struct
	apiKargoAgent := kargoAgent.ToKargoAgentAPIModel(ctx, diagnostics)
	s, err := marshal.ApiModelToPBStruct(apiKargoAgent)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Kargo agent. %s", err))
		return nil
	}
	cs = append(cs, s)
	return cs
}

func getKargoManifests(ctx context.Context, client kargov1.KargoServiceGatewayClient, orgId string, kargoAgent *types.KargoAgent) (string, error) {
	kargoAgentReq := &kargov1.GetKargoInstanceAgentRequest{
		OrganizationId: orgId,
		InstanceId:     kargoAgent.InstanceID.ValueString(),
		Id:             kargoAgent.Name.ValueString(),
		WorkspaceId:    kargoAgent.WorkspaceID.ValueString(),
	}
	kargoAgentResp, err := client.GetKargoInstanceAgent(ctx, kargoAgentReq)
	if err != nil {
		return "", errors.Wrap(err, "Unable to read instance kargo agent")
	}
	k, err := waitKargoAgentReconStatus(ctx, client, kargoAgentResp.GetAgent(), orgId, kargoAgent.InstanceID.ValueString(), kargoAgent.WorkspaceID.ValueString())
	if err != nil {
		return "", errors.Wrap(err, "Unable to check kargo agent health status")
	}
	apiReq := &kargov1.GetKargoInstanceAgentManifestsRequest{
		OrganizationId: orgId,
		InstanceId:     kargoAgent.InstanceID.ValueString(),
		Id:             k.Id,
	}
	resChan, errChan, err := client.GetKargoInstanceAgentManifests(ctx, apiReq)
	if err != nil {
		return "", errors.Wrap(err, "Unable to download manifests")
	}
	res, err := readStream(resChan, errChan)
	if err != nil {
		return "", errors.Wrap(err, "Unable to parse manifests")
	}

	return string(res), nil
}

func waitKargoAgentHealthStatus(ctx context.Context, client kargov1.KargoServiceGatewayClient, orgID string, c *types.KargoAgent) error {
	kargoAgent := &kargov1.KargoAgent{}
	healthStatus := kargoAgent.GetHealthStatus()
	breakStatusesHealth := []healthv1.StatusCode{healthv1.StatusCode_STATUS_CODE_HEALTHY, healthv1.StatusCode_STATUS_CODE_DEGRADED}

	for !slices.Contains(breakStatusesHealth, healthStatus.GetCode()) {
		time.Sleep(1 * time.Second)
		apiResp, err := client.GetKargoInstanceAgent(ctx, &kargov1.GetKargoInstanceAgentRequest{
			OrganizationId: orgID,
			InstanceId:     c.InstanceID.ValueString(),
			Id:             c.Name.ValueString(),
			WorkspaceId:    c.WorkspaceID.ValueString(),
		})
		if err != nil {
			return err
		}
		kargoAgent = apiResp.GetAgent()
		healthStatus = kargoAgent.GetHealthStatus()
		tflog.Debug(ctx, fmt.Sprintf("Kargo agent health status: %s", healthStatus.String()))
	}
	return nil
}

func waitKargoAgentReconStatus(ctx context.Context, client kargov1.KargoServiceGatewayClient, kargoAgent *kargov1.KargoAgent, orgId, instanceId, workspaceId string) (*kargov1.KargoAgent, error) {
	reconStatus := kargoAgent.GetReconciliationStatus()
	breakStatusesRecon := []reconv1.StatusCode{reconv1.StatusCode_STATUS_CODE_SUCCESSFUL, reconv1.StatusCode_STATUS_CODE_FAILED}

	for !slices.Contains(breakStatusesRecon, reconStatus.GetCode()) {
		time.Sleep(1 * time.Second)
		apiResp, err := client.GetKargoInstanceAgent(ctx, &kargov1.GetKargoInstanceAgentRequest{
			OrganizationId: orgId,
			InstanceId:     instanceId,
			Id:             kargoAgent.Id,
			WorkspaceId:    workspaceId,
		})
		if err != nil {
			return nil, err
		}
		kargoAgent = apiResp.GetAgent()
		reconStatus = kargoAgent.GetReconciliationStatus()
		tflog.Debug(ctx, fmt.Sprintf("Kargo agent recon status: %s", reconStatus.String()))
	}
	return kargoAgent, nil
}
