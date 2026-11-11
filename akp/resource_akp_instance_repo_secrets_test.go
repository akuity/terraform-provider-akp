//go:build !acc

package akp

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/akuity/terraform-provider-akp/akp/types"
)

var repoSecretsType = tftypes.MapType{ElemType: tftypes.StringType}

func repoSecrets(names ...string) tftypes.Map {
	elements := make(map[string]attr.Value, len(names))
	for _, name := range names {
		elements[name] = tftypes.MapValueMust(tftypes.StringType, map[string]attr.Value{
			"url": tftypes.StringValue("https://example.test/" + name),
		})
	}
	return tftypes.MapValueMust(repoSecretsType, elements)
}

func TestWithRemovedRepoCredentialSecrets(t *testing.T) {
	null := tftypes.MapNull(repoSecretsType)
	tests := map[string]struct {
		stateRepo, stateTemplate, planRepo, planTemplate tftypes.Map
		want                                             []string
	}{
		"removed repository and template entries": {
			stateRepo: repoSecrets("repo-a", "repo-b"), stateTemplate: repoSecrets("repo-t"),
			planRepo: repoSecrets("repo-a"), planTemplate: null,
			want: []string{"repo-b", "repo-t"},
		},
		"whole attribute removed": {
			stateRepo: repoSecrets("repo-a", "repo-b"), stateTemplate: null,
			planRepo: null, planTemplate: null,
			want: []string{"repo-a", "repo-b"},
		},
		"entry moved to the other map": {
			stateRepo: repoSecrets("repo-a"), stateTemplate: null,
			planRepo: null, planTemplate: repoSecrets("repo-a"),
		},
		"entries only added": {
			stateRepo: repoSecrets("repo-a"), stateTemplate: null,
			planRepo: repoSecrets("repo-a", "repo-new"), planTemplate: null,
		},
		"unknown plan": {
			stateRepo: repoSecrets("repo-a"), stateTemplate: null,
			planRepo: tftypes.MapUnknown(repoSecretsType), planTemplate: null,
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			state := &types.Instance{RepoCredentialSecrets: tc.stateRepo, RepoTemplateCredentialSecrets: tc.stateTemplate}
			plan := &types.Instance{RepoCredentialSecrets: tc.planRepo, RepoTemplateCredentialSecrets: tc.planTemplate}
			names, _ := withRemovedRepoCredentialSecrets(context.Background(), state, plan).Value(removedRepoCredentialSecretsKey{}).([]string)
			assert.Equal(t, tc.want, names)
		})
	}
}

func TestBuildApplyRequest_DeleteRepoCredentialSecretNames(t *testing.T) {
	state := &types.Instance{RepoCredentialSecrets: repoSecrets("repo-a", "repo-removed")}
	plan := &types.Instance{
		Name:                          tftypes.StringValue("example"),
		ArgoCD:                        &types.ArgoCD{},
		ArgoCDResources:               tftypes.MapNull(tftypes.StringType),
		RepoCredentialSecrets:         repoSecrets("repo-a"),
		RepoTemplateCredentialSecrets: tftypes.MapNull(repoSecretsType),
	}
	var diags diag.Diagnostics

	updateReq := buildApplyRequest(withRemovedRepoCredentialSecrets(context.Background(), state, plan), &diags, plan, "org", "ws")
	require.False(t, diags.HasError(), "%v", diags)
	assert.Equal(t, []string{"repo-removed"}, updateReq.GetDeleteRepoCredentialSecretNames())
	assert.Len(t, updateReq.GetRepoCredentialSecrets(), 1)

	createReq := buildApplyRequest(context.Background(), &diags, plan, "org", "ws")
	require.False(t, diags.HasError(), "%v", diags)
	assert.Empty(t, createReq.GetDeleteRepoCredentialSecretNames())
}
