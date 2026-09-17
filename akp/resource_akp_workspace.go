package akp

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	orgcv1 "github.com/akuity/api-client-go/pkg/api/gen/organization/v1"
	"github.com/akuity/terraform-provider-akp/akp/types"
)

func NewAkpWorkspaceResource() resource.Resource {
	return &GenericResource[types.Workspace]{
		TypeNameSuffix: "workspace",
		SchemaFunc:     workspaceSchema,
		CreateFunc:     workspaceCreate,
		ReadFunc:       workspaceRead,
		UpdateFunc:     workspaceUpdate,
		DeleteFunc:     workspaceDelete,
		ImportStateFunc: func(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
		},
	}
}

func workspaceCreate(ctx context.Context, cli *AkpCli, _ *diag.Diagnostics, plan *types.Workspace) (*types.Workspace, error) {
	createReq := &orgcv1.CreateWorkspaceRequest{
		OrganizationId: cli.OrgId,
		Name:           plan.Name.ValueString(),
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		desc := plan.Description.ValueString()
		createReq.Description = &desc
	}

	resp, err := retryWithBackoff(ctx, func(ctx context.Context) (*orgcv1.CreateWorkspaceResponse, error) {
		return cli.OrgCli.CreateWorkspace(ctx, createReq)
	}, "CreateWorkspace")
	if err != nil {
		return nil, fmt.Errorf("unable to create workspace: %w", err)
	}
	applyWorkspaceResponse(plan, resp.GetWorkspace())
	return plan, nil
}

func workspaceRead(ctx context.Context, cli *AkpCli, _ *diag.Diagnostics, data *types.Workspace) error {
	resp, err := retryWithBackoff(ctx, func(ctx context.Context) (*orgcv1.GetWorkspaceResponse, error) {
		return cli.OrgCli.GetWorkspace(ctx, &orgcv1.GetWorkspaceRequest{
			OrganizationId: cli.OrgId,
			Id:             data.ID.ValueString(),
		})
	}, "GetWorkspace")
	if err != nil {
		return err
	}
	if resp.GetWorkspace() == nil {
		return status.Error(codes.NotFound, "workspace not found")
	}
	applyWorkspaceResponse(data, resp.GetWorkspace())
	return nil
}

func workspaceUpdate(ctx context.Context, cli *AkpCli, _ *diag.Diagnostics, plan *types.Workspace) (*types.Workspace, error) {
	updateReq := &orgcv1.UpdateWorkspaceRequest{
		OrganizationId: cli.OrgId,
		Id:             plan.ID.ValueString(),
		Name:           plan.Name.ValueString(),
	}
	if !plan.Description.IsNull() && !plan.Description.IsUnknown() {
		desc := plan.Description.ValueString()
		updateReq.Description = &desc
	}

	resp, err := retryWithBackoff(ctx, func(ctx context.Context) (*orgcv1.UpdateWorkspaceResponse, error) {
		return cli.OrgCli.UpdateWorkspace(ctx, updateReq)
	}, "UpdateWorkspace")
	if err != nil {
		return nil, fmt.Errorf("unable to update workspace: %w", err)
	}
	applyWorkspaceResponse(plan, resp.GetWorkspace())
	return plan, nil
}

func workspaceDelete(ctx context.Context, cli *AkpCli, _ *diag.Diagnostics, state *types.Workspace) error {
	if state.IsDefault.ValueBool() {
		return fmt.Errorf("cannot delete the default workspace %q via Terraform; remove it from state with `terraform state rm` instead", state.Name.ValueString())
	}
	_, err := retryWithBackoff(ctx, func(ctx context.Context) (*orgcv1.DeleteWorkspaceResponse, error) {
		resp, err := cli.OrgCli.DeleteWorkspace(ctx, &orgcv1.DeleteWorkspaceRequest{
			OrganizationId: cli.OrgId,
			Id:             state.ID.ValueString(),
		})
		if isGoneErr(err) {
			return resp, nil
		}
		return resp, err
	}, "DeleteWorkspace")
	if err != nil {
		return fmt.Errorf("unable to delete workspace: %w", err)
	}
	return nil
}

func applyWorkspaceResponse(data *types.Workspace, ws *orgcv1.Workspace) {
	if ws == nil {
		return
	}
	data.ID = tftypes.StringValue(ws.GetId())
	data.Name = tftypes.StringValue(ws.GetName())
	data.Description = tftypes.StringValue(ws.GetDescription())
	if t := ws.GetCreateTime(); t != nil {
		data.CreateTime = tftypes.StringValue(t.AsTime().Format("2006-01-02T15:04:05Z07:00"))
	}
	data.IsDefault = tftypes.BoolValue(ws.GetIsDefault())
}
