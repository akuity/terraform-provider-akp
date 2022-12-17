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
	status "google.golang.org/grpc/status"
	codes "google.golang.org/grpc/codes"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"golang.org/x/exp/slices"
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
	apiResp, err := r.akpCli.Cli.CreateOrganizationInstance(ctx, &argocdv1.CreateOrganizationInstanceRequest{
		OrganizationId: r.akpCli.OrgId,
		Name:           plan.Name.ValueString(),
		Version:        plan.Version.ValueString(),
		Description:    &description,
	})

	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create Argo CD instance. %s", err))
		return
	}
	breakStatusesHealth := []healthv1.StatusCode{healthv1.StatusCode_STATUS_CODE_HEALTHY, healthv1.StatusCode_STATUS_CODE_DEGRADED}
	instance := apiResp.GetInstance()
	healthStatus := instance.GetHealthStatus()
	for !slices.Contains(breakStatusesHealth, healthStatus.GetCode()) {
		time.Sleep(2 * time.Second)
		apiResp2, err := r.akpCli.Cli.GetOrganizationInstance(ctx, &argocdv1.GetOrganizationInstanceRequest{
			OrganizationId: r.akpCli.OrgId,
			Id:             instance.GetId(),
			IdType:         idv1.Type_ID,
		})
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to check health of Argo CD instance. %s", err))
			return
		}
		instance = apiResp2.GetInstance()
		healthStatus = instance.GetHealthStatus()
		tflog.Debug(ctx, fmt.Sprintf("Argo CD instance status: %s", healthStatus.String()))
	}
	if instance.GetHealthStatus().GetCode() != healthv1.StatusCode_STATUS_CODE_HEALTHY {
		resp.Diagnostics.AddError("Server Error", fmt.Sprintf("Instance is not healthy. %s", err))
		return
	}
	tflog.Info(ctx, "Argo CD instance created")

	protoInstance := &akptypes.ProtoInstance{Instance: instance}
	state := protoInstance.FromProto()
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
	apiResp, err := r.akpCli.Cli.GetOrganizationInstance(ctx, &argocdv1.GetOrganizationInstanceRequest{
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

	tflog.Info(ctx, "Got Argo CD instance")
	instance := apiResp.GetInstance()
	protoInstance := &akptypes.ProtoInstance{Instance: instance}
	state = protoInstance.FromProto()

	tflog.Debug(ctx, "Updating State")
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *AkpInstanceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan *akptypes.AkpInstance

	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	protoPlan, diag := plan.ToProto()
	resp.Diagnostics.Append(diag...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx = ctxutil.SetClientCredential(ctx, r.akpCli.Cred)
	apiReq := &argocdv1.UpdateOrganizationInstanceRequest{
		OrganizationId: r.akpCli.OrgId,
		Id:             plan.Id.ValueString(),
		Instance:       protoPlan,
	}
	tflog.Debug(ctx, fmt.Sprintf("apiReq: %s", apiReq))
	apiResp, err := r.akpCli.Cli.UpdateOrganizationInstance(ctx, apiReq)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update Argo CD instance, got error: %s", err))
		return
	}
	breakStatusesRecon := []reconv1.StatusCode{reconv1.StatusCode_STATUS_CODE_SUCCESSFUL, reconv1.StatusCode_STATUS_CODE_FAILED}
	instance := apiResp.GetInstance()
	reconStatus := instance.GetReconciliationStatus()
	for !slices.Contains(breakStatusesRecon, reconStatus.GetCode()) {
		time.Sleep(2 * time.Second)
		apiResp2, err := r.akpCli.Cli.GetOrganizationInstance(ctx, &argocdv1.GetOrganizationInstanceRequest{
			OrganizationId: r.akpCli.OrgId,
			Id:             instance.GetId(),
			IdType:         idv1.Type_ID,
		})
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to check health of Argo CD instance. %s", err))
			return
		}
		instance = apiResp2.GetInstance()
		reconStatus = instance.GetReconciliationStatus()
		tflog.Debug(ctx, fmt.Sprintf("Argo CD instance status: %s", reconStatus.String()))
	}
	protoInstance := &akptypes.ProtoInstance{Instance: instance}
	state := protoInstance.FromProto()
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
	_, err := r.akpCli.Cli.DeleteOrganizationInstance(ctx, &argocdv1.DeleteOrganizationInstanceRequest{
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
