package types

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"

	"github.com/akuity/terraform-provider-akp/akp/apis/v1alpha1"
)

var (
	clusterCustomizationAttrTypes = map[string]attr.Type{
		"auto_upgrade_disabled": tftypes.BoolType,
		"kustomization":         tftypes.StringType,
		"app_replication":       tftypes.BoolType,
		"redis_tunneling":       tftypes.BoolType,
	}
)

func ToConfigMapAPIModel(ctx context.Context, diagnostics *diag.Diagnostics, configMap *ConfigMap, name string) *v1.ConfigMap {
	var data map[string]string
	diagnostics.Append(configMap.Data.ElementsAs(ctx, &data, true)...)
	return &v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Data: data,
	}
}

func ToSecretAPIModel(ctx context.Context, diagnostics *diag.Diagnostics, secret *Secret, name string) *v1.Secret {
	var labels map[string]string
	var data map[string][]byte
	var stringData map[string]string
	diagnostics.Append(secret.Labels.ElementsAs(ctx, &labels, true)...)
	diagnostics.Append(secret.Data.ElementsAs(ctx, &data, true)...)
	diagnostics.Append(secret.StringData.ElementsAs(ctx, &stringData, true)...)
	n := name
	if !secret.Name.IsNull() && !secret.Name.IsUnknown() {
		n = secret.Name.ValueString()
	}
	return &v1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   n,
			Labels: labels,
		},
		Data:       data,
		StringData: stringData,
	}
}

func ToClusterAPIModel(ctx context.Context, diagnostics *diag.Diagnostics, cluster *Cluster) *v1alpha1.Cluster {
	var labels map[string]string
	var annotations map[string]string
	diagnostics.Append(cluster.Labels.ElementsAs(ctx, &labels, true)...)
	diagnostics.Append(cluster.Annotations.ElementsAs(ctx, &annotations, true)...)
	return &v1alpha1.Cluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Cluster",
			APIVersion: "argocd.akuity.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        cluster.Name.ValueString(),
			Namespace:   cluster.Namespace.ValueString(),
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: v1alpha1.ClusterSpec{
			Description:     cluster.Spec.Description.ValueString(),
			NamespaceScoped: cluster.Spec.NamespaceScoped.ValueBool(),
			Data:            toClusterDataAPIModel(diagnostics, cluster.Spec.Data),
		},
	}
}

func ToArgoCDAPIModel(ctx context.Context, diag *diag.Diagnostics, cd *ArgoCD) *v1alpha1.ArgoCD {
	return &v1alpha1.ArgoCD{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ArgoCD",
			APIVersion: "argocd.akuity.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: cd.Name.ValueString(),
		},
		Spec: v1alpha1.ArgoCDSpec{
			Description: cd.Spec.Description.ValueString(),
			Version:     cd.Spec.Version.ValueString(),
			InstanceSpec: v1alpha1.InstanceSpec{
				IpAllowList:                  toIPAllowListAPIModel(cd.Spec.InstanceSpec.IpAllowList),
				Subdomain:                    cd.Spec.InstanceSpec.Subdomain.ValueString(),
				DeclarativeManagementEnabled: cd.Spec.InstanceSpec.DeclarativeManagementEnabled.ValueBool(),
				Extensions:                   toExtensionsAPIModel(cd.Spec.InstanceSpec.Extensions),
				ClusterCustomizationDefaults: toClusterCustomizationAPIModel(ctx, diag, cd.Spec.InstanceSpec.ClusterCustomizationDefaults),
				ImageUpdaterEnabled:          cd.Spec.InstanceSpec.ImageUpdaterEnabled.ValueBool(),
				BackendIpAllowListEnabled:    cd.Spec.InstanceSpec.BackendIpAllowListEnabled.ValueBool(),
				RepoServerDelegate:           toRepoServerDelegateAPIModel(cd.Spec.InstanceSpec.RepoServerDelegate),
				AuditExtensionEnabled:        cd.Spec.InstanceSpec.AuditExtensionEnabled.ValueBool(),
				SyncHistoryExtensionEnabled:  cd.Spec.InstanceSpec.SyncHistoryExtensionEnabled.ValueBool(),
				ImageUpdaterDelegate:         toImageUpdaterDelegateAPIModel(cd.Spec.InstanceSpec.ImageUpdaterDelegate),
				AppSetDelegate:               toAppSetDelegateAPIModel(cd.Spec.InstanceSpec.AppSetDelegate),
			},
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
		RedisTunneling:      clusterData.RedisTunneling.ValueBool(),
	}
}

func toRepoServerDelegateAPIModel(repoServerDelegate *RepoServerDelegate) *v1alpha1.RepoServerDelegate {
	if repoServerDelegate == nil {
		return nil
	}
	return &v1alpha1.RepoServerDelegate{
		ControlPlane:   repoServerDelegate.ControlPlane.ValueBool(),
		ManagedCluster: toManagedClusterAPIModel(repoServerDelegate.ManagedCluster),
	}
}

func toImageUpdaterDelegateAPIModel(imageUpdaterDelegate *ImageUpdaterDelegate) *v1alpha1.ImageUpdaterDelegate {
	if imageUpdaterDelegate == nil {
		return nil
	}
	return &v1alpha1.ImageUpdaterDelegate{
		ControlPlane:   imageUpdaterDelegate.ControlPlane.ValueBool(),
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
	diagnostics.Append(clusterCustomization.As(ctx, &customization, basetypes.ObjectAsOptions{})...)
	raw := runtime.RawExtension{}
	if err := yaml.Unmarshal([]byte(customization.Kustomization.ValueString()), &raw); err != nil {
		diagnostics.AddError("failed unmarshal kustomization string to yaml", err.Error())
	}
	return &v1alpha1.ClusterCustomization{
		AutoUpgradeDisabled: customization.AutoUpgradeDisabled.ValueBool(),
		Kustomization:       raw,
		AppReplication:      customization.AppReplication.ValueBool(),
		RedisTunneling:      customization.RedisTunneling.ValueBool(),
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

func ToConfigMapTFModel(ctx context.Context, diagnostics *diag.Diagnostics, configMap *v1.ConfigMap) ConfigMap {
	data, diag := tftypes.MapValueFrom(ctx, tftypes.StringType, &configMap.Data)
	diagnostics.Append(diag...)
	return ConfigMap{
		Data: data,
	}
}

func ToClusterTFModel(ctx context.Context, diagnostics *diag.Diagnostics, cluster *v1alpha1.Cluster) Cluster {
	labels, diag := tftypes.MapValueFrom(ctx, tftypes.StringType, &cluster.Labels)
	diagnostics.Append(diag...)
	annotations, diag := tftypes.MapValueFrom(ctx, tftypes.StringType, &cluster.Annotations)
	diagnostics.Append(diag...)
	return Cluster{
		Name:        tftypes.StringValue(cluster.Name),
		Namespace:   tftypes.StringValue(cluster.Namespace),
		Labels:      labels,
		Annotations: annotations,
		Spec: ClusterSpec{
			Description:     tftypes.StringValue(cluster.Spec.Description),
			NamespaceScoped: tftypes.BoolValue(cluster.Spec.NamespaceScoped),
			Data:            toClusterDataTFModel(cluster.Spec.Data),
		},
	}
}

func ToArgoCDTFModel(ctx context.Context, diagnostics *diag.Diagnostics, cd *v1alpha1.ArgoCD) ArgoCD {
	return ArgoCD{
		Name: tftypes.StringValue(cd.Name),
		Spec: ArgoCDSpec{
			Description: tftypes.StringValue(cd.Spec.Description),
			Version:     tftypes.StringValue(cd.Spec.Version),
			InstanceSpec: InstanceSpec{
				IpAllowList:                  toIPAllowListTFModel(cd.Spec.InstanceSpec.IpAllowList),
				Subdomain:                    tftypes.StringValue(cd.Spec.InstanceSpec.Subdomain),
				DeclarativeManagementEnabled: tftypes.BoolValue(cd.Spec.InstanceSpec.DeclarativeManagementEnabled),
				Extensions:                   toExtensionsTFModel(cd.Spec.InstanceSpec.Extensions),
				ClusterCustomizationDefaults: toClusterCustomizationTFModel(ctx, diagnostics, cd.Spec.InstanceSpec.ClusterCustomizationDefaults),
				ImageUpdaterEnabled:          tftypes.BoolValue(cd.Spec.InstanceSpec.ImageUpdaterEnabled),
				BackendIpAllowListEnabled:    tftypes.BoolValue(cd.Spec.InstanceSpec.BackendIpAllowListEnabled),
				RepoServerDelegate:           toRepoServerDelegateTFModel(cd.Spec.InstanceSpec.RepoServerDelegate),
				AuditExtensionEnabled:        tftypes.BoolValue(cd.Spec.InstanceSpec.AuditExtensionEnabled),
				SyncHistoryExtensionEnabled:  tftypes.BoolValue(cd.Spec.InstanceSpec.SyncHistoryExtensionEnabled),
				ImageUpdaterDelegate:         toImageUpdaterDelegateTFModel(cd.Spec.InstanceSpec.ImageUpdaterDelegate),
				AppSetDelegate:               toAppSetDelegateTFModel(cd.Spec.InstanceSpec.AppSetDelegate),
			},
		},
	}
}

func toClusterDataTFModel(clusterData v1alpha1.ClusterData) ClusterData {
	yamlData, err := yaml.JSONToYAML(clusterData.Kustomization.Raw)
	if err != nil {
		fmt.Printf("Error converting JSON to YAML: %v", err)
	}
	return ClusterData{
		Size:                tftypes.StringValue(string(clusterData.Size)),
		AutoUpgradeDisabled: tftypes.BoolPointerValue(clusterData.AutoUpgradeDisabled),
		Kustomization:       tftypes.StringValue(string(yamlData)),
		AppReplication:      tftypes.BoolPointerValue(clusterData.AppReplication),
		TargetVersion:       tftypes.StringValue(clusterData.TargetVersion),
		RedisTunneling:      tftypes.BoolValue(clusterData.RedisTunneling),
	}
}

func toRepoServerDelegateTFModel(repoServerDelegate *v1alpha1.RepoServerDelegate) *RepoServerDelegate {
	if repoServerDelegate == nil {
		return nil
	}
	return &RepoServerDelegate{
		ControlPlane:   tftypes.BoolValue(repoServerDelegate.ControlPlane),
		ManagedCluster: toManagedClusterTFModel(repoServerDelegate.ManagedCluster),
	}
}

func toImageUpdaterDelegateTFModel(imageUpdaterDelegate *v1alpha1.ImageUpdaterDelegate) *ImageUpdaterDelegate {
	if imageUpdaterDelegate == nil {
		return nil
	}
	return &ImageUpdaterDelegate{
		ControlPlane:   tftypes.BoolValue(imageUpdaterDelegate.ControlPlane),
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
	c := ClusterCustomization{
		AutoUpgradeDisabled: tftypes.BoolValue(customization.AutoUpgradeDisabled),
		Kustomization:       tftypes.StringValue(string(yamlData)),
		AppReplication:      tftypes.BoolValue(customization.AppReplication),
		RedisTunneling:      tftypes.BoolValue(customization.RedisTunneling),
	}
	clusterCustomization, d := tftypes.ObjectValueFrom(ctx, clusterCustomizationAttrTypes, c)
	diagnostics.Append(d...)
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
