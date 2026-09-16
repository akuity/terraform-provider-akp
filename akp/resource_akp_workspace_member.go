package akp

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/resourcevalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	orgcv1 "github.com/akuity/api-client-go/pkg/api/gen/organization/v1"
	"github.com/akuity/terraform-provider-akp/akp/types"
)

func NewAkpWorkspaceMemberResource() resource.Resource {
	return &GenericResource[types.WorkspaceMember]{
		TypeNameSuffix: "workspace_member",
		SchemaFunc:     workspaceMemberSchema,
		CreateFunc:     workspaceMemberCreate,
		ReadFunc:       workspaceMemberRead,
		UpdateFunc:     workspaceMemberUpdate,
		DeleteFunc:     workspaceMemberDelete,
		ConfigValidatorsFunc: func() []resource.ConfigValidator {
			// Exactly one member identifier must be set (the API's
			// `oneof member`, minus the unexposed user_id).
			return []resource.ConfigValidator{
				resourcevalidator.ExactlyOneOf(
					path.MatchRoot("user_email"),
					path.MatchRoot("team_name"),
				),
			}
		},
		ImportStateFunc: func(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
			// Import ID: <workspace_name>/<member_id>. Read resolves the role
			// and member identity; the operator must then write config whose
			// member identifier (user_email/team_name) matches.
			parts := strings.Split(req.ID, "/")
			if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
				resp.Diagnostics.AddError(
					"Unexpected Import Identifier",
					fmt.Sprintf("Expected `workspace_name/member_id`. Got: %q", req.ID),
				)
				return
			}
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("workspace"), parts[0])...)
			resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), parts[1])...)
		},
	}
}

func workspaceMemberCreate(ctx context.Context, cli *AkpCli, _ *diag.Diagnostics, plan *types.WorkspaceMember) (*types.WorkspaceMember, error) {
	if err := requireKnownWorkspace(plan.Workspace, "workspace_member"); err != nil {
		return nil, err
	}
	role, err := workspaceMemberRoleFromString(plan.Role.ValueString())
	if err != nil {
		return nil, err
	}
	ref, err := buildWorkspaceMemberRef(plan, role)
	if err != nil {
		return nil, err
	}

	workspace, err := getWorkspace(ctx, cli.OrgCli, cli.OrgId, plan.Workspace.ValueString())
	if err != nil {
		return nil, fmt.Errorf("unable to resolve workspace %q: %w", plan.Workspace.ValueString(), err)
	}

	resp, err := retryWithBackoff(ctx, func(ctx context.Context) (*orgcv1.AddWorkspaceMemberResponse, error) {
		return cli.OrgCli.AddWorkspaceMember(ctx, &orgcv1.AddWorkspaceMemberRequest{
			OrganizationId: cli.OrgId,
			WorkspaceId:    workspace.GetId(),
			MemberRef:      ref,
		})
	}, "AddWorkspaceMember")
	if err != nil {
		return nil, fmt.Errorf("unable to add workspace member: %w", err)
	}
	plan.WorkspaceID = tftypes.StringValue(workspace.GetId())
	applyWorkspaceMemberResponse(plan, resp.GetWorkspaceMember())
	return plan, nil
}

func workspaceMemberRead(ctx context.Context, cli *AkpCli, _ *diag.Diagnostics, data *types.WorkspaceMember) error {
	workspaceID, err := resolveWorkspaceID(ctx, cli, data)
	if err != nil {
		return fmt.Errorf("unable to resolve workspace %q: %w", data.Workspace.ValueString(), err)
	}
	data.WorkspaceID = tftypes.StringValue(workspaceID)

	resp, err := retryWithBackoff(ctx, func(ctx context.Context) (*orgcv1.GetWorkspaceMemberResponse, error) {
		return cli.OrgCli.GetWorkspaceMember(ctx, &orgcv1.GetWorkspaceMemberRequest{
			OrganizationId: cli.OrgId,
			WorkspaceId:    workspaceID,
			Id:             data.ID.ValueString(),
		})
	}, "GetWorkspaceMember")
	if err != nil {
		return err
	}
	if resp.GetWorkspaceMember() == nil {
		return status.Error(codes.NotFound, "workspace member not found")
	}
	applyWorkspaceMemberResponse(data, resp.GetWorkspaceMember())
	return nil
}

func workspaceMemberUpdate(ctx context.Context, cli *AkpCli, _ *diag.Diagnostics, plan *types.WorkspaceMember) (*types.WorkspaceMember, error) {
	// Only role is mutable in place; workspace and the member identifier are all
	// RequiresReplace. UpdateWorkspaceMember takes the membership ID and role.
	role, err := workspaceMemberRoleFromString(plan.Role.ValueString())
	if err != nil {
		return nil, err
	}
	workspaceID, err := resolveWorkspaceID(ctx, cli, plan)
	if err != nil {
		return nil, fmt.Errorf("unable to resolve workspace %q: %w", plan.Workspace.ValueString(), err)
	}
	plan.WorkspaceID = tftypes.StringValue(workspaceID)
	resp, err := retryWithBackoff(ctx, func(ctx context.Context) (*orgcv1.UpdateWorkspaceMemberResponse, error) {
		return cli.OrgCli.UpdateWorkspaceMember(ctx, &orgcv1.UpdateWorkspaceMemberRequest{
			OrganizationId: cli.OrgId,
			WorkspaceId:    workspaceID,
			Id:             plan.ID.ValueString(),
			Role:           role,
		})
	}, "UpdateWorkspaceMember")
	if err != nil {
		return nil, fmt.Errorf("unable to update workspace member: %w", err)
	}
	applyWorkspaceMemberResponse(plan, resp.GetWorkspaceMember())
	return plan, nil
}

func workspaceMemberDelete(ctx context.Context, cli *AkpCli, _ *diag.Diagnostics, state *types.WorkspaceMember) error {
	workspaceID, err := resolveWorkspaceID(ctx, cli, state)
	if err != nil {
		// If the workspace itself is gone, its members were cascade-removed, so
		// this member is already gone.
		if isGoneErr(err) {
			return nil
		}
		return fmt.Errorf("unable to resolve workspace %q: %w", state.Workspace.ValueString(), err)
	}

	_, err = retryWithBackoff(ctx, func(ctx context.Context) (*orgcv1.RemoveWorkspaceMemberResponse, error) {
		resp, err := cli.OrgCli.RemoveWorkspaceMember(ctx, &orgcv1.RemoveWorkspaceMemberRequest{
			OrganizationId: cli.OrgId,
			WorkspaceId:    workspaceID,
			Id:             state.ID.ValueString(),
		})
		if isGoneErr(err) || isLastWorkspaceMemberErr(err) {
			return resp, nil
		}
		return resp, err
	}, "RemoveWorkspaceMember")
	if err != nil {
		return fmt.Errorf("unable to remove workspace member: %w", err)
	}
	return nil
}

// resolveWorkspaceID returns the member's workspace_id, resolving it from the
// workspace name when empty (older state, imports, or `-refresh=false`).
// Get/Update/RemoveWorkspaceMember all need the ID for their permission checks.
// The raw getWorkspace error is returned so callers apply their own
// gone-handling (Read surfaces it, Delete treats it as already-gone).
func resolveWorkspaceID(ctx context.Context, cli *AkpCli, m *types.WorkspaceMember) (string, error) {
	if id := m.WorkspaceID.ValueString(); id != "" {
		return id, nil
	}
	workspace, err := getWorkspace(ctx, cli.OrgCli, cli.OrgId, m.Workspace.ValueString())
	if err != nil {
		return "", err
	}
	return workspace.GetId(), nil
}

func buildWorkspaceMemberRef(plan *types.WorkspaceMember, role orgcv1.WorkspaceMemberRole) (*orgcv1.WorkspaceMemberRef, error) {
	ref := &orgcv1.WorkspaceMemberRef{Role: role}
	switch {
	case !plan.UserEmail.IsNull() && plan.UserEmail.ValueString() != "":
		ref.Member = &orgcv1.WorkspaceMemberRef_UserEmail{UserEmail: plan.UserEmail.ValueString()}
	case !plan.TeamName.IsNull() && plan.TeamName.ValueString() != "":
		ref.Member = &orgcv1.WorkspaceMemberRef_TeamName{TeamName: plan.TeamName.ValueString()}
	default:
		return nil, fmt.Errorf("exactly one of user_email or team_name must be set")
	}
	return ref, nil
}

func workspaceMemberRoleFromString(role string) (orgcv1.WorkspaceMemberRole, error) {
	switch role {
	case "member":
		return orgcv1.WorkspaceMemberRole_WORKSPACE_MEMBER_ROLE_MEMBER, nil
	case "admin":
		return orgcv1.WorkspaceMemberRole_WORKSPACE_MEMBER_ROLE_ADMIN, nil
	default:
		return orgcv1.WorkspaceMemberRole_WORKSPACE_MEMBER_ROLE_UNSPECIFIED, fmt.Errorf("invalid role %q: must be one of \"member\" or \"admin\"", role)
	}
}

func workspaceMemberRoleToString(role orgcv1.WorkspaceMemberRole) string {
	switch role {
	case orgcv1.WorkspaceMemberRole_WORKSPACE_MEMBER_ROLE_ADMIN:
		return "admin"
	case orgcv1.WorkspaceMemberRole_WORKSPACE_MEMBER_ROLE_MEMBER:
		return "member"
	default:
		return ""
	}
}

func applyWorkspaceMemberResponse(data *types.WorkspaceMember, member *orgcv1.WorkspaceMember) {
	if member == nil {
		return
	}
	data.ID = tftypes.StringValue(member.GetId())
	data.Role = tftypes.StringValue(workspaceMemberRoleToString(member.GetRole()))

	// Preserve operator config; hydrate only on import (see hydrateIfUnset).
	switch m := member.GetMember().(type) {
	case *orgcv1.WorkspaceMember_User:
		data.UserEmail = hydrateIfUnset(data.UserEmail, m.User.GetEmail())
	case *orgcv1.WorkspaceMember_Team:
		data.TeamName = hydrateIfUnset(data.TeamName, m.Team.GetName())
	}
}
