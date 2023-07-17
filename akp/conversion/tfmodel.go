package conversion

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"

	"github.com/akuity/terraform-provider-akp/akp/apis/v1alpha1"
	"github.com/akuity/terraform-provider-akp/akp/types"
)

func ToConfigMapTFModel(ctx context.Context, diagnostics diag.Diagnostics, configMap *v1.ConfigMap) *types.ConfigMap {
	data, diag := tftypes.MapValueFrom(ctx, tftypes.StringType, &configMap.Data)
	diagnostics.Append(diag...)
	return &types.ConfigMap{
		ObjectMeta: types.ObjectMeta{
			Name: tftypes.StringValue(configMap.Name),
		},
		Data: data,
	}
}

func ToClusterTFModel(ctx context.Context, diagnostics diag.Diagnostics, cluster *v1alpha1.Cluster) *types.Cluster {
	labels, diag := tftypes.MapValueFrom(ctx, tftypes.StringType, &cluster.Labels)
	diagnostics.Append(diag...)
	annotations, diag := tftypes.MapValueFrom(ctx, tftypes.StringType, &cluster.Annotations)
	diagnostics.Append(diag...)
	return &types.Cluster{
		ClusterObjectMeta: types.ClusterObjectMeta{
			Name:        tftypes.StringValue(cluster.Name),
			Namespace:   tftypes.StringValue(cluster.Namespace),
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: types.ClusterSpec{
			Description:     tftypes.StringValue(cluster.Spec.Description),
			NamespaceScoped: tftypes.BoolValue(cluster.Spec.NamespaceScoped),
			Data:            ToClusterDataTFModel(cluster.Spec.Data),
		},
	}
}

func ToArgoCDTFModel(ctx context.Context, diagnostics diag.Diagnostics, cd *v1alpha1.ArgoCD) *types.ArgoCD {
	return &types.ArgoCD{
		ObjectMeta: types.ObjectMeta{
			Name: tftypes.StringValue(cd.Name),
		},
		Spec: types.ArgoCDSpec{
			Description: tftypes.StringValue(cd.Spec.Description),
			Version:     tftypes.StringValue(cd.Spec.Version),
			InstanceSpec: types.InstanceSpec{
				IpAllowList:                  ToIPAllowListTFModel(cd.Spec.InstanceSpec.IpAllowList),
				Subdomain:                    tftypes.StringValue(cd.Spec.InstanceSpec.Subdomain),
				DeclarativeManagementEnabled: tftypes.BoolValue(cd.Spec.InstanceSpec.DeclarativeManagementEnabled),
				Extensions:                   ToExtensionsTFModel(cd.Spec.InstanceSpec.Extensions),
				ClusterCustomizationDefaults: ToClusterCustomizationTFModel(ctx, diagnostics, cd.Spec.InstanceSpec.ClusterCustomizationDefaults),
				ImageUpdaterEnabled:          tftypes.BoolValue(cd.Spec.InstanceSpec.ImageUpdaterEnabled),
				BackendIpAllowListEnabled:    tftypes.BoolValue(cd.Spec.InstanceSpec.BackendIpAllowListEnabled),
				RepoServerDelegate:           ToRepoServerDelegateTFModel(cd.Spec.InstanceSpec.RepoServerDelegate),
				AuditExtensionEnabled:        tftypes.BoolValue(cd.Spec.InstanceSpec.AuditExtensionEnabled),
				SyncHistoryExtensionEnabled:  tftypes.BoolValue(cd.Spec.InstanceSpec.SyncHistoryExtensionEnabled),
				ImageUpdaterDelegate:         ToImageUpdaterDelegateTFModel(cd.Spec.InstanceSpec.ImageUpdaterDelegate),
				AppSetDelegate:               ToAppSetDelegateTFModel(cd.Spec.InstanceSpec.AppSetDelegate),
			},
		},
	}
}

func ToClusterDataTFModel(clusterData v1alpha1.ClusterData) types.ClusterData {
	yamlData, err := yaml.JSONToYAML(clusterData.Kustomization.Raw)
	if err != nil {
		fmt.Printf("Error converting JSON to YAML: %v", err)
	}
	return types.ClusterData{
		Size:                tftypes.StringValue(string(clusterData.Size)),
		AutoUpgradeDisabled: tftypes.BoolPointerValue(clusterData.AutoUpgradeDisabled),
		Kustomization:       tftypes.StringValue(string(yamlData)),
		AppReplication:      tftypes.BoolPointerValue(clusterData.AppReplication),
		TargetVersion:       tftypes.StringValue(clusterData.TargetVersion),
		RedisTunneling:      tftypes.BoolValue(clusterData.RedisTunneling),
	}
}

func ToRepoServerDelegateTFModel(repoServerDelegate *v1alpha1.RepoServerDelegate) *types.RepoServerDelegate {
	if repoServerDelegate == nil {
		return nil
	}
	return &types.RepoServerDelegate{
		ControlPlane:   tftypes.BoolValue(repoServerDelegate.ControlPlane),
		ManagedCluster: ToManagedClusterTFModel(repoServerDelegate.ManagedCluster),
	}
}

func ToImageUpdaterDelegateTFModel(imageUpdaterDelegate *v1alpha1.ImageUpdaterDelegate) *types.ImageUpdaterDelegate {
	if imageUpdaterDelegate == nil {
		return nil
	}
	return &types.ImageUpdaterDelegate{
		ControlPlane:   tftypes.BoolValue(imageUpdaterDelegate.ControlPlane),
		ManagedCluster: ToManagedClusterTFModel(imageUpdaterDelegate.ManagedCluster),
	}
}

func ToAppSetDelegateTFModel(appSetDelegate *v1alpha1.AppSetDelegate) *types.AppSetDelegate {
	if appSetDelegate == nil {
		return nil
	}
	return &types.AppSetDelegate{
		ManagedCluster: ToManagedClusterTFModel(appSetDelegate.ManagedCluster),
	}
}

func ToManagedClusterTFModel(cluster *v1alpha1.ManagedCluster) *types.ManagedCluster {
	if cluster == nil {
		return nil
	}
	return &types.ManagedCluster{
		ClusterName: tftypes.StringValue(cluster.ClusterName),
	}
}

func ToClusterCustomizationTFModel(ctx context.Context, diag diag.Diagnostics, customization *v1alpha1.ClusterCustomization) tftypes.Object {
	if customization == nil {
		return tftypes.ObjectNull(clusterCustomizationAttrTypes)
	}
	yamlData, err := yaml.JSONToYAML(customization.Kustomization.Raw)
	if err != nil {
		fmt.Printf("Error converting JSON to YAML: %v", err)
	}
	c := types.ClusterCustomization{
		AutoUpgradeDisabled: tftypes.BoolValue(customization.AutoUpgradeDisabled),
		Kustomization:       tftypes.StringValue(string(yamlData)),
		AppReplication:      tftypes.BoolValue(customization.AppReplication),
		RedisTunneling:      tftypes.BoolValue(customization.RedisTunneling),
	}
	clusterCustomization, d := tftypes.ObjectValueFrom(ctx, clusterCustomizationAttrTypes, c)
	diag.Append(d...)
	return clusterCustomization
}

func ToIPAllowListTFModel(entries []*v1alpha1.IPAllowListEntry) []*types.IPAllowListEntry {
	var ipAllowList []*types.IPAllowListEntry
	for _, entry := range entries {
		ipAllowList = append(ipAllowList, &types.IPAllowListEntry{
			Ip:          tftypes.StringValue(entry.Ip),
			Description: tftypes.StringValue(entry.Description),
		})
	}
	return ipAllowList
}

func ToExtensionsTFModel(entries []*v1alpha1.ArgoCDExtensionInstallEntry) []*types.ArgoCDExtensionInstallEntry {
	var extensions []*types.ArgoCDExtensionInstallEntry
	for _, entry := range entries {
		extensions = append(extensions, &types.ArgoCDExtensionInstallEntry{
			Id:      tftypes.StringValue(entry.Id),
			Version: tftypes.StringValue(entry.Version),
		})
	}
	return extensions
}
