package akp

import (
	"context"
	"errors"
	"testing"

	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"

	kargov1 "github.com/akuity/api-client-go/pkg/api/gen/kargo/v1"
	orgcv1 "github.com/akuity/api-client-go/pkg/api/gen/organization/v1"
	tfakptypes "github.com/akuity/terraform-provider-akp/akp/types"
)

func TestPreferredKargoAgentWorkspaceName(t *testing.T) {
	testCases := map[string]struct {
		agent *tfakptypes.KargoAgent
		want  string
	}{
		"nil agent": {
			agent: nil,
			want:  "",
		},
		"null workspace": {
			agent: &tfakptypes.KargoAgent{Workspace: tftypes.StringNull()},
			want:  "",
		},
		"unknown workspace": {
			agent: &tfakptypes.KargoAgent{Workspace: tftypes.StringUnknown()},
			want:  "",
		},
		"empty string workspace": {
			agent: &tfakptypes.KargoAgent{Workspace: tftypes.StringValue("")},
			want:  "",
		},
		"populated workspace": {
			agent: &tfakptypes.KargoAgent{Workspace: tftypes.StringValue("default")},
			want:  "default",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tc.want, preferredKargoAgentWorkspaceName(tc.agent))
		})
	}
}

// fakeOrgCli provides the minimal OrganizationServiceGatewayClient surface that
// the workspace resolver needs. All other methods panic so test bugs surface
// loudly instead of silently skipping.
type fakeOrgCli struct {
	orgcv1.OrganizationServiceGatewayClient

	workspaces []*orgcv1.Workspace
	err        error
}

func (f *fakeOrgCli) ListWorkspaces(context.Context, *orgcv1.ListWorkspacesRequest) (*orgcv1.ListWorkspacesResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &orgcv1.ListWorkspacesResponse{Workspaces: f.workspaces}, nil
}

// fakeKargoCli mirrors the server bug that motivated this fix: for every
// workspace queried, the same full instance list is returned, regardless of
// the WorkspaceId filter. This is the exact behavior the terraform provider
// must no longer rely on.
type fakeKargoCli struct {
	kargov1.KargoServiceGatewayClient

	allInstances []*kargov1.KargoInstance
}

func (f *fakeKargoCli) ListKargoInstances(context.Context, *kargov1.ListKargoInstancesRequest) (*kargov1.ListKargoInstancesResponse, error) {
	return &kargov1.ListKargoInstancesResponse{Instances: f.allInstances}, nil
}

func TestResolveKargoAgentWorkspace(t *testing.T) {
	const (
		targetWorkspaceID   = "ws-default"
		targetWorkspaceName = "default"
		newerWorkspaceID    = "ws-newer"
		newerWorkspaceName  = "platform"
		instanceID          = "kargo-instance-1"
	)

	multiWorkspaces := []*orgcv1.Workspace{
		// Server returns newest-first. The instance lives in the OLDER workspace.
		{Id: newerWorkspaceID, Name: newerWorkspaceName},
		{Id: targetWorkspaceID, Name: targetWorkspaceName},
	}
	allInstances := []*kargov1.KargoInstance{{Id: instanceID, WorkspaceId: targetWorkspaceID}}

	t.Run("workspace name in state is trusted without scanning", func(t *testing.T) {
		cli := &AkpCli{
			OrgId:    "org-1",
			OrgCli:   &fakeOrgCli{workspaces: multiWorkspaces},
			KargoCli: &fakeKargoCli{allInstances: allInstances},
		}
		agent := &tfakptypes.KargoAgent{
			InstanceID: tftypes.StringValue(instanceID),
			Workspace:  tftypes.StringValue(targetWorkspaceName),
		}

		gotID, gotName := resolveKargoAgentWorkspace(context.Background(), cli, agent)

		require.Equal(t, targetWorkspaceID, gotID)
		require.Equal(t, targetWorkspaceName, gotName)
	})

	t.Run("scan fallback returns the first-seen workspace (pre-fix behavior)", func(t *testing.T) {
		cli := &AkpCli{
			OrgId:    "org-1",
			OrgCli:   &fakeOrgCli{workspaces: multiWorkspaces},
			KargoCli: &fakeKargoCli{allInstances: allInstances},
		}
		agent := &tfakptypes.KargoAgent{
			InstanceID: tftypes.StringValue(instanceID),
			Workspace:  tftypes.StringNull(),
		}

		gotID, gotName := resolveKargoAgentWorkspace(context.Background(), cli, agent)

		// Documents the broken behavior we are intentionally avoiding when the
		// workspace name is known: the scan picks the first-iterated workspace,
		// which is NOT the instance's actual workspace.
		require.Equal(t, newerWorkspaceID, gotID)
		require.Equal(t, newerWorkspaceName, gotName)
	})

	t.Run("name lookup failure falls back to scan", func(t *testing.T) {
		cli := &AkpCli{
			OrgId:    "org-1",
			OrgCli:   &fakeOrgCli{err: errors.New("boom")},
			KargoCli: &fakeKargoCli{allInstances: allInstances},
		}
		agent := &tfakptypes.KargoAgent{
			InstanceID: tftypes.StringValue(instanceID),
			Workspace:  tftypes.StringValue(targetWorkspaceName),
		}

		gotID, gotName := resolveKargoAgentWorkspace(context.Background(), cli, agent)

		// Both the name lookup and the scan share the same failing ListWorkspaces
		// call, so we just verify no panic and empty return.
		require.Empty(t, gotID)
		require.Empty(t, gotName)
	})

	t.Run("nil cli or agent returns empty", func(t *testing.T) {
		gotID, gotName := resolveKargoAgentWorkspace(context.Background(), nil, &tfakptypes.KargoAgent{})
		require.Empty(t, gotID)
		require.Empty(t, gotName)

		gotID, gotName = resolveKargoAgentWorkspace(context.Background(), &AkpCli{}, nil)
		require.Empty(t, gotID)
		require.Empty(t, gotName)
	})
}
