//go:build !acc

package types

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
)

func TestToManagedSecretUpsertAPIModel(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics

	labels, d := types.MapValueFrom(ctx, types.StringType, map[string]string{"team": "platform"})
	require.False(t, d.HasError())
	allowedClusters, d := types.ListValueFrom(ctx, types.StringType, []string{"cluster-a", "cluster-b"})
	require.False(t, d.HasError())
	data, d := types.MapValueFrom(ctx, types.StringType, map[string]string{"token": "s3cr3t"})
	require.False(t, d.HasError())

	upsert := ToManagedSecretUpsertAPIModel(ctx, &diags, "my-secret", &ManagedSecret{
		Labels:          labels,
		AllowedClusters: allowedClusters,
		ClusterSelector: types.StringValue("env=prod"),
		Data:            data,
		DataVersion:     types.StringValue("1"),
	})
	require.False(t, diags.HasError())
	require.NotNil(t, upsert)

	assert.Equal(t, "my-secret", upsert.Secret.GetName())
	assert.Equal(t, "platform", upsert.Secret.GetLabels()["team"])
	assert.Equal(t, []string{"cluster-a", "cluster-b"}, upsert.Secret.GetAllowedClusters())
	assert.Equal(t, map[string]string{"env": "prod"}, upsert.Secret.GetClusterSelector().GetMatchLabels())
	assert.Equal(t, map[string]string{"token": "s3cr3t"}, upsert.Data)
	assert.False(t, upsert.ClearData)
}

func TestToManagedSecretUpsertAPIModelMinimal(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics

	upsert := ToManagedSecretUpsertAPIModel(ctx, &diags, "my-secret", &ManagedSecret{
		Labels:          types.MapNull(types.StringType),
		AllowedClusters: types.ListNull(types.StringType),
		ClusterSelector: types.StringNull(),
		Data:            types.MapNull(types.StringType),
		DataVersion:     types.StringNull(),
	})
	require.False(t, diags.HasError())
	require.NotNil(t, upsert)

	assert.Equal(t, "my-secret", upsert.Secret.GetName())
	assert.Empty(t, upsert.Secret.GetLabels())
	assert.Empty(t, upsert.Secret.GetAllowedClusters())
	assert.Nil(t, upsert.Secret.GetClusterSelector())
	assert.Nil(t, upsert.Data)
	assert.False(t, upsert.ClearData)
}

func TestToManagedSecretUpsertAPIModelEmptyDataClears(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics

	data, d := types.MapValueFrom(ctx, types.StringType, map[string]string{})
	require.False(t, d.HasError())

	upsert := ToManagedSecretUpsertAPIModel(ctx, &diags, "my-secret", &ManagedSecret{
		Labels:          types.MapNull(types.StringType),
		AllowedClusters: types.ListNull(types.StringType),
		ClusterSelector: types.StringNull(),
		Data:            data,
		DataVersion:     types.StringValue("2"),
	})
	require.False(t, diags.HasError())
	require.NotNil(t, upsert)
	assert.True(t, upsert.ClearData)
	assert.Empty(t, upsert.Data)
}

func TestObjectSelectorFromString(t *testing.T) {
	selector, err := objectSelectorFromString("env=prod,tier in (web,api),!legacy")
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"env": "prod"}, selector.GetMatchLabels())
	require.Len(t, selector.GetMatchExpressions(), 2)

	_, err = objectSelectorFromString("env in (prod")
	require.Error(t, err)
}

func TestToManagedSecretUpsertAPIModelInvalidSelector(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics

	upsert := ToManagedSecretUpsertAPIModel(ctx, &diags, "my-secret", &ManagedSecret{
		Labels:          types.MapNull(types.StringType),
		AllowedClusters: types.ListNull(types.StringType),
		ClusterSelector: types.StringValue("env in (prod"),
		Data:            types.MapNull(types.StringType),
		DataVersion:     types.StringNull(),
	})
	require.True(t, diags.HasError())
	assert.Nil(t, upsert)
}

func TestToManagedSecretsTFModel(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics

	oldLabels, d := types.MapValueFrom(ctx, types.StringType, map[string]string{"team": "old"})
	require.False(t, d.HasError())
	oldAllowedClusters, d := types.ListValueFrom(ctx, types.StringType, []string{"old-cluster"})
	require.False(t, d.HasError())
	oldData, d := types.MapValueFrom(ctx, types.StringType, map[string]string{"token": "s3cr3t"})
	require.False(t, d.HasError())
	prior := map[string]*ManagedSecret{
		"configured": {
			Labels:          oldLabels,
			AllowedClusters: oldAllowedClusters,
			ClusterSelector: types.StringValue("env=old"),
			Data:            oldData,
			DataVersion:     types.StringValue("7"),
		},
		"deleted": {
			Labels:          types.MapNull(types.StringType),
			AllowedClusters: types.ListNull(types.StringType),
			ClusterSelector: types.StringNull(),
			Data:            types.MapNull(types.StringType),
			DataVersion:     types.StringValue("1"),
		},
	}
	key := "tier"
	operator := "NotIn"
	actual := []*argocdv1.ManagedSecret{
		{
			Name:            "configured",
			Labels:          map[string]string{"team": "new"},
			AllowedClusters: []string{"ALL"},
			ClusterSelector: &argocdv1.ObjectSelector{
				MatchLabels: map[string]string{"env": "prod"},
				MatchExpressions: []*argocdv1.LabelSelectorRequirement{
					{Key: &key, Operator: &operator, Values: []string{"dev"}},
				},
			},
		},
		{Name: "unconfigured", Labels: map[string]string{"ignored": "true"}},
	}

	result := ToManagedSecretsTFModel(ctx, &diags, prior, actual)
	require.False(t, diags.HasError(), "%v", diags)
	require.Len(t, result, 1)
	assert.NotContains(t, result, "deleted")
	assert.NotContains(t, result, "unconfigured")
	secret := result["configured"]
	require.NotNil(t, secret)
	var labels map[string]string
	diags.Append(secret.Labels.ElementsAs(ctx, &labels, true)...)
	assert.Equal(t, map[string]string{"team": "new"}, labels)
	var allowedClusters []string
	diags.Append(secret.AllowedClusters.ElementsAs(ctx, &allowedClusters, true)...)
	assert.Equal(t, []string{"ALL"}, allowedClusters)
	assert.Equal(t, "env=prod,tier notin (dev)", secret.ClusterSelector.ValueString())
	assert.True(t, secret.Data.IsNull())
	assert.Equal(t, "7", secret.DataVersion.ValueString())
	assert.False(t, diags.HasError(), "%v", diags)
}

func TestToManagedSecretsTFModelPreservesNullAndEmpty(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	emptyLabels, d := types.MapValueFrom(ctx, types.StringType, map[string]string{})
	require.False(t, d.HasError())
	emptyAllowedClusters, d := types.ListValueFrom(ctx, types.StringType, []string{})
	require.False(t, d.HasError())
	prior := map[string]*ManagedSecret{
		"null": {
			Labels:          types.MapNull(types.StringType),
			AllowedClusters: types.ListNull(types.StringType),
			ClusterSelector: types.StringNull(),
		},
		"empty": {
			Labels:          emptyLabels,
			AllowedClusters: emptyAllowedClusters,
			ClusterSelector: types.StringValue(""),
		},
	}
	actual := []*argocdv1.ManagedSecret{{Name: "null"}, {Name: "empty"}}

	result := ToManagedSecretsTFModel(ctx, &diags, prior, actual)
	require.False(t, diags.HasError(), "%v", diags)
	assert.True(t, result["null"].Labels.IsNull())
	assert.True(t, result["null"].AllowedClusters.IsNull())
	assert.True(t, result["null"].ClusterSelector.IsNull())
	assert.False(t, result["empty"].Labels.IsNull())
	assert.Empty(t, result["empty"].Labels.Elements())
	assert.False(t, result["empty"].AllowedClusters.IsNull())
	assert.Empty(t, result["empty"].AllowedClusters.Elements())
	assert.False(t, result["empty"].ClusterSelector.IsNull())
	assert.Empty(t, result["empty"].ClusterSelector.ValueString())
}

func TestToManagedSecretsDataSourceModel(t *testing.T) {
	ctx := context.Background()
	var diags diag.Diagnostics
	key := "tier"
	operator := "NotIn"

	result := ToManagedSecretsDataSourceModel(ctx, &diags, []*argocdv1.ManagedSecret{
		{
			Name:            "configured",
			Labels:          map[string]string{"team": "platform"},
			AllowedClusters: []string{"ALL"},
			ClusterSelector: &argocdv1.ObjectSelector{
				MatchLabels: map[string]string{"env": "prod"},
				MatchExpressions: []*argocdv1.LabelSelectorRequirement{
					{Key: &key, Operator: &operator, Values: []string{"dev"}},
				},
			},
			SecretKeys: []string{"password", "token"},
		},
		nil,
	})
	require.False(t, diags.HasError(), "%v", diags)
	require.Len(t, result, 1)
	secret := result["configured"]
	require.NotNil(t, secret)
	var labels map[string]string
	diags.Append(secret.Labels.ElementsAs(ctx, &labels, true)...)
	assert.Equal(t, map[string]string{"team": "platform"}, labels)
	var allowedClusters []string
	diags.Append(secret.AllowedClusters.ElementsAs(ctx, &allowedClusters, true)...)
	assert.Equal(t, []string{"ALL"}, allowedClusters)
	assert.Equal(t, "env=prod,tier notin (dev)", secret.ClusterSelector.ValueString())
	var secretKeys []string
	diags.Append(secret.SecretKeys.ElementsAs(ctx, &secretKeys, true)...)
	assert.Equal(t, []string{"password", "token"}, secretKeys)
	assert.False(t, diags.HasError(), "%v", diags)
}
