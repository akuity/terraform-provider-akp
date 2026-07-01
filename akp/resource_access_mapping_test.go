//go:build !acc

package akp

import (
	"testing"

	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"

	accesscontrolv1 "github.com/akuity/api-client-go/pkg/api/gen/accesscontrol/v1"
	apikeyv1 "github.com/akuity/api-client-go/pkg/api/gen/apikey/v1"
	orgcv1 "github.com/akuity/api-client-go/pkg/api/gen/organization/v1"
	"github.com/akuity/terraform-provider-akp/akp/types"
)

// These tests cover the hand-written "access" resource response mappers, which
// share the canonical helpers in resource_access_mapping.go (applyStringList /
// hydrateIfUnset). The common thread: the affected attributes are Optional but
// NOT Computed, so their post-apply value must equal the planned value exactly —
// the mappers must not silently change the operator's null-vs-empty encoding or
// overwrite operator config with a server-canonicalized value.

// TestApplyTeamResponseCustomRoles pins how applyTeamResponse maps custom_roles.
// An empty server response must preserve the operator's representation of "no
// roles" — an explicit `[]` stays an empty (non-nil) list, an omitted field
// stays null — otherwise an explicit `custom_roles = []` trips "provider
// produced inconsistent result after apply".
func TestApplyTeamResponseCustomRoles(t *testing.T) {
	newUserTeam := func(roles []string) *orgcv1.UserTeam {
		return &orgcv1.UserTeam{
			Team:        &orgcv1.Team{Name: "platform"},
			CustomRoles: roles,
		}
	}

	t.Run("server returns roles -> taken from response", func(t *testing.T) {
		data := &types.Team{CustomRoles: nil}
		applyTeamResponse(data, newUserTeam([]string{"role-a", "role-b"}))
		assert.Len(t, data.CustomRoles, 2)
		assert.Equal(t, "role-a", data.CustomRoles[0].ValueString())
		assert.Equal(t, "role-b", data.CustomRoles[1].ValueString())
	})

	t.Run("empty response preserves explicit empty list", func(t *testing.T) {
		// Plan had `custom_roles = []` (non-nil empty slice).
		data := &types.Team{CustomRoles: []tftypes.String{}}
		applyTeamResponse(data, newUserTeam(nil))
		assert.NotNil(t, data.CustomRoles, "explicit [] must stay an empty list, not become null")
		assert.Len(t, data.CustomRoles, 0)
	})

	t.Run("empty response preserves omitted (null)", func(t *testing.T) {
		// Plan omitted custom_roles (nil slice -> null list).
		data := &types.Team{CustomRoles: nil}
		applyTeamResponse(data, newUserTeam(nil))
		assert.Nil(t, data.CustomRoles, "omitted custom_roles must stay null")
	})
}

// TestApplyWorkspaceMemberResponse pins the member-identifier handling in
// applyWorkspaceMemberResponse. The backend canonicalizes the user email
// (GetUserByEmail lowercases it and the response echoes the stored, lowercased
// value). The mapper (via hydrateIfUnset) must preserve an already-set
// identifier and only hydrate from the response when it is unset (the import
// case, which has no config).
func TestApplyWorkspaceMemberResponse(t *testing.T) {
	testCases := map[string]struct {
		data              *types.WorkspaceMember
		member            *orgcv1.WorkspaceMember
		expectedUserEmail string
		expectedTeamName  string
	}{
		"user email preserved when set, despite server lowercasing": {
			data: &types.WorkspaceMember{
				UserEmail: tftypes.StringValue("Alice@Example.com"),
			},
			member: &orgcv1.WorkspaceMember{
				Id:   "wm-1",
				Role: orgcv1.WorkspaceMemberRole_WORKSPACE_MEMBER_ROLE_ADMIN,
				Member: &orgcv1.WorkspaceMember_User{
					User: &orgcv1.WorkspaceUserMember{Email: "alice@example.com"},
				},
			},
			expectedUserEmail: "Alice@Example.com",
		},
		"user email hydrated from response when unset (import)": {
			data: &types.WorkspaceMember{
				UserEmail: tftypes.StringNull(),
			},
			member: &orgcv1.WorkspaceMember{
				Id:   "wm-2",
				Role: orgcv1.WorkspaceMemberRole_WORKSPACE_MEMBER_ROLE_MEMBER,
				Member: &orgcv1.WorkspaceMember_User{
					User: &orgcv1.WorkspaceUserMember{Email: "alice@example.com"},
				},
			},
			expectedUserEmail: "alice@example.com",
		},
		"team name preserved when set": {
			data: &types.WorkspaceMember{
				TeamName: tftypes.StringValue("Platform"),
			},
			member: &orgcv1.WorkspaceMember{
				Id:   "wm-3",
				Role: orgcv1.WorkspaceMemberRole_WORKSPACE_MEMBER_ROLE_MEMBER,
				Member: &orgcv1.WorkspaceMember_Team{
					Team: &orgcv1.WorkspaceTeamMember{Name: "platform"},
				},
			},
			expectedTeamName: "Platform",
		},
		"team name hydrated from response when unset (import)": {
			data: &types.WorkspaceMember{
				TeamName: tftypes.StringNull(),
			},
			member: &orgcv1.WorkspaceMember{
				Id:   "wm-4",
				Role: orgcv1.WorkspaceMemberRole_WORKSPACE_MEMBER_ROLE_ADMIN,
				Member: &orgcv1.WorkspaceMember_Team{
					Team: &orgcv1.WorkspaceTeamMember{Name: "platform"},
				},
			},
			expectedTeamName: "platform",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			applyWorkspaceMemberResponse(tc.data, tc.member)

			// id and role are always taken from the server response.
			assert.Equal(t, tc.member.GetId(), tc.data.ID.ValueString())
			assert.Equal(t, workspaceMemberRoleToString(tc.member.GetRole()), tc.data.Role.ValueString())

			if tc.expectedUserEmail != "" {
				assert.Equal(t, tc.expectedUserEmail, tc.data.UserEmail.ValueString())
			} else {
				assert.True(t, tc.data.UserEmail.IsNull(), "expected user_email to remain null")
			}

			if tc.expectedTeamName != "" {
				assert.Equal(t, tc.expectedTeamName, tc.data.TeamName.ValueString())
			} else {
				assert.True(t, tc.data.TeamName.IsNull(), "expected team_name to remain null")
			}
		})
	}
}

// TestApplyApiKeyResponsePermissions pins the per-list empty/null preservation
// in applyApiKeyResponse. permissions.{actions,roles,custom_roles} are Optional
// (not Computed), so an explicit empty list must stay an empty list and an
// omitted (null) list must stay null when the server returns nothing for it.
func TestApplyApiKeyResponsePermissions(t *testing.T) {
	// Org-scoped key: prior plan has an explicit empty actions list, a
	// non-empty roles list, and an omitted (null) custom_roles list.
	data := &types.ApiKey{
		Workspace: tftypes.StringNull(),
		Permissions: &types.ApiKeyPermissions{
			Actions:     []tftypes.String{}, // explicit []
			Roles:       []tftypes.String{tftypes.StringValue("member")},
			CustomRoles: nil, // omitted -> null
		},
	}

	key := &apikeyv1.APIKey{
		Id: "key-1",
		Permissions: &accesscontrolv1.Permissions{
			Actions:     nil,
			Roles:       []string{"organization/member"}, // server namespaces roles
			CustomRoles: nil,
		},
	}

	applyApiKeyResponse(data, key)

	assert.NotNil(t, data.Permissions.Actions, "explicit empty actions must stay an empty list, not null")
	assert.Len(t, data.Permissions.Actions, 0)

	assert.Len(t, data.Permissions.Roles, 1)
	assert.Equal(t, "member", data.Permissions.Roles[0].ValueString(), "role namespace must be stripped back to config form")

	assert.Nil(t, data.Permissions.CustomRoles, "omitted custom_roles must stay null")
}

// TestApplyStringList pins the helper directly: a non-empty server slice is
// converted and overwrites; an empty/nil server slice preserves `current`
// (so an explicit `[]` stays `[]` and an omitted/null field stays null).
func TestApplyStringList(t *testing.T) {
	t.Run("server returns elements -> converted", func(t *testing.T) {
		got := applyStringList(nil, []string{"a", "b"})
		assert.Len(t, got, 2)
		assert.Equal(t, "a", got[0].ValueString())
		assert.Equal(t, "b", got[1].ValueString())
	})

	t.Run("empty server preserves explicit empty list", func(t *testing.T) {
		current := []tftypes.String{}
		got := applyStringList(current, nil)
		assert.NotNil(t, got)
		assert.Len(t, got, 0)
	})

	t.Run("empty server preserves null", func(t *testing.T) {
		got := applyStringList(nil, []string{})
		assert.Nil(t, got)
	})
}

// TestStringSliceFromTF pins the TF->Go conversion: an empty input yields nil,
// and null/unknown elements are dropped (shared with akp_api_key).
func TestStringSliceFromTF(t *testing.T) {
	t.Run("empty yields nil", func(t *testing.T) {
		assert.Nil(t, stringSliceFromTF(nil))
		assert.Nil(t, stringSliceFromTF([]tftypes.String{}))
	})

	t.Run("known values kept in order", func(t *testing.T) {
		got := stringSliceFromTF([]tftypes.String{tftypes.StringValue("a"), tftypes.StringValue("b")})
		assert.Equal(t, []string{"a", "b"}, got)
	})

	t.Run("null and unknown elements are dropped", func(t *testing.T) {
		got := stringSliceFromTF([]tftypes.String{
			tftypes.StringValue("a"),
			tftypes.StringNull(),
			tftypes.StringUnknown(),
			tftypes.StringValue("b"),
		})
		assert.Equal(t, []string{"a", "b"}, got)
	})
}
