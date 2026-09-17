//go:build !acc

package akp

import (
	"context"
	"maps"
	"slices"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	orgcv1 "github.com/akuity/api-client-go/pkg/api/gen/organization/v1"
	"github.com/akuity/terraform-provider-akp/akp/types"
)

type managedSecretsArgoCDClient struct {
	argocdv1.ArgoCDServiceGatewayClient
	response          *argocdv1.ListInstanceManagedSecretsResponse
	responses         []*argocdv1.ListInstanceManagedSecretsResponse
	listCalls         int
	request           *argocdv1.ListInstanceManagedSecretsRequest
	deleteRequests    []*argocdv1.DeleteManagedSecretRequest
	deleteErrors      map[string]error
	createRequests    []*argocdv1.CreateManagedSecretRequest
	createErrors      map[string]error
	createErrorQueues map[string][]error
	updateRequests    []*argocdv1.UpdateManagedSecretRequest
	updateErrors      map[string]error
	patchRequests     []*argocdv1.PatchManagedSecretRequest
	patchErrors       map[string]error
}

func (c *managedSecretsArgoCDClient) CreateManagedSecret(_ context.Context, req *argocdv1.CreateManagedSecretRequest) (*argocdv1.CreateManagedSecretResponse, error) {
	c.createRequests = append(c.createRequests, req)
	name := req.GetManagedSecret().GetName()
	if queue := c.createErrorQueues[name]; len(queue) > 0 {
		err := queue[0]
		c.createErrorQueues[name] = queue[1:]
		if err != nil {
			return nil, err
		}
		return &argocdv1.CreateManagedSecretResponse{}, nil
	}
	if err := c.createErrors[name]; err != nil {
		return nil, err
	}
	return &argocdv1.CreateManagedSecretResponse{}, nil
}

func (c *managedSecretsArgoCDClient) UpdateManagedSecret(_ context.Context, req *argocdv1.UpdateManagedSecretRequest) (*argocdv1.UpdateManagedSecretResponse, error) {
	c.updateRequests = append(c.updateRequests, req)
	if err := c.updateErrors[req.GetName()]; err != nil {
		return nil, err
	}
	return &argocdv1.UpdateManagedSecretResponse{}, nil
}

func (c *managedSecretsArgoCDClient) PatchManagedSecret(_ context.Context, req *argocdv1.PatchManagedSecretRequest) (*argocdv1.PatchManagedSecretResponse, error) {
	c.patchRequests = append(c.patchRequests, req)
	if err := c.patchErrors[req.GetName()]; err != nil {
		return nil, err
	}
	return &argocdv1.PatchManagedSecretResponse{}, nil
}

func newTestManagedSecret(data map[string]string) *types.ManagedSecret {
	secret := &types.ManagedSecret{
		Labels:          tftypes.MapNull(tftypes.StringType),
		AllowedClusters: tftypes.ListNull(tftypes.StringType),
		ClusterSelector: tftypes.StringNull(),
		Data:            tftypes.MapNull(tftypes.StringType),
		DataVersion:     tftypes.StringNull(),
	}
	if data != nil {
		value, diags := tftypes.MapValueFrom(context.Background(), tftypes.StringType, data)
		if diags.HasError() {
			panic(diags)
		}
		secret.Data = value
	}
	return secret
}

func newVersionedManagedSecret(data map[string]string, version string) *types.ManagedSecret {
	secret := newTestManagedSecret(data)
	secret.DataVersion = tftypes.StringValue(version)
	return secret
}

func TestSyncManagedSecrets(t *testing.T) {
	ctx := context.Background()

	t.Run("creates new, updates tracked, deletes removed", func(t *testing.T) {
		var diags diag.Diagnostics
		client := &managedSecretsArgoCDClient{}
		cli := &AkpCli{Cli: client, OrgId: "org-id"}
		state := map[string]*types.ManagedSecret{
			"kept":    newTestManagedSecret(map[string]string{"token": "old"}),
			"removed": newTestManagedSecret(nil),
		}
		plan := map[string]*types.ManagedSecret{
			"kept": newVersionedManagedSecret(map[string]string{"token": "new"}, "2"),
			"new":  newTestManagedSecret(map[string]string{"token": "fresh"}),
		}

		synced, err := syncManagedSecrets(ctx, cli, &diags, "instance-id", "workspace-id", state, plan)
		require.NoError(t, err)
		require.False(t, diags.HasError(), "%v", diags)

		require.Len(t, client.deleteRequests, 1)
		assert.Equal(t, "removed", client.deleteRequests[0].GetName())
		require.Len(t, client.updateRequests, 1)
		assert.Equal(t, "kept", client.updateRequests[0].GetName())
		assert.Equal(t, map[string]string{"token": "new"}, client.updateRequests[0].GetManagedSecretData())
		require.Len(t, client.createRequests, 1)
		assert.Equal(t, "new", client.createRequests[0].GetManagedSecret().GetName())
		assert.Equal(t, "org-id", client.createRequests[0].GetOrganizationId())
		assert.Equal(t, "workspace-id", client.createRequests[0].GetWorkspaceId())
		assert.Equal(t, "instance-id", client.createRequests[0].GetInstanceId())
		assert.Empty(t, client.patchRequests)
		assert.ElementsMatch(t, []string{"kept", "new"}, slices.Collect(maps.Keys(synced)))
	})

	t.Run("existing untracked secret is not adopted", func(t *testing.T) {
		var diags diag.Diagnostics
		client := &managedSecretsArgoCDClient{createErrors: map[string]error{
			"ui-secret": status.Error(codes.AlreadyExists, "managed secret \"ui-secret\" already exists"),
		}}
		cli := &AkpCli{Cli: client, OrgId: "org-id"}
		plan := map[string]*types.ManagedSecret{"ui-secret": newTestManagedSecret(nil)}

		synced, err := syncManagedSecrets(ctx, cli, &diags, "instance-id", "workspace-id", nil, plan)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not adopted")
		assert.Empty(t, synced)
	})

	t.Run("not found on update falls back to create", func(t *testing.T) {
		var diags diag.Diagnostics
		client := &managedSecretsArgoCDClient{updateErrors: map[string]error{
			"kept": status.Error(codes.NotFound, "managed secret \"kept\" not found"),
		}}
		cli := &AkpCli{Cli: client, OrgId: "org-id"}
		state := map[string]*types.ManagedSecret{"kept": newTestManagedSecret(nil)}
		plan := map[string]*types.ManagedSecret{"kept": newTestManagedSecret(nil)}

		synced, err := syncManagedSecrets(ctx, cli, &diags, "instance-id", "workspace-id", state, plan)
		require.NoError(t, err)
		require.Len(t, client.updateRequests, 1)
		require.Len(t, client.createRequests, 1)
		assert.Equal(t, "kept", client.createRequests[0].GetManagedSecret().GetName())
		assert.Contains(t, synced, "kept")
	})

	t.Run("empty data clears keys via patch", func(t *testing.T) {
		var diags diag.Diagnostics
		client := &managedSecretsArgoCDClient{response: &argocdv1.ListInstanceManagedSecretsResponse{
			ManagedSecrets: []*argocdv1.ManagedSecret{
				{Name: "kept", SecretKeys: []string{"a", "b"}},
			},
		}}
		cli := &AkpCli{Cli: client, OrgId: "org-id"}
		state := map[string]*types.ManagedSecret{"kept": newTestManagedSecret(nil)}
		plan := map[string]*types.ManagedSecret{"kept": newVersionedManagedSecret(map[string]string{}, "2")}

		synced, err := syncManagedSecrets(ctx, cli, &diags, "instance-id", "workspace-id", state, plan)
		require.NoError(t, err)
		assert.Empty(t, client.updateRequests)
		require.Len(t, client.patchRequests, 1)
		assert.Equal(t, "kept", client.patchRequests[0].GetName())
		assert.Equal(t, map[string]string{"a": "", "b": ""}, client.patchRequests[0].GetManagedSecretData())
		assert.Contains(t, synced, "kept")
	})

	t.Run("create with empty data clears keys of a recovered secret", func(t *testing.T) {
		var diags diag.Diagnostics
		client := &managedSecretsArgoCDClient{response: &argocdv1.ListInstanceManagedSecretsResponse{
			ManagedSecrets: []*argocdv1.ManagedSecret{
				{Name: "orphan", SecretKeys: []string{"token"}},
			},
		}}
		cli := &AkpCli{Cli: client, OrgId: "org-id"}
		plan := map[string]*types.ManagedSecret{"orphan": newVersionedManagedSecret(map[string]string{}, "9")}

		synced, err := syncManagedSecrets(ctx, cli, &diags, "instance-id", "workspace-id", nil, plan)
		require.NoError(t, err)
		require.Len(t, client.createRequests, 1)
		assert.Empty(t, client.createRequests[0].GetManagedSecretData())
		require.Len(t, client.patchRequests, 1)
		assert.Equal(t, "orphan", client.patchRequests[0].GetName())
		assert.Equal(t, map[string]string{"token": ""}, client.patchRequests[0].GetManagedSecretData())
		assert.Contains(t, synced, "orphan")
	})

	t.Run("create re-lists keys instead of trusting an earlier snapshot", func(t *testing.T) {
		var diags diag.Diagnostics
		client := &managedSecretsArgoCDClient{responses: []*argocdv1.ListInstanceManagedSecretsResponse{
			{ManagedSecrets: []*argocdv1.ManagedSecret{{Name: "kept", SecretKeys: []string{"a"}}}},
			{ManagedSecrets: []*argocdv1.ManagedSecret{{Name: "kept"}, {Name: "orphan", SecretKeys: []string{"token"}}}},
		}}
		cli := &AkpCli{Cli: client, OrgId: "org-id"}
		state := map[string]*types.ManagedSecret{"kept": newTestManagedSecret(nil)}
		plan := map[string]*types.ManagedSecret{
			"kept":   newVersionedManagedSecret(map[string]string{}, "2"),
			"orphan": newVersionedManagedSecret(map[string]string{}, "1"),
		}

		_, err := syncManagedSecrets(ctx, cli, &diags, "instance-id", "workspace-id", state, plan)
		require.NoError(t, err)
		assert.Equal(t, 2, client.listCalls)
		require.Len(t, client.patchRequests, 2)
		assert.Equal(t, "kept", client.patchRequests[0].GetName())
		assert.Equal(t, map[string]string{"a": ""}, client.patchRequests[0].GetManagedSecretData())
		assert.Equal(t, "orphan", client.patchRequests[1].GetName())
		assert.Equal(t, map[string]string{"token": ""}, client.patchRequests[1].GetManagedSecretData())
	})

	t.Run("create with empty data skips the patch when the server has no keys", func(t *testing.T) {
		var diags diag.Diagnostics
		client := &managedSecretsArgoCDClient{response: &argocdv1.ListInstanceManagedSecretsResponse{}}
		cli := &AkpCli{Cli: client, OrgId: "org-id"}
		plan := map[string]*types.ManagedSecret{"fresh": newVersionedManagedSecret(map[string]string{}, "1")}

		_, err := syncManagedSecrets(ctx, cli, &diags, "instance-id", "workspace-id", nil, plan)
		require.NoError(t, err)
		require.Len(t, client.createRequests, 1)
		require.NotNil(t, client.request, "keys must be listed before deciding to skip the patch")
		assert.Empty(t, client.patchRequests)
	})

	t.Run("create with omitted data never patches", func(t *testing.T) {
		var diags diag.Diagnostics
		client := &managedSecretsArgoCDClient{}
		cli := &AkpCli{Cli: client, OrgId: "org-id"}
		plan := map[string]*types.ManagedSecret{"orphan": newTestManagedSecret(nil)}

		_, err := syncManagedSecrets(ctx, cli, &diags, "instance-id", "workspace-id", nil, plan)
		require.NoError(t, err)
		require.Len(t, client.createRequests, 1)
		assert.Nil(t, client.request)
		assert.Empty(t, client.patchRequests)
	})

	t.Run("unchanged data_version sends metadata-only update", func(t *testing.T) {
		var diags diag.Diagnostics
		client := &managedSecretsArgoCDClient{}
		cli := &AkpCli{Cli: client, OrgId: "org-id"}
		state := map[string]*types.ManagedSecret{"kept": newVersionedManagedSecret(nil, "1")}
		plan := map[string]*types.ManagedSecret{"kept": newVersionedManagedSecret(map[string]string{"token": "changed-without-bump"}, "1")}

		synced, err := syncManagedSecrets(ctx, cli, &diags, "instance-id", "workspace-id", state, plan)
		require.NoError(t, err)
		require.Len(t, client.updateRequests, 1)
		assert.Empty(t, client.updateRequests[0].GetManagedSecretData())
		assert.Empty(t, client.patchRequests)
		assert.Empty(t, client.createRequests)
		assert.Contains(t, synced, "kept")
	})

	t.Run("unchanged data_version skips clearing data", func(t *testing.T) {
		var diags diag.Diagnostics
		client := &managedSecretsArgoCDClient{}
		cli := &AkpCli{Cli: client, OrgId: "org-id"}
		state := map[string]*types.ManagedSecret{"kept": newVersionedManagedSecret(nil, "1")}
		plan := map[string]*types.ManagedSecret{"kept": newVersionedManagedSecret(map[string]string{}, "1")}

		_, err := syncManagedSecrets(ctx, cli, &diags, "instance-id", "workspace-id", state, plan)
		require.NoError(t, err)
		require.Len(t, client.updateRequests, 1)
		assert.Empty(t, client.updateRequests[0].GetManagedSecretData())
		assert.Empty(t, client.patchRequests)
	})

	t.Run("create fallback sends data despite unchanged data_version", func(t *testing.T) {
		var diags diag.Diagnostics
		client := &managedSecretsArgoCDClient{updateErrors: map[string]error{
			"kept": status.Error(codes.NotFound, "managed secret \"kept\" not found"),
		}}
		cli := &AkpCli{Cli: client, OrgId: "org-id"}
		state := map[string]*types.ManagedSecret{"kept": newVersionedManagedSecret(nil, "1")}
		plan := map[string]*types.ManagedSecret{"kept": newVersionedManagedSecret(map[string]string{"token": "value"}, "1")}

		synced, err := syncManagedSecrets(ctx, cli, &diags, "instance-id", "workspace-id", state, plan)
		require.NoError(t, err)
		require.Len(t, client.updateRequests, 1)
		assert.Empty(t, client.updateRequests[0].GetManagedSecretData())
		require.Len(t, client.createRequests, 1)
		assert.Equal(t, map[string]string{"token": "value"}, client.createRequests[0].GetManagedSecretData())
		assert.Contains(t, synced, "kept")
	})

	t.Run("delete failure keeps the name tracked", func(t *testing.T) {
		var diags diag.Diagnostics
		client := &managedSecretsArgoCDClient{deleteErrors: map[string]error{
			"stuck": status.Error(codes.InvalidArgument, "cannot delete reserved secret"),
		}}
		cli := &AkpCli{Cli: client, OrgId: "org-id"}
		state := map[string]*types.ManagedSecret{"stuck": newTestManagedSecret(nil)}

		synced, err := syncManagedSecrets(ctx, cli, &diags, "instance-id", "workspace-id", state, nil)
		require.Error(t, err)
		assert.Contains(t, synced, "stuck")
	})

	t.Run("upsert failure prevents deletions", func(t *testing.T) {
		var diags diag.Diagnostics
		client := &managedSecretsArgoCDClient{createErrors: map[string]error{
			"new": status.Error(codes.InvalidArgument, "cannot use reserved secret \"new\" as a managed secret"),
		}}
		cli := &AkpCli{Cli: client, OrgId: "org-id"}
		state := map[string]*types.ManagedSecret{"removed": newTestManagedSecret(nil)}
		plan := map[string]*types.ManagedSecret{"new": newTestManagedSecret(nil)}

		synced, err := syncManagedSecrets(ctx, cli, &diags, "instance-id", "workspace-id", state, plan)
		require.Error(t, err)
		assert.Contains(t, err.Error(), `Unable to apply managed secret "new"`)
		assert.Empty(t, client.deleteRequests)
		assert.Contains(t, synced, "removed")
	})

	t.Run("create failure commits earlier progress", func(t *testing.T) {
		var diags diag.Diagnostics
		client := &managedSecretsArgoCDClient{createErrors: map[string]error{
			"b-fails": status.Error(codes.InvalidArgument, "invalid managed secret"),
		}}
		cli := &AkpCli{Cli: client, OrgId: "org-id"}
		plan := map[string]*types.ManagedSecret{
			"a-ok":    newTestManagedSecret(nil),
			"b-fails": newTestManagedSecret(nil),
		}

		synced, err := syncManagedSecrets(ctx, cli, &diags, "instance-id", "workspace-id", nil, plan)
		require.Error(t, err)
		assert.Contains(t, synced, "a-ok")
		assert.NotContains(t, synced, "b-fails")
	})

	t.Run("partial failure nulls write-only data on upserted secrets", func(t *testing.T) {
		var diags diag.Diagnostics
		client := &managedSecretsArgoCDClient{createErrors: map[string]error{
			"z-fails": status.Error(codes.InvalidArgument, "cannot use reserved name prefix"),
		}}
		cli := &AkpCli{Cli: client, OrgId: "org-id"}
		state := map[string]*types.ManagedSecret{"tracked": newTestManagedSecret(nil)}
		plan := map[string]*types.ManagedSecret{
			"tracked": newTestManagedSecret(map[string]string{"token": "new"}),
			"created": newTestManagedSecret(map[string]string{"token": "fresh"}),
			"z-fails": newTestManagedSecret(nil),
		}

		synced, err := syncManagedSecrets(ctx, cli, &diags, "instance-id", "workspace-id", state, plan)
		require.Error(t, err)
		require.Contains(t, synced, "tracked")
		require.Contains(t, synced, "created")
		assert.True(t, synced["tracked"].Data.IsNull())
		assert.True(t, synced["created"].Data.IsNull())
		assert.False(t, plan["tracked"].Data.IsNull())
	})
}

func TestCreateManagedSecretOwner(t *testing.T) {
	ctx := context.Background()

	t.Run("create carries the terraform owner across retries", func(t *testing.T) {
		client := &managedSecretsArgoCDClient{createErrorQueues: map[string][]error{
			"racy": {status.Error(codes.Unavailable, "connection reset"), nil},
		}}
		cli := &AkpCli{Cli: client, OrgId: "org-id"}
		upsert := &types.ManagedSecretUpsert{
			Secret: &argocdv1.ManagedSecret{Name: "racy"},
			Data:   map[string]string{"token": "value"},
		}

		require.NoError(t, createManagedSecret(ctx, cli, "instance-id", "workspace-id", upsert))
		require.Len(t, client.createRequests, 2)
		for _, req := range client.createRequests {
			assert.Equal(t, managedSecretOwner, req.GetOwner())
			assert.Equal(t, map[string]string{"token": "value"}, req.GetManagedSecretData())
		}
		assert.Empty(t, client.updateRequests)
	})

	t.Run("already exists after a retry is returned without a client-side fallback", func(t *testing.T) {
		client := &managedSecretsArgoCDClient{createErrorQueues: map[string][]error{
			"taken": {
				status.Error(codes.Unavailable, "connection reset"),
				status.Error(codes.AlreadyExists, "managed secret \"taken\" already exists"),
			},
		}}
		cli := &AkpCli{Cli: client, OrgId: "org-id"}
		upsert := &types.ManagedSecretUpsert{Secret: &argocdv1.ManagedSecret{Name: "taken"}}

		err := createManagedSecret(ctx, cli, "instance-id", "workspace-id", upsert)
		assert.Equal(t, codes.AlreadyExists, status.Code(err))
		require.Len(t, client.createRequests, 2)
		assert.Empty(t, client.updateRequests)
	})
}

func (c *managedSecretsArgoCDClient) ListInstanceManagedSecrets(_ context.Context, req *argocdv1.ListInstanceManagedSecretsRequest) (*argocdv1.ListInstanceManagedSecretsResponse, error) {
	c.request = req
	c.listCalls++
	if len(c.responses) > 0 {
		resp := c.responses[0]
		c.responses = c.responses[1:]
		return resp, nil
	}
	return c.response, nil
}

func (c *managedSecretsArgoCDClient) DeleteManagedSecret(_ context.Context, req *argocdv1.DeleteManagedSecretRequest) (*argocdv1.DeleteManagedSecretResponse, error) {
	c.deleteRequests = append(c.deleteRequests, req)
	if err := c.deleteErrors[req.GetName()]; err != nil {
		return nil, err
	}
	return &argocdv1.DeleteManagedSecretResponse{}, nil
}

func TestInstanceSchemaValidateImplementation(t *testing.T) {
	diags := instanceSchema().ValidateImplementation(context.Background())
	assert.False(t, diags.HasError(), "%v", diags)
}

func TestManagedSecretsNilEntry(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	instance := &types.Instance{ManagedSecrets: map[string]*types.ManagedSecret{"null": nil}}

	assert.Empty(t, instance.GetSensitiveStrings(ctx, &diags))
	require.False(t, diags.HasError())
}

func TestRefreshManagedSecrets(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	client := &managedSecretsArgoCDClient{
		response: &argocdv1.ListInstanceManagedSecretsResponse{
			ManagedSecrets: []*argocdv1.ManagedSecret{
				{Name: "configured", Labels: map[string]string{"team": "platform"}},
				{Name: "unconfigured", Labels: map[string]string{"team": "other"}},
			},
		},
	}
	instance := &types.Instance{
		ID: tftypes.StringValue("instance-id"),
		ManagedSecrets: map[string]*types.ManagedSecret{
			"configured": {
				Labels:          tftypes.MapNull(tftypes.StringType),
				AllowedClusters: tftypes.ListNull(tftypes.StringType),
				ClusterSelector: tftypes.StringNull(),
				Data:            tftypes.MapNull(tftypes.StringType),
				DataVersion:     tftypes.StringValue("1"),
			},
			"deleted": {
				Labels:          tftypes.MapNull(tftypes.StringType),
				AllowedClusters: tftypes.ListNull(tftypes.StringType),
				ClusterSelector: tftypes.StringNull(),
				Data:            tftypes.MapNull(tftypes.StringType),
				DataVersion:     tftypes.StringValue("1"),
			},
		},
	}
	cli := &AkpCli{Cli: client, OrgId: "org-id"}

	require.NoError(t, refreshManagedSecrets(ctx, &diags, cli, instance, "workspace-id"))
	require.False(t, diags.HasError(), "%v", diags)
	assert.Equal(t, "org-id", client.request.GetOrganizationId())
	assert.Equal(t, "workspace-id", client.request.GetWorkspaceId())
	assert.Equal(t, "instance-id", client.request.GetInstanceId())
	require.Len(t, instance.ManagedSecrets, 1)
	assert.Contains(t, instance.ManagedSecrets, "configured")
	assert.NotContains(t, instance.ManagedSecrets, "deleted")
	assert.NotContains(t, instance.ManagedSecrets, "unconfigured")
}

func TestRemovedManagedSecretNames(t *testing.T) {
	secret := &types.ManagedSecret{}
	tests := map[string]struct {
		state  map[string]*types.ManagedSecret
		plan   map[string]*types.ManagedSecret
		expect []string
	}{
		"removed names are sorted": {
			state: map[string]*types.ManagedSecret{
				"removed-b": secret,
				"kept":      secret,
				"removed-a": secret,
				"null-plan": secret,
				"null":      nil,
			},
			plan:   map[string]*types.ManagedSecret{"kept": secret, "new": secret, "null-plan": nil},
			expect: []string{"null-plan", "removed-a", "removed-b"},
		},
		"nil state": {
			plan: map[string]*types.ManagedSecret{"new": secret},
		},
		"nil plan removes state names": {
			state:  map[string]*types.ManagedSecret{"removed": secret},
			expect: []string{"removed"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, tc.expect, removedManagedSecretNames(tc.state, tc.plan))
		})
	}
}

func TestDeleteManagedSecret(t *testing.T) {
	t.Run("deletes the named secret", func(t *testing.T) {
		client := &managedSecretsArgoCDClient{}
		cli := &AkpCli{Cli: client, OrgId: "org-id"}

		require.NoError(t, deleteManagedSecret(context.Background(), cli, "instance-id", "workspace-id", "first"))
		require.Len(t, client.deleteRequests, 1)
		assert.Equal(t, "org-id", client.deleteRequests[0].GetOrganizationId())
		assert.Equal(t, "workspace-id", client.deleteRequests[0].GetWorkspaceId())
		assert.Equal(t, "instance-id", client.deleteRequests[0].GetInstanceId())
		assert.Equal(t, "first", client.deleteRequests[0].GetName())
	})

	t.Run("delete failure is returned", func(t *testing.T) {
		client := &managedSecretsArgoCDClient{deleteErrors: map[string]error{
			"reserved": status.Error(codes.InvalidArgument, "cannot delete reserved secret"),
		}}
		cli := &AkpCli{Cli: client, OrgId: "org-id"}

		err := deleteManagedSecret(context.Background(), cli, "instance-id", "workspace-id", "reserved")
		require.Error(t, err)
		assert.Contains(t, err.Error(), `Unable to delete managed secret "reserved"`)
	})
}

func TestApplyManagedSecretChangesWorkspaceLookupFailure(t *testing.T) {
	var diags diag.Diagnostics
	client := &managedSecretsArgoCDClient{}
	cli := &AkpCli{Cli: client, OrgCli: &stubWorkspaceClient{}, OrgId: "org-id"}
	result := &types.Instance{
		ID:        tftypes.StringValue("instance-id"),
		Workspace: tftypes.StringValue("missing-workspace"),
	}
	stateSecrets := map[string]*types.ManagedSecret{"kept": newTestManagedSecret(nil)}
	plannedSecrets := map[string]*types.ManagedSecret{
		"kept": newTestManagedSecret(nil),
		"new":  newTestManagedSecret(nil),
	}

	got, err := applyManagedSecretChanges(context.Background(), cli, &diags, result, stateSecrets, plannedSecrets, nil)
	require.Error(t, err)
	require.ErrorContains(t, err, "Unable to get workspace for managed secret changes")
	require.Same(t, result, got)
	assert.Equal(t, stateSecrets, got.ManagedSecrets)
	assert.Empty(t, client.createRequests)
	assert.Empty(t, client.updateRequests)
	assert.Empty(t, client.deleteRequests)

	t.Run("keeps already created secrets tracked", func(t *testing.T) {
		var diags diag.Diagnostics
		created := map[string]*types.ManagedSecret{"new": newTestManagedSecret(nil)}

		got, err := applyManagedSecretChanges(context.Background(), cli, &diags, result, stateSecrets, plannedSecrets, created)
		require.Error(t, err)
		assert.Contains(t, got.ManagedSecrets, "kept")
		assert.Contains(t, got.ManagedSecrets, "new")
	})
}

func TestApplyManagedSecretChangesSkipsCreatedSecrets(t *testing.T) {
	var diags diag.Diagnostics
	client := &managedSecretsArgoCDClient{response: &argocdv1.ListInstanceManagedSecretsResponse{
		ManagedSecrets: []*argocdv1.ManagedSecret{{Name: "kept"}, {Name: "fresh"}},
	}}
	workspaces := []*orgcv1.Workspace{{Id: "workspace-id", Name: "default", IsDefault: true}}
	cli := &AkpCli{Cli: client, OrgCli: &stubWorkspaceClient{workspaces: workspaces}, OrgId: "org-id"}
	result := &types.Instance{
		ID:        tftypes.StringValue("instance-id"),
		Workspace: tftypes.StringNull(),
	}
	stateSecrets := map[string]*types.ManagedSecret{"kept": newVersionedManagedSecret(nil, "1")}
	plannedSecrets := map[string]*types.ManagedSecret{
		"kept":  newVersionedManagedSecret(map[string]string{"token": "new"}, "2"),
		"fresh": newTestManagedSecret(map[string]string{"token": "value"}),
	}
	created := map[string]*types.ManagedSecret{"fresh": newTestManagedSecret(nil)}

	got, err := applyManagedSecretChanges(context.Background(), cli, &diags, result, stateSecrets, plannedSecrets, created)
	require.NoError(t, err)
	require.False(t, diags.HasError(), "%v", diags)
	assert.Empty(t, client.createRequests)
	require.Len(t, client.updateRequests, 1)
	assert.Equal(t, "kept", client.updateRequests[0].GetName())
	assert.Empty(t, client.deleteRequests)
	assert.Contains(t, got.ManagedSecrets, "kept")
	assert.Contains(t, got.ManagedSecrets, "fresh")
}

func TestApplyManagedSecretChangesReportsWrittenValues(t *testing.T) {
	var diags diag.Diagnostics
	client := &managedSecretsArgoCDClient{response: &argocdv1.ListInstanceManagedSecretsResponse{
		ManagedSecrets: []*argocdv1.ManagedSecret{{Name: "kept", Labels: map[string]string{"team": "stale"}}},
	}}
	workspaces := []*orgcv1.Workspace{{Id: "workspace-id", Name: "default", IsDefault: true}}
	cli := &AkpCli{Cli: client, OrgCli: &stubWorkspaceClient{workspaces: workspaces}, OrgId: "org-id"}
	result := &types.Instance{ID: tftypes.StringValue("instance-id"), Workspace: tftypes.StringNull()}
	planned := newTestManagedSecret(nil)
	planned.Labels, _ = tftypes.MapValueFrom(context.Background(), tftypes.StringType, map[string]string{"team": "sre"})
	stateSecrets := map[string]*types.ManagedSecret{"kept": newTestManagedSecret(nil)}
	plannedSecrets := map[string]*types.ManagedSecret{"kept": planned}

	got, err := applyManagedSecretChanges(context.Background(), cli, &diags, result, stateSecrets, plannedSecrets, nil)
	require.NoError(t, err)
	require.Len(t, client.updateRequests, 1)
	assert.Equal(t, 0, client.listCalls, "post-apply state must not depend on a re-read")
	assert.Equal(t, planned.Labels, got.ManagedSecrets["kept"].Labels)
	assert.True(t, got.ManagedSecrets["kept"].Data.IsNull())
}

func TestCreateAddedManagedSecrets(t *testing.T) {
	ctx := context.Background()
	workspaces := []*orgcv1.Workspace{{Id: "workspace-id", Name: "default", IsDefault: true}}
	newInstance := func(secrets map[string]*types.ManagedSecret) *types.Instance {
		return &types.Instance{
			ID:             tftypes.StringValue("instance-id"),
			Workspace:      tftypes.StringNull(),
			ManagedSecrets: secrets,
		}
	}

	t.Run("creates only added names before the instance apply", func(t *testing.T) {
		var diags diag.Diagnostics
		client := &managedSecretsArgoCDClient{}
		cli := &AkpCli{Cli: client, OrgCli: &stubWorkspaceClient{workspaces: workspaces}, OrgId: "org-id"}
		state := newInstance(map[string]*types.ManagedSecret{"kept": newTestManagedSecret(nil)})
		plan := newInstance(map[string]*types.ManagedSecret{
			"kept":  newTestManagedSecret(nil),
			"fresh": newTestManagedSecret(map[string]string{"token": "value"}),
		})

		created, err := createAddedManagedSecrets(ctx, cli, &diags, state, plan)
		require.NoError(t, err)
		require.False(t, diags.HasError(), "%v", diags)
		require.Len(t, client.createRequests, 1)
		assert.Equal(t, "fresh", client.createRequests[0].GetManagedSecret().GetName())
		assert.Equal(t, "workspace-id", client.createRequests[0].GetWorkspaceId())
		assert.Equal(t, "instance-id", client.createRequests[0].GetInstanceId())
		assert.Equal(t, managedSecretOwner, client.createRequests[0].GetOwner())
		assert.Empty(t, client.updateRequests)
		assert.Empty(t, client.deleteRequests)
		require.Contains(t, created, "fresh")
		assert.NotContains(t, created, "kept")
		assert.True(t, created["fresh"].Data.IsNull())
	})

	t.Run("no added names skips the lookup", func(t *testing.T) {
		var diags diag.Diagnostics
		client := &managedSecretsArgoCDClient{}
		cli := &AkpCli{Cli: client, OrgId: "org-id"}
		state := newInstance(map[string]*types.ManagedSecret{"kept": newTestManagedSecret(nil)})
		plan := newInstance(map[string]*types.ManagedSecret{"kept": newTestManagedSecret(nil)})

		created, err := createAddedManagedSecrets(ctx, cli, &diags, state, plan)
		require.NoError(t, err)
		assert.Nil(t, created)
		assert.Empty(t, client.createRequests)
	})

	t.Run("conflict fails before the instance apply", func(t *testing.T) {
		var diags diag.Diagnostics
		client := &managedSecretsArgoCDClient{createErrors: map[string]error{
			"ui-secret": status.Error(codes.AlreadyExists, "managed secret \"ui-secret\" already exists"),
		}}
		cli := &AkpCli{Cli: client, OrgCli: &stubWorkspaceClient{workspaces: workspaces}, OrgId: "org-id"}
		state := newInstance(nil)
		plan := newInstance(map[string]*types.ManagedSecret{"ui-secret": newTestManagedSecret(nil)})

		created, err := createAddedManagedSecrets(ctx, cli, &diags, state, plan)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not adopted")
		assert.Empty(t, created)
	})

	t.Run("conflict keeps earlier creates tracked", func(t *testing.T) {
		var diags diag.Diagnostics
		client := &managedSecretsArgoCDClient{createErrors: map[string]error{
			"ui-secret": status.Error(codes.AlreadyExists, "managed secret \"ui-secret\" already exists"),
		}}
		cli := &AkpCli{Cli: client, OrgCli: &stubWorkspaceClient{workspaces: workspaces}, OrgId: "org-id"}
		state := newInstance(nil)
		plan := newInstance(map[string]*types.ManagedSecret{
			"fresh":     newTestManagedSecret(map[string]string{"token": "value"}),
			"ui-secret": newTestManagedSecret(nil),
		})

		created, err := createAddedManagedSecrets(ctx, cli, &diags, state, plan)
		require.Error(t, err)
		require.Len(t, client.createRequests, 2)
		require.Contains(t, created, "fresh")
		assert.NotContains(t, created, "ui-secret")
		assert.True(t, created["fresh"].Data.IsNull())
	})
}

func TestWithCreatedManagedSecrets(t *testing.T) {
	state := &types.Instance{
		ID:             tftypes.StringValue("instance-id"),
		ManagedSecrets: map[string]*types.ManagedSecret{"kept": newTestManagedSecret(nil)},
	}

	t.Run("nothing created keeps the prior state untouched", func(t *testing.T) {
		assert.Nil(t, withCreatedManagedSecrets(state, nil))
		assert.Nil(t, withCreatedManagedSecrets(state, map[string]*types.ManagedSecret{}))
	})

	t.Run("created secrets are added to the prior state", func(t *testing.T) {
		created := map[string]*types.ManagedSecret{"fresh": newTestManagedSecret(nil)}

		got := withCreatedManagedSecrets(state, created)
		require.NotNil(t, got)
		assert.Equal(t, "instance-id", got.ID.ValueString())
		assert.ElementsMatch(t, []string{"kept", "fresh"}, slices.Collect(maps.Keys(got.ManagedSecrets)))
		assert.ElementsMatch(t, []string{"kept"}, slices.Collect(maps.Keys(state.ManagedSecrets)))
	})

	t.Run("works when the prior state has no secrets", func(t *testing.T) {
		got := withCreatedManagedSecrets(&types.Instance{}, map[string]*types.ManagedSecret{"fresh": newTestManagedSecret(nil)})
		require.NotNil(t, got)
		assert.Contains(t, got.ManagedSecrets, "fresh")
	})
}

func TestTrackedManagedSecrets(t *testing.T) {
	stateSecrets := map[string]*types.ManagedSecret{"kept": newTestManagedSecret(nil)}

	t.Run("nothing created returns the state secrets as-is", func(t *testing.T) {
		assert.Nil(t, trackedManagedSecrets(nil, nil))
		got := trackedManagedSecrets(stateSecrets, nil)
		assert.Equal(t, stateSecrets, got)
	})

	t.Run("created secrets are merged without touching the state map", func(t *testing.T) {
		created := map[string]*types.ManagedSecret{"fresh": newTestManagedSecret(nil)}
		got := trackedManagedSecrets(stateSecrets, created)
		assert.ElementsMatch(t, []string{"kept", "fresh"}, slices.Collect(maps.Keys(got)))
		assert.ElementsMatch(t, []string{"kept"}, slices.Collect(maps.Keys(stateSecrets)))
		assert.True(t, got["fresh"].Data.IsNull())
	})
}
