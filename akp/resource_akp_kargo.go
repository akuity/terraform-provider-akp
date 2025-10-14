package akp

import (
	"context"
	"encoding/json"
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
	httpctx "github.com/akuity/grpc-gateway-client/pkg/http/context"
	"github.com/akuity/terraform-provider-akp/akp/types"
)

// Ensure provider defined types fully satisfy framework interfaces
var (
	_ resource.Resource                = &AkpKargoInstanceResource{}
	_ resource.ResourceWithImportState = &AkpKargoInstanceResource{}
)

func NewAkpKargoInstanceResource() resource.Resource {
	return &AkpKargoInstanceResource{}
}

type AkpKargoInstanceResource struct {
	akpCli *AkpCli
}

func (r *AkpKargoInstanceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kargo_instance"
}

func (r *AkpKargoInstanceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *AkpKargoInstanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Debug(ctx, "Creating an instance")
	var plan types.KargoInstance

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.upsert(ctx, &resp.Diagnostics, &plan)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
	} else {
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	}
}

func (r *AkpKargoInstanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Debug(ctx, "Reading a Kargo instance")
	var data types.KargoInstance
	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = httpctx.SetAuthorizationHeader(ctx, r.akpCli.Cred.Scheme(), r.akpCli.Cred.Credential())

	err := refreshKargoState(ctx, &resp.Diagnostics, r.akpCli.KargoCli, &data, r.akpCli.OrgId, false)
	if err != nil {
		handleReadResourceError(ctx, resp, err)
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AkpKargoInstanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Debug(ctx, "Updating a Kargo instance")
	var plan types.KargoInstance

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.upsert(ctx, &resp.Diagnostics, &plan)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
	} else {
		resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	}
}

func (r *AkpKargoInstanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Debug(ctx, "Deleting a Kargo instance")
	var state types.KargoInstance

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx = httpctx.SetAuthorizationHeader(ctx, r.akpCli.Cred.Scheme(), r.akpCli.Cred.Credential())
	_, err := retryWithBackoff(ctx, func(ctx context.Context) (*kargov1.DeleteInstanceResponse, error) {
		return r.akpCli.KargoCli.DeleteInstance(ctx, &kargov1.DeleteInstanceRequest{
			Id:             state.ID.ValueString(),
			OrganizationId: r.akpCli.OrgId,
		})
	}, "DeleteInstance")
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete Argo CD instance, got error: %s", err))
		return
	}
	// Give it some time to remove the Kargo instance. This is useful when the terraform provider is performing a replace operation, to give it enough time to destroy the previous instance.
	time.Sleep(2 * time.Second)
}

func (r *AkpKargoInstanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
}

func (r *AkpKargoInstanceResource) validateAKIntelligenceFeatures(ctx context.Context, plan *types.KargoInstance) error {
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

func (r *AkpKargoInstanceResource) upsert(ctx context.Context, diagnostics *diag.Diagnostics, plan *types.KargoInstance) error {
	ctx = httpctx.SetAuthorizationHeader(ctx, r.akpCli.Cred.Scheme(), r.akpCli.Cred.Credential())

	if err := r.validateAKIntelligenceFeatures(ctx, plan); err != nil {
		return err
	}

	workspace, err := getWorkspace(ctx, r.akpCli.OrgCli, r.akpCli.OrgId, plan.Workspace.ValueString())
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get workspace. %s", err))
		return errors.New("Unable to get workspace")
	}
	apiReq := buildKargoApplyRequest(ctx, diagnostics, r.akpCli.KargoCli, plan, r.akpCli.OrgId, workspace.GetId())
	if diagnostics.HasError() {
		return errors.New("Unable to build Kargo instance request")
	}
	tflog.Debug(ctx, fmt.Sprintf("Apply instance request: %s", apiReq))

	_, err = retryWithBackoff(ctx, func(ctx context.Context) (*kargov1.ApplyKargoInstanceResponse, error) {
		return r.akpCli.KargoCli.ApplyKargoInstance(ctx, apiReq)
	}, "ApplyKargoInstance")
	if err != nil {
		return errors.Wrap(err, "Unable to upsert Kargo instance")
	}

	if plan.Workspace.ValueString() == "" {
		plan.Workspace = tftypes.StringValue(workspace.GetName())
	}

	getResourceFunc := func(ctx context.Context) (*kargov1.GetKargoInstanceResponse, error) {
		return retryWithBackoff(ctx, func(ctx context.Context) (*kargov1.GetKargoInstanceResponse, error) {
			return r.akpCli.KargoCli.GetKargoInstance(ctx, &kargov1.GetKargoInstanceRequest{
				OrganizationId: r.akpCli.OrgId,
				Name:           plan.Name.ValueString(),
				WorkspaceId:    plan.Workspace.ValueString(),
			})
		}, "GetKargoInstance")
	}

	getStatusFunc := func(resp *kargov1.GetKargoInstanceResponse) healthv1.StatusCode {
		if resp == nil || resp.Instance == nil {
			return healthv1.StatusCode_STATUS_CODE_UNKNOWN
		}
		return resp.Instance.GetHealthStatus().GetCode()
	}

	waitErr := waitForStatus(
		ctx,
		getResourceFunc,
		getStatusFunc,
		[]healthv1.StatusCode{healthv1.StatusCode_STATUS_CODE_HEALTHY},
		10*time.Second,
		5*time.Minute,
		fmt.Sprintf("Instance %s", plan.Name.ValueString()),
		"health",
	)

	if waitErr != nil {
		diagnostics.AddError("Instance Wait Error", fmt.Sprintf("Instance '%s' did not become healthy: %s", plan.Name.ValueString(), waitErr.Error()))
		return waitErr
	}

	return refreshKargoState(ctx, diagnostics, r.akpCli.KargoCli, plan, r.akpCli.OrgId, false)
}

func buildKargoApplyRequest(ctx context.Context, diagnostics *diag.Diagnostics, client kargov1.KargoServiceGatewayClient, kargo *types.KargoInstance, orgID, workspaceID string) *kargov1.ApplyKargoInstanceRequest {
	idType := idv1.Type_NAME
	id := kargo.Name.ValueString()

	if !kargo.ID.IsNull() && kargo.ID.ValueString() != "" {
		idType = idv1.Type_ID
		id = kargo.ID.ValueString()
	}

	agentMaps := buildAgentMaps(ctx, client, id, orgID)

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
}

// kargoSupportedGroupKinds maps supported GroupKinds to optional per-object validators.
var kargoSupportedGroupKinds = map[schema.GroupKind]func(*unstructured.Unstructured) error{
	{Group: "kargo.akuity.io", Kind: "Project"}:               nil,
	{Group: "kargo.akuity.io", Kind: "Warehouse"}:             nil,
	{Group: "kargo.akuity.io", Kind: "Stage"}:                 nil,
	{Group: "kargo.akuity.io", Kind: "PromotionTask"}:         nil,
	{Group: "kargo.akuity.io", Kind: "ClusterPromotionTask"}:  nil,
	{Group: "argoproj.io", Kind: "AnalysisTemplate"}:          nil,
	{Group: "rbac.authorization.k8s.io", Kind: "Role"}:        nil,
	{Group: "rbac.authorization.k8s.io", Kind: "RoleBinding"}: nil,
	{Group: "", Kind: "ServiceAccount"}:                       nil,
	{Group: "", Kind: "ConfigMap"}:                            nil,
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

func buildKargo(ctx context.Context, diagnostics *diag.Diagnostics, kargo *types.KargoInstance, agentMaps *types.AgentMaps) *structpb.Struct {
	apiKargo := kargo.Kargo.ToKargoAPIModel(ctx, diagnostics, kargo.Name.ValueString(), agentMaps)
	if diagnostics.HasError() {
		return nil
	}
	jsonBytes, err := json.Marshal(apiKargo)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to marshal Kargo instance. %s", err))
		return nil
	}

	var rawMap map[string]any
	if err = json.Unmarshal(jsonBytes, &rawMap); err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to unmarshal Kargo instance. %s", err))
		return nil
	}

	if spec, ok := rawMap["spec"].(map[string]any); ok {
		_, fok := spec["fqdn"].(string)
		if !fok {
			spec["fqdn"] = ""
		}
	}

	s, err := structpb.NewStruct(rawMap)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Kargo instance struct. %s", err))
	}
	return s
}

func refreshKargoState(ctx context.Context, diagnostics *diag.Diagnostics, client kargov1.KargoServiceGatewayClient, kargo *types.KargoInstance, orgID string, isDataSource bool) error {
	req := &kargov1.GetKargoInstanceRequest{
		OrganizationId: orgID,
		Name:           kargo.Name.ValueString(),
	}
	tflog.Debug(ctx, fmt.Sprintf("Get Kargo instance request: %s", req))
	resp, err := retryWithBackoff(ctx, func(ctx context.Context) (*kargov1.GetKargoInstanceResponse, error) {
		return client.GetKargoInstance(ctx, req)
	}, "GetKargoInstance")
	if err != nil {
		return errors.Wrap(err, "Unable to read Kargo instance")
	}
	tflog.Debug(ctx, fmt.Sprintf("Get Kargo instance response: %s", resp))
	kargo.ID = tftypes.StringValue(resp.Instance.Id)

	agentMaps := buildAgentMaps(ctx, client, kargo.ID.ValueString(), orgID)

	exportReq := &kargov1.ExportKargoInstanceRequest{
		OrganizationId: orgID,
		Id:             kargo.ID.ValueString(),
		WorkspaceId:    resp.Instance.WorkspaceId,
	}
	tflog.Debug(ctx, fmt.Sprintf("Export Kargo instance request: %s", exportReq))
	exportResp, err := retryWithBackoff(ctx, func(ctx context.Context) (*kargov1.ExportKargoInstanceResponse, error) {
		return client.ExportKargoInstance(ctx, exportReq)
	}, "ExportKargoInstance")
	if err != nil {
		return errors.Wrap(err, "Unable to export Kargo instance")
	}
	tflog.Debug(ctx, fmt.Sprintf("Export Kargo instance response: %s", exportResp))
	return kargo.Update(ctx, diagnostics, exportResp, agentMaps, isDataSource)
}

func buildAgentMaps(ctx context.Context, client kargov1.KargoServiceGatewayClient, instanceID, orgID string) *types.AgentMaps {
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
			// if no workspace name is provided, return the default workspace
			return w, nil
		}
		if w.Name == name {
			return w, nil
		}
	}

	return nil, fmt.Errorf("workspace %s not found", name)
}
