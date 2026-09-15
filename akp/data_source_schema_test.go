//go:build !acc

package akp

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	kargov1 "github.com/akuity/api-client-go/pkg/api/gen/kargo/v1"
	"github.com/akuity/terraform-provider-akp/akp/types"
)

// Every data source schema must be valid, with all attributes Computed except
// its lookup keys. TestDataSourceWholeObjectOutput covers the omitted secrets.
func TestDataSourceSchemas(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		ds       datasource.DataSource
		required []string
	}{
		{&AkpInstanceDataSource{}, []string{"name"}},
		{&AkpClusterDataSource{}, []string{"instance_id", "name"}},
		{&AkpClustersDataSource{}, []string{"instance_id"}},
		{&AkpKargoDataSource{}, []string{"name"}},
		{&AkpKargoAgentDataSource{}, []string{"instance_id", "name"}},
		{&AkpKargoAgentsDataSource{}, []string{"instance_id"}},
	} {
		var resp datasource.SchemaResponse
		tc.ds.Schema(ctx, datasource.SchemaRequest{}, &resp)
		require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics)
		require.Empty(t, resp.Schema.ValidateImplementation(ctx).Errors())
		for name, a := range resp.Schema.Attributes {
			required := false
			for _, r := range tc.required {
				required = required || r == name
			}
			require.Equal(t, required, a.IsRequired(), "%T %s", tc.ds, name)
			require.Equal(t, !required, a.IsComputed(), "%T %s", tc.ds, name)
		}
	}
	var instanceSchema datasource.SchemaResponse
	(&AkpInstanceDataSource{}).Schema(ctx, datasource.SchemaRequest{}, &instanceSchema)
	instance := instanceSchema.Schema.Attributes
	spec := instance["argocd"].(schema.SingleNestedAttribute).Attributes["spec"].(schema.SingleNestedAttribute).Attributes["instance_spec"].(schema.SingleNestedAttribute)
	require.NotEmpty(t, spec.Attributes["assistant_extension_enabled"].GetDeprecationMessage())

	// Cluster data sources keep their existing sensitive kubeconfig fields.
	for _, ds := range []datasource.DataSource{&AkpClusterDataSource{}, &AkpKargoAgentDataSource{}} {
		var resp datasource.SchemaResponse
		ds.Schema(ctx, datasource.SchemaRequest{}, &resp)
		kubeconfig := resp.Schema.Attributes["kube_config"].(schema.SingleNestedAttribute)
		for _, name := range []string{"password", "client_key", "token"} {
			require.True(t, kubeconfig.Attributes[name].IsSensitive())
		}
	}
	require.Equal(t, reflect.TypeFor[types.ManagedSecretDataSource]().NumField(), len(getManagedSecretDataSourceAttributes()))
}

func TestInstanceDataSourceModelRoundTrip(t *testing.T) {
	ctx := context.Background()
	config := dataSourceConfig(t, &AkpInstanceDataSource{})
	var data types.InstanceDataSource
	require.Empty(t, config.Get(ctx, &data))
	instance := &types.Instance{Name: data.Name}
	argocd, err := structpb.NewStruct(map[string]any{
		"spec": map[string]any{
			"version": "v3.0.0",
			"instanceSpec": map[string]any{
				"subdomain":                  "example",
				"metricsIngressUsername":     "metrics-user",
				"metricsIngressPasswordHash": "secret-hash",
			},
		},
	})
	require.NoError(t, err)
	var diags diag.Diagnostics
	require.NoError(t, instance.Update(ctx, &diags, &argocdv1.ExportInstanceResponse{Argocd: argocd}, true))
	require.Empty(t, diags)
	managedSecrets := types.ToManagedSecretsDataSourceModel(ctx, &diags, []*argocdv1.ManagedSecret{
		{
			Name:            "registry",
			SecretKeys:      []string{"token"},
			ClusterSelector: &argocdv1.ObjectSelector{MatchLabels: map[string]string{"env": "prod"}},
		},
	})
	require.Empty(t, diags)
	data = types.NewInstanceDataSourceModel(instance, managedSecrets)
	state := tfsdk.State{Schema: config.Schema}
	require.Empty(t, state.Set(ctx, &data))
	var result types.InstanceDataSource
	require.Empty(t, state.Get(ctx, &result))
	require.Equal(t, "example", result.Name.ValueString())
	require.Equal(t, "v3.0.0", result.ArgoCD.Spec.Version.ValueString())
	require.Equal(t, "metrics-user", result.ArgoCD.Spec.InstanceSpec.MetricsIngressUsername.ValueString())
	require.Equal(t, "env=prod", result.ManagedSecrets["registry"].ClusterSelector.ValueString())
	var secretKeys []string
	require.Empty(t, result.ManagedSecrets["registry"].SecretKeys.ElementsAs(ctx, &secretKeys, false))
	require.Equal(t, []string{"token"}, secretKeys)
}

func TestKargoDataSourceModelRoundTrip(t *testing.T) {
	ctx := context.Background()
	config := dataSourceConfig(t, &AkpKargoDataSource{})
	var data types.KargoInstanceDataSource
	require.Empty(t, config.Get(ctx, &data))
	instance := &types.KargoInstance{Name: data.Name}
	kargo, err := structpb.NewStruct(map[string]any{
		"spec": map[string]any{
			"version":           "v1.7.0",
			"kargoInstanceSpec": map[string]any{},
			"oidcConfig": map[string]any{
				"enabled":         true,
				"clientId":        "example-client",
				"dexConfigSecret": map[string]any{"token": map[string]any{"value": "secret-token"}},
				"adminAccount": map[string]any{
					"claims": map[string]any{"groups": map[string]any{"values": []any{"admins"}}},
				},
			},
		},
	})
	require.NoError(t, err)
	var diags diag.Diagnostics
	require.NoError(t, instance.Update(ctx, &diags, &kargov1.ExportKargoInstanceResponse{Kargo: kargo}, true))
	require.Empty(t, diags)
	data = types.NewKargoInstanceDataSourceModel(instance)
	state := tfsdk.State{Schema: config.Schema}
	require.Empty(t, state.Set(ctx, &data))
	var result types.KargoInstanceDataSource
	require.Empty(t, state.Get(ctx, &result))
	require.Equal(t, "example", result.Name.ValueString())
	require.Equal(t, "v1.7.0", result.Kargo.Spec.Version.ValueString())
	require.True(t, result.Kargo.Spec.OidcConfig.Enabled.ValueBool())
	require.Equal(t, "example-client", result.Kargo.Spec.OidcConfig.ClientID.ValueString())
	require.Contains(t, result.Kargo.Spec.OidcConfig.AdminAccount.Claims.Elements(), "groups")
}

func dataSourceConfig(t *testing.T, ds datasource.DataSource) tfsdk.Config {
	t.Helper()
	ctx := context.Background()
	var resp datasource.SchemaResponse
	ds.Schema(ctx, datasource.SchemaRequest{}, &resp)
	require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics)
	typ := resp.Schema.Type().TerraformType(ctx)
	values := make(map[string]tftypes.Value)
	for name, typ := range typ.(tftypes.Object).AttributeTypes {
		values[name] = tftypes.NewValue(typ, nil)
	}
	values["name"] = tftypes.NewValue(tftypes.String, "example")
	return tfsdk.Config{Schema: resp.Schema, Raw: tftypes.NewValue(typ, values)}
}
