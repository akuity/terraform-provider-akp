package types

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
	"sigs.k8s.io/yaml"

	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
)

func TestGenerateExpectedKustomization_MergesCustomAndUser(t *testing.T) {
	custom := &CustomAgentSizeConfig{
		ApplicationController: &AppControllerCustomAgentSizeConfig{
			Cpu:    tftypes.StringValue("1000m"),
			Memory: tftypes.StringValue("2Gi"),
		},
		RepoServer: &RepoServerCustomAgentSizeConfig{
			Cpu:      tftypes.StringValue("1000m"),
			Memory:   tftypes.StringValue("0.5Gi"),
			Replicas: tftypes.Int64Value(5),
		},
	}

	user := `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
patches:
  - patch: |
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: argocd-repo-server
      spec:
        template:
          spec:
            nodeSelector:
              argocd: "true"
            tolerations:
            - key: argocd
              operator: Exists
              effect: "NoSchedule"
    target:
      kind: Deployment
  - patch: |
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: argocd-repo-server
      spec:
        template:
          spec:
            containers:
              - name: argocd-repo-server
                env:
                  - name: TEST_ENV_VAR
                    value: 100
    target:
      kind: Deployment
      name: argocd-repo-server
`

	out, err := GenerateExpectedKustomization(custom, user)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Ensure the user content is a subset of the generated content
	if !isKustomizationSubset(user, out) {
		t.Fatalf("expected user kustomization to be subset of output")
	}

	// Ensure custom patches and replicas are present
	customOnly := map[string]any{
		"patches": []any{
			map[string]any{
				"patch":  generateResourcePatch("argocd-application-controller", custom.ApplicationController.Memory.ValueString(), custom.ApplicationController.Cpu.ValueString()),
				"target": map[string]string{"kind": "Deployment", "name": "argocd-application-controller"},
			},
			map[string]any{
				"patch":  generateResourcePatch("argocd-repo-server", custom.RepoServer.Memory.ValueString(), custom.RepoServer.Cpu.ValueString()),
				"target": map[string]string{"kind": "Deployment", "name": "argocd-repo-server"},
			},
		},
		"replicas": []any{
			map[string]any{"count": int64(5), "name": "argocd-repo-server"},
		},
	}
	customOnlyYAML, _ := yaml.Marshal(customOnly)
	if !isKustomizationSubset(string(customOnlyYAML), out) {
		t.Fatalf("expected custom patches/replicas to be included in output")
	}
}

func TestGenerateExpectedKustomization_ResourcePatchConflict_RepoServer(t *testing.T) {
	custom := &CustomAgentSizeConfig{
		RepoServer: &RepoServerCustomAgentSizeConfig{
			Cpu:      tftypes.StringValue("1000m"),
			Memory:   tftypes.StringValue("0.5Gi"),
			Replicas: tftypes.Int64Value(3),
		},
	}

	// User patch contains resources for repo server which conflicts
	user := `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
patches:
  - patch: |
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: argocd-repo-server
      spec:
        template:
          spec:
            containers:
              - name: argocd-repo-server
                resources:
                  requests:
                    cpu: 500m
                    memory: 256Mi
    target:
      kind: Deployment
      name: argocd-repo-server
`

	_, err := GenerateExpectedKustomization(custom, user)
	if err == nil {
		t.Fatalf("expected conflict error, got nil")
	}
}

func TestGenerateExpectedKustomization_ResourcePatchConflict_AppController(t *testing.T) {
	custom := &CustomAgentSizeConfig{
		ApplicationController: &AppControllerCustomAgentSizeConfig{
			Cpu:    tftypes.StringValue("1000m"),
			Memory: tftypes.StringValue("2Gi"),
		},
	}

	user := `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
patches:
  - patch: |
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: argocd-application-controller
      spec:
        template:
          spec:
            containers:
              - name: argocd-application-controller
                resources:
                  requests:
                    cpu: 500m
                    memory: 256Mi
    target:
      kind: Deployment
      name: argocd-application-controller
`

	_, err := GenerateExpectedKustomization(custom, user)
	if err == nil {
		t.Fatalf("expected conflict error, got nil")
	}
}

func TestGenerateExpectedKustomization_CustomNil_ReturnsUser(t *testing.T) {
	user := `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
patches: []
`
	out, err := GenerateExpectedKustomization(nil, user)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !normalizedYAMLEqual(user, out) {
		t.Fatalf("expected output to equal user input")
	}
}

func TestGenerateExpectedKustomization_EmptyUser_GeneratesDefaults(t *testing.T) {
	custom := &CustomAgentSizeConfig{
		RepoServer: &RepoServerCustomAgentSizeConfig{
			Cpu:      tftypes.StringValue("1000m"),
			Memory:   tftypes.StringValue("0.5Gi"),
			Replicas: tftypes.Int64Value(2),
		},
	}
	out, err := GenerateExpectedKustomization(custom, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var obj map[string]any
	if err := yaml.Unmarshal([]byte(out), &obj); err != nil {
		t.Fatalf("failed to unmarshal output: %v", err)
	}
	if obj["apiVersion"] != "kustomize.config.k8s.io/v1beta1" || obj["kind"] != "Kustomization" {
		t.Fatalf("expected default apiVersion/kind, got: %v", obj)
	}
	// Ensure replicas include the repo server count
	replicas, ok := obj["replicas"].([]any)
	if !ok || len(replicas) != 1 {
		t.Fatalf("expected one replicas entry")
	}
}

func TestIsKustomizationSubset(t *testing.T) {
	superset := `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
patches:
  - patch: |
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: a
    target:
      kind: Deployment
      name: a
  - patch: |
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: b
    target:
      kind: Deployment
      name: b
replicas:
  - name: argocd-repo-server
    count: 2
`
	subset := `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
patches:
  - patch: |
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: a
    target:
      kind: Deployment
      name: a
`
	if !isKustomizationSubset(subset, superset) {
		t.Fatalf("expected subset to be true")
	}

	notSubset := `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
patches:
  - patch: |
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: c
    target:
      kind: Deployment
      name: c
`
	if isKustomizationSubset(notSubset, superset) {
		t.Fatalf("expected not subset to be false")
	}
}

func TestGenerateExpectedKustomization_ResourcePatch_OtherDeployment_NoConflict(t *testing.T) {
	custom := &CustomAgentSizeConfig{
		RepoServer: &RepoServerCustomAgentSizeConfig{
			Cpu:      tftypes.StringValue("1000m"),
			Memory:   tftypes.StringValue("0.5Gi"),
			Replicas: tftypes.Int64Value(1),
		},
	}

	user := `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
patches:
  - patch: |
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: not-argocd
      spec:
        template:
          spec:
            containers:
              - name: foo
                resources:
                  limits:
                    memory: 512Mi
                  requests:
                    cpu: 200m
                    memory: 256Mi
    target:
      kind: Deployment
      name: not-argocd
`

	out, err := GenerateExpectedKustomization(custom, user)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !isKustomizationSubset(user, out) {
		t.Fatalf("expected user resources patch to be preserved in output")
	}
}

func TestIsKustomizationSubset_WithResourcesPatch_ExactMatch(t *testing.T) {
	patch := `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
patches:
  - patch: |
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: not-argocd
      spec:
        template:
          spec:
            containers:
              - name: foo
                resources:
                  limits:
                    memory: 512Mi
                  requests:
                    cpu: 200m
                    memory: 256Mi
    target:
      kind: Deployment
      name: not-argocd
`
	superset := patch
	subset := patch
	if !isKustomizationSubset(subset, superset) {
		t.Fatalf("expected exact same resources patch to be subset")
	}
}

// A custom-size cluster's kustomization in state must hold only what the operator
// wrote. The API echoes the patches generated from custom_agent_size_config; feeding
// those back as user input made the next apply reject them as conflicting patches.
func TestClusterUpdate_CustomSizeKustomizationIsNotUserInput(t *testing.T) {
	custom := &CustomAgentSizeConfig{
		ApplicationController: &AppControllerCustomAgentSizeConfig{Cpu: tftypes.StringValue("1000m"), Memory: tftypes.StringValue("2Gi")},
		RepoServer:            &RepoServerCustomAgentSizeConfig{Cpu: tftypes.StringValue("2000m"), Memory: tftypes.StringValue("4Gi"), Replicas: tftypes.Int64Value(3)},
	}
	user := "apiVersion: kustomize.config.k8s.io/v1beta1\nkind: Kustomization\npatches:\n- patch: |\n    apiVersion: apps/v1\n    kind: Deployment\n    metadata:\n      name: argocd-repo-server\n    spec:\n      template:\n        spec:\n          nodeSelector:\n            argocd: \"true\"\n  target:\n    kind: Deployment\n"
	generated, err := GenerateExpectedKustomization(custom, "")
	require.NoError(t, err)
	withUser, err := GenerateExpectedKustomization(custom, user)
	require.NoError(t, err)

	apiCluster := func(kustomization string) *argocdv1.Cluster {
		var m map[string]any
		require.NoError(t, yaml.Unmarshal([]byte(kustomization), &m))
		k, err := structpb.NewStruct(m)
		require.NoError(t, err)
		return &argocdv1.Cluster{Id: "id", Name: "custom", Data: &argocdv1.ClusterData{Namespace: "argocd", Kustomization: k}}
	}

	for name, tc := range map[string]struct {
		plan     tftypes.String
		api      string
		expected tftypes.String
	}{
		"unset kustomization is planned unknown":     {plan: tftypes.StringUnknown(), api: generated, expected: tftypes.StringNull()},
		"state written before the fix holds patches": {plan: tftypes.StringValue(generated), api: generated, expected: tftypes.StringNull()},
		"operator kustomization is kept":             {plan: tftypes.StringValue(user), api: withUser, expected: tftypes.StringValue(user)},
	} {
		t.Run(name, func(t *testing.T) {
			plan := &Cluster{Spec: &ClusterSpec{Data: ClusterData{
				Size: tftypes.StringValue("custom"), CustomAgentSizeConfig: custom, Kustomization: tc.plan,
			}}}
			state := &Cluster{}
			var diags diag.Diagnostics
			state.Update(context.Background(), &diags, apiCluster(tc.api), plan)
			require.False(t, diags.HasError(), "%v", diags)
			require.Equal(t, "custom", state.Spec.Data.Size.ValueString())
			require.Equal(t, tc.expected, state.Spec.Data.Kustomization)
		})
	}

	t.Run("unknown plan never reaches state when the API returns no patches", func(t *testing.T) {
		plan := &Cluster{Spec: &ClusterSpec{Data: ClusterData{
			Size: tftypes.StringValue("custom"), CustomAgentSizeConfig: custom, Kustomization: tftypes.StringUnknown(),
		}}}
		state := &Cluster{}
		var diags diag.Diagnostics
		state.Update(context.Background(), &diags, &argocdv1.Cluster{Id: "id", Name: "custom", Data: &argocdv1.ClusterData{Namespace: "argocd"}}, plan)
		require.False(t, diags.HasError(), "%v", diags)
		require.False(t, state.Spec.Data.Kustomization.IsUnknown())
	})
}
