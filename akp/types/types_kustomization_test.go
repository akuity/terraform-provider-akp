package types

import (
	"testing"

	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	"sigs.k8s.io/yaml"
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

	out, err := generateExpectedKustomization(custom, user)
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
				"patch":  generateAppControllerPatch(custom.ApplicationController),
				"target": map[string]string{"kind": "Deployment", "name": "argocd-application-controller"},
			},
			map[string]any{
				"patch":  generateRepoServerPatch(custom.RepoServer),
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

	_, err := generateExpectedKustomization(custom, user)
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

	_, err := generateExpectedKustomization(custom, user)
	if err == nil {
		t.Fatalf("expected conflict error, got nil")
	}
}

func TestGenerateExpectedKustomization_CustomNil_ReturnsUser(t *testing.T) {
	user := `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
patches: []
`
	out, err := generateExpectedKustomization(nil, user)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !yamlEqual(user, out) {
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
	out, err := generateExpectedKustomization(custom, "")
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

	out, err := generateExpectedKustomization(custom, user)
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
