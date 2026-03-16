package akp

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"

	httpctx "github.com/akuity/grpc-gateway-client/pkg/http/context"
	kargov1 "github.com/akuity/api-client-go/pkg/api/gen/kargo/v1"
)

var (
	_ resource.Resource                = &AkpKargoDefaultShardAgentResource{}
	_ resource.ResourceWithImportState = &AkpKargoDefaultShardAgentResource{}
)

func NewAkpKargoDefaultShardAgentResource() resource.Resource {
	return &AkpKargoDefaultShardAgentResource{}
}

type AkpKargoDefaultShardAgentResource struct {
	akpCli *AkpCli
}

type KargoDefaultShardAgentResourceModel struct {
	ID              tftypes.String `tfsdk:"id"`
	KargoInstanceID tftypes.String `tfsdk:"kargo_instance_id"`
	AgentID         tftypes.String `tfsdk:"agent_id"`
}

func (r *AkpKargoDefaultShardAgentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_kargo_default_shard_agent"
}

func (r *AkpKargoDefaultShardAgentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *AkpKargoDefaultShardAgentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	tflog.Debug(ctx, "Creating Kargo default shard agent binding")
	var plan KargoDefaultShardAgentResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = httpctx.SetAuthorizationHeader(ctx, r.akpCli.Cred.Scheme(), r.akpCli.Cred.Credential())

	if err := r.setDefaultShardAgent(ctx, plan.KargoInstanceID.ValueString(), plan.AgentID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to set default shard agent: %s", err))
		return
	}

	plan.ID = tftypes.StringValue(plan.KargoInstanceID.ValueString())
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AkpKargoDefaultShardAgentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	tflog.Debug(ctx, "Reading Kargo default shard agent binding")
	var data KargoDefaultShardAgentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = httpctx.SetAuthorizationHeader(ctx, r.akpCli.Cred.Scheme(), r.akpCli.Cred.Credential())

	instance, err := r.getKargoInstance(ctx, data.KargoInstanceID.ValueString())
	if err != nil {
		if status.Code(err) == codes.NotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get Kargo instance: %s", err))
		return
	}

	currentAgentID := instance.GetSpec().GetDefaultShardAgent()
	if currentAgentID == "" {
		// Default shard agent was cleared externally
		resp.State.RemoveResource(ctx)
		return
	}

	data.AgentID = tftypes.StringValue(currentAgentID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AkpKargoDefaultShardAgentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Debug(ctx, "Updating Kargo default shard agent binding")
	var plan KargoDefaultShardAgentResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = httpctx.SetAuthorizationHeader(ctx, r.akpCli.Cred.Scheme(), r.akpCli.Cred.Credential())

	if err := r.setDefaultShardAgent(ctx, plan.KargoInstanceID.ValueString(), plan.AgentID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update default shard agent: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AkpKargoDefaultShardAgentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	tflog.Debug(ctx, "Deleting Kargo default shard agent binding")
	var state KargoDefaultShardAgentResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = httpctx.SetAuthorizationHeader(ctx, r.akpCli.Cred.Scheme(), r.akpCli.Cred.Credential())

	if err := r.setDefaultShardAgent(ctx, state.KargoInstanceID.ValueString(), ""); err != nil {
		if status.Code(err) == codes.NotFound {
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to clear default shard agent: %s", err))
		return
	}
}

func (r *AkpKargoDefaultShardAgentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: kargo_instance_id/agent_id
	idParts := strings.Split(req.ID, "/")
	if len(idParts) != 2 || idParts[0] == "" || idParts[1] == "" {
		resp.Diagnostics.AddError(
			"Unexpected Import Identifier",
			fmt.Sprintf("Expected import identifier with format: kargo_instance_id/agent_id. Got: %q", req.ID),
		)
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("kargo_instance_id"), idParts[0])...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("agent_id"), idParts[1])...)
}

func (r *AkpKargoDefaultShardAgentResource) getKargoInstance(ctx context.Context, instanceID string) (*kargov1.KargoInstance, error) {
	instancesResp, err := retryWithBackoff(ctx, func(ctx context.Context) (*kargov1.ListKargoInstancesResponse, error) {
		return r.akpCli.KargoCli.ListKargoInstances(ctx, &kargov1.ListKargoInstancesRequest{
			OrganizationId: r.akpCli.OrgId,
		})
	}, "ListKargoInstances")
	if err != nil {
		return nil, errors.Wrap(err, "failed to list kargo instances")
	}

	for _, instance := range instancesResp.GetInstances() {
		if instance.GetId() == instanceID {
			return instance, nil
		}
	}

	return nil, status.Errorf(codes.NotFound, "kargo instance %s not found", instanceID)
}

func (r *AkpKargoDefaultShardAgentResource) setDefaultShardAgent(ctx context.Context, instanceID, agentID string) error {
	patchStruct, err := structpb.NewStruct(map[string]any{
		"spec": map[string]any{
			"defaultShardAgent": agentID,
		},
	})
	if err != nil {
		return errors.Wrap(err, "failed to create patch struct")
	}

	_, err = retryWithBackoff(ctx, func(ctx context.Context) (*kargov1.PatchKargoInstanceResponse, error) {
		return r.akpCli.KargoCli.PatchKargoInstance(ctx, &kargov1.PatchKargoInstanceRequest{
			OrganizationId: r.akpCli.OrgId,
			Id:             instanceID,
			Patch:          patchStruct,
		})
	}, "PatchKargoInstance")
	if err != nil {
		return errors.Wrap(err, "failed to patch kargo instance")
	}

	return nil
}
