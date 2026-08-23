//go:build !acc

package akp

import (
	"context"
	"testing"

	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	kargov1 "github.com/akuity/api-client-go/pkg/api/gen/kargo/v1"
	orgcv1 "github.com/akuity/api-client-go/pkg/api/gen/organization/v1"
	idv1 "github.com/akuity/api-client-go/pkg/api/gen/types/id/v1"
)

type stubArgoCDClient struct {
	argocdv1.ArgoCDServiceGatewayClient
	instance          *argocdv1.Instance
	getErr            error
	moveErrors        []error
	mutateOnMoveError bool
	getRequests       []*argocdv1.GetInstanceRequest
	moveRequests      []*argocdv1.UpdateInstanceWorkspaceRequest
}

func (s *stubArgoCDClient) GetInstance(_ context.Context, req *argocdv1.GetInstanceRequest) (*argocdv1.GetInstanceResponse, error) {
	s.getRequests = append(s.getRequests, req)
	if s.getErr != nil {
		return nil, s.getErr
	}
	return &argocdv1.GetInstanceResponse{Instance: s.instance}, nil
}

func (s *stubArgoCDClient) UpdateInstanceWorkspace(_ context.Context, req *argocdv1.UpdateInstanceWorkspaceRequest) (*argocdv1.UpdateInstanceWorkspaceResponse, error) {
	s.moveRequests = append(s.moveRequests, req)
	call := len(s.moveRequests) - 1
	var err error
	if call < len(s.moveErrors) {
		err = s.moveErrors[call]
	}
	if err == nil || s.mutateOnMoveError {
		s.instance.WorkspaceId = req.NewWorkspaceId
	}
	if err != nil {
		return nil, err
	}
	return &argocdv1.UpdateInstanceWorkspaceResponse{}, nil
}

type stubKargoClient struct {
	kargov1.KargoServiceGatewayClient
	instance          *kargov1.KargoInstance
	instances         []*kargov1.KargoInstance
	getErr            error
	listErr           error
	moveErrors        []error
	mutateOnMoveError bool
	getRequests       []*kargov1.GetKargoInstanceRequest
	listRequests      []*kargov1.ListKargoInstancesRequest
	moveRequests      []*kargov1.UpdateKargoInstanceWorkspaceRequest
}

type stubWorkspaceClient struct {
	orgcv1.OrganizationServiceGatewayClient
	workspaces   []*orgcv1.Workspace
	listErr      error
	listRequests []*orgcv1.ListWorkspacesRequest
}

func (s *stubWorkspaceClient) ListWorkspaces(_ context.Context, req *orgcv1.ListWorkspacesRequest) (*orgcv1.ListWorkspacesResponse, error) {
	s.listRequests = append(s.listRequests, req)
	if s.listErr != nil {
		return nil, s.listErr
	}
	return &orgcv1.ListWorkspacesResponse{Workspaces: s.workspaces}, nil
}

func (s *stubKargoClient) GetKargoInstance(_ context.Context, req *kargov1.GetKargoInstanceRequest) (*kargov1.GetKargoInstanceResponse, error) {
	s.getRequests = append(s.getRequests, req)
	if s.getErr != nil {
		return nil, s.getErr
	}
	return &kargov1.GetKargoInstanceResponse{Instance: s.instance}, nil
}

func (s *stubKargoClient) ListKargoInstances(_ context.Context, req *kargov1.ListKargoInstancesRequest) (*kargov1.ListKargoInstancesResponse, error) {
	s.listRequests = append(s.listRequests, req)
	if s.listErr != nil {
		return nil, s.listErr
	}
	return &kargov1.ListKargoInstancesResponse{Instances: s.instances}, nil
}

func (s *stubKargoClient) UpdateKargoInstanceWorkspace(_ context.Context, req *kargov1.UpdateKargoInstanceWorkspaceRequest) (*kargov1.UpdateKargoInstanceWorkspaceResponse, error) {
	s.moveRequests = append(s.moveRequests, req)
	call := len(s.moveRequests) - 1
	var err error
	if call < len(s.moveErrors) {
		err = s.moveErrors[call]
	}
	if err == nil || s.mutateOnMoveError {
		if s.instance != nil && s.instance.GetId() == req.GetId() {
			s.instance.WorkspaceId = req.GetNewWorkspaceId()
		}
		for _, instance := range s.instances {
			if instance.GetId() == req.GetId() {
				instance.WorkspaceId = req.GetNewWorkspaceId()
			}
		}
	}
	if err != nil {
		return nil, err
	}
	return &kargov1.UpdateKargoInstanceWorkspaceResponse{}, nil
}

func TestEnsureInstanceWorkspace(t *testing.T) {
	target := &orgcv1.Workspace{Id: "ws-new", Name: "justin-team"}
	defaultWorkspace := &orgcv1.Workspace{Id: "ws-default", Name: "default", IsDefault: true}
	testCases := map[string]struct {
		client          *stubArgoCDClient
		instanceID      string
		workspace       *orgcv1.Workspace
		errExpected     bool
		attemptExpected bool
		movesExpected   int
	}{
		"moves instance in another workspace by stable ID": {
			client: &stubArgoCDClient{instance: &argocdv1.Instance{Id: "inst-1", WorkspaceId: "ws-old"}}, instanceID: "inst-1",
			workspace: target, attemptExpected: true, movesExpected: 1,
		},
		"skips instance already in target workspace": {
			client: &stubArgoCDClient{instance: &argocdv1.Instance{Id: "inst-1", WorkspaceId: "ws-new"}}, instanceID: "inst-1", workspace: target,
		},
		"skips unassigned instance targeting default workspace": {
			client: &stubArgoCDClient{instance: &argocdv1.Instance{Id: "inst-1"}}, instanceID: "inst-1", workspace: defaultWorkspace,
		},
		"moves unassigned instance targeting non-default workspace": {
			client: &stubArgoCDClient{instance: &argocdv1.Instance{Id: "inst-1"}}, instanceID: "inst-1",
			workspace: target, attemptExpected: true, movesExpected: 1,
		},
		"propagates get error": {
			client: &stubArgoCDClient{getErr: status.Error(codes.InvalidArgument, "boom")}, instanceID: "inst-1", workspace: target, errExpected: true,
		},
		"propagates non-retryable move error": {
			client:     &stubArgoCDClient{instance: &argocdv1.Instance{Id: "inst-1", WorkspaceId: "ws-old"}, moveErrors: []error{status.Error(codes.PermissionDenied, "denied")}},
			instanceID: "inst-1", workspace: target, errExpected: true, attemptExpected: true, movesExpected: 1,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			attempted, err := ensureInstanceWorkspace(context.Background(), tc.client, "org-1", tc.instanceID, "planned-new-name", tc.workspace)
			if tc.errExpected {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tc.attemptExpected, attempted)
			assert.Len(t, tc.client.moveRequests, tc.movesExpected)
			if tc.instanceID != "" {
				require.NotEmpty(t, tc.client.getRequests)
				assert.Equal(t, idv1.Type_ID, tc.client.getRequests[0].GetIdType())
				assert.Equal(t, tc.instanceID, tc.client.getRequests[0].GetId())
			}
			if tc.movesExpected > 0 {
				moveReq := tc.client.moveRequests[0]
				assert.Equal(t, "org-1", moveReq.GetOrganizationId())
				assert.Equal(t, "inst-1", moveReq.GetId())
				assert.Equal(t, tc.workspace.GetId(), moveReq.GetNewWorkspaceId())
			}
		})
	}
}

func TestEnsureInstanceWorkspaceRechecksAfterTransientMoveError(t *testing.T) {
	target := &orgcv1.Workspace{Id: "ws-new", Name: "target"}
	client := &stubArgoCDClient{
		instance: &argocdv1.Instance{Id: "inst-1", WorkspaceId: "ws-old"}, moveErrors: []error{status.Error(codes.Unavailable, "response lost")}, mutateOnMoveError: true,
	}
	attempted, err := ensureInstanceWorkspace(context.Background(), client, "org-1", "inst-1", "renamed", target)
	require.NoError(t, err)
	assert.True(t, attempted)
	assert.Len(t, client.getRequests, 2)
	assert.Len(t, client.moveRequests, 1, "the retry must observe the completed move before issuing another update")
}

func TestEnsureInstanceWorkspaceRetriesWhenMoveDidNotCommit(t *testing.T) {
	target := &orgcv1.Workspace{Id: "ws-new", Name: "target"}
	client := &stubArgoCDClient{
		instance: &argocdv1.Instance{Id: "inst-1", WorkspaceId: "ws-old"}, moveErrors: []error{status.Error(codes.Unavailable, "not committed")},
	}
	attempted, err := ensureInstanceWorkspace(context.Background(), client, "org-1", "inst-1", "renamed", target)
	require.NoError(t, err)
	assert.True(t, attempted)
	assert.Len(t, client.getRequests, 2)
	assert.Len(t, client.moveRequests, 2)
	assert.Equal(t, "ws-new", client.instance.GetWorkspaceId())
}

func TestEnsureKargoInstanceWorkspace(t *testing.T) {
	target := &orgcv1.Workspace{Id: "ws-new", Name: "justin-team"}
	defaultWorkspace := &orgcv1.Workspace{Id: "ws-default", Name: "default", IsDefault: true}

	t.Run("uses stable ID instead of a colliding planned name", func(t *testing.T) {
		managed := &kargov1.KargoInstance{Id: "inst-managed", Name: "old-name", WorkspaceId: "ws-old"}
		collision := &kargov1.KargoInstance{Id: "inst-collision", Name: "planned-new-name", WorkspaceId: "ws-other"}
		client := &stubKargoClient{instances: []*kargov1.KargoInstance{managed, collision}}
		attempted, err := ensureKargoInstanceWorkspace(context.Background(), client, "org-1", managed.GetId(), collision.GetName(), target)
		require.NoError(t, err)
		assert.True(t, attempted)
		require.Len(t, client.listRequests, 1)
		assert.Equal(t, "org-1", client.listRequests[0].GetOrganizationId())
		assert.Empty(t, client.getRequests, "a known ID must never fall back to a name lookup")
		require.Len(t, client.moveRequests, 1)
		assert.Equal(t, managed.GetId(), client.moveRequests[0].GetId())
		assert.Equal(t, "ws-old", client.moveRequests[0].GetWorkspaceId())
		assert.Equal(t, "ws-new", managed.GetWorkspaceId())
		assert.Equal(t, "ws-other", collision.GetWorkspaceId())
	})

	t.Run("known missing ID is an error without name fallback", func(t *testing.T) {
		collision := &kargov1.KargoInstance{Id: "inst-collision", Name: "planned-new-name", WorkspaceId: "ws-old"}
		client := &stubKargoClient{instances: []*kargov1.KargoInstance{collision}}
		attempted, err := ensureKargoInstanceWorkspace(context.Background(), client, "org-1", "missing-id", collision.GetName(), target)
		require.Error(t, err)
		assert.Equal(t, codes.NotFound, status.Code(err))
		assert.False(t, attempted)
		assert.Empty(t, client.getRequests)
		assert.Empty(t, client.moveRequests)
	})

	t.Run("create path skips an instance that does not exist yet", func(t *testing.T) {
		client := &stubKargoClient{getErr: status.Error(codes.NotFound, "no such instance")}
		attempted, err := ensureKargoInstanceWorkspace(context.Background(), client, "org-1", "", "new-instance", target)
		require.NoError(t, err)
		assert.False(t, attempted)
		require.Len(t, client.getRequests, 1)
		assert.Equal(t, "new-instance", client.getRequests[0].GetName())
		assert.Empty(t, client.listRequests)
		assert.Empty(t, client.moveRequests)
	})

	t.Run("create path moves an existing instance found by name", func(t *testing.T) {
		client := &stubKargoClient{instance: &kargov1.KargoInstance{Id: "inst-1", Name: "existing", WorkspaceId: "ws-old"}}
		attempted, err := ensureKargoInstanceWorkspace(context.Background(), client, "org-1", "", "existing", target)
		require.NoError(t, err)
		assert.True(t, attempted)
		require.Len(t, client.moveRequests, 1)
		assert.Equal(t, "inst-1", client.moveRequests[0].GetId())
	})

	t.Run("skips unassigned instance targeting default workspace", func(t *testing.T) {
		instance := &kargov1.KargoInstance{Id: "inst-1", Name: "existing"}
		client := &stubKargoClient{instances: []*kargov1.KargoInstance{instance}}
		attempted, err := ensureKargoInstanceWorkspace(context.Background(), client, "org-1", "inst-1", "renamed", defaultWorkspace)
		require.NoError(t, err)
		assert.False(t, attempted)
		assert.Empty(t, client.moveRequests)
	})

	t.Run("propagates a non-retryable move error", func(t *testing.T) {
		instance := &kargov1.KargoInstance{Id: "inst-1", Name: "existing", WorkspaceId: "ws-old"}
		client := &stubKargoClient{instances: []*kargov1.KargoInstance{instance}, moveErrors: []error{status.Error(codes.PermissionDenied, "denied")}}
		attempted, err := ensureKargoInstanceWorkspace(context.Background(), client, "org-1", "inst-1", "renamed", target)
		require.Error(t, err)
		assert.True(t, attempted)
		assert.Len(t, client.moveRequests, 1)
	})
}

func TestEnsureKargoInstanceWorkspaceRechecksAfterTransientMoveError(t *testing.T) {
	target := &orgcv1.Workspace{Id: "ws-new", Name: "target"}
	instance := &kargov1.KargoInstance{Id: "inst-1", Name: "old-name", WorkspaceId: "ws-old"}
	client := &stubKargoClient{
		instances: []*kargov1.KargoInstance{instance}, moveErrors: []error{status.Error(codes.Unavailable, "response lost")}, mutateOnMoveError: true,
	}
	attempted, err := ensureKargoInstanceWorkspace(context.Background(), client, "org-1", "inst-1", "renamed", target)
	require.NoError(t, err)
	assert.True(t, attempted)
	assert.Len(t, client.listRequests, 2)
	assert.Len(t, client.moveRequests, 1, "the retry must observe the completed move before issuing another update")
}

func TestWorkspaceMoveNotNeeded(t *testing.T) {
	testCases := map[string]struct {
		current string
		target  *orgcv1.Workspace
		want    bool
	}{
		"same workspace": {current: "ws-1", target: &orgcv1.Workspace{Id: "ws-1"}, want: true},
		"legacy unassigned instance is in default workspace":   {target: &orgcv1.Workspace{Id: "ws-default", IsDefault: true}, want: true},
		"legacy unassigned instance is not in named workspace": {target: &orgcv1.Workspace{Id: "ws-1"}},
		"different workspace": {current: "ws-old", target: &orgcv1.Workspace{Id: "ws-new"}},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, workspaceMoveNotNeeded(tc.current, tc.target))
		})
	}
}

func TestWorkspaceStateValue(t *testing.T) {
	defaultWorkspace := &orgcv1.Workspace{Id: "ws-default", Name: "default", IsDefault: true}
	namedWorkspace := &orgcv1.Workspace{Id: "ws-named", Name: "team"}

	testCases := map[string]struct {
		current   tftypes.String
		workspace *orgcv1.Workspace
		want      tftypes.String
	}{
		"preserves explicit empty string for default workspace": {
			current: tftypes.StringValue(""), workspace: defaultWorkspace, want: tftypes.StringValue(""),
		},
		"hydrates null default workspace": {
			current: tftypes.StringNull(), workspace: defaultWorkspace, want: tftypes.StringValue("default"),
		},
		"hydrates unknown default workspace": {
			current: tftypes.StringUnknown(), workspace: defaultWorkspace, want: tftypes.StringValue("default"),
		},
		"hydrates drift to default workspace from named state": {
			current: tftypes.StringValue("team"), workspace: defaultWorkspace, want: tftypes.StringValue("default"),
		},
		"hydrates named workspace": {
			current: tftypes.StringValue(""), workspace: namedWorkspace, want: tftypes.StringValue("team"),
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.want, workspaceStateValue(tc.current, tc.workspace))
		})
	}
}

func TestWorkspaceStateFromAPI(t *testing.T) {
	defaultWorkspace := &orgcv1.Workspace{Id: "ws-default", Name: "default", IsDefault: true}
	namedWorkspace := &orgcv1.Workspace{Id: "ws-named", Name: "team"}
	current := tftypes.StringValue("planned")

	testCases := map[string]struct {
		client      *stubWorkspaceClient
		workspaceID string
		current     tftypes.String
		strict      bool
		want        tftypes.String
		wantErr     bool
		wantRequest bool
	}{
		"resolves named workspace in strict mode": {
			client:      &stubWorkspaceClient{workspaces: []*orgcv1.Workspace{defaultWorkspace, namedWorkspace}},
			workspaceID: namedWorkspace.GetId(),
			current:     current,
			strict:      true,
			want:        tftypes.StringValue(namedWorkspace.GetName()),
			wantRequest: true,
		},
		"preserves explicit empty default spelling": {
			client:      &stubWorkspaceClient{workspaces: []*orgcv1.Workspace{defaultWorkspace}},
			workspaceID: defaultWorkspace.GetId(),
			current:     tftypes.StringValue(""),
			strict:      true,
			want:        tftypes.StringValue(""),
			wantRequest: true,
		},
		"tolerant mode preserves state for empty workspace ID": {
			client:  &stubWorkspaceClient{},
			current: current,
			want:    current,
		},
		"strict mode rejects empty workspace ID": {
			client:  &stubWorkspaceClient{},
			current: current,
			strict:  true,
			want:    current,
			wantErr: true,
		},
		"tolerant mode preserves state on lookup error": {
			client:      &stubWorkspaceClient{listErr: status.Error(codes.PermissionDenied, "denied")},
			workspaceID: namedWorkspace.GetId(),
			current:     current,
			want:        current,
			wantRequest: true,
		},
		"strict mode propagates lookup error": {
			client:      &stubWorkspaceClient{listErr: status.Error(codes.PermissionDenied, "denied")},
			workspaceID: namedWorkspace.GetId(),
			current:     current,
			strict:      true,
			want:        current,
			wantErr:     true,
			wantRequest: true,
		},
		"tolerant mode preserves state for unknown workspace": {
			client:      &stubWorkspaceClient{workspaces: []*orgcv1.Workspace{defaultWorkspace}},
			workspaceID: namedWorkspace.GetId(),
			current:     current,
			want:        current,
			wantRequest: true,
		},
		"strict mode rejects unknown workspace": {
			client:      &stubWorkspaceClient{workspaces: []*orgcv1.Workspace{defaultWorkspace}},
			workspaceID: namedWorkspace.GetId(),
			current:     current,
			strict:      true,
			want:        current,
			wantErr:     true,
			wantRequest: true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got, err := workspaceStateFromAPI(
				context.Background(),
				tc.client,
				"org-1",
				tc.workspaceID,
				tc.current,
				tc.strict,
			)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tc.want, got)
			if tc.wantRequest {
				require.Len(t, tc.client.listRequests, 1)
				assert.Equal(t, "org-1", tc.client.listRequests[0].GetOrganizationId())
			} else {
				assert.Empty(t, tc.client.listRequests)
			}
		})
	}
}
