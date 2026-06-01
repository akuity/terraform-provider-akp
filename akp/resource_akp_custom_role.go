package akp

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	orgcv1 "github.com/akuity/api-client-go/pkg/api/gen/organization/v1"
	"github.com/akuity/terraform-provider-akp/akp/types"
)

func NewAkpCustomRoleResource() resource.Resource {
	return &GenericResource[types.CustomRole]{
		TypeNameSuffix: "custom_role",
		SchemaFunc:     customRoleSchema,
		CreateFunc:     customRoleCreate,
		ReadFunc:       customRoleRead,
		UpdateFunc:     customRoleUpdate,
		DeleteFunc:     customRoleDelete,
		ImportStateFunc: func(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
			// Import IDs:
			//   org-scoped:       <custom_role_id>
			//   workspace-scoped: <workspace_name>/<custom_role_id>
			parts := strings.Split(req.ID, "/")
			badID := func() {
				resp.Diagnostics.AddError(
					"Unexpected Import Identifier",
					fmt.Sprintf("Expected `custom_role_id` or `workspace_name/custom_role_id`. Got: %q", req.ID),
				)
			}
			switch len(parts) {
			case 1:
				if parts[0] == "" {
					badID()
					return
				}
				resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[0])...)
			case 2:
				if parts[0] == "" || parts[1] == "" {
					badID()
					return
				}
				resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("workspace"), parts[0])...)
				resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
			default:
				badID()
			}
		},
	}
}

func customRoleCreate(ctx context.Context, cli *AkpCli, _ *diag.Diagnostics, plan *types.CustomRole) (*types.CustomRole, error) {
	if err := requireKnownWorkspace(plan.Workspace, "custom_role"); err != nil {
		return nil, err
	}
	if !plan.Workspace.IsNull() && plan.Workspace.ValueString() != "" {
		workspace, err := getWorkspace(ctx, cli.OrgCli, cli.OrgId, plan.Workspace.ValueString())
		if err != nil {
			return nil, fmt.Errorf("unable to resolve workspace %q: %w", plan.Workspace.ValueString(), err)
		}
		resp, err := retryWithBackoff(ctx, func(ctx context.Context) (*orgcv1.CreateWorkspaceCustomRoleResponse, error) {
			return cli.OrgCli.CreateWorkspaceCustomRole(ctx, &orgcv1.CreateWorkspaceCustomRoleRequest{
				OrganizationId: cli.OrgId,
				WorkspaceId:    workspace.GetId(),
				Name:           plan.Name.ValueString(),
				Description:    plan.Description.ValueString(),
				Policy:         plan.Policy.ValueString(),
			})
		}, "CreateWorkspaceCustomRole")
		if err != nil {
			return nil, fmt.Errorf("unable to create workspace custom role: %w", err)
		}
		applyCustomRoleResponse(plan, resp.GetCustomRole())
		return plan, nil
	}

	resp, err := retryWithBackoff(ctx, func(ctx context.Context) (*orgcv1.CreateCustomRoleResponse, error) {
		return cli.OrgCli.CreateCustomRole(ctx, &orgcv1.CreateCustomRoleRequest{
			OrganizationId: cli.OrgId,
			Name:           plan.Name.ValueString(),
			Description:    plan.Description.ValueString(),
			Policy:         plan.Policy.ValueString(),
		})
	}, "CreateCustomRole")
	if err != nil {
		return nil, fmt.Errorf("unable to create custom role: %w", err)
	}
	applyCustomRoleResponse(plan, resp.GetCustomRole())
	return plan, nil
}

func customRoleRead(ctx context.Context, cli *AkpCli, _ *diag.Diagnostics, data *types.CustomRole) error {
	if !data.Workspace.IsNull() && data.Workspace.ValueString() != "" {
		workspace, err := getWorkspace(ctx, cli.OrgCli, cli.OrgId, data.Workspace.ValueString())
		if err != nil {
			return err
		}
		resp, err := retryWithBackoff(ctx, func(ctx context.Context) (*orgcv1.GetWorkspaceCustomRoleResponse, error) {
			return cli.OrgCli.GetWorkspaceCustomRole(ctx, &orgcv1.GetWorkspaceCustomRoleRequest{
				OrganizationId: cli.OrgId,
				WorkspaceId:    workspace.GetId(),
				Id:             data.ID.ValueString(),
			})
		}, "GetWorkspaceCustomRole")
		if err != nil {
			return err
		}
		if resp.GetCustomRole() == nil {
			return status.Error(codes.NotFound, "custom role not found")
		}
		applyCustomRoleResponse(data, resp.GetCustomRole())
		return nil
	}

	resp, err := retryWithBackoff(ctx, func(ctx context.Context) (*orgcv1.GetCustomRoleResponse, error) {
		return cli.OrgCli.GetCustomRole(ctx, &orgcv1.GetCustomRoleRequest{
			OrganizationId: cli.OrgId,
			Id:             data.ID.ValueString(),
		})
	}, "GetCustomRole")
	if err != nil {
		return err
	}
	if resp.GetCustomRole() == nil {
		return status.Error(codes.NotFound, "custom role not found")
	}
	applyCustomRoleResponse(data, resp.GetCustomRole())
	return nil
}

func customRoleUpdate(ctx context.Context, cli *AkpCli, _ *diag.Diagnostics, plan *types.CustomRole) (*types.CustomRole, error) {
	if err := requireKnownWorkspace(plan.Workspace, "custom_role"); err != nil {
		return nil, err
	}
	if !plan.Workspace.IsNull() && plan.Workspace.ValueString() != "" {
		workspace, err := getWorkspace(ctx, cli.OrgCli, cli.OrgId, plan.Workspace.ValueString())
		if err != nil {
			return nil, fmt.Errorf("unable to resolve workspace %q: %w", plan.Workspace.ValueString(), err)
		}
		resp, err := retryWithBackoff(ctx, func(ctx context.Context) (*orgcv1.UpdateWorkspaceCustomRoleResponse, error) {
			return cli.OrgCli.UpdateWorkspaceCustomRole(ctx, &orgcv1.UpdateWorkspaceCustomRoleRequest{
				OrganizationId: cli.OrgId,
				WorkspaceId:    workspace.GetId(),
				Id:             plan.ID.ValueString(),
				Name:           plan.Name.ValueString(),
				Description:    plan.Description.ValueString(),
				Policy:         plan.Policy.ValueString(),
			})
		}, "UpdateWorkspaceCustomRole")
		if err != nil {
			return nil, fmt.Errorf("unable to update workspace custom role: %w", err)
		}
		applyCustomRoleResponse(plan, resp.GetCustomRole())
		return plan, nil
	}

	resp, err := retryWithBackoff(ctx, func(ctx context.Context) (*orgcv1.UpdateCustomRoleResponse, error) {
		return cli.OrgCli.UpdateCustomRole(ctx, &orgcv1.UpdateCustomRoleRequest{
			OrganizationId: cli.OrgId,
			Id:             plan.ID.ValueString(),
			Name:           plan.Name.ValueString(),
			Description:    plan.Description.ValueString(),
			Policy:         plan.Policy.ValueString(),
		})
	}, "UpdateCustomRole")
	if err != nil {
		return nil, fmt.Errorf("unable to update custom role: %w", err)
	}
	applyCustomRoleResponse(plan, resp.GetCustomRole())
	return plan, nil
}

func customRoleDelete(ctx context.Context, cli *AkpCli, _ *diag.Diagnostics, state *types.CustomRole) error {
	if !state.Workspace.IsNull() && state.Workspace.ValueString() != "" {
		workspace, err := getWorkspace(ctx, cli.OrgCli, cli.OrgId, state.Workspace.ValueString())
		if err != nil {
			// Workspace gone or no longer visible: fall back to the org-level
			// delete by ID. Server filters by (org, id) regardless of which
			// endpoint is called, so this still removes the workspace-scoped
			// role instead of orphaning it.
			if !isGoneErr(err) {
				return fmt.Errorf("unable to resolve workspace %q: %w", state.Workspace.ValueString(), err)
			}
		} else {
			_, err := retryWithBackoff(ctx, func(ctx context.Context) (*orgcv1.DeleteWorkspaceCustomRoleResponse, error) {
				resp, err := cli.OrgCli.DeleteWorkspaceCustomRole(ctx, &orgcv1.DeleteWorkspaceCustomRoleRequest{
					OrganizationId: cli.OrgId,
					WorkspaceId:    workspace.GetId(),
					Id:             state.ID.ValueString(),
				})
				if isGoneErr(err) {
					return resp, nil
				}
				return resp, err
			}, "DeleteWorkspaceCustomRole")
			if err != nil {
				return fmt.Errorf("unable to delete workspace custom role: %w", err)
			}
			return nil
		}
	}

	_, err := retryWithBackoff(ctx, func(ctx context.Context) (*orgcv1.DeleteCustomRoleResponse, error) {
		resp, err := cli.OrgCli.DeleteCustomRole(ctx, &orgcv1.DeleteCustomRoleRequest{
			OrganizationId: cli.OrgId,
			Id:             state.ID.ValueString(),
		})
		if isGoneErr(err) {
			return resp, nil
		}
		return resp, err
	}, "DeleteCustomRole")
	if err != nil {
		return fmt.Errorf("unable to delete custom role: %w", err)
	}
	return nil
}

// requireKnownWorkspace guards against Unknown leaking into scope routing.
// The plan framework is supposed to resolve Unknown values before Apply, but
// `Workspace.ValueString()` silently returns "" for both null and unknown —
// so a missed resolution would route a workspace-scoped resource to the
// org-scoped endpoint without anyone noticing. Failing loudly is cheaper
// than debugging "the api key ended up in the wrong scope".
func requireKnownWorkspace(workspace tftypes.String, resourceKind string) error {
	if workspace.IsUnknown() {
		return fmt.Errorf("%s: workspace must be known at apply time; received an unresolved value", resourceKind)
	}
	return nil
}

func applyCustomRoleResponse(data *types.CustomRole, role *orgcv1.CustomRole) {
	if role == nil {
		return
	}
	data.ID = tftypes.StringValue(role.GetId())
	data.Name = tftypes.StringValue(role.GetName())
	data.Description = tftypes.StringValue(role.GetDescription())
	data.Policy = tftypes.StringValue(role.GetPolicy())
}
