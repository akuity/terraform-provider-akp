package types

import (
	"context"
	"fmt"
	"reflect"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"google.golang.org/protobuf/types/known/structpb"
	yamlv3 "gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"

	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	"github.com/akuity/terraform-provider-akp/akp/apis/v1alpha1"
	"github.com/akuity/terraform-provider-akp/akp/marshal"
)

var (
	clusterCustomizationAttrTypes = map[string]attr.Type{
		"auto_upgrade_disabled":    types.BoolType,
		"kustomization":            types.StringType,
		"app_replication":          types.BoolType,
		"redis_tunneling":          types.BoolType,
		"server_side_diff_enabled": types.BoolType,
	}

	appsetPolicyAttrTypes = map[string]attr.Type{
		"policy":          types.StringType,
		"override_policy": types.BoolType,
	}

	resourcesAttrTypes = map[string]attr.Type{
		"memory": types.StringType,
		"cpu":    types.StringType,
	}

	appControllerAutoScalingAttrTypes = map[string]attr.Type{
		"resource_minimum": types.ObjectType{AttrTypes: resourcesAttrTypes},
		"resource_maximum": types.ObjectType{AttrTypes: resourcesAttrTypes},
	}

	repoServerAutoScalingAttrTypes = map[string]attr.Type{
		"resource_minimum": types.ObjectType{AttrTypes: resourcesAttrTypes},
		"resource_maximum": types.ObjectType{AttrTypes: resourcesAttrTypes},
		"replicas_maximum": types.Int64Type,
		"replicas_minimum": types.Int64Type,
	}

	autoScalerConfigAttrTypes = map[string]attr.Type{
		"application_controller": types.ObjectType{AttrTypes: appControllerAutoScalingAttrTypes},
		"repo_server":            types.ObjectType{AttrTypes: repoServerAutoScalingAttrTypes},
	}

	ClusterSizeString = map[argocdv1.ClusterSize]string{
		argocdv1.ClusterSize_CLUSTER_SIZE_SMALL:       "small",
		argocdv1.ClusterSize_CLUSTER_SIZE_MEDIUM:      "medium",
		argocdv1.ClusterSize_CLUSTER_SIZE_LARGE:       "large",
		argocdv1.ClusterSize_CLUSTER_SIZE_AUTO:        "auto",
		argocdv1.ClusterSize_CLUSTER_SIZE_UNSPECIFIED: "unspecified",
	}

	DirectClusterTypeString = map[argocdv1.DirectClusterType]string{
		argocdv1.DirectClusterType_DIRECT_CLUSTER_TYPE_KARGO: "kargo",
	}
)

func (a *ArgoCD) Update(ctx context.Context, diagnostics *diag.Diagnostics, cd *v1alpha1.ArgoCD) {
	declarativeManagementEnabled := cd.Spec.InstanceSpec.DeclarativeManagementEnabled != nil && *cd.Spec.InstanceSpec.DeclarativeManagementEnabled

	imageUpdaterEnabled := cd.Spec.InstanceSpec.ImageUpdaterEnabled != nil && *cd.Spec.InstanceSpec.ImageUpdaterEnabled

	backendIpAllowListEnabled := cd.Spec.InstanceSpec.BackendIpAllowListEnabled != nil && *cd.Spec.InstanceSpec.BackendIpAllowListEnabled

	auditExtensionEnabled := cd.Spec.InstanceSpec.AuditExtensionEnabled != nil && *cd.Spec.InstanceSpec.AuditExtensionEnabled

	syncHistoryExtensionEnabled := cd.Spec.InstanceSpec.SyncHistoryExtensionEnabled != nil && *cd.Spec.InstanceSpec.SyncHistoryExtensionEnabled

	assistantExtensionEnabled := cd.Spec.InstanceSpec.AssistantExtensionEnabled != nil && *cd.Spec.InstanceSpec.AssistantExtensionEnabled

	fqdn := ""
	if cd.Spec.InstanceSpec.Fqdn != nil {
		fqdn = *cd.Spec.InstanceSpec.Fqdn
	}
	multiClusterK8SDashboardEnabled := false
	if cd.Spec.InstanceSpec.MultiClusterK8SDashboardEnabled != nil {
		multiClusterK8SDashboardEnabled = *cd.Spec.InstanceSpec.MultiClusterK8SDashboardEnabled
	}

	var appInAnyNamespaceConfig *AppInAnyNamespaceConfig
	if a.Spec.InstanceSpec.AppInAnyNamespaceConfig != nil &&
		!a.Spec.InstanceSpec.AppInAnyNamespaceConfig.Enabled.ValueBool() &&
		(cd.Spec.InstanceSpec.AppInAnyNamespaceConfig == nil ||
			cd.Spec.InstanceSpec.AppInAnyNamespaceConfig.Enabled == nil ||
			!*cd.Spec.InstanceSpec.AppInAnyNamespaceConfig.Enabled) {
		appInAnyNamespaceConfig = a.Spec.InstanceSpec.AppInAnyNamespaceConfig
	} else {
		appInAnyNamespaceConfig = toAppInAnyNamespaceConfigTFModel(cd.Spec.InstanceSpec.AppInAnyNamespaceConfig)
	}

	var applicationSetExtension *ApplicationSetExtension
	if a.Spec.InstanceSpec.ApplicationSetExtension != nil &&
		!a.Spec.InstanceSpec.ApplicationSetExtension.Enabled.ValueBool() &&
		(cd.Spec.InstanceSpec.ApplicationSetExtension == nil ||
			cd.Spec.InstanceSpec.ApplicationSetExtension.Enabled == nil ||
			!*cd.Spec.InstanceSpec.ApplicationSetExtension.Enabled) {
		applicationSetExtension = a.Spec.InstanceSpec.ApplicationSetExtension
	} else {
		applicationSetExtension = toApplicationSetExtensionTFModel(cd.Spec.InstanceSpec.ApplicationSetExtension)
	}

	metricsIngressUsername := types.StringNull()
	if cd.Spec.InstanceSpec.MetricsIngressUsername != nil {
		metricsIngressUsername = types.StringValue(*cd.Spec.InstanceSpec.MetricsIngressUsername)
	}
	metricsIngressPasswordHash := types.StringNull()
	if cd.Spec.InstanceSpec.MetricsIngressPasswordHash != nil {
		metricsIngressPasswordHash = types.StringValue(*cd.Spec.InstanceSpec.MetricsIngressPasswordHash)
	}
	privilegedNotificationCluster := types.StringNull()
	if cd.Spec.InstanceSpec.PrivilegedNotificationCluster != nil {
		privilegedNotificationCluster = types.StringValue(*cd.Spec.InstanceSpec.PrivilegedNotificationCluster)
	}

	a.Spec = ArgoCDSpec{
		Description: types.StringValue(cd.Spec.Description),
		Version:     types.StringValue(cd.Spec.Version),
		InstanceSpec: InstanceSpec{
			IpAllowList:                     toIPAllowListTFModel(cd.Spec.InstanceSpec.IpAllowList),
			Subdomain:                       types.StringValue(cd.Spec.InstanceSpec.Subdomain),
			DeclarativeManagementEnabled:    types.BoolValue(declarativeManagementEnabled),
			Extensions:                      toExtensionsTFModel(cd.Spec.InstanceSpec.Extensions),
			ClusterCustomizationDefaults:    a.toClusterCustomizationTFModel(ctx, diagnostics, cd.Spec.InstanceSpec.ClusterCustomizationDefaults),
			ImageUpdaterEnabled:             types.BoolValue(imageUpdaterEnabled),
			BackendIpAllowListEnabled:       types.BoolValue(backendIpAllowListEnabled),
			RepoServerDelegate:              toRepoServerDelegateTFModel(cd.Spec.InstanceSpec.RepoServerDelegate),
			AuditExtensionEnabled:           types.BoolValue(auditExtensionEnabled),
			SyncHistoryExtensionEnabled:     types.BoolValue(syncHistoryExtensionEnabled),
			CrossplaneExtension:             toCrossplaneExtensionTFModel(cd.Spec.InstanceSpec.CrossplaneExtension),
			ImageUpdaterDelegate:            toImageUpdaterDelegateTFModel(cd.Spec.InstanceSpec.ImageUpdaterDelegate),
			AppSetDelegate:                  toAppSetDelegateTFModel(cd.Spec.InstanceSpec.AppSetDelegate),
			AssistantExtensionEnabled:       types.BoolValue(assistantExtensionEnabled),
			AppsetPolicy:                    toAppsetPolicyTFModel(ctx, diagnostics, cd.Spec.InstanceSpec.AppsetPolicy),
			HostAliases:                     toHostAliasesTFModel(cd.Spec.InstanceSpec.HostAliases),
			AgentPermissionsRules:           toAgentPermissionsRulesTFModel(cd.Spec.InstanceSpec.AgentPermissionsRules),
			Fqdn:                            types.StringValue(fqdn),
			MultiClusterK8SDashboardEnabled: types.BoolValue(multiClusterK8SDashboardEnabled),
			AkuityIntelligenceExtension:     toAkuityIntelligenceExtensionTFModel(cd.Spec.InstanceSpec.AkuityIntelligenceExtension, a),
			KubeVisionConfig:                toKubeVisionConfigTFModel(cd.Spec.InstanceSpec.KubeVisionConfig, a),
			AppInAnyNamespaceConfig:         appInAnyNamespaceConfig,
			AppsetPlugins:                   toAppsetPluginsTFModel(cd.Spec.InstanceSpec.AppsetPlugins),
			ApplicationSetExtension:         applicationSetExtension,
			MetricsIngressUsername:          metricsIngressUsername,
			MetricsIngressPasswordHash:      metricsIngressPasswordHash,
			PrivilegedNotificationCluster:   privilegedNotificationCluster,
			ClusterAddonsExtension:          toClusterAddonsExtensionTFModel(cd.Spec.InstanceSpec.ClusterAddonsExtension, a),
		},
	}
}

func (a *ArgoCD) ToArgoCDAPIModel(ctx context.Context, diag *diag.Diagnostics, name string) *v1alpha1.ArgoCD {
	return &v1alpha1.ArgoCD{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ArgoCD",
			APIVersion: "argocd.akuity.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1alpha1.ArgoCDSpec{
			Description: a.Spec.Description.ValueString(),
			Version:     a.Spec.Version.ValueString(),
			InstanceSpec: v1alpha1.InstanceSpec{
				IpAllowList:                     toIPAllowListAPIModel(a.Spec.InstanceSpec.IpAllowList),
				Subdomain:                       a.Spec.InstanceSpec.Subdomain.ValueString(),
				DeclarativeManagementEnabled:    toBoolPointer(a.Spec.InstanceSpec.DeclarativeManagementEnabled),
				Extensions:                      toExtensionsAPIModel(a.Spec.InstanceSpec.Extensions),
				ClusterCustomizationDefaults:    toClusterCustomizationAPIModel(ctx, diag, a.Spec.InstanceSpec.ClusterCustomizationDefaults),
				ImageUpdaterEnabled:             toBoolPointer(a.Spec.InstanceSpec.ImageUpdaterEnabled),
				BackendIpAllowListEnabled:       toBoolPointer(a.Spec.InstanceSpec.BackendIpAllowListEnabled),
				RepoServerDelegate:              toRepoServerDelegateAPIModel(a.Spec.InstanceSpec.RepoServerDelegate),
				AuditExtensionEnabled:           toBoolPointer(a.Spec.InstanceSpec.AuditExtensionEnabled),
				SyncHistoryExtensionEnabled:     toBoolPointer(a.Spec.InstanceSpec.SyncHistoryExtensionEnabled),
				CrossplaneExtension:             toCrossplaneExtensionAPIModel(a.Spec.InstanceSpec.CrossplaneExtension),
				ImageUpdaterDelegate:            toImageUpdaterDelegateAPIModel(a.Spec.InstanceSpec.ImageUpdaterDelegate),
				AppSetDelegate:                  toAppSetDelegateAPIModel(a.Spec.InstanceSpec.AppSetDelegate),
				AssistantExtensionEnabled:       toBoolPointer(a.Spec.InstanceSpec.AssistantExtensionEnabled),
				AppsetPolicy:                    toAppsetPolicyAPIModel(ctx, diag, a.Spec.InstanceSpec.AppsetPolicy),
				HostAliases:                     toHostAliasesAPIModel(a.Spec.InstanceSpec.HostAliases),
				AgentPermissionsRules:           toAgentPermissionsRuleAPIModel(a.Spec.InstanceSpec.AgentPermissionsRules),
				Fqdn:                            a.Spec.InstanceSpec.Fqdn.ValueStringPointer(),
				MultiClusterK8SDashboardEnabled: toBoolPointer(a.Spec.InstanceSpec.MultiClusterK8SDashboardEnabled),
				AkuityIntelligenceExtension:     toAkuityIntelligenceExtensionAPIModel(a.Spec.InstanceSpec.AkuityIntelligenceExtension),
				KubeVisionConfig:                toKubeVisionConfigAPIModel(a.Spec.InstanceSpec.KubeVisionConfig),
				AppInAnyNamespaceConfig:         toAppInAnyNamespaceConfigAPIModel(a.Spec.InstanceSpec.AppInAnyNamespaceConfig),
				AppsetPlugins:                   toAppsetPluginsAPIModel(a.Spec.InstanceSpec.AppsetPlugins),
				ApplicationSetExtension:         toApplicationSetExtensionAPIModel(a.Spec.InstanceSpec.ApplicationSetExtension),
				MetricsIngressUsername:          a.Spec.InstanceSpec.MetricsIngressUsername.ValueStringPointer(),
				MetricsIngressPasswordHash:      a.Spec.InstanceSpec.MetricsIngressPasswordHash.ValueStringPointer(),
				PrivilegedNotificationCluster:   a.Spec.InstanceSpec.PrivilegedNotificationCluster.ValueStringPointer(),
				ClusterAddonsExtension:          toClusterAddonsExtensionAPIModel(a.Spec.InstanceSpec.ClusterAddonsExtension),
			},
		},
	}
}

func toBoolPointer(b types.Bool) *bool {
	if b.IsUnknown() {
		return nil
	}
	return b.ValueBoolPointer()
}

func (c *Cluster) Update(ctx context.Context, diagnostics *diag.Diagnostics, apiCluster *argocdv1.Cluster, plan *Cluster) {
	c.ID = types.StringValue(apiCluster.GetId())
	c.Name = types.StringValue(apiCluster.GetName())
	c.Namespace = types.StringValue(apiCluster.GetData().Namespace)
	if c.RemoveAgentResourcesOnDestroy.IsUnknown() || c.RemoveAgentResourcesOnDestroy.IsNull() {
		c.RemoveAgentResourcesOnDestroy = types.BoolValue(true)
	}
	if c.ReapplyManifestsOnUpdate.IsUnknown() || c.ReapplyManifestsOnUpdate.IsNull() {
		c.ReapplyManifestsOnUpdate = types.BoolValue(false)
	} else {
		c.ReapplyManifestsOnUpdate = plan.ReapplyManifestsOnUpdate
	}
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
	jsonData, err := apiCluster.GetData().GetKustomization().MarshalJSON()
	if err != nil {
		diagnostics.AddError("getting cluster kustomization", err.Error())
	}
	yamlData, err := yaml.JSONToYAML(jsonData)
	if err != nil {
		diagnostics.AddError("getting cluster kustomization", err.Error())
	}

	var kustomization types.String
	if plan != nil && plan.Spec != nil && !plan.Spec.Data.Kustomization.IsNull() && !plan.Spec.Data.Kustomization.IsUnknown() {
		yamlMap := map[string]any{}
		if err := yaml.Unmarshal(yamlData, &yamlMap); err != nil {
			diagnostics.AddError("failed to unmarshal plan kustomization", err.Error())
		}
		// Ensure the kustomization is valid, api only returns patches and replicas.
		yamlMap["apiVersion"] = "kustomize.config.k8s.io/v1beta1"
		yamlMap["kind"] = "Kustomization"
		yamlData, err = yaml.Marshal(yamlMap)
		if err != nil {
			diagnostics.AddError("failed to marshal yaml map", err.Error())
		}
		if isKustomizationSubset(plan.Spec.Data.Kustomization.ValueString(), string(yamlData)) {
			kustomization = plan.Spec.Data.Kustomization
		}
	} else if apiCluster.GetData().GetKustomization() != nil {
		// When no kustomization is specified in the plan, set it to obtained value from the API
		kustomization = types.StringValue(string(yamlData))
	} else {
		kustomization = types.StringNull()
	}

	var size types.String
	var customConfig *CustomAgentSizeConfig
	if plan != nil && plan.Spec != nil && plan.Spec.Data.CustomAgentSizeConfig != nil && plan.Spec.Data.Size.ValueString() == "custom" {
		size = plan.Spec.Data.Size
		customKustomization, err := generateExpectedKustomization(plan.Spec.Data.CustomAgentSizeConfig, "")
		if err != nil {
			diagnostics.AddError("failed to generate expected kustomization", err.Error())
		} else {
			if isKustomizationSubset(customKustomization, string(yamlData)) {
				customConfig = plan.Spec.Data.CustomAgentSizeConfig
				size = types.StringValue("custom")
			} else {
				size = types.StringValue(ClusterSizeString[apiCluster.GetData().GetSize()])
			}
		}
	} else {
		size = types.StringValue(ClusterSizeString[apiCluster.GetData().GetSize()])
	}

	c.Labels = labels
	c.Annotations = annotations

	var directClusterSpec *DirectClusterSpec
	if plan != nil && plan.Spec != nil && plan.Spec.Data.DirectClusterSpec != nil {
		clusterType := DirectClusterTypeString[apiCluster.GetData().DirectClusterSpec.GetClusterType()]
		if clusterType == DirectClusterTypeString[argocdv1.DirectClusterType_DIRECT_CLUSTER_TYPE_KARGO] {
			directClusterSpec = &DirectClusterSpec{
				ClusterType:     types.StringValue(clusterType),
				KargoInstanceId: types.StringValue(apiCluster.GetData().DirectClusterSpec.GetKargoInstanceId()),
			}
		}
	}

	// Handle EksAddonEnabled field
	var eksAddonEnabled types.Bool
	if plan != nil && plan.Spec != nil && !plan.Spec.Data.EksAddonEnabled.IsUnknown() {
		// If the plan has a value (including null), preserve the plan value to avoid inconsistency
		eksAddonEnabled = plan.Spec.Data.EksAddonEnabled
	} else {
		// Otherwise, use the API value
		eksAddonEnabled = types.BoolValue(apiCluster.GetData().GetEksAddonEnabled())
	}

	// Handle DatadogAnnotationsEnabled field
	var datadogAnnotationsEnabled types.Bool
	if plan != nil && plan.Spec != nil && !plan.Spec.Data.DatadogAnnotationsEnabled.IsUnknown() {
		// If the plan has a value (including null), preserve the plan value to avoid inconsistency
		datadogAnnotationsEnabled = plan.Spec.Data.DatadogAnnotationsEnabled
	} else {
		// Otherwise, use the API value
		datadogAnnotationsEnabled = types.BoolValue(apiCluster.GetData().GetDatadogAnnotationsEnabled())
	}

	var namespaceScoped types.Bool
	if plan != nil && plan.Spec != nil && !plan.Spec.NamespaceScoped.IsUnknown() {
		namespaceScoped = plan.Spec.NamespaceScoped
	} else {
		namespaceScoped = types.BoolValue(apiCluster.GetNamespaceScoped()) //nolint:staticcheck
	}

	/*
		var serverSideDiffEnabled types.Bool
		if plan != nil && plan.Spec != nil && !plan.Spec.Data.ServerSideDiffEnabled.IsUnknown() {
			serverSideDiffEnabled = plan.Spec.Data.ServerSideDiffEnabled
		} else {
			serverSideDiffEnabled = types.BoolValue(apiCluster.GetData().GetServerSideDiffEnabled())
		}

	*/

	c.Spec = &ClusterSpec{
		NamespaceScoped: namespaceScoped,
		Data: ClusterData{
			Size:                            size,
			AutoUpgradeDisabled:             types.BoolValue(apiCluster.GetData().GetAutoUpgradeDisabled()),
			Kustomization:                   kustomization,
			AppReplication:                  types.BoolValue(apiCluster.GetData().GetAppReplication()),
			TargetVersion:                   types.StringValue(apiCluster.GetData().GetTargetVersion()),
			RedisTunneling:                  types.BoolValue(apiCluster.GetData().GetRedisTunneling()),
			EksAddonEnabled:                 eksAddonEnabled,
			DatadogAnnotationsEnabled:       datadogAnnotationsEnabled,
			ManagedClusterConfig:            toManagedClusterConfigTFModel(apiCluster.GetData().GetManagedClusterConfig()),
			MultiClusterK8SDashboardEnabled: types.BoolValue(apiCluster.GetData().GetMultiClusterK8SDashboardEnabled()),
			AutoscalerConfig:                toAutoScalerConfigTFModel(plan, apiCluster.GetData().GetAutoscalerConfig()),
			CustomAgentSizeConfig:           customConfig,
			Compatibility:                   toCompatibilityTFModel(plan, apiCluster.GetData().GetCompatibility()),
			ArgocdNotificationsSettings:     toArgoCDNotificationsSettingsTFModel(plan, apiCluster.GetData().GetArgocdNotificationsSettings()),
			DirectClusterSpec:               directClusterSpec,
			// ServerSideDiffEnabled:           serverSideDiffEnabled,
		},
	}

	if apiCluster.GetDescription() != "" {
		c.Spec.Description = types.StringValue(apiCluster.GetDescription())
	}

	if apiCluster.GetData().GetProject() != "" {
		c.Spec.Data.Project = types.StringValue(apiCluster.GetData().GetProject())
	}
}

func (c *Cluster) ToClusterAPIModel(ctx context.Context, diagnostics *diag.Diagnostics) *v1alpha1.Cluster {
	var labels map[string]string
	var annotations map[string]string
	diagnostics.Append(c.Labels.ElementsAs(ctx, &labels, true)...)
	diagnostics.Append(c.Annotations.ElementsAs(ctx, &annotations, true)...)
	return &v1alpha1.Cluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Cluster",
			APIVersion: "argocd.akuity.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        c.Name.ValueString(),
			Namespace:   c.Namespace.ValueString(),
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: v1alpha1.ClusterSpec{
			Description:     c.Spec.Description.ValueString(),
			NamespaceScoped: c.Spec.NamespaceScoped.ValueBool(),
			Data:            toClusterDataAPIModel(diagnostics, c.Spec.Data),
		},
	}
}

func (c *ConfigManagementPlugin) Update(ctx context.Context, diagnostics *diag.Diagnostics, cmp *v1alpha1.ConfigManagementPlugin) {
	version := types.StringNull()
	if cmp.Spec.Version != "" {
		version = types.StringValue(cmp.Spec.Version)
	}
	c.Enabled = types.BoolValue(cmp.Annotations[v1alpha1.AnnotationCMPEnabled] == "true")
	c.Image = types.StringValue(cmp.Annotations[v1alpha1.AnnotationCMPImage])
	c.Spec = &PluginSpec{
		Version:          version,
		Init:             toCommandTFModel(cmp.Spec.Init),
		Generate:         toCommandTFModel(cmp.Spec.Generate),
		Discover:         toDiscoverTFModel(cmp.Spec.Discover),
		Parameters:       toParametersTFModel(ctx, diagnostics, cmp.Spec.Parameters),
		PreserveFileMode: types.BoolValue(cmp.Spec.PreserveFileMode),
	}
}

func (c *ConfigManagementPlugin) ToConfigManagementPluginAPIModel(ctx context.Context, diagnostics *diag.Diagnostics, name string) *v1alpha1.ConfigManagementPlugin {
	return &v1alpha1.ConfigManagementPlugin{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigManagementPlugin",
			APIVersion: "argoproj.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Annotations: map[string]string{
				v1alpha1.AnnotationCMPImage:   c.Image.ValueString(),
				v1alpha1.AnnotationCMPEnabled: strconv.FormatBool(c.Enabled.ValueBool()),
			},
		},
		Spec: v1alpha1.PluginSpec{
			Version:          c.Spec.Version.ValueString(),
			Init:             toCommandAPIModel(c.Spec.Init),
			Generate:         toCommandAPIModel(c.Spec.Generate),
			Discover:         toDiscoverAPIModel(c.Spec.Discover),
			Parameters:       toParametersAPIModel(ctx, diagnostics, c.Spec.Parameters),
			PreserveFileMode: c.Spec.PreserveFileMode.ValueBool(),
		},
	}
}

func ToConfigManagementPluginsTFModel(ctx context.Context, diagnostics *diag.Diagnostics, cmps []*structpb.Struct, oldCMPs map[string]*ConfigManagementPlugin) map[string]*ConfigManagementPlugin {
	if len(cmps) == 0 && len(oldCMPs) == 0 {
		return oldCMPs
	}
	newCMPs := make(map[string]*ConfigManagementPlugin)
	for _, plugin := range cmps {
		var apiCMP *v1alpha1.ConfigManagementPlugin
		if err := marshal.RemarshalTo(plugin.AsMap(), &apiCMP); err != nil {
			diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get ConfigManagementPlugin. %s", err))
			return nil
		}
		cmp := &ConfigManagementPlugin{}
		cmp.Update(ctx, diagnostics, apiCMP)
		newCMPs[apiCMP.Name] = cmp
	}
	return newCMPs
}

func toClusterDataAPIModel(diagnostics *diag.Diagnostics, clusterData ClusterData) v1alpha1.ClusterData {
	var managedConfig *v1alpha1.ManagedClusterConfig
	if clusterData.ManagedClusterConfig != nil {
		managedConfig = &v1alpha1.ManagedClusterConfig{
			SecretName: clusterData.ManagedClusterConfig.SecretName.ValueString(),
			SecretKey:  clusterData.ManagedClusterConfig.SecretKey.ValueString(),
		}
	}
	var directClusterSpec *v1alpha1.DirectClusterSpec
	if clusterData.DirectClusterSpec != nil {
		clusterType := clusterData.DirectClusterSpec.ClusterType.ValueString()
		if clusterType != "" && clusterType == DirectClusterTypeString[argocdv1.DirectClusterType_DIRECT_CLUSTER_TYPE_KARGO] {
			directClusterSpec = &v1alpha1.DirectClusterSpec{
				ClusterType:     v1alpha1.DirectClusterType(clusterType),
				KargoInstanceId: clusterData.DirectClusterSpec.KargoInstanceId.ValueStringPointer(),
			}
		} else {
			diagnostics.AddError("unsupported cluster type", fmt.Sprintf("cluster_type %s is not supported, supported cluster_type: `kargo`", clusterData.DirectClusterSpec.ClusterType.String()))
			return v1alpha1.ClusterData{}
		}
	}

	clusterSize := clusterData.Size.ValueString()
	raw := runtime.RawExtension{}
	if clusterSize != "custom" {
		if clusterData.Kustomization.ValueString() != "" {
			if err := yaml.Unmarshal([]byte(clusterData.Kustomization.ValueString()), &raw); err != nil {
				diagnostics.AddError("failed unmarshal kustomization string to yaml", err.Error())
				return v1alpha1.ClusterData{}
			}
		}
	} else {
		expectedKustomization, err := generateExpectedKustomization(clusterData.CustomAgentSizeConfig, clusterData.Kustomization.ValueString())
		if err != nil {
			diagnostics.AddError("failed to generate expected kustomization", err.Error())
			return v1alpha1.ClusterData{}
		}

		if err = yaml.Unmarshal([]byte(expectedKustomization), &raw); err != nil {
			diagnostics.AddError("failed unmarshal kustomization string to yaml", err.Error())
			return v1alpha1.ClusterData{}
		}
	}

	// TODO: This is present since the API rejects custom size (to be exact any other sizes than small,medium,large,auto).
	// We should probably address this in the API ultimately by allowing for the setting of custom, but for now we hack it in the provider.
	var apiSize string
	if clusterSize == "custom" {
		apiSize = "large"
	} else {
		apiSize = clusterSize
	}

	return v1alpha1.ClusterData{
		Size:                            v1alpha1.ClusterSize(apiSize),
		AutoUpgradeDisabled:             toBoolPointer(clusterData.AutoUpgradeDisabled),
		ServerSideDiffEnabled:           toBoolPointer(clusterData.ServerSideDiffEnabled),
		Kustomization:                   raw,
		AppReplication:                  toBoolPointer(clusterData.AppReplication),
		TargetVersion:                   clusterData.TargetVersion.ValueString(),
		RedisTunneling:                  toBoolPointer(clusterData.RedisTunneling),
		DatadogAnnotationsEnabled:       toBoolPointer(clusterData.DatadogAnnotationsEnabled),
		EksAddonEnabled:                 toBoolPointer(clusterData.EksAddonEnabled),
		ManagedClusterConfig:            managedConfig,
		MultiClusterK8SDashboardEnabled: toBoolPointer(clusterData.MultiClusterK8SDashboardEnabled),
		AutoscalerConfig:                toAutoAgentSizeConfigAPIModel(extractConfigFromObjectValue(clusterData.AutoscalerConfig)),
		Project:                         clusterData.Project.ValueString(),
		Compatibility:                   toCompatibilityAPIModel(clusterData.Compatibility),
		ArgocdNotificationsSettings:     toArgoCDNotificationsSettingsAPIModel(clusterData.ArgocdNotificationsSettings),
		DirectClusterSpec:               directClusterSpec,
	}
}

func toRepoServerDelegateAPIModel(repoServerDelegate *RepoServerDelegate) *v1alpha1.RepoServerDelegate {
	if repoServerDelegate == nil {
		return nil
	}
	return &v1alpha1.RepoServerDelegate{
		ControlPlane:   toBoolPointer(repoServerDelegate.ControlPlane),
		ManagedCluster: toManagedClusterAPIModel(repoServerDelegate.ManagedCluster),
	}
}

func toCrossplaneExtensionAPIModel(extension *CrossplaneExtension) *v1alpha1.CrossplaneExtension {
	if extension == nil {
		return nil
	}
	return &v1alpha1.CrossplaneExtension{
		Resources: convertSlice(extension.Resources, func(t *CrossplaneExtensionResource) *v1alpha1.CrossplaneExtensionResource {
			return &v1alpha1.CrossplaneExtensionResource{
				Group: t.Group.ValueString(),
			}
		}),
	}
}

func toAgentPermissionsRuleAPIModel(extensions []*AgentPermissionsRule) []*v1alpha1.AgentPermissionsRule {
	var agentPermissionsRules []*v1alpha1.AgentPermissionsRule
	for _, extension := range extensions {
		agentPermissionsRules = append(agentPermissionsRules, &v1alpha1.AgentPermissionsRule{
			ApiGroups: convertSlice(extension.ApiGroups, tfStringToString),
			Resources: convertSlice(extension.Resources, tfStringToString),
			Verbs:     convertSlice(extension.Verbs, tfStringToString),
		})
	}
	return agentPermissionsRules
}

func toImageUpdaterDelegateAPIModel(imageUpdaterDelegate *ImageUpdaterDelegate) *v1alpha1.ImageUpdaterDelegate {
	if imageUpdaterDelegate == nil {
		return nil
	}
	return &v1alpha1.ImageUpdaterDelegate{
		ControlPlane:   toBoolPointer(imageUpdaterDelegate.ControlPlane),
		ManagedCluster: toManagedClusterAPIModel(imageUpdaterDelegate.ManagedCluster),
	}
}

func toAppSetDelegateAPIModel(appSetDelegate *AppSetDelegate) *v1alpha1.AppSetDelegate {
	if appSetDelegate == nil {
		return nil
	}
	return &v1alpha1.AppSetDelegate{
		ManagedCluster: toManagedClusterAPIModel(appSetDelegate.ManagedCluster),
	}
}

func toManagedClusterAPIModel(cluster *ManagedCluster) *v1alpha1.ManagedCluster {
	if cluster == nil {
		return nil
	}
	return &v1alpha1.ManagedCluster{
		ClusterName: cluster.ClusterName.ValueString(),
	}
}

func toClusterCustomizationAPIModel(ctx context.Context, diagnostics *diag.Diagnostics, clusterCustomization types.Object) *v1alpha1.ClusterCustomization {
	var customization ClusterCustomization
	diagnostics.Append(clusterCustomization.As(ctx, &customization, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	})...)
	if diagnostics.HasError() {
		return nil
	}
	raw := runtime.RawExtension{}
	if err := yaml.Unmarshal([]byte(customization.Kustomization.ValueString()), &raw); err != nil {
		diagnostics.AddError("failed unmarshal kustomization string to yaml", err.Error())
	}
	return &v1alpha1.ClusterCustomization{
		AutoUpgradeDisabled:   toBoolPointer(customization.AutoUpgradeDisabled),
		Kustomization:         raw,
		AppReplication:        toBoolPointer(customization.AppReplication),
		RedisTunneling:        toBoolPointer(customization.RedisTunneling),
		ServerSideDiffEnabled: toBoolPointer(customization.ServerSideDiffEnabled),
	}
}

func toIPAllowListAPIModel(entries []*IPAllowListEntry) []*v1alpha1.IPAllowListEntry {
	ipAllowList := []*v1alpha1.IPAllowListEntry{}
	for _, entry := range entries {
		ipAllowList = append(ipAllowList, &v1alpha1.IPAllowListEntry{
			Ip:          entry.Ip.ValueString(),
			Description: entry.Description.ValueString(),
		})
	}
	return ipAllowList
}

func toExtensionsAPIModel(entries basetypes.ListValue) []*v1alpha1.ArgoCDExtensionInstallEntry {
	if entries.IsNull() {
		return nil
	}

	extensions := make([]*v1alpha1.ArgoCDExtensionInstallEntry, 0)
	for _, entry := range entries.Elements() {
		obj := entry.(basetypes.ObjectValue)
		id := obj.Attributes()["id"].(basetypes.StringValue).ValueString()
		version := obj.Attributes()["version"].(basetypes.StringValue).ValueString()
		if id == "" || version == "" {
			continue
		}
		extensions = append(extensions, &v1alpha1.ArgoCDExtensionInstallEntry{
			Id:      id,
			Version: version,
		})
	}
	return extensions
}

func toAppsetPolicyAPIModel(ctx context.Context, diagnostics *diag.Diagnostics, appsetPolicy types.Object) *v1alpha1.AppsetPolicy {
	var policy AppsetPolicy
	diagnostics.Append(appsetPolicy.As(ctx, &policy, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	})...)
	if diagnostics.HasError() {
		return nil
	}
	return &v1alpha1.AppsetPolicy{
		Policy:         policy.Policy.ValueString(),
		OverridePolicy: toBoolPointer(policy.OverridePolicy),
	}
}

func toHostAliasesAPIModel(hostAliases []*HostAliases) []*v1alpha1.HostAliases {
	var hostAliasesAPI []*v1alpha1.HostAliases
	for _, entry := range hostAliases {
		var hostnames []string
		for _, hostname := range entry.Hostnames {
			hostnames = append(hostnames, hostname.ValueString())
		}
		hostAliasesAPI = append(hostAliasesAPI, &v1alpha1.HostAliases{
			Ip:        entry.Ip.ValueString(),
			Hostnames: hostnames,
		})
	}
	return hostAliasesAPI
}

func toParametersAPIModel(ctx context.Context, diagnostics *diag.Diagnostics, parameters *Parameters) *v1alpha1.Parameters {
	if parameters == nil {
		return nil
	}
	var static []*v1alpha1.ParameterAnnouncement
	for _, s := range parameters.Static {
		var array []string
		for _, a := range s.Array {
			array = append(array, a.ValueString())
		}
		var m map[string]string
		diagnostics.Append(s.Map.ElementsAs(ctx, &m, true)...)

		static = append(static, &v1alpha1.ParameterAnnouncement{
			Name:           s.Name.ValueString(),
			Title:          s.Title.ValueString(),
			Tooltip:        s.Tooltip.ValueString(),
			Required:       s.Required.ValueBool(),
			ItemType:       s.ItemType.ValueString(),
			CollectionType: s.CollectionType.ValueString(),
			String_:        s.String_.ValueString(),
			Array:          array,
			Map:            m,
		})
	}
	return &v1alpha1.Parameters{
		Static:  static,
		Dynamic: toDynamicAPIModel(parameters.Dynamic),
	}
}

func toDynamicAPIModel(dynamic *Dynamic) *v1alpha1.Dynamic {
	if dynamic == nil {
		return nil
	}
	var commands []string
	for _, c := range dynamic.Command {
		commands = append(commands, c.ValueString())
	}
	var args []string
	for _, a := range dynamic.Args {
		args = append(args, a.ValueString())
	}
	return &v1alpha1.Dynamic{
		Command: commands,
		Args:    args,
	}
}

func toDiscoverAPIModel(discover *Discover) *v1alpha1.Discover {
	if discover == nil {
		return nil
	}
	return &v1alpha1.Discover{
		Find:     toFindAPIModel(discover.Find),
		FileName: discover.FileName.ValueString(),
	}
}

func toFindAPIModel(find *Find) *v1alpha1.Find {
	if find == nil {
		return nil
	}
	var commands []string
	for _, c := range find.Command {
		commands = append(commands, c.ValueString())
	}
	var args []string
	for _, a := range find.Args {
		args = append(args, a.ValueString())
	}
	return &v1alpha1.Find{
		Command: commands,
		Args:    args,
		Glob:    find.Glob.ValueString(),
	}
}

func toCommandAPIModel(command *Command) *v1alpha1.Command {
	if command == nil {
		return nil
	}
	var commands []string
	for _, c := range command.Command {
		commands = append(commands, c.ValueString())
	}
	var args []string
	for _, a := range command.Args {
		args = append(args, a.ValueString())
	}
	return &v1alpha1.Command{
		Command: commands,
		Args:    args,
	}
}

func toRepoServerDelegateTFModel(repoServerDelegate *v1alpha1.RepoServerDelegate) *RepoServerDelegate {
	if repoServerDelegate == nil {
		return nil
	}
	controlPlane := repoServerDelegate.ControlPlane != nil && *repoServerDelegate.ControlPlane

	return &RepoServerDelegate{
		ControlPlane:   types.BoolValue(controlPlane),
		ManagedCluster: toManagedClusterTFModel(repoServerDelegate.ManagedCluster),
	}
}

func toImageUpdaterDelegateTFModel(imageUpdaterDelegate *v1alpha1.ImageUpdaterDelegate) *ImageUpdaterDelegate {
	if imageUpdaterDelegate == nil {
		return nil
	}
	controlPlane := imageUpdaterDelegate.ControlPlane != nil && *imageUpdaterDelegate.ControlPlane

	return &ImageUpdaterDelegate{
		ControlPlane:   types.BoolValue(controlPlane),
		ManagedCluster: toManagedClusterTFModel(imageUpdaterDelegate.ManagedCluster),
	}
}

func toAppSetDelegateTFModel(appSetDelegate *v1alpha1.AppSetDelegate) *AppSetDelegate {
	if appSetDelegate == nil {
		return nil
	}
	return &AppSetDelegate{
		ManagedCluster: toManagedClusterTFModel(appSetDelegate.ManagedCluster),
	}
}

func toManagedClusterTFModel(cluster *v1alpha1.ManagedCluster) *ManagedCluster {
	if cluster == nil {
		return nil
	}
	return &ManagedCluster{
		ClusterName: types.StringValue(cluster.ClusterName),
	}
}

func (a *ArgoCD) toClusterCustomizationTFModel(ctx context.Context, diagnostics *diag.Diagnostics, customization *v1alpha1.ClusterCustomization) types.Object {
	if customization == nil {
		return types.ObjectNull(clusterCustomizationAttrTypes)
	}
	yamlData, err := yaml.JSONToYAML(customization.Kustomization.Raw)
	if err != nil {
		diagnostics.AddError("failed to convert json to yaml", err.Error())
	}

	if !a.Spec.InstanceSpec.ClusterCustomizationDefaults.IsNull() && !a.Spec.InstanceSpec.ClusterCustomizationDefaults.IsUnknown() {
		var existingCustomization ClusterCustomization
		diagnostics.Append(a.Spec.InstanceSpec.ClusterCustomizationDefaults.As(ctx, &existingCustomization, basetypes.ObjectAsOptions{
			UnhandledNullAsEmpty:    true,
			UnhandledUnknownAsEmpty: true,
		})...)

		if !diagnostics.HasError() {
			existingYaml := existingCustomization.Kustomization.ValueString()
			newYaml := string(yamlData)
			if yamlEqual(existingYaml, newYaml) {
				yamlData = []byte(existingYaml)
			}
		}
	}

	autoUpgradeDisabled := customization.AutoUpgradeDisabled != nil && *customization.AutoUpgradeDisabled

	appReplication := customization.AppReplication != nil && *customization.AppReplication

	redisTunneling := customization.RedisTunneling != nil && *customization.RedisTunneling

	serverSideDiffEnabled := customization.ServerSideDiffEnabled != nil && *customization.ServerSideDiffEnabled

	c := &ClusterCustomization{
		AutoUpgradeDisabled:   types.BoolValue(autoUpgradeDisabled),
		Kustomization:         types.StringValue(string(yamlData)),
		AppReplication:        types.BoolValue(appReplication),
		RedisTunneling:        types.BoolValue(redisTunneling),
		ServerSideDiffEnabled: types.BoolValue(serverSideDiffEnabled),
	}
	clusterCustomization, d := types.ObjectValueFrom(ctx, clusterCustomizationAttrTypes, c)
	diagnostics.Append(d...)
	if diagnostics.HasError() {
		return types.ObjectNull(clusterCustomizationAttrTypes)
	}
	return clusterCustomization
}

func toIPAllowListTFModel(entries []*v1alpha1.IPAllowListEntry) []*IPAllowListEntry {
	var ipAllowList []*IPAllowListEntry
	for _, entry := range entries {
		description := types.StringNull()
		if entry.Description != "" {
			description = types.StringValue(entry.Description)
		}
		ipAllowList = append(ipAllowList, &IPAllowListEntry{
			Ip:          types.StringValue(entry.Ip),
			Description: description,
		})
	}
	return ipAllowList
}

func toExtensionsTFModel(entries []*v1alpha1.ArgoCDExtensionInstallEntry) types.List {
	extensions := make([]attr.Value, 0, len(entries))
	for _, entry := range entries {
		extensions = append(extensions, types.ObjectValueMust(
			map[string]attr.Type{
				"id":      types.StringType,
				"version": types.StringType,
			},
			map[string]attr.Value{
				"id":      types.StringValue(entry.Id),
				"version": types.StringValue(entry.Version),
			},
		))
	}

	return types.ListValueMust(
		types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"id":      types.StringType,
				"version": types.StringType,
			},
		},
		extensions,
	)
}

func toAppsetPolicyTFModel(ctx context.Context, diagnostics *diag.Diagnostics, appsetPolicy *v1alpha1.AppsetPolicy) types.Object {
	if appsetPolicy == nil {
		return types.ObjectNull(appsetPolicyAttrTypes)
	}

	overridePolicy := appsetPolicy.OverridePolicy != nil && *appsetPolicy.OverridePolicy

	a := &AppsetPolicy{
		Policy:         types.StringValue(appsetPolicy.Policy),
		OverridePolicy: types.BoolValue(overridePolicy),
	}
	policy, d := types.ObjectValueFrom(ctx, appsetPolicyAttrTypes, a)
	diagnostics.Append(d...)
	if diagnostics.HasError() {
		return types.ObjectNull(appsetPolicyAttrTypes)
	}
	return policy
}

func toHostAliasesTFModel(entries []*v1alpha1.HostAliases) []*HostAliases {
	var hostAliases []*HostAliases
	for _, entry := range entries {
		var hostnames []types.String
		for _, hostname := range entry.Hostnames {
			hostnames = append(hostnames, types.StringValue(hostname))
		}
		hostAliases = append(hostAliases, &HostAliases{
			Ip:        types.StringValue(entry.Ip),
			Hostnames: hostnames,
		})
	}
	return hostAliases
}

func toParametersTFModel(ctx context.Context, diagnostics *diag.Diagnostics, parameters *v1alpha1.Parameters) *Parameters {
	if parameters == nil {
		return nil
	}
	var static []*ParameterAnnouncement
	for _, s := range parameters.Static {
		static = append(static, toParameterAnnouncementTFModel(ctx, diagnostics, s))
	}
	return &Parameters{
		Static:  static,
		Dynamic: toDynamicTFModel(parameters.Dynamic),
	}
}

func toDynamicTFModel(dynamic *v1alpha1.Dynamic) *Dynamic {
	if dynamic == nil {
		return nil
	}
	var commands []types.String
	for _, c := range dynamic.Command {
		commands = append(commands, types.StringValue(c))
	}
	var args []types.String
	for _, a := range dynamic.Args {
		args = append(args, types.StringValue(a))
	}
	return &Dynamic{
		Command: commands,
		Args:    args,
	}
}

func toParameterAnnouncementTFModel(ctx context.Context, diagnostics *diag.Diagnostics, parameter *v1alpha1.ParameterAnnouncement) *ParameterAnnouncement {
	if parameter == nil {
		return nil
	}
	var array []types.String
	for _, a := range parameter.Array {
		array = append(array, types.StringValue(a))
	}
	m, diag := types.MapValueFrom(ctx, types.StringType, &parameter.Map)
	diagnostics.Append(diag...)
	name := types.StringNull()
	if parameter.Name != "" {
		name = types.StringValue(parameter.Name)
	}
	title := types.StringNull()
	if parameter.Title != "" {
		title = types.StringValue(parameter.Title)
	}
	tooltip := types.StringNull()
	if parameter.Tooltip != "" {
		tooltip = types.StringValue(parameter.Tooltip)
	}
	itemType := types.StringNull()
	if parameter.ItemType != "" {
		itemType = types.StringValue(parameter.ItemType)
	}
	collectionType := types.StringNull()
	if parameter.CollectionType != "" {
		collectionType = types.StringValue(parameter.CollectionType)
	}
	string_ := types.StringNull()
	if parameter.String_ != "" {
		string_ = types.StringValue(parameter.String_)
	}
	return &ParameterAnnouncement{
		Name:           name,
		Title:          title,
		Tooltip:        tooltip,
		Required:       types.BoolValue(parameter.Required),
		ItemType:       itemType,
		CollectionType: collectionType,
		String_:        string_,
		Array:          array,
		Map:            m,
	}
}

func toDiscoverTFModel(discover *v1alpha1.Discover) *Discover {
	if discover == nil {
		return nil
	}
	fileName := types.StringNull()
	if discover.FileName != "" {
		fileName = types.StringValue(discover.FileName)
	}
	return &Discover{
		Find:     toFindTFModel(discover.Find),
		FileName: fileName,
	}
}

func toFindTFModel(find *v1alpha1.Find) *Find {
	if find == nil {
		return nil
	}
	var commands []types.String
	for _, c := range find.Command {
		commands = append(commands, types.StringValue(c))
	}
	var args []types.String
	for _, a := range find.Args {
		args = append(args, types.StringValue(a))
	}
	glob := types.StringNull()
	if find.Glob != "" {
		glob = types.StringValue(find.Glob)
	}
	return &Find{
		Command: commands,
		Args:    args,
		Glob:    glob,
	}
}

func toCommandTFModel(command *v1alpha1.Command) *Command {
	if command == nil {
		return nil
	}
	var commands []types.String
	for _, c := range command.Command {
		commands = append(commands, types.StringValue(c))
	}
	var args []types.String
	for _, a := range command.Args {
		args = append(args, types.StringValue(a))
	}
	return &Command{
		Command: commands,
		Args:    args,
	}
}

func toCrossplaneExtensionTFModel(extension *v1alpha1.CrossplaneExtension) *CrossplaneExtension {
	if extension == nil || len(extension.Resources) == 0 {
		return nil
	}
	return &CrossplaneExtension{
		Resources: convertSlice(extension.Resources, crossplaneExtensionResourceToTFModel),
	}
}

func toAgentPermissionsRulesTFModel(rules []*v1alpha1.AgentPermissionsRule) []*AgentPermissionsRule {
	var agentPermissionsRules []*AgentPermissionsRule
	for _, rule := range rules {
		tfRule := &AgentPermissionsRule{
			ApiGroups: convertSlice(rule.ApiGroups, stringToTFString),
			Resources: convertSlice(rule.Resources, stringToTFString),
			Verbs:     convertSlice(rule.Verbs, stringToTFString),
		}
		agentPermissionsRules = append(agentPermissionsRules, tfRule)
	}
	return agentPermissionsRules
}

func toAppInAnyNamespaceConfigTFModel(config *v1alpha1.AppInAnyNamespaceConfig) *AppInAnyNamespaceConfig {
	if config == nil {
		return nil
	}
	if config.Enabled == nil {
		return nil
	}
	return &AppInAnyNamespaceConfig{
		Enabled: types.BoolValue(*config.Enabled),
	}
}

func convertSlice[T, U any](s []T, conv func(T) U) []U {
	var tfSlice []U
	for _, item := range s {
		tfSlice = append(tfSlice, conv(item))
	}
	return tfSlice
}

func stringToTFString(str string) types.String {
	return types.StringValue(str)
}

func tfStringToString(str types.String) string {
	return str.ValueString()
}

func crossplaneExtensionResourceToTFModel(resource *v1alpha1.CrossplaneExtensionResource) *CrossplaneExtensionResource {
	if resource == nil {
		return nil
	}
	return &CrossplaneExtensionResource{
		Group: types.StringValue(resource.Group),
	}
}

func toManagedClusterConfigTFModel(cfg *argocdv1.ManagedClusterConfig) *ManagedClusterConfig {
	if cfg == nil {
		return nil
	}
	return &ManagedClusterConfig{
		SecretName: types.StringValue(cfg.SecretName),
		SecretKey:  types.StringValue(cfg.SecretKey),
	}
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

func toResourcesAPIModel(resources *Resources) *v1alpha1.Resources {
	if resources == nil {
		return nil
	}
	return &v1alpha1.Resources{
		Mem: resources.Memory.ValueString(),
		Cpu: resources.Cpu.ValueString(),
	}
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

func extractConfigFromObjectValue(obj basetypes.ObjectValue) *AutoScalerConfig {
	if obj.IsNull() || obj.IsUnknown() {
		return nil
	}

	attrs := obj.Attributes()
	config := &AutoScalerConfig{}
	if appCtrl, ok := attrs["application_controller"].(basetypes.ObjectValue); ok {
		appCtrlAttrs := appCtrl.Attributes()
		if resMin, ok := appCtrlAttrs["resource_minimum"].(basetypes.ObjectValue); ok {
			resMinAttrs := resMin.Attributes()
			if cpu, ok := resMinAttrs["cpu"].(basetypes.StringValue); ok {
				if mem, ok := resMinAttrs["memory"].(basetypes.StringValue); ok {
					if resMax, ok := appCtrlAttrs["resource_maximum"].(basetypes.ObjectValue); ok {
						resMaxAttrs := resMax.Attributes()
						if maxCpu, ok := resMaxAttrs["cpu"].(basetypes.StringValue); ok {
							if maxMem, ok := resMaxAttrs["memory"].(basetypes.StringValue); ok {
								config.ApplicationController = &AppControllerAutoScalingConfig{
									ResourceMinimum: &Resources{
										Cpu:    cpu,
										Memory: mem,
									},
									ResourceMaximum: &Resources{
										Cpu:    maxCpu,
										Memory: maxMem,
									},
								}
							}
						}
					}
				}
			}
		}
	}
	if repoServer, ok := attrs["repo_server"].(basetypes.ObjectValue); ok {
		repoServerAttrs := repoServer.Attributes()
		if resMin, ok := repoServerAttrs["resource_minimum"].(basetypes.ObjectValue); ok {
			resMinAttrs := resMin.Attributes()
			if cpu, ok := resMinAttrs["cpu"].(basetypes.StringValue); ok {
				if mem, ok := resMinAttrs["memory"].(basetypes.StringValue); ok {
					if resMax, ok := repoServerAttrs["resource_maximum"].(basetypes.ObjectValue); ok {
						resMaxAttrs := resMax.Attributes()
						if maxCpu, ok := resMaxAttrs["cpu"].(basetypes.StringValue); ok {
							if maxMem, ok := resMaxAttrs["memory"].(basetypes.StringValue); ok {
								if replMax, ok := repoServerAttrs["replicas_maximum"].(basetypes.Int64Value); ok {
									if replMin, ok := repoServerAttrs["replicas_minimum"].(basetypes.Int64Value); ok {
										config.RepoServer = &RepoServerAutoScalingConfig{
											ResourceMinimum: &Resources{
												Cpu:    cpu,
												Memory: mem,
											},
											ResourceMaximum: &Resources{
												Cpu:    maxCpu,
												Memory: maxMem,
											},
											ReplicasMaximum: replMax,
											ReplicasMinimum: replMin,
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
	return config
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

func toAppsetPluginsTFModel(plugins []*v1alpha1.AppsetPlugins) []*AppsetPlugins {
	var tfPlugins []*AppsetPlugins
	for _, plugin := range plugins {
		tfPlugins = append(tfPlugins, &AppsetPlugins{
			Name:           types.StringValue(plugin.Name),
			Token:          types.StringValue(plugin.Token),
			BaseUrl:        types.StringValue(plugin.BaseUrl),
			RequestTimeout: types.Int64Value(int64(plugin.RequestTimeout)),
		})
	}
	return tfPlugins
}

func toAppsetPluginsAPIModel(plugins []*AppsetPlugins) []*v1alpha1.AppsetPlugins {
	var apiPlugins []*v1alpha1.AppsetPlugins
	for _, plugin := range plugins {
		apiPlugins = append(apiPlugins, &v1alpha1.AppsetPlugins{
			Name:           plugin.Name.ValueString(),
			Token:          plugin.Token.ValueString(),
			BaseUrl:        plugin.BaseUrl.ValueString(),
			RequestTimeout: int32(plugin.RequestTimeout.ValueInt64()),
		})
	}
	return apiPlugins
}

func toAppInAnyNamespaceConfigAPIModel(config *AppInAnyNamespaceConfig) *v1alpha1.AppInAnyNamespaceConfig {
	if config == nil || config.Enabled.IsNull() {
		disable := false
		return &v1alpha1.AppInAnyNamespaceConfig{
			Enabled: &disable,
		}
	}
	return &v1alpha1.AppInAnyNamespaceConfig{
		Enabled: config.Enabled.ValueBoolPointer(),
	}
}

func toAkuityIntelligenceExtensionTFModel(extension *v1alpha1.AkuityIntelligenceExtension, plan *ArgoCD) *AkuityIntelligenceExtension {
	if plan != nil {
		if plan.Spec.InstanceSpec.AkuityIntelligenceExtension == nil {
			return nil
		}
	}

	if extension == nil {
		return nil
	}
	return &AkuityIntelligenceExtension{
		Enabled:                  types.BoolValue(extension.Enabled != nil && *extension.Enabled),
		AllowedUsernames:         convertSlice(extension.AllowedUsernames, func(s string) types.String { return types.StringValue(s) }),
		AllowedGroups:            convertSlice(extension.AllowedGroups, func(s string) types.String { return types.StringValue(s) }),
		AiSupportEngineerEnabled: types.BoolValue(extension.AiSupportEngineerEnabled != nil && *extension.AiSupportEngineerEnabled),
	}
}

func toAkuityIntelligenceExtensionAPIModel(extension *AkuityIntelligenceExtension) *v1alpha1.AkuityIntelligenceExtension {
	if extension == nil {
		return nil
	}
	return &v1alpha1.AkuityIntelligenceExtension{
		Enabled:                  toBoolPointer(extension.Enabled),
		AllowedUsernames:         convertSlice(extension.AllowedUsernames, tfStringToString),
		AllowedGroups:            convertSlice(extension.AllowedGroups, tfStringToString),
		AiSupportEngineerEnabled: toBoolPointer(extension.AiSupportEngineerEnabled),
	}
}

func toClusterAddonsExtensionTFModel(extension *v1alpha1.ClusterAddonsExtension, plan *ArgoCD) *ClusterAddonsExtension {
	if plan != nil {
		if plan.Spec.InstanceSpec.ClusterAddonsExtension == nil {
			return nil
		}
	}
	if extension == nil {
		return nil
	}
	return &ClusterAddonsExtension{
		Enabled:          types.BoolValue(extension.Enabled != nil && *extension.Enabled),
		AllowedUsernames: convertSlice(extension.AllowedUsernames, func(s string) types.String { return types.StringValue(s) }),
		AllowedGroups:    convertSlice(extension.AllowedGroups, func(s string) types.String { return types.StringValue(s) }),
	}
}

func toClusterAddonsExtensionAPIModel(extension *ClusterAddonsExtension) *v1alpha1.ClusterAddonsExtension {
	if extension == nil {
		return nil
	}
	return &v1alpha1.ClusterAddonsExtension{
		Enabled:          toBoolPointer(extension.Enabled),
		AllowedUsernames: convertSlice(extension.AllowedUsernames, tfStringToString),
		AllowedGroups:    convertSlice(extension.AllowedGroups, tfStringToString),
	}
}

func toKubeVisionConfigTFModel(config *v1alpha1.KubeVisionConfig, plan *ArgoCD) *KubeVisionConfig {
	if plan == nil {
		return nil
	}
	if plan.Spec.InstanceSpec.KubeVisionConfig == nil || config == nil {
		return plan.Spec.InstanceSpec.KubeVisionConfig
	}
	res := &KubeVisionConfig{}
	if plan.Spec.InstanceSpec.KubeVisionConfig.CveScanConfig != nil {
		res.CveScanConfig = toCveScanConfigTFModel(config.CveScanConfig)
	}
	if plan.Spec.InstanceSpec.KubeVisionConfig.AiConfig != nil {
		res.AiConfig = toAIConfigTFModel(config.AiConfig, plan.Spec.InstanceSpec.KubeVisionConfig.AiConfig)
	}
	if plan.Spec.InstanceSpec.KubeVisionConfig.AdditionalAttributes != nil {
		res.AdditionalAttributes = toAdditionalAttributesTFModel(config.AdditionalAttributes)
	}
	return res
}

func toKubeVisionConfigAPIModel(config *KubeVisionConfig) *v1alpha1.KubeVisionConfig {
	if config == nil {
		return nil
	}
	return &v1alpha1.KubeVisionConfig{
		CveScanConfig:        toCveScanConfigAPIModel(config.CveScanConfig),
		AiConfig:             toAIConfigAPIModel(config.AiConfig),
		AdditionalAttributes: toAdditionalAttributesAPIModel(config.AdditionalAttributes),
	}
}

func toCveScanConfigTFModel(config *v1alpha1.CveScanConfig) *CveScanConfig {
	if config == nil {
		return nil
	}
	return &CveScanConfig{
		ScanEnabled:    types.BoolValue(config.ScanEnabled != nil && *config.ScanEnabled),
		RescanInterval: types.StringValue(config.RescanInterval),
	}
}

func toCveScanConfigAPIModel(config *CveScanConfig) *v1alpha1.CveScanConfig {
	if config == nil {
		return nil
	}
	return &v1alpha1.CveScanConfig{
		ScanEnabled:    toBoolPointer(config.ScanEnabled),
		RescanInterval: config.RescanInterval.ValueString(),
	}
}

func toAIConfigTFModel(config *v1alpha1.AIConfig, plan *AIConfig) *AIConfig {
	if config == nil {
		return nil
	}
	var incidents *IncidentsConfig
	if config.Incidents != nil {
		incidents = toIncidentsConfigTFModel(config.Incidents, plan)
	}
	return &AIConfig{
		Runbooks:            convertSlice(config.Runbooks, toRunbookTFModel),
		Incidents:           incidents,
		ArgocdSlackService:  types.StringPointerValue(config.ArgocdSlackService),
		ArgocdSlackChannels: convertSlice(config.ArgocdSlackChannels, stringToTFString),
	}
}

func toAdditionalAttributesTFModel(config []*v1alpha1.AdditionalAttributeRule) []*AdditionalAttributeRule {
	if config == nil {
		return nil
	}
	return convertSlice(config, func(rule *v1alpha1.AdditionalAttributeRule) *AdditionalAttributeRule {
		return &AdditionalAttributeRule{
			Group:       types.StringValue(rule.Group),
			Kind:        types.StringValue(rule.Kind),
			Namespace:   types.StringValue(rule.Namespace),
			Labels:      convertSlice(rule.Labels, stringToTFString),
			Annotations: convertSlice(rule.Annotations, stringToTFString),
		}
	})
}

func toAIConfigAPIModel(config *AIConfig) *v1alpha1.AIConfig {
	if config == nil {
		return nil
	}
	return &v1alpha1.AIConfig{
		Runbooks:            convertSlice(config.Runbooks, toRunbookAPIModel),
		Incidents:           toIncidentsConfigAPIModel(config.Incidents),
		ArgocdSlackService:  config.ArgocdSlackService.ValueStringPointer(),
		ArgocdSlackChannels: convertSlice(config.ArgocdSlackChannels, tfStringToString),
	}
}

func toAdditionalAttributesAPIModel(config []*AdditionalAttributeRule) []*v1alpha1.AdditionalAttributeRule {
	if config == nil {
		return nil
	}
	return convertSlice(config, func(rule *AdditionalAttributeRule) *v1alpha1.AdditionalAttributeRule {
		return &v1alpha1.AdditionalAttributeRule{
			Group:       rule.Group.ValueString(),
			Kind:        rule.Kind.ValueString(),
			Namespace:   rule.Namespace.ValueString(),
			Labels:      convertSlice(rule.Labels, tfStringToString),
			Annotations: convertSlice(rule.Annotations, tfStringToString),
		}
	})
}

func toRunbookTFModel(runbook *v1alpha1.Runbook) *Runbook {
	if runbook == nil {
		return nil
	}
	return &Runbook{
		Name:              types.StringValue(runbook.Name),
		Content:           types.StringValue(runbook.Content),
		AppliedTo:         toTargetSelectorTFModel(runbook.AppliedTo),
		SlackChannelNames: convertSlice(runbook.SlackChannelNames, stringToTFString),
	}
}

func toRunbookAPIModel(runbook *Runbook) *v1alpha1.Runbook {
	if runbook == nil {
		return nil
	}
	return &v1alpha1.Runbook{
		Name:              runbook.Name.ValueString(),
		Content:           runbook.Content.ValueString(),
		AppliedTo:         toTargetSelectorAPIModel(runbook.AppliedTo),
		SlackChannelNames: convertSlice(runbook.SlackChannelNames, tfStringToString),
	}
}

func toTargetSelectorTFModel(selector *v1alpha1.TargetSelector) *TargetSelector {
	if selector == nil {
		return nil
	}
	return &TargetSelector{
		ArgocdApplications: convertSlice(selector.ArgocdApplications, func(s string) types.String { return types.StringValue(s) }),
		K8SNamespaces:      convertSlice(selector.K8SNamespaces, func(s string) types.String { return types.StringValue(s) }),
		Clusters:           convertSlice(selector.Clusters, func(s string) types.String { return types.StringValue(s) }),
		DegradedFor:        types.StringPointerValue(selector.DegradedFor),
	}
}

func toTargetSelectorAPIModel(selector *TargetSelector) *v1alpha1.TargetSelector {
	if selector == nil {
		return nil
	}
	return &v1alpha1.TargetSelector{
		ArgocdApplications: convertSlice(selector.ArgocdApplications, tfStringToString),
		K8SNamespaces:      convertSlice(selector.K8SNamespaces, tfStringToString),
		Clusters:           convertSlice(selector.Clusters, tfStringToString),
		DegradedFor:        selector.DegradedFor.ValueStringPointer(),
	}
}

func toIncidentsConfigTFModel(config *v1alpha1.IncidentsConfig, plan *AIConfig) *IncidentsConfig {
	if config == nil {
		return nil
	}
	var grouping *IncidentsGroupingConfig
	if plan != nil && plan.Incidents != nil && plan.Incidents.Grouping != nil {
		grouping = toIncidentsGroupingConfigTFModel(config.Grouping)
	}
	return &IncidentsConfig{
		Triggers: convertSlice(config.Triggers, toTargetSelectorTFModel),
		Webhooks: convertSlice(config.Webhooks, toIncidentWebhookConfigTFModel),
		Grouping: grouping,
	}
}

func toIncidentsGroupingConfigTFModel(config *v1alpha1.IncidentsGroupingConfig) *IncidentsGroupingConfig {
	if config == nil {
		return nil
	}
	return &IncidentsGroupingConfig{
		ArgocdApplicationNames: toStringArrayTFModel(config.ArgocdApplicationNames),
		K8SNamespaces:          toStringArrayTFModel(config.K8SNamespaces),
	}
}

func toIncidentsGroupingConfigAPIMode(config *IncidentsGroupingConfig) *v1alpha1.IncidentsGroupingConfig {
	if config == nil {
		return nil
	}
	return &v1alpha1.IncidentsGroupingConfig{
		ArgocdApplicationNames: toStringArrayAPIModel(config.ArgocdApplicationNames),
		K8SNamespaces:          toStringArrayAPIModel(config.K8SNamespaces),
	}
}

func toIncidentsConfigAPIModel(config *IncidentsConfig) *v1alpha1.IncidentsConfig {
	if config == nil {
		return nil
	}
	return &v1alpha1.IncidentsConfig{
		Triggers: convertSlice(config.Triggers, toTargetSelectorAPIModel),
		Webhooks: convertSlice(config.Webhooks, toIncidentWebhookConfigAPIModel),
		Grouping: toIncidentsGroupingConfigAPIMode(config.Grouping),
	}
}

func toIncidentWebhookConfigTFModel(webhook *v1alpha1.IncidentWebhookConfig) *IncidentWebhookConfig {
	if webhook == nil {
		return nil
	}
	return &IncidentWebhookConfig{
		Name:                      types.StringValue(webhook.Name),
		DescriptionPath:           types.StringValue(webhook.DescriptionPath),
		ClusterPath:               types.StringValue(webhook.ClusterPath),
		K8SNamespacePath:          types.StringValue(webhook.K8SNamespacePath),
		ArgocdApplicationNamePath: types.StringValue(webhook.ArgocdApplicationNamePath),
	}
}

func toIncidentWebhookConfigAPIModel(webhook *IncidentWebhookConfig) *v1alpha1.IncidentWebhookConfig {
	if webhook == nil {
		return nil
	}
	return &v1alpha1.IncidentWebhookConfig{
		Name:                      webhook.Name.ValueString(),
		DescriptionPath:           webhook.DescriptionPath.ValueString(),
		ClusterPath:               webhook.ClusterPath.ValueString(),
		K8SNamespacePath:          webhook.K8SNamespacePath.ValueString(),
		ArgocdApplicationNamePath: webhook.ArgocdApplicationNamePath.ValueString(),
	}
}

func toCompatibilityTFModel(plan *Cluster, cfg *argocdv1.ClusterCompatibility) *ClusterCompatibility {
	if plan != nil && plan.Spec != nil {
		if plan.Spec.Data.Compatibility == nil {
			if cfg == nil || !cfg.Ipv6Only {
				return nil
			}
		}
	}
	var cc *ClusterCompatibility
	if cfg != nil {
		cc = &ClusterCompatibility{
			Ipv6Only: types.BoolValue(cfg.Ipv6Only),
		}
	}
	return cc
}

func toCompatibilityAPIModel(cfg *ClusterCompatibility) *v1alpha1.ClusterCompatibility {
	if cfg == nil {
		return nil
	}
	if !cfg.Ipv6Only.ValueBool() {
		return nil
	}
	return &v1alpha1.ClusterCompatibility{
		Ipv6Only: cfg.Ipv6Only.ValueBool(),
	}
}

func toArgoCDNotificationsSettingsTFModel(plan *Cluster, cfg *argocdv1.ClusterArgoCDNotificationsSettings) *ClusterArgoCDNotificationsSettings {
	if plan != nil && plan.Spec != nil {
		if plan.Spec.Data.ArgocdNotificationsSettings == nil {
			if cfg == nil || !cfg.InClusterSettings {
				return nil
			}
		}
	}
	var cs *ClusterArgoCDNotificationsSettings
	if cfg != nil {
		cs = &ClusterArgoCDNotificationsSettings{
			InClusterSettings: types.BoolValue(cfg.InClusterSettings),
		}
	}
	return cs
}

func toArgoCDNotificationsSettingsAPIModel(cfg *ClusterArgoCDNotificationsSettings) *v1alpha1.ClusterArgoCDNotificationsSettings {
	if cfg == nil {
		return nil
	}
	return &v1alpha1.ClusterArgoCDNotificationsSettings{
		InClusterSettings: cfg.InClusterSettings.ValueBool(),
	}
}

func generateExpectedKustomization(customConfig *CustomAgentSizeConfig, userKustomization string) (string, error) {
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

func toAutoAgentSizeConfigAPIModel(autoscalerConfig *AutoScalerConfig) *v1alpha1.AutoScalerConfig {
	if autoscalerConfig == nil {
		return nil
	}

	var appController *v1alpha1.AppControllerAutoScalingConfig
	if autoscalerConfig.ApplicationController != nil {
		appController = &v1alpha1.AppControllerAutoScalingConfig{
			ResourceMinimum: toResourcesAPIModel(autoscalerConfig.ApplicationController.ResourceMinimum),
			ResourceMaximum: toResourcesAPIModel(autoscalerConfig.ApplicationController.ResourceMaximum),
		}
	}

	var repoServer *v1alpha1.RepoServerAutoScalingConfig
	if autoscalerConfig.RepoServer != nil {
		repoServer = &v1alpha1.RepoServerAutoScalingConfig{
			ResourceMinimum: toResourcesAPIModel(autoscalerConfig.RepoServer.ResourceMinimum),
			ResourceMaximum: toResourcesAPIModel(autoscalerConfig.RepoServer.ResourceMaximum),
			ReplicaMinimum:  int32(autoscalerConfig.RepoServer.ReplicasMinimum.ValueInt64()),
			ReplicaMaximum:  int32(autoscalerConfig.RepoServer.ReplicasMaximum.ValueInt64()),
		}
	}

	return &v1alpha1.AutoScalerConfig{
		ApplicationController: appController,
		RepoServer:            repoServer,
	}
}

func toAutoScalerConfigTFModel(plan *Cluster, apiConfig *argocdv1.AutoScalerConfig) types.Object {
	// If plan is nil, use API value
	if plan == nil || plan.Spec == nil {
		if apiConfig == nil {
			return types.ObjectNull(autoScalerConfigAttrTypes)
		}
		// Continue processing to return API config
	} else {
		// Get the size value
		sizeValue := ""
		if !plan.Spec.Data.Size.IsNull() {
			sizeValue = plan.Spec.Data.Size.ValueString()
		}

		// Check auto_agent_size_config status
		autoscalerIsUnknown := plan.Spec.Data.AutoscalerConfig.IsUnknown()
		autoscalerIsNull := plan.Spec.Data.AutoscalerConfig.IsNull()

		// Main logic: decide whether to return null or continue processing
		if sizeValue != "auto" {
			// For non-auto sizes, auto_agent_size_config should generally be null
			if autoscalerIsUnknown || autoscalerIsNull {
				// Not specified or explicitly null - return null
				return types.ObjectNull(autoScalerConfigAttrTypes)
			}
			// Has explicit non-null values - preserve the planned values instead of API values
			// This handles the case where we transition from "auto" to another size
			ctx := context.Background()
			plannedConfig := &AutoScalerConfig{}
			plan.Spec.Data.AutoscalerConfig.As(ctx, plannedConfig, basetypes.ObjectAsOptions{})
			return plan.Spec.Data.AutoscalerConfig
		} else {
			// For auto size, show config from API if not explicitly set to null
			if autoscalerIsNull {
				return types.ObjectNull(autoScalerConfigAttrTypes)
			}
			// Continue processing to show API defaults or planned values
		}
	}

	// If the plan doesn't include auto scaler config but size is "auto", show API defaults
	if plan != nil && plan.Spec != nil && plan.Spec.Data.AutoscalerConfig.IsNull() {
		// If size is "auto", we should show the API defaults even if not explicitly configured
		if !plan.Spec.Data.Size.IsNull() && plan.Spec.Data.Size.ValueString() == "auto" {
			// Show API defaults in state - don't return null
		} else {
			// For other sizes, don't show autoscaler config
			return types.ObjectNull(autoScalerConfigAttrTypes)
		}
	}

	if apiConfig == nil {
		return types.ObjectNull(autoScalerConfigAttrTypes)
	}

	// Get the planned auto scaler config to preserve original values when equivalent
	var plannedConfig *AutoScalerConfig
	if plan != nil && plan.Spec != nil && !plan.Spec.Data.AutoscalerConfig.IsNull() {
		ctx := context.Background()
		plannedConfig = &AutoScalerConfig{}
		plan.Spec.Data.AutoscalerConfig.As(ctx, plannedConfig, basetypes.ObjectAsOptions{})
	}

	configAttrs := make(map[string]attr.Value)

	if apiConfig.ApplicationController != nil {
		appControllerAttrs := make(map[string]attr.Value)

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

			resourceMinAttrs := map[string]attr.Value{
				"memory": types.StringValue(memoryValue),
				"cpu":    types.StringValue(cpuValue),
			}
			appControllerAttrs["resource_minimum"] = types.ObjectValueMust(resourcesAttrTypes, resourceMinAttrs)
		} else {
			appControllerAttrs["resource_minimum"] = types.ObjectNull(resourcesAttrTypes)
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

			resourceMaxAttrs := map[string]attr.Value{
				"memory": types.StringValue(memoryValue),
				"cpu":    types.StringValue(cpuValue),
			}
			appControllerAttrs["resource_maximum"] = types.ObjectValueMust(resourcesAttrTypes, resourceMaxAttrs)
		} else {
			appControllerAttrs["resource_maximum"] = types.ObjectNull(resourcesAttrTypes)
		}

		configAttrs["application_controller"] = types.ObjectValueMust(appControllerAutoScalingAttrTypes, appControllerAttrs)
	} else {
		configAttrs["application_controller"] = types.ObjectNull(appControllerAutoScalingAttrTypes)
	}

	if apiConfig.RepoServer != nil {
		repoServerAttrs := make(map[string]attr.Value)

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

			resourceMinAttrs := map[string]attr.Value{
				"memory": types.StringValue(memoryValue),
				"cpu":    types.StringValue(cpuValue),
			}
			repoServerAttrs["resource_minimum"] = types.ObjectValueMust(resourcesAttrTypes, resourceMinAttrs)
		} else {
			repoServerAttrs["resource_minimum"] = types.ObjectNull(resourcesAttrTypes)
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

			resourceMaxAttrs := map[string]attr.Value{
				"memory": types.StringValue(memoryValue),
				"cpu":    types.StringValue(cpuValue),
			}
			repoServerAttrs["resource_maximum"] = types.ObjectValueMust(resourcesAttrTypes, resourceMaxAttrs)
		} else {
			repoServerAttrs["resource_maximum"] = types.ObjectNull(resourcesAttrTypes)
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

		repoServerAttrs["replicas_minimum"] = types.Int64Value(replicasMin)
		repoServerAttrs["replicas_maximum"] = types.Int64Value(replicasMax)

		configAttrs["repo_server"] = types.ObjectValueMust(repoServerAutoScalingAttrTypes, repoServerAttrs)
	} else {
		configAttrs["repo_server"] = types.ObjectNull(repoServerAutoScalingAttrTypes)
	}

	return types.ObjectValueMust(autoScalerConfigAttrTypes, configAttrs)
}

func toApplicationSetExtensionTFModel(extension *v1alpha1.ApplicationSetExtension) *ApplicationSetExtension {
	if extension == nil {
		return nil
	}
	if extension.Enabled == nil {
		return nil
	}

	return &ApplicationSetExtension{
		Enabled: types.BoolValue(*extension.Enabled),
	}
}

func toApplicationSetExtensionAPIModel(extension *ApplicationSetExtension) *v1alpha1.ApplicationSetExtension {
	if extension == nil || extension.Enabled.IsNull() {
		// When the block is removed from config, explicitly disable it
		disabled := false
		return &v1alpha1.ApplicationSetExtension{
			Enabled: &disabled,
		}
	}
	return &v1alpha1.ApplicationSetExtension{
		Enabled: extension.Enabled.ValueBoolPointer(),
	}
}
