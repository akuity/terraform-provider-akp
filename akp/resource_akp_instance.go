package akp

import (
	"context"
	"fmt"
	"time"

	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	idv1 "github.com/akuity/api-client-go/pkg/api/gen/types/id/v1"
	healthv1 "github.com/akuity/api-client-go/pkg/api/gen/types/status/health/v1"
	reconv1 "github.com/akuity/api-client-go/pkg/api/gen/types/status/reconciliation/v1"
	ctxutil "github.com/akuity/api-client-go/pkg/utils/context"
	akptypes "github.com/akuity/terraform-provider-akp/akp/types"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"golang.org/x/exp/slices"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// Ensure provider defined types fully satisfy framework interfaces
var _ resource.Resource = &AkpInstanceResource{}
var _ resource.ResourceWithImportState = &AkpInstanceResource{}

func NewAkpInstanceResource() resource.Resource {
	return &AkpInstanceResource{}
}

// AkpInstanceResource defines the resource implementation.
type AkpInstanceResource struct {
	akpCli *AkpCli
}

func (r *AkpInstanceResource) waitInstanceHealthStatus(ctx context.Context, instance *argocdv1.Instance) (*argocdv1.Instance, error) {
	healthStatus := instance.GetHealthStatus()
	breakStatusesHealth := []healthv1.StatusCode{healthv1.StatusCode_STATUS_CODE_HEALTHY, healthv1.StatusCode_STATUS_CODE_DEGRADED}

	for !slices.Contains(breakStatusesHealth, healthStatus.GetCode()) {
		time.Sleep(2 * time.Second)
		apiReq := &argocdv1.GetInstanceRequest{
			OrganizationId: r.akpCli.OrgId,
			Id:             instance.GetId(),
			IdType:         idv1.Type_ID,
		}
		tflog.Debug(ctx, fmt.Sprintf("Api Req: %s", apiReq.String()))
		apiResp, err := r.akpCli.Cli.GetInstance(ctx, apiReq)
		tflog.Debug(ctx, fmt.Sprintf("Api Resp: %s", apiResp.String()))
		if err != nil {
			return nil, err
		}
		instance = apiResp.GetInstance()
		healthStatus = instance.GetHealthStatus()
		tflog.Info(ctx, fmt.Sprintf("Instance health status: %s", healthStatus.String()))
	}
	return instance, nil
}

func (r *AkpInstanceResource) waitInstanceReconStatus(ctx context.Context, instance *argocdv1.Instance) (*argocdv1.Instance, error) {
	reconStatus := instance.GetReconciliationStatus()
	breakStatusesRecon := []reconv1.StatusCode{reconv1.StatusCode_STATUS_CODE_SUCCESSFUL, reconv1.StatusCode_STATUS_CODE_FAILED}

	for !slices.Contains(breakStatusesRecon, reconStatus.GetCode()) {
		time.Sleep(1 * time.Second)
		apiReq := &argocdv1.GetInstanceRequest{
			OrganizationId: r.akpCli.OrgId,
			Id:             instance.GetId(),
			IdType:         idv1.Type_ID,
		}
		tflog.Debug(ctx, fmt.Sprintf("Api Req: %s", apiReq.String()))
		apiResp, err := r.akpCli.Cli.GetInstance(ctx, apiReq)
		tflog.Debug(ctx, fmt.Sprintf("Api Resp: %s", apiResp.String()))
		if err != nil {
			return nil, err
		}
		instance = apiResp.GetInstance()
		reconStatus = instance.GetReconciliationStatus()
		tflog.Info(ctx, fmt.Sprintf("Instance reconciliation status: %s", reconStatus.String()))
	}
	return instance, nil
}

func (r *AkpInstanceResource) UpdateInstance(ctx context.Context, id string, to *argocdv1.Instance) diag.Diagnostics {
	diag := diag.Diagnostics{}
	apiReq := &argocdv1.UpdateInstanceRequest{
		OrganizationId: r.akpCli.OrgId,
		Id:             id,
		Instance:       to,
	}
	tflog.Debug(ctx, fmt.Sprintf("Api Req: %s", apiReq.String()))
	apiResp, err := r.akpCli.Cli.UpdateInstance(ctx, apiReq)
	tflog.Debug(ctx, fmt.Sprintf("Api Resp: %s", apiResp.String()))
	if err != nil {
		diag.AddError("Client Error", fmt.Sprintf("Unable to update Argo CD instance: %s", err))
		return diag
	}
	to, err = r.waitInstanceReconStatus(ctx, apiResp.GetInstance())
	if err != nil {
		diag.AddError("Client Error", fmt.Sprintf("Unable to check Argo CD reconciliation instance status: %s", err))
	}
	return diag
}

func (r *AkpInstanceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_instance"
}

func (r *AkpInstanceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *AkpInstanceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Debug(ctx, "Creating an Argo CD Instance")
	var plan *akptypes.AkpInstance

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)

	if resp.Diagnostics.HasError() {
		return
	}

	ctx = ctxutil.SetClientCredential(ctx, r.akpCli.Cred)
	description := plan.Description.ValueString()
	apiReq := &argocdv1.CreateInstanceRequest{
		OrganizationId: r.akpCli.OrgId,
		Name:           plan.Name.ValueString(),
		Version:        plan.Version.ValueString(),
		Description:    &description,
	}
	tflog.Debug(ctx, fmt.Sprintf("Api Req: %s", apiReq.String()))
	apiResp, err := r.akpCli.Cli.CreateInstance(ctx, apiReq)
	tflog.Debug(ctx, fmt.Sprintf("Api Resp: %s", apiResp.String()))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Argo CD instance. %s", err))
		return
	}
	instance, err := r.waitInstanceHealthStatus(ctx, apiResp.GetInstance())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to check instance health. %s", err))
		return
	}
	tflog.Info(ctx, "Argo CD instance created")
	if instance.GetHealthStatus().GetCode() != healthv1.StatusCode_STATUS_CODE_HEALTHY {
		resp.Diagnostics.AddError("Server Error", fmt.Sprintf("Instance is not healthy. %s", err))
	} else {
		// Update the instance
	    resp.Diagnostics.Append(plan.As(instance)...)
		tflog.Debug(ctx, fmt.Sprintf("Updating Instance to %s", instance))
		resp.Diagnostics.Append(r.UpdateInstance(ctx, instance.Id, instance)...)
		tflog.Info(ctx, "Argo CD instance updated")
	}
	state := &akptypes.AkpInstance{}
	resp.Diagnostics.Append(state.UpdateFrom(instance)...)
	tflog.Debug(ctx, "Updating State")
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *AkpInstanceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Debug(ctx, "Reading an Argo CD Instance")
	var state *akptypes.AkpInstance

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = ctxutil.SetClientCredential(ctx, r.akpCli.Cred)
	apiResp, err := r.akpCli.Cli.GetInstance(ctx, &argocdv1.GetInstanceRequest{
		Id:             state.Id.ValueString(),
		IdType:         idv1.Type_ID,
		OrganizationId: r.akpCli.OrgId,
	})
	switch status.Code(err) {
	case codes.OK:
		tflog.Debug(ctx, fmt.Sprintf("Api Response: %s", apiResp))
	case codes.NotFound:
		resp.State.RemoveResource(ctx)
		return
	default:
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read Argo CD instance. %s", err))
		return
	}

	tflog.Debug(ctx, "Got Argo CD instance")
	resp.Diagnostics.Append(state.UpdateFrom(apiResp.GetInstance())...)

	tflog.Debug(ctx, "Updating State")
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *AkpInstanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan *akptypes.AkpInstance

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	instance := &argocdv1.Instance{}
	resp.Diagnostics.Append(plan.As(instance)...)
	ctx = ctxutil.SetClientCredential(ctx, r.akpCli.Cred)
	resp.Diagnostics.Append(r.UpdateInstance(ctx, plan.Id.ValueString(), instance)...)
	state := &akptypes.AkpInstance{}
	resp.Diagnostics.Append(state.UpdateFrom(instance)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *AkpInstanceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Debug(ctx, "Deleting an Argo CD Instance")
	var state *akptypes.AkpInstance

	diags := req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx = ctxutil.SetClientCredential(ctx, r.akpCli.Cred)
	_, err := r.akpCli.Cli.DeleteInstance(ctx, &argocdv1.DeleteInstanceRequest{
		Id:             state.Id.ValueString(),
		OrganizationId: r.akpCli.OrgId,
	})

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete Argo CD instance, got error: %s", err))
		return
	}
	tflog.Info(ctx, "Instance deleted")
}

func (r *AkpInstanceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
