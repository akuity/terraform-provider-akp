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
	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/pkg/errors"
	"golang.org/x/exp/slices"
	"google.golang.org/protobuf/types/known/structpb"

	kargov1 "github.com/akuity/api-client-go/pkg/api/gen/kargo/v1"
	healthv1 "github.com/akuity/api-client-go/pkg/api/gen/types/status/health/v1"
	reconv1 "github.com/akuity/api-client-go/pkg/api/gen/types/status/reconciliation/v1"
	httpctx "github.com/akuity/grpc-gateway-client/pkg/http/context"
	"github.com/akuity/terraform-provider-akp/akp/marshal"
	"github.com/akuity/terraform-provider-akp/akp/types"
)

// Ensure provider defined types fully satisfy framework interfaces
var (
	_ resource.Resource                = &AkpKargoAgentResource{}
	_ resource.ResourceWithImportState = &AkpKargoAgentResource{}
)

func NewAkpKargoAgentResource() resource.Resource {
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
	kubeconfig, err := getKubeconfig(ctx, plan.KubeConfig)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	// Delete the manifests
	if kubeconfig != nil && plan.RemoveAgentResourcesOnDestroy.ValueBool() {
		manifests, _, err := getKargoManifests(ctx, r.akpCli.KargoCli, r.akpCli.OrgId, &plan)
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
	}
	_, err = r.akpCli.KargoCli.DeleteInstanceAgent(ctx, apiReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete Kargo agent. %s", err))
		return
	}
	// Give it some time to remove the cluster. This is useful when the terraform provider is performing a replace operation, to give it enough time to destroy the previous cluster.
	time.Sleep(5 * time.Second)
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

	workspace, err := getWorkspace(ctx, r.akpCli.OrgCli, r.akpCli.OrgId, plan.Workspace.ValueString())
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get workspace. %s", err))
		return nil, errors.New("Unable to get workspace")
	}
	apiReq := buildKargoAgentApplyRequest(ctx, diagnostics, plan, r.akpCli.OrgId, workspace.Id)
	if diagnostics.HasError() {
		return nil, nil
	}
	result, err := r.applyKargoInstance(ctx, plan, apiReq, isCreate, r.akpCli.KargoCli.ApplyKargoInstance, r.upsertKubeConfig)
	if err != nil {
		return result, err
	}

	if plan.Workspace.ValueString() == "" {
		plan.Workspace = tftypes.StringValue(workspace.GetName())
	}
	return result, refreshKargoAgentState(ctx, diagnostics, r.akpCli.KargoCli, result, r.akpCli.OrgId, nil, plan)
}

func (r *AkpKargoAgentResource) applyKargoInstance(ctx context.Context, plan *types.KargoAgent, apiReq *kargov1.ApplyKargoInstanceRequest, isCreate bool, applyKargoInstance func(context.Context, *kargov1.ApplyKargoInstanceRequest) (*kargov1.ApplyKargoInstanceResponse, error), upsertKubeConfig func(ctx context.Context, plan *types.KargoAgent, isCreate bool) error) (*types.KargoAgent, error) {
	kubeconfig := plan.KubeConfig
	plan.KubeConfig = nil
	tflog.Debug(ctx, fmt.Sprintf("Apply Kargo agent request: %s", apiReq))
	_, err := applyKargoInstance(ctx, apiReq)
	if err != nil {
		return nil, fmt.Errorf("unable to create Kargo agent: %s", err)
	}

	if kubeconfig != nil {
		plan.KubeConfig = kubeconfig
		err = upsertKubeConfig(ctx, plan, isCreate)
		if err != nil {
			// Ensure kubeconfig won't be committed to state by setting it to nil
			plan.KubeConfig = nil
			return plan, fmt.Errorf("unable to apply kargo manifests: %s", err)
		}
	}

	return plan, nil
}

func (r *AkpKargoAgentResource) upsertKubeConfig(ctx context.Context, plan *types.KargoAgent, isCreate bool) error {
	// Apply agent manifests to clusters if the kubeconfig is specified for cluster.
	kubeconfig, err := getKubeconfig(ctx, plan.KubeConfig)
	if err != nil {
		return err
	}

	// Apply the manifests
	if kubeconfig != nil && isCreate {
		manifests, id, err := getKargoManifests(ctx, r.akpCli.KargoCli, r.akpCli.OrgId, plan)
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
		return waitKargoAgentHealthStatus(ctx, r.akpCli.KargoCli, r.akpCli.OrgId, plan)
	}
	return nil
}

func refreshKargoAgentState(ctx context.Context, diagnostics *diag.Diagnostics, client kargov1.KargoServiceGatewayClient, kargoAgent *types.KargoAgent,
	orgID string, state *tfsdk.State, plan *types.KargoAgent,
) error {
	agents, err := client.ListKargoInstanceAgents(ctx, &kargov1.ListKargoInstanceAgentsRequest{
		OrganizationId: orgID,
		InstanceId:     kargoAgent.InstanceID.ValueString(),
	})
	if err != nil {
		return errors.Wrap(err, "Unable to read Kargo agents")
	}
	var agent *kargov1.KargoAgent
	for _, a := range agents.GetAgents() {
		if a.GetName() == kargoAgent.Name.ValueString() {
			agent = a
			break
		}
	}
	if agent == nil {
		state.RemoveResource(ctx)
		return nil
	}
	tflog.Debug(ctx, fmt.Sprintf("current kargo agent: %s", agent))
	kargoAgent.Update(ctx, diagnostics, agent, plan)
	return nil
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

func getKargoManifests(ctx context.Context, client kargov1.KargoServiceGatewayClient, orgId string, kargoAgent *types.KargoAgent) (string, string, error) {
	agents, err := client.ListKargoInstanceAgents(ctx, &kargov1.ListKargoInstanceAgentsRequest{
		OrganizationId: orgId,
		InstanceId:     kargoAgent.InstanceID.ValueString(),
	})
	if err != nil {
		return "", "", errors.Wrap(err, "Unable to read Kargo agents")
	}
	var agent *kargov1.KargoAgent
	for _, a := range agents.GetAgents() {
		if a.GetName() == kargoAgent.Name.ValueString() {
			agent = a
			break
		}
	}
	if agent == nil {
		return "", "", errors.New("Unable to find Kargo agent")
	}

	k, err := waitKargoAgentReconStatus(ctx, client, agent, orgId, kargoAgent.InstanceID.ValueString())
	if err != nil {
		return "", "", errors.Wrap(err, "Unable to check kargo agent health status")
	}
	apiReq := &kargov1.GetKargoInstanceAgentManifestsRequest{
		OrganizationId: orgId,
		InstanceId:     kargoAgent.InstanceID.ValueString(),
		Id:             k.Id,
	}
	resChan, errChan, err := client.GetKargoInstanceAgentManifests(ctx, apiReq)
	if err != nil {
		return "", "", errors.Wrap(err, "Unable to download manifests")
	}
	res, err := readStream(resChan, errChan)
	if err != nil {
		return "", "", errors.Wrap(err, "Unable to parse manifests")
	}

	return string(res), k.Id, nil
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
			Id:             c.ID.ValueString(),
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

func waitKargoAgentReconStatus(ctx context.Context, client kargov1.KargoServiceGatewayClient, kargoAgent *kargov1.KargoAgent, orgId, instanceId string) (*kargov1.KargoAgent, error) {
	reconStatus := kargoAgent.GetReconciliationStatus()
	breakStatusesRecon := []reconv1.StatusCode{reconv1.StatusCode_STATUS_CODE_SUCCESSFUL, reconv1.StatusCode_STATUS_CODE_FAILED}

	for !slices.Contains(breakStatusesRecon, reconStatus.GetCode()) {
		time.Sleep(1 * time.Second)
		apiResp, err := client.GetKargoInstanceAgent(ctx, &kargov1.GetKargoInstanceAgentRequest{
			OrganizationId: orgId,
			InstanceId:     instanceId,
			Id:             kargoAgent.Id,
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
