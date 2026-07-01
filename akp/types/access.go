package types

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ApiKey struct {
	ID               types.String       `tfsdk:"id"`
	Workspace        types.String       `tfsdk:"workspace"`
	Description      types.String       `tfsdk:"description"`
	ExpireInDuration types.String       `tfsdk:"expire_in_duration"`
	Permissions      *ApiKeyPermissions `tfsdk:"permissions"`
	Secret           types.String       `tfsdk:"secret"`
	OrganizationID   types.String       `tfsdk:"organization_id"`
	CreateTime       types.String       `tfsdk:"create_time"`
	ExpireTime       types.String       `tfsdk:"expire_time"`
}

type ApiKeyPermissions struct {
	Actions     []types.String `tfsdk:"actions"`
	Roles       []types.String `tfsdk:"roles"`
	CustomRoles []types.String `tfsdk:"custom_roles"`
}

type CustomRole struct {
	ID          types.String `tfsdk:"id"`
	Workspace   types.String `tfsdk:"workspace"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Policy      types.String `tfsdk:"policy"`
}

type Workspace struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	CreateTime  types.String `tfsdk:"create_time"`
	IsDefault   types.Bool   `tfsdk:"is_default"`
}

// Team maps the API's Team/UserTeam pair. A team is identified by Name (its
// natural key — the API has no separate team ID); Description and CustomRoles
// are mutable in place. CreateTime and MemberCount are server-assigned.
type Team struct {
	Name        types.String   `tfsdk:"name"`
	Description types.String   `tfsdk:"description"`
	CustomRoles []types.String `tfsdk:"custom_roles"`
	CreateTime  types.String   `tfsdk:"create_time"`
	MemberCount types.Int64    `tfsdk:"member_count"`
}

// WorkspaceMember maps the API's WorkspaceMember/WorkspaceMemberRef pair. A
// member is a role plus exactly one of UserEmail or TeamName — the API's
// `oneof member` also allows user_id, but operators have no way to discover a
// user ID through this provider, so only the email identifier is exposed. ID
// is the server-assigned membership record ID used to read, update, and remove
// the member.
type WorkspaceMember struct {
	ID          types.String `tfsdk:"id"`
	Workspace   types.String `tfsdk:"workspace"`
	WorkspaceID types.String `tfsdk:"workspace_id"`
	Role        types.String `tfsdk:"role"`
	UserEmail   types.String `tfsdk:"user_email"`
	TeamName    types.String `tfsdk:"team_name"`
}
