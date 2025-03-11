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

	kargov1 "github.com/akuity/api-client-go/pkg/api/gen/kargo/v1"
	orgcv1 "github.com/akuity/api-client-go/pkg/api/gen/organization/v1"
	idv1 "github.com/akuity/api-client-go/pkg/api/gen/types/id/v1"
	httpctx "github.com/akuity/grpc-gateway-client/pkg/http/context"
	"github.com/akuity/terraform-provider-akp/akp/types"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ resource.Resource = &AkpKargoInstanceResource{}
var _ resource.ResourceWithImportState = &AkpKargoInstanceResource{}

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

	err := refreshKargoState(ctx, &resp.Diagnostics, r.akpCli.KargoCli, &data, r.akpCli.OrgId)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
	} else {
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	}
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
	_, err := r.akpCli.KargoCli.DeleteInstance(ctx, &kargov1.DeleteInstanceRequest{
		Id:             state.ID.ValueString(),
		OrganizationId: r.akpCli.OrgId,
	})

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

func (r *AkpKargoInstanceResource) upsert(ctx context.Context, diagnostics *diag.Diagnostics, plan *types.KargoInstance) error {
	ctx = httpctx.SetAuthorizationHeader(ctx, r.akpCli.Cred.Scheme(), r.akpCli.Cred.Credential())
	workspaces, err := r.akpCli.OrgCli.ListWorkspaces(ctx, &orgcv1.ListWorkspacesRequest{
		OrganizationId: r.akpCli.OrgId,
	})
	if err != nil {
		return errors.Wrap(err, "Unable to read workspaces")
	}
	var workspaceId string
	for _, w := range workspaces.GetWorkspaces() {
		if w.GetIsDefault() {
			workspaceId = w.GetId()
			break
		}
	}
	if workspaceId == "" {
		return errors.New("Default workspace not found")
	}
	apiReq := buildKargoApplyRequest(ctx, diagnostics, plan, r.akpCli.OrgId, workspaceId)
	if diagnostics.HasError() {
		return errors.New("Unable to build Kargo instance request")
	}
	tflog.Debug(ctx, fmt.Sprintf("Apply instance request: %s", apiReq))
	_, err = r.akpCli.KargoCli.ApplyKargoInstance(ctx, apiReq)
	if err != nil {
		return errors.Wrap(err, "Unable to upsert Kargo instance")
	}

	return refreshKargoState(ctx, diagnostics, r.akpCli.KargoCli, plan, r.akpCli.OrgId)
}

func buildKargoApplyRequest(ctx context.Context, diagnostics *diag.Diagnostics, kargo *types.KargoInstance, orgID, workspaceID string) *kargov1.ApplyKargoInstanceRequest {
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
	return applyReq
}

func buildKargo(ctx context.Context, diagnostics *diag.Diagnostics, kargo *types.KargoInstance) *structpb.Struct {
	apiKargo := kargo.Kargo.ToKargoAPIModel(ctx, diagnostics, kargo.Name.ValueString())
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

func refreshKargoState(ctx context.Context, diagnostics *diag.Diagnostics, client kargov1.KargoServiceGatewayClient, kargo *types.KargoInstance, orgID string) error {
	req := &kargov1.GetKargoInstanceRequest{
		OrganizationId: orgID,
		Name:           kargo.Name.ValueString(),
	}
	tflog.Debug(ctx, fmt.Sprintf("Get Kargo instance request: %s", req))
	resp, err := client.GetKargoInstance(ctx, req)
	if err != nil {
		return errors.Wrap(err, "Unable to read Kargo instance")
	}
	tflog.Debug(ctx, fmt.Sprintf("Get Kargo instance response: %s", resp))
	kargo.ID = tftypes.StringValue(resp.Instance.Id)
	exportReq := &kargov1.ExportKargoInstanceRequest{
		OrganizationId: orgID,
		Id:             kargo.ID.ValueString(),
		WorkspaceId:    resp.Instance.WorkspaceId,
	}
	tflog.Debug(ctx, fmt.Sprintf("Export Kargo instance request: %s", exportReq))
	exportResp, err := client.ExportKargoInstance(ctx, exportReq)
	if err != nil {
		return errors.Wrap(err, "Unable to export Kargo instance")
	}
	tflog.Debug(ctx, fmt.Sprintf("Export Kargo instance response: %s", exportResp))
	return kargo.Update(ctx, diagnostics, exportResp)
}
