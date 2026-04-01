package akp

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"

	kargov1 "github.com/akuity/api-client-go/pkg/api/gen/kargo/v1"
)

type KargoDefaultShardAgentResourceModel struct {
	ID              tftypes.String `tfsdk:"id"`
	KargoInstanceID tftypes.String `tfsdk:"kargo_instance_id"`
	AgentID         tftypes.String `tfsdk:"agent_id"`
}

func NewAkpKargoDefaultShardAgentResource() resource.Resource {
	return &GenericResource[KargoDefaultShardAgentResourceModel]{
		TypeNameSuffix: "kargo_default_shard_agent",
		SchemaFunc:     kargoDefaultShardAgentSchema,
		CreateFunc:     kargoDefaultShardAgentCreate,
		ReadFunc:       kargoDefaultShardAgentRead,
		UpdateFunc:     kargoDefaultShardAgentUpdate,
		DeleteFunc:     kargoDefaultShardAgentDelete,
		ImportStateFunc: func(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
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
		},
	}
}

func kargoDefaultShardAgentCreate(ctx context.Context, cli *AkpCli, _ *diag.Diagnostics, plan *KargoDefaultShardAgentResourceModel) (*KargoDefaultShardAgentResourceModel, error) {
	if err := setDefaultShardAgent(ctx, cli, plan.KargoInstanceID.ValueString(), plan.AgentID.ValueString()); err != nil {
		return nil, fmt.Errorf("unable to set default shard agent: %w", err)
	}
	plan.ID = tftypes.StringValue(plan.KargoInstanceID.ValueString())
	return plan, nil
}

func kargoDefaultShardAgentRead(ctx context.Context, cli *AkpCli, _ *diag.Diagnostics, data *KargoDefaultShardAgentResourceModel) error {
	instance, err := getKargoInstanceForDefaultShard(ctx, cli, data.KargoInstanceID.ValueString())
	if err != nil {
		return err
	}

	currentAgentID := instance.GetSpec().GetDefaultShardAgent()
	if currentAgentID == "" {
		return status.Errorf(codes.NotFound, "default shard agent was cleared externally")
	}

	data.AgentID = tftypes.StringValue(currentAgentID)
	return nil
}

func kargoDefaultShardAgentUpdate(ctx context.Context, cli *AkpCli, _ *diag.Diagnostics, plan *KargoDefaultShardAgentResourceModel) (*KargoDefaultShardAgentResourceModel, error) {
	if err := setDefaultShardAgent(ctx, cli, plan.KargoInstanceID.ValueString(), plan.AgentID.ValueString()); err != nil {
		return nil, fmt.Errorf("unable to update default shard agent: %w", err)
	}
	return plan, nil
}

func kargoDefaultShardAgentDelete(ctx context.Context, cli *AkpCli, _ *diag.Diagnostics, state *KargoDefaultShardAgentResourceModel) error {
	if err := setDefaultShardAgent(ctx, cli, state.KargoInstanceID.ValueString(), ""); err != nil {
		if status.Code(err) == codes.NotFound {
			return nil
		}
		return fmt.Errorf("unable to clear default shard agent: %w", err)
	}
	return nil
}

func getKargoInstanceForDefaultShard(ctx context.Context, cli *AkpCli, instanceID string) (*kargov1.KargoInstance, error) {
	instancesResp, err := retryWithBackoff(ctx, func(ctx context.Context) (*kargov1.ListKargoInstancesResponse, error) {
		return cli.KargoCli.ListKargoInstances(ctx, &kargov1.ListKargoInstancesRequest{
			OrganizationId: cli.OrgId,
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

func setDefaultShardAgent(ctx context.Context, cli *AkpCli, instanceID, agentID string) error {
	patchStruct, err := structpb.NewStruct(map[string]any{
		"spec": map[string]any{
			"defaultShardAgent": agentID,
		},
	})
	if err != nil {
		return errors.Wrap(err, "failed to create patch struct")
	}

	_, err = retryWithBackoff(ctx, func(ctx context.Context) (*kargov1.PatchKargoInstanceResponse, error) {
		return cli.KargoCli.PatchKargoInstance(ctx, &kargov1.PatchKargoInstanceRequest{
			OrganizationId: cli.OrgId,
			Id:             instanceID,
			Patch:          patchStruct,
		})
	}, "PatchKargoInstance")
	if err != nil {
		return errors.Wrap(err, "failed to patch kargo instance")
	}

	return nil
}
