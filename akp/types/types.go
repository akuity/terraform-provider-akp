package types

import (
	"context"
	"fmt"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"google.golang.org/protobuf/types/known/structpb"
	"k8s.io/apimachinery/pkg/api/resource"
	"sigs.k8s.io/yaml"

	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	"github.com/akuity/terraform-provider-akp/akp/apis/v1alpha1"
	"github.com/akuity/terraform-provider-akp/akp/marshal"
)

var (
	DirectClusterTypeString = map[argocdv1.DirectClusterType]string{
		argocdv1.DirectClusterType_DIRECT_CLUSTER_TYPE_KARGO: "kargo",
	}

	clusterSizeProtoToTF = map[string]string{
		"CLUSTER_SIZE_SMALL":       "small",
		"CLUSTER_SIZE_MEDIUM":      "medium",
		"CLUSTER_SIZE_LARGE":       "large",
		"CLUSTER_SIZE_AUTO":        "auto",
		"CLUSTER_SIZE_UNSPECIFIED": "unspecified",
	}

	directClusterTypeProtoToTF = map[string]string{
		"DIRECT_CLUSTER_TYPE_KARGO": "kargo",
	}
	OverridesMap = overrideMap{
		"spec.instance_spec.extensions":                                   emptyListIfSet(),
		"spec.instance_spec.cluster_customization_defaults.kustomization": yamlStringToObject(),
		"data.kustomization":                                              excludeField(),
		"data.custom_agent_size_config":                                   excludeField(),
		"data.size":                                                       stringWithMapping(map[string]string{"custom": "large"}),
		"data.maintenance_mode_expiry":                                    suppressEmptyString(),
	}

	RenamesMap = renameMap{
		"data.auto_agent_size_config": "autoscalerConfig",
		"data.auto_agent_size_config.application_controller.resource_minimum.memory": "mem",
		"data.auto_agent_size_config.application_controller.resource_maximum.memory": "mem",
		"data.auto_agent_size_config.repo_server.resource_minimum.memory":            "mem",
		"data.auto_agent_size_config.repo_server.resource_maximum.memory":            "mem",
		"data.auto_agent_size_config.repo_server.replicas_maximum":                   "replicaMaximum",
		"data.auto_agent_size_config.repo_server.replicas_minimum":                   "replicaMinimum",

		// The instance-level agent-size default reuses the same attribute shapes as
		// the per-cluster block above, so it needs the same renames. Without them
		// the TF names go to the API verbatim and UpdateInstance rejects the patch
		// with `unknown field "memory"`.
		"spec.instance_spec.cluster_customization_defaults.autoscaler_config.application_controller.resource_minimum.memory": "mem",
		"spec.instance_spec.cluster_customization_defaults.autoscaler_config.application_controller.resource_maximum.memory": "mem",
		"spec.instance_spec.cluster_customization_defaults.autoscaler_config.repo_server.resource_minimum.memory":            "mem",
		"spec.instance_spec.cluster_customization_defaults.autoscaler_config.repo_server.resource_maximum.memory":            "mem",
		"spec.instance_spec.cluster_customization_defaults.autoscaler_config.repo_server.replicas_minimum":                   "replicaMinimum",
		"spec.instance_spec.cluster_customization_defaults.autoscaler_config.repo_server.replicas_maximum":                   "replicaMaximum",
	}

	// ReverseOverridesMap defines custom API→TF conversion logic for ArgoCD and Cluster resources.
	// Keys are tfsdk tag paths (dot-separated for nesting).
	ReverseOverridesMap = reverseOverrideMap{
		// Kustomization is an object in the API but a YAML string in TF
		"spec.instance_spec.cluster_customization_defaults.kustomization": ObjectToYAMLString(),
		"data.kustomization": ObjectToYAMLString(),
		// TF-only fields with no API equivalent
		"data.custom_agent_size_config":     ExcludeFromAPI(),
		"remove_agent_resources_on_destroy": TFOnlyField(types.BoolValue(true)),
		"reapply_manifests_on_update":       TFOnlyField(types.BoolValue(false)),
		"ensure_healthy":                    TFOnlyField(types.BoolValue(false)),
		"namespace_scoped":                  HydrateFromAPIWhenPlanNull(),
		// Write-only secret field
		"spec.instance_spec.metrics_ingress_password_hash": PreserveFromPlan(),
		// The platform reports a disabled MCP server by omitting enabled; without this the
		// portal toggle turning it off never shows up as drift.
		"spec.instance_spec.mcp_server.enabled": EnabledFromAPIWhenConfigured(),
		// Enum fields: protojson outputs proto names (e.g., "CLUSTER_SIZE_SMALL"), TF expects lowercase
		"data.size":                             ProtoEnumToLowerString(clusterSizeProtoToTF),
		"data.direct_cluster_spec.cluster_type": ProtoEnumToLowerString(directClusterTypeProtoToTF),
		// Connectivity enum: normalize proto name to public/private, defaulting to public when unset
		"data.connectivity":               ConnectivityReverseOverride(),
		"spec.instance_spec.connectivity": ConnectivityReverseOverride(),
		// Customization defaults connectivity is an optional nested object: normalize the
		// enum name but decline when absent so the object is not materialized.
		"spec.instance_spec.cluster_customization_defaults.connectivity": ProtoEnumToLowerString(connectivityProtoToTF),
		// The instance-level default agent size has the same problem: the API
		// returns CLUSTER_SIZE_*, and the raw name would leak into a sensitive
		// block. "custom" is not a value here — the schema allows only
		// small/medium/large/auto, with Custom expressed as large plus patches.
		"spec.instance_spec.cluster_customization_defaults.size": ProtoEnumToLowerString(clusterSizeProtoToTF),
		// The server renders these through a %0.2fGi formatter, so a configured "4Gi"
		// returns as "4.00Gi"; inside the sensitive argocd block that surfaces as
		// "inconsistent values for sensitive attribute" naming no field at all.
		"spec.instance_spec.cluster_customization_defaults.autoscaler_config.application_controller.resource_minimum.cpu":    PreserveEquivalentQuantity(),
		"spec.instance_spec.cluster_customization_defaults.autoscaler_config.application_controller.resource_minimum.memory": PreserveEquivalentQuantity(),
		"spec.instance_spec.cluster_customization_defaults.autoscaler_config.application_controller.resource_maximum.cpu":    PreserveEquivalentQuantity(),
		"spec.instance_spec.cluster_customization_defaults.autoscaler_config.application_controller.resource_maximum.memory": PreserveEquivalentQuantity(),
		"spec.instance_spec.cluster_customization_defaults.autoscaler_config.repo_server.resource_minimum.cpu":               PreserveEquivalentQuantity(),
		"spec.instance_spec.cluster_customization_defaults.autoscaler_config.repo_server.resource_minimum.memory":            PreserveEquivalentQuantity(),
		"spec.instance_spec.cluster_customization_defaults.autoscaler_config.repo_server.resource_maximum.cpu":               PreserveEquivalentQuantity(),
		"spec.instance_spec.cluster_customization_defaults.autoscaler_config.repo_server.resource_maximum.memory":            PreserveEquivalentQuantity(),
	}

	// ReverseRenamesMap maps tfsdk tags to API camelCase keys for the reverse direction.
	// Uses the same mapping as RenamesMap since both map snake_case → camelCase.
	ReverseRenamesMap = RenamesMap
)

func (c *Cluster) Update(ctx context.Context, diagnostics *diag.Diagnostics, apiCluster *argocdv1.Cluster, plan *Cluster) {
	c.ID = types.StringValue(apiCluster.GetId())
	c.Name = types.StringValue(apiCluster.GetName())
	c.Namespace = types.StringValue(apiCluster.GetData().Namespace)
	labels, d := types.MapValueFrom(ctx, types.StringType, apiCluster.GetData().GetLabels())
	if d.HasError() {
		labels = types.MapNull(types.StringType)
	}
	diagnostics.Append(d...)
	annotations, d := types.MapValueFrom(ctx, types.StringType, apiCluster.GetData().GetAnnotations())
	if d.HasError() {
		annotations = types.MapNull(types.StringType)
	}
	diagnostics.Append(d...)
	c.Labels = labels
	c.Annotations = annotations

	if c.RemoveAgentResourcesOnDestroy.IsUnknown() || c.RemoveAgentResourcesOnDestroy.IsNull() {
		c.RemoveAgentResourcesOnDestroy = types.BoolValue(true)
	}
	if c.ReapplyManifestsOnUpdate.IsUnknown() || c.ReapplyManifestsOnUpdate.IsNull() {
		c.ReapplyManifestsOnUpdate = types.BoolValue(false)
	}
	if c.EnsureHealthy.IsUnknown() || c.EnsureHealthy.IsNull() {
		c.EnsureHealthy = types.BoolValue(false)
	}

	if c.Spec == nil {
		c.Spec = &ClusterSpec{}
	}
	apiMap, err := marshal.ProtoToMap(apiCluster)
	if err != nil {
		diagnostics.AddError("failed to marshal cluster proto to map", err.Error())
		return
	}

	var planArg *ClusterSpec
	var planData ClusterData
	if plan != nil {
		planArg = DeepCopy(plan.Spec)
		planData = planArg.Data
	}
	diagnostics.Append(BuildStateFromAPI(ctx, apiMap, c.Spec, planArg, ReverseOverridesMap, ReverseRenamesMap, "spec")...)

	var planForPostProcessing *Cluster
	if planArg != nil {
		planForPostProcessing = &Cluster{Spec: &ClusterSpec{Data: planData}}
	}
	c.Spec.Data.AutoscalerConfig = toAutoScalerConfigTFModel(planForPostProcessing, apiCluster.GetData().GetAutoscalerConfig())

	customConfig := inferCustomAgentSizeConfig(apiCluster, planData, diagnostics)
	c.Spec.Data.CustomAgentSizeConfig = customConfig
	if customConfig != nil {
		c.Spec.Data.Size = types.StringValue("custom")
	}

	if plan != nil && planData.Size.ValueString() == "custom" {
		switch {
		// An unset kustomization is planned unknown (UnknownWhenCustomSize) and the API
		// then returns only the patches generated from custom_agent_size_config. Those
		// are not user input: stored as such, the next apply would reject them as a
		// conflicting user patch. The equality check migrates states that stored them.
		case customConfig != nil && (planData.Kustomization.IsUnknown() || isGeneratedKustomization(planData.Kustomization.ValueString(), customConfig)):
			c.Spec.Data.Kustomization = types.StringNull()
		case !planData.Kustomization.IsUnknown():
			c.Spec.Data.Kustomization = planData.Kustomization
		}
	}
}

// isGeneratedKustomization reports whether kustomization holds exactly what
// GenerateExpectedKustomization derives from customConfig, i.e. no user content.
func isGeneratedKustomization(kustomization string, customConfig *CustomAgentSizeConfig) bool {
	generated, err := GenerateExpectedKustomization(customConfig, "")
	return err == nil && normalizedYAMLEqual(kustomization, generated)
}

func inferCustomAgentSizeConfig(apiCluster *argocdv1.Cluster, planData ClusterData, diagnostics *diag.Diagnostics) *CustomAgentSizeConfig {
	kustomization, err := clusterKustomizationYAML(apiCluster)
	if err != nil {
		diagnostics.AddError("getting cluster kustomization for custom size", err.Error())
		return nil
	}
	if kustomization == "" {
		return nil
	}

	if planData.Size.ValueString() != "custom" {
		return nil
	}

	if planData.CustomAgentSizeConfig != nil && planData.Size.ValueString() == "custom" {
		customKustomization, err := GenerateExpectedKustomization(planData.CustomAgentSizeConfig, "")
		if err != nil {
			diagnostics.AddError("failed to generate expected kustomization", err.Error())
			return nil
		}
		if isKustomizationSubset(customKustomization, kustomization) {
			return planData.CustomAgentSizeConfig
		}
	}

	customConfig, err := inferCustomAgentSizeConfigFromKustomization(kustomization)
	if err != nil {
		diagnostics.AddError("failed to infer custom cluster sizing from kustomization", err.Error())
		return nil
	}
	return customConfig
}

func clusterKustomizationYAML(apiCluster *argocdv1.Cluster) (string, error) {
	if apiCluster == nil || apiCluster.GetData() == nil || apiCluster.GetData().GetKustomization() == nil {
		return "", nil
	}

	jsonData, err := apiCluster.GetData().GetKustomization().MarshalJSON()
	if err != nil {
		return "", err
	}
	yamlData, err := yaml.JSONToYAML(jsonData)
	if err != nil {
		return "", err
	}
	return string(yamlData), nil
}

func inferCustomAgentSizeConfigFromKustomization(kustomization string) (*CustomAgentSizeConfig, error) {
	if kustomization == "" {
		return nil, nil
	}

	var config map[string]any
	if err := yaml.Unmarshal([]byte(kustomization), &config); err != nil {
		return nil, err
	}

	result := &CustomAgentSizeConfig{}
	if patches, ok := config["patches"].([]any); ok {
		for _, patchEntry := range patches {
			patchMap, ok := patchEntry.(map[string]any)
			if !ok {
				continue
			}
			target, ok := patchMap["target"].(map[string]any)
			if !ok {
				continue
			}
			name, _ := target["name"].(string)
			patchStr, _ := patchMap["patch"].(string)
			if patchStr == "" {
				continue
			}

			memory, cpu, ok := extractCustomAgentPatchResources(patchStr)
			if !ok {
				continue
			}

			switch name {
			case "argocd-application-controller":
				result.ApplicationController = &AppControllerCustomAgentSizeConfig{
					Memory: types.StringValue(memory),
					Cpu:    types.StringValue(cpu),
				}
			case "argocd-repo-server":
				if result.RepoServer == nil {
					result.RepoServer = &RepoServerCustomAgentSizeConfig{}
				}
				result.RepoServer.Memory = types.StringValue(memory)
				result.RepoServer.Cpu = types.StringValue(cpu)
			}
		}
	}

	if replicas, ok := config["replicas"].([]any); ok {
		for _, replicaEntry := range replicas {
			replicaMap, ok := replicaEntry.(map[string]any)
			if !ok {
				continue
			}
			name, _ := replicaMap["name"].(string)
			if name != "argocd-repo-server" {
				continue
			}
			count, ok := replicaMap["count"].(float64)
			if !ok {
				continue
			}
			if result.RepoServer == nil {
				result.RepoServer = &RepoServerCustomAgentSizeConfig{}
			}
			result.RepoServer.Replicas = types.Int64Value(int64(count))
		}
	}

	if result.ApplicationController == nil && result.RepoServer == nil {
		return nil, nil
	}

	return result, nil
}

func extractCustomAgentPatchResources(patch string) (memory, cpu string, ok bool) {
	for _, container := range patchContainers(patch) {
		resources, ok := container["resources"].(map[string]any)
		if !ok {
			continue
		}
		requests, _ := resources["requests"].(map[string]any)
		limits, _ := resources["limits"].(map[string]any)
		memory, _ = limits["memory"].(string)
		if memory == "" {
			memory, _ = requests["memory"].(string)
		}
		cpu, _ = requests["cpu"].(string)
		if memory != "" && cpu != "" {
			return memory, cpu, true
		}
	}

	return "", "", false
}

var (
	CMPOverridesMap = overrideMap{
		"enabled": excludeField(),
		"image":   excludeField(),
	}

	CMPReverseOverridesMap = reverseOverrideMap{}
	CMPRenamesMap          = renameMap{}
	CMPReverseRenamesMap   = CMPRenamesMap
)

func ToConfigManagementPluginsTFModel(ctx context.Context, diagnostics *diag.Diagnostics, cmps []*structpb.Struct, oldCMPs map[string]*ConfigManagementPlugin) map[string]*ConfigManagementPlugin {
	if len(cmps) == 0 && len(oldCMPs) == 0 {
		return oldCMPs
	}
	newCMPs := make(map[string]*ConfigManagementPlugin)
	for _, plugin := range cmps {
		apiMap := plugin.AsMap()

		name := ""
		if metadata, ok := apiMap["metadata"].(map[string]any); ok {
			name, _ = metadata["name"].(string)
			if annotations, ok := metadata["annotations"].(map[string]any); ok {
				if enabled, ok := annotations[v1alpha1.AnnotationCMPEnabled]; ok {
					apiMap["enabled"] = enabled == "true"
				}
				if image, ok := annotations[v1alpha1.AnnotationCMPImage]; ok {
					apiMap["image"] = image
				}
			}
		}

		cmp := &ConfigManagementPlugin{}
		var plan *ConfigManagementPlugin
		if old, ok := oldCMPs[name]; ok {
			plan = old
		}
		diagnostics.Append(BuildStateFromAPI(WithoutReadContext(ctx), apiMap, cmp, plan, CMPReverseOverridesMap, CMPReverseRenamesMap, "")...)
		newCMPs[name] = cmp
	}
	return newCMPs
}

func BuildCMPMap(cmp *ConfigManagementPlugin, name string) map[string]any {
	rawMap := TFToMapWithOverrides(cmp, CMPOverridesMap, CMPRenamesMap)
	if rawMap == nil {
		rawMap = make(map[string]any)
	}
	rawMap["kind"] = "ConfigManagementPlugin"
	rawMap["apiVersion"] = "argoproj.io/v1alpha1"
	rawMap["metadata"] = map[string]any{
		"name": name,
		"annotations": map[string]any{
			v1alpha1.AnnotationCMPImage:   cmp.Image.ValueString(),
			v1alpha1.AnnotationCMPEnabled: fmt.Sprintf("%t", cmp.Enabled.ValueBool()),
		},
	}
	return rawMap
}

// patchContainers returns the pod containers addressed by a Deployment patch.
func patchContainers(patch string) []map[string]any {
	var patchObj map[string]any
	if err := yaml.Unmarshal([]byte(patch), &patchObj); err != nil {
		return nil
	}
	spec, _ := patchObj["spec"].(map[string]any)
	template, _ := spec["template"].(map[string]any)
	templateSpec, _ := template["spec"].(map[string]any)
	containers, _ := templateSpec["containers"].([]any)
	result := make([]map[string]any, 0, len(containers))
	for _, container := range containers {
		if m, ok := container.(map[string]any); ok {
			result = append(result, m)
		}
	}
	return result
}

func isResourcePatch(patch map[string]any) bool {
	patchContent, ok := patch["patch"].(string)
	if !ok {
		return false
	}
	for _, container := range patchContainers(patchContent) {
		if _, hasResources := container["resources"]; hasResources {
			return true
		}
	}
	return false
}

func generateResourcePatch(name, memory, cpu string) string {
	return fmt.Sprintf(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: %s
spec:
  template:
    spec:
      containers:
        - name: %s
          resources:
            limits:
              memory: %s
            requests:
              cpu: %s
              memory: %s`, name, name, memory, cpu, memory)
}

func areResourcesEquivalent(plan, new string) bool {
	if plan == "" {
		return true
	}
	planQ, err1 := resource.ParseQuantity(plan)
	newQ, err2 := resource.ParseQuantity(new)
	if err1 != nil || err2 != nil {
		return plan == new
	}

	planVal := planQ.Value()
	newVal := newQ.Value()

	var diff float64
	if planVal > newVal {
		diff = float64(planVal-newVal) / float64(planVal)
	} else {
		diff = float64(newVal-planVal) / float64(newVal)
	}

	// there maybe Mi to Gi conversion with some rounding difference
	return diff <= 0.05
}

func GenerateExpectedKustomization(customConfig *CustomAgentSizeConfig, userKustomization string) (string, error) {
	if customConfig == nil {
		return userKustomization, nil
	}

	var expectedConfig map[string]any
	var userPatches []any
	var userReplicas []any

	if userKustomization != "" {
		if err := yaml.Unmarshal([]byte(userKustomization), &expectedConfig); err != nil {
			return "", fmt.Errorf("failed to parse user kustomization: %s", err.Error())
		}

		if patches, ok := expectedConfig["patches"].([]any); ok {
			userPatches = patches
		}
		if replicas, ok := expectedConfig["replicas"].([]any); ok {
			userReplicas = replicas
		}
	} else {
		expectedConfig = map[string]any{
			"apiVersion": "kustomize.config.k8s.io/v1beta1",
			"kind":       "Kustomization",
		}
	}

	for _, p := range userPatches {
		if patch, ok := p.(map[string]any); ok {
			if isResourcePatch(patch) {
				target := patch["target"].(map[string]any)
				name := target["name"].(string)
				if name == "argocd-application-controller" || name == "argocd-repo-server" {
					return "", fmt.Errorf("kustomization contains resource patches for %s, which conflicts with custom_agent_size_config. Please use only custom_agent_size_config for resource configuration or change the size to other value", name)
				}
			}
		}
	}

	customPatches := make([]map[string]any, 0)
	customReplicas := make([]map[string]any, 0)

	if customConfig.ApplicationController != nil {
		customPatches = append(customPatches, map[string]any{
			"patch": generateResourcePatch("argocd-application-controller", customConfig.ApplicationController.Memory.ValueString(), customConfig.ApplicationController.Cpu.ValueString()),
			"target": map[string]string{
				"kind": "Deployment",
				"name": "argocd-application-controller",
			},
		})
	}

	if customConfig.RepoServer != nil {
		customPatches = append(customPatches, map[string]any{
			"patch": generateResourcePatch("argocd-repo-server", customConfig.RepoServer.Memory.ValueString(), customConfig.RepoServer.Cpu.ValueString()),
			"target": map[string]string{
				"kind": "Deployment",
				"name": "argocd-repo-server",
			},
		})

		customReplicas = append(customReplicas, map[string]any{
			"count": customConfig.RepoServer.Replicas.ValueInt64(),
			"name":  "argocd-repo-server",
		})
	}

	allPatches := customPatches
	for _, p := range userPatches {
		allPatches = append(allPatches, p.(map[string]any))
	}

	allReplicas := customReplicas
	for _, r := range userReplicas {
		allReplicas = append(allReplicas, r.(map[string]any))
	}

	expectedConfig["patches"] = allPatches
	if len(allReplicas) > 0 {
		expectedConfig["replicas"] = allReplicas
	}

	yamlData, err := yaml.Marshal(expectedConfig)
	if err != nil {
		return "", fmt.Errorf("failed to marshal expected kustomization: %s", err.Error())
	}

	return string(yamlData), nil
}

// isKustomizationSubset checks if the API response is a subset of the expected kustomization
func isKustomizationSubset(subset, superset string) bool {
	var subsetObj, supersetObj map[string]any
	if err := yaml.Unmarshal([]byte(subset), &subsetObj); err != nil {
		return false
	}
	if err := yaml.Unmarshal([]byte(superset), &supersetObj); err != nil {
		return false
	}

	return isMapSubset(subsetObj, supersetObj)
}

func isMapSubset(subset, superset map[string]any) bool {
	for key, subValue := range subset {
		superValue, exists := superset[key]
		if !exists {
			return false
		}

		switch subVal := subValue.(type) {
		case map[string]any:
			if superVal, ok := superValue.(map[string]any); ok {
				if !isMapSubset(subVal, superVal) {
					return false
				}
			} else {
				return false
			}
		case []any:
			if superVal, ok := superValue.([]any); ok {
				if !isSliceSubset(subVal, superVal) {
					return false
				}
			} else {
				return false
			}
		default:
			if !reflect.DeepEqual(subValue, superValue) {
				return false
			}
		}
	}
	return true
}

func isSliceSubset(subset, superset []any) bool {
	for _, subItem := range subset {
		found := false
		for _, superItem := range superset {
			if reflect.DeepEqual(subItem, superItem) {
				found = true
				break
			}
			if subMap, ok := subItem.(map[string]any); ok {
				if superMap, ok := superItem.(map[string]any); ok {
					if isMapSubset(subMap, superMap) {
						found = true
						break
					}
				}
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// resourcesFromAPI converts API resources to the TF model, keeping the planned
// spelling of equivalent quantities (e.g. "1Gi" vs "1024Mi") to avoid spurious diffs.
func resourcesFromAPI(api *argocdv1.Resources, plan *Resources) *Resources {
	if api == nil {
		return nil
	}
	mem, cpu := api.Mem, api.Cpu
	if plan != nil {
		if areResourcesEquivalent(plan.Memory.ValueString(), mem) {
			mem = plan.Memory.ValueString()
		}
		if areResourcesEquivalent(plan.Cpu.ValueString(), cpu) {
			cpu = plan.Cpu.ValueString()
		}
	}
	return &Resources{Memory: types.StringValue(mem), Cpu: types.StringValue(cpu)}
}

func toAutoScalerConfigTFModel(plan *Cluster, apiConfig *argocdv1.AutoScalerConfig) *AutoScalerConfig {
	var planned *AutoScalerConfig
	if plan != nil && plan.Spec != nil {
		planned = plan.Spec.Data.AutoscalerConfig
		if planned == nil {
			return nil
		}
		// Only size "auto" takes its config from the API; other sizes keep the planned values.
		if plan.Spec.Data.Size.ValueString() != "auto" {
			return planned
		}
	}
	if apiConfig == nil {
		return nil
	}

	result := &AutoScalerConfig{}
	if ac := apiConfig.ApplicationController; ac != nil {
		var planMin, planMax *Resources
		if planned != nil && planned.ApplicationController != nil {
			planMin, planMax = planned.ApplicationController.ResourceMinimum, planned.ApplicationController.ResourceMaximum
		}
		result.ApplicationController = &AppControllerAutoScalingConfig{
			ResourceMinimum: resourcesFromAPI(ac.ResourceMinimum, planMin),
			ResourceMaximum: resourcesFromAPI(ac.ResourceMaximum, planMax),
		}
	}
	if rs := apiConfig.RepoServer; rs != nil {
		var planMin, planMax *Resources
		replicasMin, replicasMax := int64(rs.ReplicaMinimum), int64(rs.ReplicaMaximum)
		if planned != nil && planned.RepoServer != nil {
			planMin, planMax = planned.RepoServer.ResourceMinimum, planned.RepoServer.ResourceMaximum
			if !planned.RepoServer.ReplicasMinimum.IsNull() {
				replicasMin = planned.RepoServer.ReplicasMinimum.ValueInt64()
			}
			if !planned.RepoServer.ReplicasMaximum.IsNull() {
				replicasMax = planned.RepoServer.ReplicasMaximum.ValueInt64()
			}
		}
		result.RepoServer = &RepoServerAutoScalingConfig{
			ResourceMinimum: resourcesFromAPI(rs.ResourceMinimum, planMin),
			ResourceMaximum: resourcesFromAPI(rs.ResourceMaximum, planMax),
			ReplicasMinimum: types.Int64Value(replicasMin),
			ReplicasMaximum: types.Int64Value(replicasMax),
		}
	}
	return result
}
