package akp

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
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
		TypeNameSuffix:  "kargo_default_shard_agent",
		SchemaFunc:      kargoDefaultShardAgentSchema,
		CreateFunc:      kargoDefaultShardAgentCreate,
		ReadFunc:        kargoDefaultShardAgentRead,
		UpdateFunc:      kargoDefaultShardAgentUpdate,
		DeleteFunc:      kargoDefaultShardAgentDelete,
		ImportStateFunc: importSplitID("kargo_instance_id", "agent_id"),
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
	instance, err := getKargoInstanceByID(ctx, cli, data.KargoInstanceID.ValueString())
	if err != nil {
		return err
	}

	currentAgentID := instance.GetSpec().GetDefaultShardAgent()
	if currentAgentID == "" {
		return status.Errorf(codes.NotFound, "default shard agent was cleared externally")
	}

	data.ID = data.KargoInstanceID
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

func setDefaultShardAgent(ctx context.Context, cli *AkpCli, instanceID, agentID string) error {
	patchStruct, err := structpb.NewStruct(map[string]any{
		"spec": map[string]any{
			"defaultShardAgent": agentID,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create patch struct: %w", err)
	}

	_, err = retryWithBackoff(ctx, func(ctx context.Context) (*kargov1.PatchKargoInstanceResponse, error) {
		return cli.KargoCli.PatchKargoInstance(ctx, &kargov1.PatchKargoInstanceRequest{
			OrganizationId: cli.OrgId,
			Id:             instanceID,
			Patch:          patchStruct,
		})
	}, "PatchKargoInstance")
	if err != nil {
		return fmt.Errorf("failed to patch kargo instance: %w", err)
	}

	return nil
}
