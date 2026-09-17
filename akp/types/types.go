package types

import (
	"context"
	"fmt"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"google.golang.org/protobuf/types/known/structpb"
	yamlv3 "gopkg.in/yaml.v3"
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
		planArg = DeepCopyClusterSpec(plan.Spec)
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

	if plan != nil && planData.Size.ValueString() == "custom" && !planData.Kustomization.IsUnknown() {
		c.Spec.Data.Kustomization = planData.Kustomization
	}
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
	var patchMap map[string]any
	if err := yaml.Unmarshal([]byte(patch), &patchMap); err != nil {
		return "", "", false
	}

	spec, _ := patchMap["spec"].(map[string]any)
	template, _ := spec["template"].(map[string]any)
	templateSpec, _ := template["spec"].(map[string]any)
	containers, _ := templateSpec["containers"].([]any)
	for _, container := range containers {
		containerMap, ok := container.(map[string]any)
		if !ok {
			continue
		}
		resources, ok := containerMap["resources"].(map[string]any)
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

func isResourcePatch(patch map[string]any) bool {
	patchContent, ok := patch["patch"].(string)
	if !ok {
		return false
	}

	var patchObj map[string]any
	if err := yaml.Unmarshal([]byte(patchContent), &patchObj); err != nil {
		return false
	}

	spec, ok := patchObj["spec"].(map[string]any)
	if !ok {
		return false
	}

	template, ok := spec["template"].(map[string]any)
	if !ok {
		return false
	}

	templateSpec, ok := template["spec"].(map[string]any)
	if !ok {
		return false
	}

	containers, ok := templateSpec["containers"].([]any)
	if !ok {
		return false
	}

	for _, container := range containers {
		containerMap, ok := container.(map[string]any)
		if !ok {
			return false
		}
		_, hasResources := containerMap["resources"]
		if hasResources {
			return true
		}
	}
	return false
}

func generateAppControllerPatch(config *AppControllerCustomAgentSizeConfig) string {
	if config == nil {
		return ""
	}
	return fmt.Sprintf(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: argocd-application-controller
spec:
  template:
    spec:
      containers:
        - name: argocd-application-controller
          resources:
            limits:
              memory: %s
            requests:
              cpu: %s
              memory: %s`,
		config.Memory.ValueString(),
		config.Cpu.ValueString(),
		config.Memory.ValueString())
}

func generateRepoServerPatch(config *RepoServerCustomAgentSizeConfig) string {
	if config == nil {
		return ""
	}
	return fmt.Sprintf(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: argocd-repo-server
spec:
  template:
    spec:
      containers:
        - name: argocd-repo-server
          resources:
            limits:
              memory: %s
            requests:
              cpu: %s
              memory: %s`,
		config.Memory.ValueString(),
		config.Cpu.ValueString(),
		config.Memory.ValueString())
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

func yamlEqual(a, b string) bool {
	var objA, objB any
	if err := yamlv3.Unmarshal([]byte(a), &objA); err != nil {
		return false
	}
	if err := yamlv3.Unmarshal([]byte(b), &objB); err != nil {
		return false
	}
	return reflect.DeepEqual(objA, objB)
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
			"patch": generateAppControllerPatch(customConfig.ApplicationController),
			"target": map[string]string{
				"kind": "Deployment",
				"name": "argocd-application-controller",
			},
		})
	}

	if customConfig.RepoServer != nil {
		customPatches = append(customPatches, map[string]any{
			"patch": generateRepoServerPatch(customConfig.RepoServer),
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
	if yamlEqual(subset, superset) {
		return true
	}

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

func toAutoScalerConfigTFModel(plan *Cluster, apiConfig *argocdv1.AutoScalerConfig) *AutoScalerConfig {
	// If plan is nil, use API value
	if plan == nil || plan.Spec == nil {
		if apiConfig == nil {
			return nil
		}
		// Continue processing to return API config
	} else {
		// Get the size value
		sizeValue := ""
		if !plan.Spec.Data.Size.IsNull() {
			sizeValue = plan.Spec.Data.Size.ValueString()
		}

		// Check auto_agent_size_config status
		autoscalerIsNull := plan.Spec.Data.AutoscalerConfig == nil

		// Main logic: decide whether to return null or continue processing
		if sizeValue != "auto" {
			// For non-auto sizes, auto_agent_size_config should generally be null
			if autoscalerIsNull {
				// Not specified or explicitly null - return null
				return nil
			}
			// Has explicit non-null values - preserve the planned values instead of API values
			// This handles the case where we transition from "auto" to another size
			return plan.Spec.Data.AutoscalerConfig
		} else {
			// For auto size, show config from API if not explicitly set to null
			if autoscalerIsNull {
				return nil
			}
			// Continue processing to show API defaults or planned values
		}
	}

	// If the plan doesn't include auto scaler config but size is "auto", show API defaults
	if plan != nil && plan.Spec != nil && plan.Spec.Data.AutoscalerConfig == nil {
		// If size is "auto", we should show the API defaults even if not explicitly configured
		if !plan.Spec.Data.Size.IsNull() && plan.Spec.Data.Size.ValueString() == "auto" {
			// Show API defaults in state - don't return null
		} else {
			// For other sizes, don't show autoscaler config
			return nil
		}
	}

	if apiConfig == nil {
		return nil
	}

	// Get the planned auto scaler config to preserve original values when equivalent
	var plannedConfig *AutoScalerConfig
	if plan != nil && plan.Spec != nil && plan.Spec.Data.AutoscalerConfig != nil {
		plannedConfig = plan.Spec.Data.AutoscalerConfig
	}

	result := &AutoScalerConfig{}

	if apiConfig.ApplicationController != nil {
		appController := &AppControllerAutoScalingConfig{}

		if apiConfig.ApplicationController.ResourceMinimum != nil {
			// Use planned values if they are resource-equivalent to API values
			memoryValue := apiConfig.ApplicationController.ResourceMinimum.Mem
			cpuValue := apiConfig.ApplicationController.ResourceMinimum.Cpu

			if plannedConfig != nil && plannedConfig.ApplicationController != nil && plannedConfig.ApplicationController.ResourceMinimum != nil {
				if areResourcesEquivalent(plannedConfig.ApplicationController.ResourceMinimum.Memory.ValueString(), memoryValue) {
					memoryValue = plannedConfig.ApplicationController.ResourceMinimum.Memory.ValueString()
				}
				if areResourcesEquivalent(plannedConfig.ApplicationController.ResourceMinimum.Cpu.ValueString(), cpuValue) {
					cpuValue = plannedConfig.ApplicationController.ResourceMinimum.Cpu.ValueString()
				}
			}

			appController.ResourceMinimum = &Resources{
				Memory: types.StringValue(memoryValue),
				Cpu:    types.StringValue(cpuValue),
			}
		}

		if apiConfig.ApplicationController.ResourceMaximum != nil {
			// Use planned values if they are resource-equivalent to API values
			memoryValue := apiConfig.ApplicationController.ResourceMaximum.Mem
			cpuValue := apiConfig.ApplicationController.ResourceMaximum.Cpu

			if plannedConfig != nil && plannedConfig.ApplicationController != nil && plannedConfig.ApplicationController.ResourceMaximum != nil {
				if areResourcesEquivalent(plannedConfig.ApplicationController.ResourceMaximum.Memory.ValueString(), memoryValue) {
					memoryValue = plannedConfig.ApplicationController.ResourceMaximum.Memory.ValueString()
				}
				if areResourcesEquivalent(plannedConfig.ApplicationController.ResourceMaximum.Cpu.ValueString(), cpuValue) {
					cpuValue = plannedConfig.ApplicationController.ResourceMaximum.Cpu.ValueString()
				}
			}

			appController.ResourceMaximum = &Resources{
				Memory: types.StringValue(memoryValue),
				Cpu:    types.StringValue(cpuValue),
			}
		}

		result.ApplicationController = appController
	}

	if apiConfig.RepoServer != nil {
		repoServer := &RepoServerAutoScalingConfig{}

		if apiConfig.RepoServer.ResourceMinimum != nil {
			// Use planned values if they are resource-equivalent to API values
			memoryValue := apiConfig.RepoServer.ResourceMinimum.Mem
			cpuValue := apiConfig.RepoServer.ResourceMinimum.Cpu

			if plannedConfig != nil && plannedConfig.RepoServer != nil && plannedConfig.RepoServer.ResourceMinimum != nil {
				if areResourcesEquivalent(plannedConfig.RepoServer.ResourceMinimum.Memory.ValueString(), memoryValue) {
					memoryValue = plannedConfig.RepoServer.ResourceMinimum.Memory.ValueString()
				}
				if areResourcesEquivalent(plannedConfig.RepoServer.ResourceMinimum.Cpu.ValueString(), cpuValue) {
					cpuValue = plannedConfig.RepoServer.ResourceMinimum.Cpu.ValueString()
				}
			}

			repoServer.ResourceMinimum = &Resources{
				Memory: types.StringValue(memoryValue),
				Cpu:    types.StringValue(cpuValue),
			}
		}

		if apiConfig.RepoServer.ResourceMaximum != nil {
			// Use planned values if they are resource-equivalent to API values
			memoryValue := apiConfig.RepoServer.ResourceMaximum.Mem
			cpuValue := apiConfig.RepoServer.ResourceMaximum.Cpu

			if plannedConfig != nil && plannedConfig.RepoServer != nil && plannedConfig.RepoServer.ResourceMaximum != nil {
				if areResourcesEquivalent(plannedConfig.RepoServer.ResourceMaximum.Memory.ValueString(), memoryValue) {
					memoryValue = plannedConfig.RepoServer.ResourceMaximum.Memory.ValueString()
				}
				if areResourcesEquivalent(plannedConfig.RepoServer.ResourceMaximum.Cpu.ValueString(), cpuValue) {
					cpuValue = plannedConfig.RepoServer.ResourceMaximum.Cpu.ValueString()
				}
			}

			repoServer.ResourceMaximum = &Resources{
				Memory: types.StringValue(memoryValue),
				Cpu:    types.StringValue(cpuValue),
			}
		}

		// For replica values, always use planned values if available, otherwise API values
		replicasMin := int64(apiConfig.RepoServer.ReplicaMinimum)
		replicasMax := int64(apiConfig.RepoServer.ReplicaMaximum)
		if plannedConfig != nil && plannedConfig.RepoServer != nil {
			if !plannedConfig.RepoServer.ReplicasMinimum.IsNull() {
				replicasMin = plannedConfig.RepoServer.ReplicasMinimum.ValueInt64()
			}
			if !plannedConfig.RepoServer.ReplicasMaximum.IsNull() {
				replicasMax = plannedConfig.RepoServer.ReplicasMaximum.ValueInt64()
			}
		}

		repoServer.ReplicasMinimum = types.Int64Value(replicasMin)
		repoServer.ReplicasMaximum = types.Int64Value(replicasMax)

		result.RepoServer = repoServer
	}

	return result
}

// preserveInstanceAutoscalerPlanQuantities keeps the operator's own spelling of
// the instance-level auto limits when the API returns an equivalent quantity.
//
// The server stores these as resource.Quantity and renders them back through a
// %0.2fGi formatter, so a configured "4Gi" comes back as "4.00Gi". Both mean the
// same thing, but Terraform compares strings — and since the whole argocd block
// is one sensitive attribute, the mismatch surfaces as "inconsistent values for
// sensitive attribute" naming no field at all.
//
// This is a post-processing pass rather than a ReverseFieldOverride because
// cluster_customization_defaults is a dynamic types.Object and buildTFObject
// only resolves overrides for an Object's direct children (see its
// "deep nested overrides" TODO); these quantities sit three levels deeper. The
// Cluster resource's equivalent limits are handled in toAutoScalerConfigTFModel,
// which only has the Cluster plan.
//
// A genuinely different quantity is left untouched so real drift stays visible.
func preserveInstanceAutoscalerPlanQuantities(state, plan *ArgoCD) {
	if state == nil || plan == nil {
		return
	}
	stateCCD := state.Spec.InstanceSpec.ClusterCustomizationDefaults
	planCCD := plan.Spec.InstanceSpec.ClusterCustomizationDefaults
	if stateCCD.IsNull() || stateCCD.IsUnknown() || planCCD.IsNull() || planCCD.IsUnknown() {
		return
	}

	stateCfg, ok := objectAttr(stateCCD, "autoscaler_config")
	if !ok {
		return
	}
	planCfg, ok := objectAttr(planCCD, "autoscaler_config")
	if !ok {
		return
	}

	rebuiltCfg, changed := preserveWorkloadQuantities(stateCfg, planCfg)
	if !changed {
		return
	}

	attrs := stateCCD.Attributes()
	attrs["autoscaler_config"] = rebuiltCfg
	rebuilt, diags := types.ObjectValue(stateCCD.AttributeTypes(context.Background()), attrs)
	if diags.HasError() {
		return
	}
	state.Spec.InstanceSpec.ClusterCustomizationDefaults = rebuilt
}

// preserveWorkloadQuantities walks autoscaler_config's workload objects
// (application_controller, repo_server) and their resource_minimum /
// resource_maximum pairs, swapping any state quantity for the planned spelling
// when the two are equivalent.
func preserveWorkloadQuantities(stateCfg, planCfg types.Object) (types.Object, bool) {
	changed := false
	cfgAttrs := stateCfg.Attributes()

	for _, workload := range []string{"application_controller", "repo_server"} {
		stateWL, ok := objectAttr(stateCfg, workload)
		if !ok {
			continue
		}
		planWL, ok := objectAttr(planCfg, workload)
		if !ok {
			continue
		}

		wlAttrs := stateWL.Attributes()
		wlChanged := false
		for _, bound := range []string{"resource_minimum", "resource_maximum"} {
			stateRes, ok := objectAttr(stateWL, bound)
			if !ok {
				continue
			}
			planRes, ok := objectAttr(planWL, bound)
			if !ok {
				continue
			}

			resAttrs := stateRes.Attributes()
			resChanged := false
			for name, stateVal := range resAttrs {
				stateStr, ok := stateVal.(types.String)
				if !ok {
					continue
				}
				planStr, ok := planRes.Attributes()[name].(types.String)
				if !ok || planStr.IsNull() || planStr.IsUnknown() || planStr.ValueString() == "" {
					continue
				}
				if planStr.ValueString() == stateStr.ValueString() ||
					!areResourcesEquivalent(planStr.ValueString(), stateStr.ValueString()) {
					continue
				}
				resAttrs[name] = planStr
				resChanged = true
			}
			if !resChanged {
				continue
			}
			rebuiltRes, diags := types.ObjectValue(stateRes.AttributeTypes(context.Background()), resAttrs)
			if diags.HasError() {
				continue
			}
			wlAttrs[bound] = rebuiltRes
			wlChanged = true
		}
		if !wlChanged {
			continue
		}
		rebuiltWL, diags := types.ObjectValue(stateWL.AttributeTypes(context.Background()), wlAttrs)
		if diags.HasError() {
			continue
		}
		cfgAttrs[workload] = rebuiltWL
		changed = true
	}

	if !changed {
		return stateCfg, false
	}
	rebuilt, diags := types.ObjectValue(stateCfg.AttributeTypes(context.Background()), cfgAttrs)
	if diags.HasError() {
		return stateCfg, false
	}
	return rebuilt, true
}

// objectAttr reads a nested types.Object attribute, reporting whether it is
// present and usable.
func objectAttr(parent types.Object, name string) (types.Object, bool) {
	val, ok := parent.Attributes()[name]
	if !ok {
		return types.Object{}, false
	}
	obj, ok := val.(types.Object)
	if !ok || obj.IsNull() || obj.IsUnknown() {
		return types.Object{}, false
	}
	return obj, true
}
