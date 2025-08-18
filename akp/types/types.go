package types

import (
	"context"
	"fmt"
	"reflect"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
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
		"auto_upgrade_disabled": tftypes.BoolType,
		"kustomization":         tftypes.StringType,
		"app_replication":       tftypes.BoolType,
		"redis_tunneling":       tftypes.BoolType,
	}

	appsetPolicyAttrTypes = map[string]attr.Type{
		"policy":          tftypes.StringType,
		"override_policy": tftypes.BoolType,
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
	declarativeManagementEnabled := false
	if cd.Spec.InstanceSpec.DeclarativeManagementEnabled != nil && *cd.Spec.InstanceSpec.DeclarativeManagementEnabled {
		declarativeManagementEnabled = true
	}
	imageUpdaterEnabled := false
	if cd.Spec.InstanceSpec.ImageUpdaterEnabled != nil && *cd.Spec.InstanceSpec.ImageUpdaterEnabled {
		imageUpdaterEnabled = true
	}
	backendIpAllowListEnabled := false
	if cd.Spec.InstanceSpec.BackendIpAllowListEnabled != nil && *cd.Spec.InstanceSpec.BackendIpAllowListEnabled {
		backendIpAllowListEnabled = true
	}
	auditExtensionEnabled := false
	if cd.Spec.InstanceSpec.AuditExtensionEnabled != nil && *cd.Spec.InstanceSpec.AuditExtensionEnabled {
		auditExtensionEnabled = true
	}
	syncHistoryExtensionEnabled := false
	if cd.Spec.InstanceSpec.SyncHistoryExtensionEnabled != nil && *cd.Spec.InstanceSpec.SyncHistoryExtensionEnabled {
		syncHistoryExtensionEnabled = true
	}
	assistantExtensionEnabled := false
	if cd.Spec.InstanceSpec.AssistantExtensionEnabled != nil && *cd.Spec.InstanceSpec.AssistantExtensionEnabled {
		assistantExtensionEnabled = true
	}
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

	a.Spec = ArgoCDSpec{
		Description: tftypes.StringValue(cd.Spec.Description),
		Version:     tftypes.StringValue(cd.Spec.Version),
		InstanceSpec: InstanceSpec{
			IpAllowList:                     toIPAllowListTFModel(cd.Spec.InstanceSpec.IpAllowList),
			Subdomain:                       tftypes.StringValue(cd.Spec.InstanceSpec.Subdomain),
			DeclarativeManagementEnabled:    tftypes.BoolValue(declarativeManagementEnabled),
			Extensions:                      toExtensionsTFModel(cd.Spec.InstanceSpec.Extensions),
			ClusterCustomizationDefaults:    a.toClusterCustomizationTFModel(ctx, diagnostics, cd.Spec.InstanceSpec.ClusterCustomizationDefaults),
			ImageUpdaterEnabled:             tftypes.BoolValue(imageUpdaterEnabled),
			BackendIpAllowListEnabled:       tftypes.BoolValue(backendIpAllowListEnabled),
			RepoServerDelegate:              toRepoServerDelegateTFModel(cd.Spec.InstanceSpec.RepoServerDelegate),
			AuditExtensionEnabled:           tftypes.BoolValue(auditExtensionEnabled),
			SyncHistoryExtensionEnabled:     tftypes.BoolValue(syncHistoryExtensionEnabled),
			CrossplaneExtension:             toCrossplaneExtensionTFModel(cd.Spec.InstanceSpec.CrossplaneExtension),
			ImageUpdaterDelegate:            toImageUpdaterDelegateTFModel(cd.Spec.InstanceSpec.ImageUpdaterDelegate),
			AppSetDelegate:                  toAppSetDelegateTFModel(cd.Spec.InstanceSpec.AppSetDelegate),
			AssistantExtensionEnabled:       tftypes.BoolValue(assistantExtensionEnabled),
			AppsetPolicy:                    toAppsetPolicyTFModel(ctx, diagnostics, cd.Spec.InstanceSpec.AppsetPolicy),
			HostAliases:                     toHostAliasesTFModel(cd.Spec.InstanceSpec.HostAliases),
			AgentPermissionsRules:           toAgentPermissionsRulesTFModel(cd.Spec.InstanceSpec.AgentPermissionsRules),
			Fqdn:                            types.StringValue(fqdn),
			MultiClusterK8SDashboardEnabled: tftypes.BoolValue(multiClusterK8SDashboardEnabled),
			AppInAnyNamespaceConfig:         appInAnyNamespaceConfig,
			AppsetPlugins:                   toAppsetPluginsTFModel(cd.Spec.InstanceSpec.AppsetPlugins),
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
				AppInAnyNamespaceConfig:         toAppInAnyNamespaceConfigAPIModel(a.Spec.InstanceSpec.AppInAnyNamespaceConfig),
				AppsetPlugins:                   toAppsetPluginsAPIModel(a.Spec.InstanceSpec.AppsetPlugins),
			},
		},
	}
}

func toBoolPointer(b tftypes.Bool) *bool {
	if b.IsUnknown() {
		return nil
	}
	return b.ValueBoolPointer()
}

func (c *Cluster) Update(ctx context.Context, diagnostics *diag.Diagnostics, apiCluster *argocdv1.Cluster, plan *Cluster) {
	c.ID = tftypes.StringValue(apiCluster.GetId())
	c.Name = tftypes.StringValue(apiCluster.GetName())
	c.Namespace = tftypes.StringValue(apiCluster.GetNamespace())
	if c.RemoveAgentResourcesOnDestroy.IsUnknown() || c.RemoveAgentResourcesOnDestroy.IsNull() {
		c.RemoveAgentResourcesOnDestroy = tftypes.BoolValue(true)
	}
	if c.ReapplyManifestsOnUpdate.IsUnknown() || c.ReapplyManifestsOnUpdate.IsNull() {
		c.ReapplyManifestsOnUpdate = tftypes.BoolValue(false)
	} else {
		c.ReapplyManifestsOnUpdate = plan.ReapplyManifestsOnUpdate
	}
	labels, d := tftypes.MapValueFrom(ctx, tftypes.StringType, apiCluster.GetData().GetLabels())
	if d.HasError() {
		labels = tftypes.MapNull(tftypes.StringType)
	}
	diagnostics.Append(d...)
	annotations, d := tftypes.MapValueFrom(ctx, tftypes.StringType, apiCluster.GetData().GetAnnotations())
	if d.HasError() {
		annotations = tftypes.MapNull(tftypes.StringType)
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

	var kustomization tftypes.String
	if plan != nil && plan.Spec != nil && !plan.Spec.Data.Kustomization.IsNull() && !plan.Spec.Data.Kustomization.IsUnknown() {
		if isKustomizationSubset(plan.Spec.Data.Kustomization.ValueString(), string(yamlData)) {
			kustomization = plan.Spec.Data.Kustomization
		}
	} else if apiCluster.GetData().GetKustomization() != nil {
		// When no kustomization is specified in the plan, set it to obtained value from the API
		kustomization = tftypes.StringValue(string(yamlData))
	} else {
		kustomization = tftypes.StringNull()
	}

	var size tftypes.String
	var customConfig *CustomAgentSizeConfig
	if plan != nil && plan.Spec != nil && plan.Spec.Data.CustomAgentSizeConfig != nil && plan.Spec.Data.Size.ValueString() == "custom" {
		size = plan.Spec.Data.Size
		customKustomization, err := generateExpectedKustomization(plan.Spec.Data.CustomAgentSizeConfig, "")
		if err != nil {
			diagnostics.AddError("failed to generate expected kustomization", err.Error())
		} else {
			if isKustomizationSubset(customKustomization, string(yamlData)) {
				customConfig = plan.Spec.Data.CustomAgentSizeConfig
				size = tftypes.StringValue("custom")
			} else {
				size = tftypes.StringValue(ClusterSizeString[apiCluster.GetData().GetSize()])
			}
		}
	} else {
		size = tftypes.StringValue(ClusterSizeString[apiCluster.GetData().GetSize()])
	}

	c.Labels = labels
	c.Annotations = annotations

	autoscalerConfig := toAutoScalerConfigTFModel(nil)
	if plan != nil && plan.Spec != nil && plan.Spec.Data.Size.ValueString() == "auto" {
		newAPIConfig := apiCluster.GetData().GetAutoscalerConfig()
		if !plan.Spec.Data.AutoscalerConfig.IsNull() && !plan.Spec.Data.AutoscalerConfig.IsUnknown() && newAPIConfig != nil &&
			newAPIConfig.RepoServer != nil && newAPIConfig.ApplicationController != nil {
			newConfig := &AutoScalerConfig{
				ApplicationController: &AppControllerAutoScalingConfig{
					ResourceMinimum: &Resources{
						Memory: tftypes.StringValue(newAPIConfig.ApplicationController.ResourceMinimum.Mem),
						Cpu:    tftypes.StringValue(newAPIConfig.ApplicationController.ResourceMinimum.Cpu),
					},
					ResourceMaximum: &Resources{
						Memory: tftypes.StringValue(newAPIConfig.ApplicationController.ResourceMaximum.Mem),
						Cpu:    tftypes.StringValue(newAPIConfig.ApplicationController.ResourceMaximum.Cpu),
					},
				},
				RepoServer: &RepoServerAutoScalingConfig{
					ResourceMinimum: &Resources{
						Memory: tftypes.StringValue(newAPIConfig.RepoServer.ResourceMinimum.Mem),
						Cpu:    tftypes.StringValue(newAPIConfig.RepoServer.ResourceMinimum.Cpu),
					},
					ResourceMaximum: &Resources{
						Memory: tftypes.StringValue(newAPIConfig.RepoServer.ResourceMaximum.Mem),
						Cpu:    tftypes.StringValue(newAPIConfig.RepoServer.ResourceMaximum.Cpu),
					},
					ReplicasMaximum: tftypes.Int64Value(int64(newAPIConfig.RepoServer.ReplicaMaximum)),
					ReplicasMinimum: tftypes.Int64Value(int64(newAPIConfig.RepoServer.ReplicaMinimum)),
				},
			}
			if areAutoScalerConfigsEquivalent(extractConfigFromObjectValue(plan.Spec.Data.AutoscalerConfig), newConfig) {
				autoscalerConfig = plan.Spec.Data.AutoscalerConfig
			} else {
				autoscalerConfig = toAutoScalerConfigTFModel(newAPIConfig)
			}
		} else {
			autoscalerConfig = toAutoScalerConfigTFModel(newAPIConfig)
		}
	}

	var directClusterSpec *DirectClusterSpec
	if plan != nil && plan.Spec != nil && plan.Spec.Data.DirectClusterSpec != nil {
		clusterType := DirectClusterTypeString[apiCluster.GetData().DirectClusterSpec.GetClusterType()]
		if clusterType == DirectClusterTypeString[argocdv1.DirectClusterType_DIRECT_CLUSTER_TYPE_KARGO] {
			directClusterSpec = &DirectClusterSpec{
				ClusterType:     tftypes.StringValue(clusterType),
				KargoInstanceId: tftypes.StringValue(apiCluster.GetData().DirectClusterSpec.GetKargoInstanceId()),
			}
		}
	}

	c.Spec = &ClusterSpec{
		Description:     tftypes.StringValue(apiCluster.GetDescription()),
		NamespaceScoped: tftypes.BoolValue(apiCluster.GetNamespaceScoped()),
		Data: ClusterData{
			Size:                            size,
			AutoUpgradeDisabled:             tftypes.BoolValue(apiCluster.GetData().GetAutoUpgradeDisabled()),
			Kustomization:                   kustomization,
			AppReplication:                  tftypes.BoolValue(apiCluster.GetData().GetAppReplication()),
			TargetVersion:                   tftypes.StringValue(apiCluster.GetData().GetTargetVersion()),
			RedisTunneling:                  tftypes.BoolValue(apiCluster.GetData().GetRedisTunneling()),
			DatadogAnnotationsEnabled:       tftypes.BoolValue(apiCluster.GetData().GetDatadogAnnotationsEnabled()),
			EksAddonEnabled:                 tftypes.BoolValue(apiCluster.GetData().GetEksAddonEnabled()),
			ManagedClusterConfig:            toManagedClusterConfigTFModel(apiCluster.GetData().GetManagedClusterConfig()),
			MultiClusterK8SDashboardEnabled: tftypes.BoolValue(apiCluster.GetData().GetMultiClusterK8SDashboardEnabled()),
			AutoscalerConfig:                autoscalerConfig,
			CustomAgentSizeConfig:           customConfig,
			Project:                         tftypes.StringValue(apiCluster.GetData().GetProject()),
			Compatibility:                   toCompatibilityTFModel(plan, apiCluster.GetData().GetCompatibility()),
			ArgocdNotificationsSettings:     toArgoCDNotificationsSettingsTFModel(plan, apiCluster.GetData().GetArgocdNotificationsSettings()),
			DirectClusterSpec:               directClusterSpec,
		},
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
			Data:            toClusterDataAPIModel(ctx, diagnostics, c.Spec.Data),
		},
	}
}

func (c *ConfigManagementPlugin) Update(ctx context.Context, diagnostics *diag.Diagnostics, cmp *v1alpha1.ConfigManagementPlugin) {
	version := tftypes.StringNull()
	if cmp.Spec.Version != "" {
		version = tftypes.StringValue(cmp.Spec.Version)
	}
	c.Enabled = tftypes.BoolValue(cmp.Annotations[v1alpha1.AnnotationCMPEnabled] == "true")
	c.Image = types.StringValue(cmp.Annotations[v1alpha1.AnnotationCMPImage])
	c.Spec = &PluginSpec{
		Version:          version,
		Init:             toCommandTFModel(cmp.Spec.Init),
		Generate:         toCommandTFModel(cmp.Spec.Generate),
		Discover:         toDiscoverTFModel(cmp.Spec.Discover),
		Parameters:       toParametersTFModel(ctx, diagnostics, cmp.Spec.Parameters),
		PreserveFileMode: tftypes.BoolValue(cmp.Spec.PreserveFileMode),
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

func toClusterDataAPIModel(ctx context.Context, diagnostics *diag.Diagnostics, clusterData ClusterData) v1alpha1.ClusterData {
	var autoscalerConfig *AutoScalerConfig
	if d := clusterData.AutoscalerConfig.As(ctx, &autoscalerConfig, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	}); d.HasError() {
		diagnostics.AddError("failed to convert autoscaler config", "")
		return v1alpha1.ClusterData{}
	}

	size, raw := handleAgentSizeAndKustomization(diagnostics, &clusterData, autoscalerConfig)
	if diagnostics.HasError() {
		return v1alpha1.ClusterData{}
	}

	var managedConfig *v1alpha1.ManagedClusterConfig
	if clusterData.ManagedClusterConfig != nil {
		managedConfig = &v1alpha1.ManagedClusterConfig{
			SecretName: clusterData.ManagedClusterConfig.SecretName.ValueString(),
			SecretKey:  clusterData.ManagedClusterConfig.SecretKey.ValueString(),
		}
	}

	autoscalerConfigAPI := &v1alpha1.AutoScalerConfig{}
	if autoscalerConfig != nil {
		if autoscalerConfig.RepoServer != nil {
			if autoscalerConfig.RepoServer.ResourceMaximum == nil || autoscalerConfig.RepoServer.ResourceMinimum == nil {
				diagnostics.AddError("repo server autoscaler config requires minimum and maximum resources", "")
				return v1alpha1.ClusterData{}
			}
			if autoscalerConfig.RepoServer.ResourceMinimum.Memory.ValueString() == "" || autoscalerConfig.RepoServer.ResourceMinimum.Cpu.ValueString() == "" ||
				autoscalerConfig.RepoServer.ResourceMaximum.Memory.ValueString() == "" || autoscalerConfig.RepoServer.ResourceMaximum.Cpu.ValueString() == "" ||
				autoscalerConfig.RepoServer.ReplicasMaximum.ValueInt64() == 0 || autoscalerConfig.RepoServer.ReplicasMinimum.ValueInt64() == 0 {
				diagnostics.AddError("repo server autoscaler config requires memory, cpu, and replicas values", "")
				return v1alpha1.ClusterData{}
			}
			if !areResourcesValid(
				autoscalerConfig.RepoServer.ResourceMinimum.Memory.ValueString(),
				autoscalerConfig.RepoServer.ResourceMaximum.Memory.ValueString(),
			) {
				diagnostics.AddError("repo server minimum memory must be less than or equal to maximum memory", "")
				return v1alpha1.ClusterData{}
			}
			if !areResourcesValid(
				autoscalerConfig.RepoServer.ResourceMinimum.Cpu.ValueString(),
				autoscalerConfig.RepoServer.ResourceMaximum.Cpu.ValueString(),
			) {
				diagnostics.AddError("repo server minimum CPU must be less than or equal to maximum CPU", "")
				return v1alpha1.ClusterData{}
			}
			if autoscalerConfig.RepoServer.ReplicasMinimum.ValueInt64() > autoscalerConfig.RepoServer.ReplicasMaximum.ValueInt64() {
				diagnostics.AddError("repo server minimum replicas must be less than or equal to maximum replicas", "")
				return v1alpha1.ClusterData{}
			}
			autoscalerConfigAPI.RepoServer = &v1alpha1.RepoServerAutoScalingConfig{
				ResourceMinimum: toResourcesAPIModel(autoscalerConfig.RepoServer.ResourceMinimum),
				ResourceMaximum: toResourcesAPIModel(autoscalerConfig.RepoServer.ResourceMaximum),
				ReplicaMaximum:  int32(autoscalerConfig.RepoServer.ReplicasMaximum.ValueInt64()),
				ReplicaMinimum:  int32(autoscalerConfig.RepoServer.ReplicasMinimum.ValueInt64()),
			}
		}
		if autoscalerConfig.ApplicationController != nil {
			if autoscalerConfig.ApplicationController.ResourceMaximum == nil || autoscalerConfig.ApplicationController.ResourceMinimum == nil {
				diagnostics.AddError("app controller autoscaler config requires minimum and maximum resources", "")
				return v1alpha1.ClusterData{}
			}
			if autoscalerConfig.ApplicationController.ResourceMinimum.Memory.ValueString() == "" || autoscalerConfig.ApplicationController.ResourceMinimum.Cpu.ValueString() == "" ||
				autoscalerConfig.ApplicationController.ResourceMaximum.Memory.ValueString() == "" || autoscalerConfig.ApplicationController.ResourceMaximum.Cpu.ValueString() == "" {
				diagnostics.AddError("app controller autoscaler config requires memory, cpu values", "")
				return v1alpha1.ClusterData{}
			}
			if !areResourcesValid(
				autoscalerConfig.ApplicationController.ResourceMinimum.Memory.ValueString(),
				autoscalerConfig.ApplicationController.ResourceMaximum.Memory.ValueString(),
			) {
				diagnostics.AddError("application controller minimum memory must be less than or equal to maximum memory", "")
				return v1alpha1.ClusterData{}
			}
			if !areResourcesValid(
				autoscalerConfig.ApplicationController.ResourceMinimum.Cpu.ValueString(),
				autoscalerConfig.ApplicationController.ResourceMaximum.Cpu.ValueString(),
			) {
				diagnostics.AddError("application controller minimum CPU must be less than or equal to maximum CPU", "")
				return v1alpha1.ClusterData{}
			}
			autoscalerConfigAPI.ApplicationController = &v1alpha1.AppControllerAutoScalingConfig{
				ResourceMinimum: toResourcesAPIModel(autoscalerConfig.ApplicationController.ResourceMinimum),
				ResourceMaximum: toResourcesAPIModel(autoscalerConfig.ApplicationController.ResourceMaximum),
			}
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
	return v1alpha1.ClusterData{
		Size:                            v1alpha1.ClusterSize(size),
		AutoUpgradeDisabled:             toBoolPointer(clusterData.AutoUpgradeDisabled),
		Kustomization:                   raw,
		AppReplication:                  toBoolPointer(clusterData.AppReplication),
		TargetVersion:                   clusterData.TargetVersion.ValueString(),
		RedisTunneling:                  toBoolPointer(clusterData.RedisTunneling),
		DatadogAnnotationsEnabled:       toBoolPointer(clusterData.DatadogAnnotationsEnabled),
		EksAddonEnabled:                 toBoolPointer(clusterData.EksAddonEnabled),
		ManagedClusterConfig:            managedConfig,
		MultiClusterK8SDashboardEnabled: toBoolPointer(clusterData.MultiClusterK8SDashboardEnabled),
		AutoscalerConfig:                autoscalerConfigAPI,
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

func toClusterCustomizationAPIModel(ctx context.Context, diagnostics *diag.Diagnostics, clusterCustomization tftypes.Object) *v1alpha1.ClusterCustomization {
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
		AutoUpgradeDisabled: toBoolPointer(customization.AutoUpgradeDisabled),
		Kustomization:       raw,
		AppReplication:      toBoolPointer(customization.AppReplication),
		RedisTunneling:      toBoolPointer(customization.RedisTunneling),
	}
}

func toIPAllowListAPIModel(entries []*IPAllowListEntry) []*v1alpha1.IPAllowListEntry {
	var ipAllowList []*v1alpha1.IPAllowListEntry
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

func toAppsetPolicyAPIModel(ctx context.Context, diagnostics *diag.Diagnostics, appsetPolicy tftypes.Object) *v1alpha1.AppsetPolicy {
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
	controlPlane := false
	if repoServerDelegate.ControlPlane != nil && *repoServerDelegate.ControlPlane {
		controlPlane = true
	}
	return &RepoServerDelegate{
		ControlPlane:   tftypes.BoolValue(controlPlane),
		ManagedCluster: toManagedClusterTFModel(repoServerDelegate.ManagedCluster),
	}
}

func toImageUpdaterDelegateTFModel(imageUpdaterDelegate *v1alpha1.ImageUpdaterDelegate) *ImageUpdaterDelegate {
	if imageUpdaterDelegate == nil {
		return nil
	}
	controlPlane := false
	if imageUpdaterDelegate.ControlPlane != nil && *imageUpdaterDelegate.ControlPlane {
		controlPlane = true
	}
	return &ImageUpdaterDelegate{
		ControlPlane:   tftypes.BoolValue(controlPlane),
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
		ClusterName: tftypes.StringValue(cluster.ClusterName),
	}
}

func (a *ArgoCD) toClusterCustomizationTFModel(ctx context.Context, diagnostics *diag.Diagnostics, customization *v1alpha1.ClusterCustomization) tftypes.Object {
	if customization == nil {
		return tftypes.ObjectNull(clusterCustomizationAttrTypes)
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

	autoUpgradeDisabled := false
	if customization.AutoUpgradeDisabled != nil && *customization.AutoUpgradeDisabled {
		autoUpgradeDisabled = true
	}
	appReplication := false
	if customization.AppReplication != nil && *customization.AppReplication {
		appReplication = true
	}
	redisTunneling := false
	if customization.RedisTunneling != nil && *customization.RedisTunneling {
		redisTunneling = true
	}
	c := &ClusterCustomization{
		AutoUpgradeDisabled: tftypes.BoolValue(autoUpgradeDisabled),
		Kustomization:       tftypes.StringValue(string(yamlData)),
		AppReplication:      tftypes.BoolValue(appReplication),
		RedisTunneling:      tftypes.BoolValue(redisTunneling),
	}
	clusterCustomization, d := tftypes.ObjectValueFrom(ctx, clusterCustomizationAttrTypes, c)
	diagnostics.Append(d...)
	if diagnostics.HasError() {
		return tftypes.ObjectNull(clusterCustomizationAttrTypes)
	}
	return clusterCustomization
}

func toIPAllowListTFModel(entries []*v1alpha1.IPAllowListEntry) []*IPAllowListEntry {
	var ipAllowList []*IPAllowListEntry
	for _, entry := range entries {
		ipAllowList = append(ipAllowList, &IPAllowListEntry{
			Ip:          tftypes.StringValue(entry.Ip),
			Description: tftypes.StringValue(entry.Description),
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

func toAppsetPolicyTFModel(ctx context.Context, diagnostics *diag.Diagnostics, appsetPolicy *v1alpha1.AppsetPolicy) tftypes.Object {
	if appsetPolicy == nil {
		return tftypes.ObjectNull(appsetPolicyAttrTypes)
	}

	overridePolicy := false
	if appsetPolicy.OverridePolicy != nil && *appsetPolicy.OverridePolicy {
		overridePolicy = true
	}
	a := &AppsetPolicy{
		Policy:         tftypes.StringValue(appsetPolicy.Policy),
		OverridePolicy: tftypes.BoolValue(overridePolicy),
	}
	policy, d := tftypes.ObjectValueFrom(ctx, appsetPolicyAttrTypes, a)
	diagnostics.Append(d...)
	if diagnostics.HasError() {
		return tftypes.ObjectNull(appsetPolicyAttrTypes)
	}
	return policy
}

func toHostAliasesTFModel(entries []*v1alpha1.HostAliases) []*HostAliases {
	var hostAliases []*HostAliases
	for _, entry := range entries {
		var hostnames []tftypes.String
		for _, hostname := range entry.Hostnames {
			hostnames = append(hostnames, tftypes.StringValue(hostname))
		}
		hostAliases = append(hostAliases, &HostAliases{
			Ip:        tftypes.StringValue(entry.Ip),
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
	var commands []tftypes.String
	for _, c := range dynamic.Command {
		commands = append(commands, tftypes.StringValue(c))
	}
	var args []tftypes.String
	for _, a := range dynamic.Args {
		args = append(args, tftypes.StringValue(a))
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
	var array []tftypes.String
	for _, a := range parameter.Array {
		array = append(array, tftypes.StringValue(a))
	}
	m, diag := tftypes.MapValueFrom(ctx, tftypes.StringType, &parameter.Map)
	diagnostics.Append(diag...)
	name := tftypes.StringNull()
	if parameter.Name != "" {
		name = tftypes.StringValue(parameter.Name)
	}
	title := tftypes.StringNull()
	if parameter.Title != "" {
		title = tftypes.StringValue(parameter.Title)
	}
	tooltip := tftypes.StringNull()
	if parameter.Tooltip != "" {
		tooltip = tftypes.StringValue(parameter.Tooltip)
	}
	itemType := tftypes.StringNull()
	if parameter.ItemType != "" {
		itemType = tftypes.StringValue(parameter.ItemType)
	}
	collectionType := tftypes.StringNull()
	if parameter.CollectionType != "" {
		collectionType = tftypes.StringValue(parameter.CollectionType)
	}
	string_ := tftypes.StringNull()
	if parameter.String_ != "" {
		string_ = tftypes.StringValue(parameter.String_)
	}
	return &ParameterAnnouncement{
		Name:           name,
		Title:          title,
		Tooltip:        tooltip,
		Required:       tftypes.BoolValue(parameter.Required),
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
	fileName := tftypes.StringNull()
	if discover.FileName != "" {
		fileName = tftypes.StringValue(discover.FileName)
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
	var commands []tftypes.String
	for _, c := range find.Command {
		commands = append(commands, tftypes.StringValue(c))
	}
	var args []tftypes.String
	for _, a := range find.Args {
		args = append(args, tftypes.StringValue(a))
	}
	glob := tftypes.StringNull()
	if find.Glob != "" {
		glob = tftypes.StringValue(find.Glob)
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
	var commands []tftypes.String
	for _, c := range command.Command {
		commands = append(commands, tftypes.StringValue(c))
	}
	var args []tftypes.String
	for _, a := range command.Args {
		args = append(args, tftypes.StringValue(a))
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
		Enabled: tftypes.BoolValue(*config.Enabled),
	}
}

func convertSlice[T any, U any](s []T, conv func(T) U) []U {
	var tfSlice []U
	for _, item := range s {
		tfSlice = append(tfSlice, conv(item))
	}
	return tfSlice
}

func stringToTFString(str string) tftypes.String {
	return tftypes.StringValue(str)
}

func tfStringToString(str tftypes.String) string {
	return str.ValueString()
}

func crossplaneExtensionResourceToTFModel(resource *v1alpha1.CrossplaneExtensionResource) *CrossplaneExtensionResource {
	if resource == nil {
		return nil
	}
	return &CrossplaneExtensionResource{
		Group: tftypes.StringValue(resource.Group),
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

func toAutoScalerConfigTFModel(cfg *argocdv1.AutoScalerConfig) basetypes.ObjectValue {
	attributeTypes := map[string]attr.Type{
		"application_controller": basetypes.ObjectType{
			AttrTypes: map[string]attr.Type{
				"resource_minimum": basetypes.ObjectType{
					AttrTypes: map[string]attr.Type{
						"memory": types.StringType,
						"cpu":    types.StringType,
					},
				},
				"resource_maximum": basetypes.ObjectType{
					AttrTypes: map[string]attr.Type{
						"memory": types.StringType,
						"cpu":    types.StringType,
					},
				},
			},
		},
		"repo_server": basetypes.ObjectType{
			AttrTypes: map[string]attr.Type{
				"resource_minimum": basetypes.ObjectType{
					AttrTypes: map[string]attr.Type{
						"memory": types.StringType,
						"cpu":    types.StringType,
					},
				},
				"resource_maximum": basetypes.ObjectType{
					AttrTypes: map[string]attr.Type{
						"memory": types.StringType,
						"cpu":    types.StringType,
					},
				},
				"replicas_maximum": types.Int64Type,
				"replicas_minimum": types.Int64Type,
			},
		},
	}

	attributes := map[string]attr.Value{}
	if cfg == nil {
		return basetypes.NewObjectNull(attributeTypes)
	}
	if cfg.ApplicationController != nil {
		attributes["application_controller"] = basetypes.NewObjectValueMust(
			attributeTypes["application_controller"].(basetypes.ObjectType).AttrTypes,
			map[string]attr.Value{
				"resource_minimum": basetypes.NewObjectValueMust(
					attributeTypes["application_controller"].(basetypes.ObjectType).AttrTypes["resource_minimum"].(basetypes.ObjectType).AttrTypes,
					map[string]attr.Value{
						"memory": basetypes.NewStringValue(cfg.ApplicationController.ResourceMinimum.Mem),
						"cpu":    basetypes.NewStringValue(cfg.ApplicationController.ResourceMinimum.Cpu),
					},
				),
				"resource_maximum": basetypes.NewObjectValueMust(
					attributeTypes["application_controller"].(basetypes.ObjectType).AttrTypes["resource_maximum"].(basetypes.ObjectType).AttrTypes,
					map[string]attr.Value{
						"memory": basetypes.NewStringValue(cfg.ApplicationController.ResourceMaximum.Mem),
						"cpu":    basetypes.NewStringValue(cfg.ApplicationController.ResourceMaximum.Cpu),
					},
				),
			})
	}
	if cfg.RepoServer != nil {
		attributes["repo_server"] = basetypes.NewObjectValueMust(
			attributeTypes["repo_server"].(basetypes.ObjectType).AttrTypes,
			map[string]attr.Value{
				"resource_minimum": basetypes.NewObjectValueMust(
					attributeTypes["repo_server"].(basetypes.ObjectType).AttrTypes["resource_minimum"].(basetypes.ObjectType).AttrTypes,
					map[string]attr.Value{
						"memory": basetypes.NewStringValue(cfg.RepoServer.ResourceMinimum.Mem),
						"cpu":    basetypes.NewStringValue(cfg.RepoServer.ResourceMinimum.Cpu),
					},
				),
				"resource_maximum": basetypes.NewObjectValueMust(
					attributeTypes["repo_server"].(basetypes.ObjectType).AttrTypes["resource_maximum"].(basetypes.ObjectType).AttrTypes,
					map[string]attr.Value{
						"memory": basetypes.NewStringValue(cfg.RepoServer.ResourceMaximum.Mem),
						"cpu":    basetypes.NewStringValue(cfg.RepoServer.ResourceMaximum.Cpu),
					},
				),
				"replicas_maximum": basetypes.NewInt64Value(int64(cfg.RepoServer.ReplicaMaximum)),
				"replicas_minimum": basetypes.NewInt64Value(int64(cfg.RepoServer.ReplicaMinimum)),
			},
		)
	}

	objectValue, diags := basetypes.NewObjectValue(attributeTypes, attributes)
	if diags.HasError() {
		return basetypes.NewObjectUnknown(attributeTypes)
	}
	return objectValue
}

func handleAgentSizeAndKustomization(diagnostics *diag.Diagnostics, clusterData *ClusterData, autoscalerConfig *AutoScalerConfig) (size string, kustomization runtime.RawExtension) {
	customSizeConfig := clusterData.CustomAgentSizeConfig
	if autoscalerConfig != nil && clusterData.Size.ValueString() != "auto" {
		diagnostics.AddError("autoscaler config should not be set when size is not auto", "")
		return clusterData.Size.ValueString(), runtime.RawExtension{}
	}
	if customSizeConfig == nil && clusterData.Size.ValueString() == "custom" {
		diagnostics.AddError("custom agent size config is required when size is custom", "")
		return clusterData.Size.ValueString(), runtime.RawExtension{}
	}
	if customSizeConfig != nil && clusterData.Size.ValueString() != "custom" {
		diagnostics.AddError("custom agent size config should not be set when size is not custom", "")
		return clusterData.Size.ValueString(), runtime.RawExtension{}
	}

	if clusterData.Size.ValueString() != "custom" {
		raw := runtime.RawExtension{}
		if clusterData.Kustomization.ValueString() != "" {
			if err := yaml.Unmarshal([]byte(clusterData.Kustomization.ValueString()), &raw); err != nil {
				diagnostics.AddError("failed unmarshal kustomization string to yaml", err.Error())
				return clusterData.Size.ValueString(), runtime.RawExtension{}
			}
		}
		return clusterData.Size.ValueString(), raw
	}

	if customSizeConfig.ApplicationController != nil {
		if customSizeConfig.ApplicationController.Memory.ValueString() == "" || customSizeConfig.ApplicationController.Cpu.ValueString() == "" {
			diagnostics.AddError("memory and cpu are required for app controller custom size", "")
			return clusterData.Size.ValueString(), runtime.RawExtension{}
		}
	}
	if customSizeConfig.RepoServer != nil {
		if customSizeConfig.RepoServer.Memory.ValueString() == "" || customSizeConfig.RepoServer.Cpu.ValueString() == "" || customSizeConfig.RepoServer.Replicas.ValueInt64() == 0 {
			diagnostics.AddError("memory, cpu and replicas are required for repo server custom size", "")
			return clusterData.Size.ValueString(), runtime.RawExtension{}
		} else if customSizeConfig.RepoServer.Replicas.ValueInt64() < 0 {
			diagnostics.AddError("replicas must be greater than or equal to 0", "")
			return clusterData.Size.ValueString(), runtime.RawExtension{}
		}
	}

	expectedKustomization, err := generateExpectedKustomization(customSizeConfig, clusterData.Kustomization.ValueString())
	if err != nil {
		diagnostics.AddError("failed to generate expected kustomization", err.Error())
		return clusterData.Size.ValueString(), runtime.RawExtension{}
	}

	raw := runtime.RawExtension{}
	if err := yaml.Unmarshal([]byte(expectedKustomization), &raw); err != nil {
		diagnostics.AddError("failed unmarshal kustomization string to yaml", err.Error())
		return clusterData.Size.ValueString(), runtime.RawExtension{}
	}

	return "large", raw
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

func areAutoScalerConfigsEquivalent(plan, now *AutoScalerConfig) bool {
	if plan == nil {
		return true
	}
	if plan.ApplicationController != nil && now.ApplicationController != nil {
		if !areResourcesEquivalent(
			plan.ApplicationController.ResourceMinimum.Cpu.ValueString(),
			now.ApplicationController.ResourceMinimum.Cpu.ValueString(),
		) || !areResourcesEquivalent(
			plan.ApplicationController.ResourceMinimum.Memory.ValueString(),
			now.ApplicationController.ResourceMinimum.Memory.ValueString(),
		) || !areResourcesEquivalent(
			plan.ApplicationController.ResourceMaximum.Cpu.ValueString(),
			now.ApplicationController.ResourceMaximum.Cpu.ValueString(),
		) || !areResourcesEquivalent(
			plan.ApplicationController.ResourceMaximum.Memory.ValueString(),
			now.ApplicationController.ResourceMaximum.Memory.ValueString(),
		) {
			return false
		}
	}
	if plan.RepoServer != nil && now.RepoServer != nil {
		if !areResourcesEquivalent(
			plan.RepoServer.ResourceMinimum.Cpu.ValueString(),
			now.RepoServer.ResourceMinimum.Cpu.ValueString(),
		) || !areResourcesEquivalent(
			plan.RepoServer.ResourceMinimum.Memory.ValueString(),
			now.RepoServer.ResourceMinimum.Memory.ValueString(),
		) || !areResourcesEquivalent(
			plan.RepoServer.ResourceMaximum.Cpu.ValueString(),
			now.RepoServer.ResourceMaximum.Cpu.ValueString(),
		) || !areResourcesEquivalent(
			plan.RepoServer.ResourceMaximum.Memory.ValueString(),
			now.RepoServer.ResourceMaximum.Memory.ValueString(),
		) || plan.RepoServer.ReplicasMaximum != now.RepoServer.ReplicasMaximum ||
			plan.RepoServer.ReplicasMinimum != now.RepoServer.ReplicasMinimum {
			return false
		}
	}
	return true
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

func areResourcesValid(min, max string) bool {
	minQ, err1 := resource.ParseQuantity(min)
	maxQ, err2 := resource.ParseQuantity(max)
	if err1 != nil || err2 != nil {
		return true
	}
	return minQ.Cmp(maxQ) <= 0
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
	if config == nil {
		return nil
	}
	return &v1alpha1.AppInAnyNamespaceConfig{
		Enabled: config.Enabled.ValueBoolPointer(),
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
			if isResourcePatch(patch) && customConfig != nil {
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
