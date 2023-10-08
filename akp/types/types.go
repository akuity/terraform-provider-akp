package types

import (
	"bytes"
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"

	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	"github.com/akuity/terraform-provider-akp/akp/apis/akuity/v1alpha1"
	argocdv1alpha1 "github.com/akuity/terraform-provider-akp/akp/apis/argocd/v1alpha1"
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
		argocdv1.ClusterSize_CLUSTER_SIZE_UNSPECIFIED: "unspecified",
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
	a.Spec = ArgoCDSpec{
		Description: tftypes.StringValue(cd.Spec.Description),
		Version:     tftypes.StringValue(cd.Spec.Version),
		InstanceSpec: InstanceSpec{
			IpAllowList:                  toIPAllowListTFModel(cd.Spec.InstanceSpec.IpAllowList),
			Subdomain:                    tftypes.StringValue(cd.Spec.InstanceSpec.Subdomain),
			DeclarativeManagementEnabled: tftypes.BoolValue(declarativeManagementEnabled),
			Extensions:                   toExtensionsTFModel(cd.Spec.InstanceSpec.Extensions),
			ClusterCustomizationDefaults: toClusterCustomizationTFModel(ctx, diagnostics, cd.Spec.InstanceSpec.ClusterCustomizationDefaults),
			ImageUpdaterEnabled:          tftypes.BoolValue(imageUpdaterEnabled),
			BackendIpAllowListEnabled:    tftypes.BoolValue(backendIpAllowListEnabled),
			RepoServerDelegate:           toRepoServerDelegateTFModel(cd.Spec.InstanceSpec.RepoServerDelegate),
			AuditExtensionEnabled:        tftypes.BoolValue(auditExtensionEnabled),
			SyncHistoryExtensionEnabled:  tftypes.BoolValue(syncHistoryExtensionEnabled),
			ImageUpdaterDelegate:         toImageUpdaterDelegateTFModel(cd.Spec.InstanceSpec.ImageUpdaterDelegate),
			AppSetDelegate:               toAppSetDelegateTFModel(cd.Spec.InstanceSpec.AppSetDelegate),
			AssistantExtensionEnabled:    tftypes.BoolValue(assistantExtensionEnabled),
			AppsetPolicy:                 toAppsetPolicyTFModel(ctx, diagnostics, cd.Spec.InstanceSpec.AppsetPolicy),
			HostAliases:                  toHostAliasesTFModel(cd.Spec.InstanceSpec.HostAliases),
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
				IpAllowList:                  toIPAllowListAPIModel(a.Spec.InstanceSpec.IpAllowList),
				Subdomain:                    a.Spec.InstanceSpec.Subdomain.ValueString(),
				DeclarativeManagementEnabled: a.Spec.InstanceSpec.DeclarativeManagementEnabled.ValueBoolPointer(),
				Extensions:                   toExtensionsAPIModel(a.Spec.InstanceSpec.Extensions),
				ClusterCustomizationDefaults: toClusterCustomizationAPIModel(ctx, diag, a.Spec.InstanceSpec.ClusterCustomizationDefaults),
				ImageUpdaterEnabled:          a.Spec.InstanceSpec.ImageUpdaterEnabled.ValueBoolPointer(),
				BackendIpAllowListEnabled:    a.Spec.InstanceSpec.BackendIpAllowListEnabled.ValueBoolPointer(),
				RepoServerDelegate:           toRepoServerDelegateAPIModel(a.Spec.InstanceSpec.RepoServerDelegate),
				AuditExtensionEnabled:        a.Spec.InstanceSpec.AuditExtensionEnabled.ValueBoolPointer(),
				SyncHistoryExtensionEnabled:  a.Spec.InstanceSpec.SyncHistoryExtensionEnabled.ValueBoolPointer(),
				ImageUpdaterDelegate:         toImageUpdaterDelegateAPIModel(a.Spec.InstanceSpec.ImageUpdaterDelegate),
				AppSetDelegate:               toAppSetDelegateAPIModel(a.Spec.InstanceSpec.AppSetDelegate),
				AssistantExtensionEnabled:    a.Spec.InstanceSpec.AssistantExtensionEnabled.ValueBoolPointer(),
				AppsetPolicy:                 toAppsetPolicyAPIModel(ctx, diag, a.Spec.InstanceSpec.AppsetPolicy),
				HostAliases:                  toHostAliasesAPIModel(a.Spec.InstanceSpec.HostAliases),
			},
		},
	}
}

func (c *Cluster) Update(ctx context.Context, diagnostics *diag.Diagnostics, apiCluster *argocdv1.Cluster) {
	c.ID = tftypes.StringValue(apiCluster.GetId())
	c.Name = tftypes.StringValue(apiCluster.GetName())
	c.Namespace = tftypes.StringValue(apiCluster.GetNamespace())
	if c.RemoveAgentResourcesOnDestroy.IsUnknown() || c.RemoveAgentResourcesOnDestroy.IsNull() {
		c.RemoveAgentResourcesOnDestroy = tftypes.BoolValue(true)
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
		diagnostics.AddError("getting cluster kustomization", fmt.Sprintf("%s", err.Error()))
	}
	yamlData, err := yaml.JSONToYAML(jsonData)
	if err != nil {
		diagnostics.AddError("getting cluster kustomization", fmt.Sprintf("%s", err.Error()))
	}

	kustomization := tftypes.StringValue(string(yamlData))
	if c.Spec != nil {
		rawPlan := runtime.RawExtension{}
		old := c.Spec.Data.Kustomization
		if err := yaml.Unmarshal([]byte(old.ValueString()), &rawPlan); err != nil {
			diagnostics.AddError("failed unmarshal kustomization string to yaml", err.Error())
		}

		oldYamlData, err := yaml.Marshal(&rawPlan)
		if err != nil {
			diagnostics.AddError("failed to convert json to yaml data", err.Error())
		}
		if bytes.Equal(oldYamlData, yamlData) {
			kustomization = old
		}
	}

	c.Labels = labels
	c.Annotations = annotations
	c.Spec = &ClusterSpec{
		Description:     tftypes.StringValue(apiCluster.GetDescription()),
		NamespaceScoped: tftypes.BoolValue(apiCluster.GetNamespaceScoped()),
		Data: ClusterData{
			Size:                tftypes.StringValue(ClusterSizeString[apiCluster.GetData().GetSize()]),
			AutoUpgradeDisabled: tftypes.BoolValue(apiCluster.GetData().GetAutoUpgradeDisabled()),
			Kustomization:       kustomization,
			AppReplication:      tftypes.BoolValue(apiCluster.GetData().GetAppReplication()),
			TargetVersion:       tftypes.StringValue(apiCluster.GetData().GetTargetVersion()),
			RedisTunneling:      tftypes.BoolValue(apiCluster.GetData().GetRedisTunneling()),
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
			Data:            toClusterDataAPIModel(diagnostics, c.Spec.Data),
		},
	}
}

func (c *ConfigManagementPlugin) Update(ctx context.Context, diagnostics *diag.Diagnostics, cmp *argocdv1alpha1.ConfigManagementPlugin) {
	version := tftypes.StringNull()
	if cmp.Spec.Version != "" {
		version = tftypes.StringValue(cmp.Spec.Version)
	}
	c.Name = tftypes.StringValue(cmp.Name)
	c.Enabled = tftypes.BoolValue(cmp.Annotations[argocdv1alpha1.AnnotationCMPEnabled] == "true")
	c.Image = types.StringValue(cmp.Annotations[argocdv1alpha1.AnnotationCMPImage])
	c.Spec = &PluginSpec{
		Version:          version,
		Init:             toCommandTFModel(cmp.Spec.Init),
		Generate:         toCommandTFModel(cmp.Spec.Generate),
		Discover:         toDiscoverTFModel(cmp.Spec.Discover),
		Parameters:       toParametersTFModel(ctx, diagnostics, cmp.Spec.Parameters),
		PreserveFileMode: tftypes.BoolValue(cmp.Spec.PreserveFileMode),
	}
}

func (c *ConfigManagementPlugin) ToConfigManagementPluginAPIModel(ctx context.Context, diagnostics *diag.Diagnostics) *argocdv1alpha1.ConfigManagementPlugin {
	return &argocdv1alpha1.ConfigManagementPlugin{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigManagementPlugin",
			APIVersion: "argoproj.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: c.Name.ValueString(),
			Annotations: map[string]string{
				argocdv1alpha1.AnnotationCMPImage:   c.Image.ValueString(),
				argocdv1alpha1.AnnotationCMPEnabled: strconv.FormatBool(c.Enabled.ValueBool()),
			},
		},
		Spec: argocdv1alpha1.PluginSpec{
			Version:          c.Spec.Version.ValueString(),
			Init:             toCommandAPIModel(c.Spec.Init),
			Generate:         toCommandAPIModel(c.Spec.Generate),
			Discover:         toDiscoverAPIModel(c.Spec.Discover),
			Parameters:       toParametersAPIModel(ctx, diagnostics, c.Spec.Parameters),
			PreserveFileMode: c.Spec.PreserveFileMode.ValueBool(),
		},
	}
}

func toClusterDataAPIModel(diagnostics *diag.Diagnostics, clusterData ClusterData) v1alpha1.ClusterData {
	raw := runtime.RawExtension{}
	if err := yaml.Unmarshal([]byte(clusterData.Kustomization.ValueString()), &raw); err != nil {
		diagnostics.AddError("failed unmarshal kustomization string to yaml", err.Error())
	}
	return v1alpha1.ClusterData{
		Size:                v1alpha1.ClusterSize(clusterData.Size.ValueString()),
		AutoUpgradeDisabled: clusterData.AutoUpgradeDisabled.ValueBoolPointer(),
		Kustomization:       raw,
		AppReplication:      clusterData.AppReplication.ValueBoolPointer(),
		TargetVersion:       clusterData.TargetVersion.ValueString(),
		RedisTunneling:      clusterData.RedisTunneling.ValueBoolPointer(),
	}
}

func toRepoServerDelegateAPIModel(repoServerDelegate *RepoServerDelegate) *v1alpha1.RepoServerDelegate {
	if repoServerDelegate == nil {
		return nil
	}
	return &v1alpha1.RepoServerDelegate{
		ControlPlane:   repoServerDelegate.ControlPlane.ValueBoolPointer(),
		ManagedCluster: toManagedClusterAPIModel(repoServerDelegate.ManagedCluster),
	}
}

func toImageUpdaterDelegateAPIModel(imageUpdaterDelegate *ImageUpdaterDelegate) *v1alpha1.ImageUpdaterDelegate {
	if imageUpdaterDelegate == nil {
		return nil
	}
	return &v1alpha1.ImageUpdaterDelegate{
		ControlPlane:   imageUpdaterDelegate.ControlPlane.ValueBoolPointer(),
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
		AutoUpgradeDisabled: customization.AutoUpgradeDisabled.ValueBoolPointer(),
		Kustomization:       raw,
		AppReplication:      customization.AppReplication.ValueBoolPointer(),
		RedisTunneling:      customization.RedisTunneling.ValueBoolPointer(),
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

func toExtensionsAPIModel(entries []*ArgoCDExtensionInstallEntry) []*v1alpha1.ArgoCDExtensionInstallEntry {
	var extensions []*v1alpha1.ArgoCDExtensionInstallEntry
	for _, entry := range entries {
		extensions = append(extensions, &v1alpha1.ArgoCDExtensionInstallEntry{
			Id:      entry.Id.ValueString(),
			Version: entry.Version.ValueString(),
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
		OverridePolicy: policy.OverridePolicy.ValueBoolPointer(),
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

func toParametersAPIModel(ctx context.Context, diagnostics *diag.Diagnostics, parameters *Parameters) *argocdv1alpha1.Parameters {
	if parameters == nil {
		return nil
	}
	var static []*argocdv1alpha1.ParameterAnnouncement
	for _, s := range parameters.Static {
		var array []string
		for _, a := range s.Array {
			array = append(array, a.ValueString())
		}
		var m map[string]string
		diagnostics.Append(s.Map.ElementsAs(ctx, &m, true)...)

		static = append(static, &argocdv1alpha1.ParameterAnnouncement{
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
	return &argocdv1alpha1.Parameters{
		Static:  static,
		Dynamic: toDynamicAPIModel(parameters.Dynamic),
	}
}

func toDynamicAPIModel(dynamic *Dynamic) *argocdv1alpha1.Dynamic {
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
	return &argocdv1alpha1.Dynamic{
		Command: commands,
		Args:    args,
	}
}

func toDiscoverAPIModel(discover *Discover) *argocdv1alpha1.Discover {
	if discover == nil {
		return nil
	}
	return &argocdv1alpha1.Discover{
		Find:     toFindAPIModel(discover.Find),
		FileName: discover.FileName.ValueString(),
	}
}

func toFindAPIModel(find *Find) *argocdv1alpha1.Find {
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
	return &argocdv1alpha1.Find{
		Command: commands,
		Args:    args,
		Glob:    find.Glob.ValueString(),
	}
}

func toCommandAPIModel(command *Command) *argocdv1alpha1.Command {
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
	return &argocdv1alpha1.Command{
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

func toClusterCustomizationTFModel(ctx context.Context, diagnostics *diag.Diagnostics, customization *v1alpha1.ClusterCustomization) tftypes.Object {
	if customization == nil {
		return tftypes.ObjectNull(clusterCustomizationAttrTypes)
	}
	yamlData, err := yaml.JSONToYAML(customization.Kustomization.Raw)
	if err != nil {
		diagnostics.AddError("failed to convert json to yaml", err.Error())
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

func toExtensionsTFModel(entries []*v1alpha1.ArgoCDExtensionInstallEntry) []*ArgoCDExtensionInstallEntry {
	var extensions []*ArgoCDExtensionInstallEntry
	for _, entry := range entries {
		extensions = append(extensions, &ArgoCDExtensionInstallEntry{
			Id:      tftypes.StringValue(entry.Id),
			Version: tftypes.StringValue(entry.Version),
		})
	}
	return extensions
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

func toParametersTFModel(ctx context.Context, diagnostics *diag.Diagnostics, parameters *argocdv1alpha1.Parameters) *Parameters {
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

func toDynamicTFModel(dynamic *argocdv1alpha1.Dynamic) *Dynamic {
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

func toParameterAnnouncementTFModel(ctx context.Context, diagnostics *diag.Diagnostics, parameter *argocdv1alpha1.ParameterAnnouncement) *ParameterAnnouncement {
	if parameter == nil {
		return nil
	}
	var array []tftypes.String
	for _, a := range parameter.Array {
		array = append(array, tftypes.StringValue(a))
	}
	m, diag := tftypes.MapValueFrom(ctx, tftypes.StringType, &parameter.Map)
	diagnostics.Append(diag...)
	return &ParameterAnnouncement{
		Name:           tftypes.StringValue(parameter.Name),
		Title:          tftypes.StringValue(parameter.Title),
		Tooltip:        tftypes.StringValue(parameter.Tooltip),
		Required:       tftypes.BoolValue(parameter.Required),
		ItemType:       tftypes.StringValue(parameter.ItemType),
		CollectionType: tftypes.StringValue(parameter.CollectionType),
		String_:        tftypes.StringValue(parameter.String_),
		Array:          array,
		Map:            m,
	}
}

func toDiscoverTFModel(discover *argocdv1alpha1.Discover) *Discover {
	if discover == nil {
		return nil
	}
	return &Discover{
		Find:     toFindTFModel(discover.Find),
		FileName: tftypes.StringValue(discover.FileName),
	}
}

func toFindTFModel(find *argocdv1alpha1.Find) *Find {
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
	return &Find{
		Command: commands,
		Args:    args,
		Glob:    tftypes.StringValue(find.Glob),
	}
}

func toCommandTFModel(command *argocdv1alpha1.Command) *Command {
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
