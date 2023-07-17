package conversion

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
	"github.com/akuity/terraform-provider-akp/akp/types"
)

var (
	clusterCustomizationAttrTypes = map[string]attr.Type{
		"auto_upgrade_disabled": tftypes.BoolType,
		"kustomization":         tftypes.StringType,
		"app_replication":       tftypes.BoolType,
		"redis_tunneling":       tftypes.BoolType,
	}
)

func ToConfigMapAPIModel(ctx context.Context, diagnostics diag.Diagnostics, configMap *types.ConfigMap, name string) *v1.ConfigMap {
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

func ToSecretAPIModel(ctx context.Context, diagnostics diag.Diagnostics, secret *types.Secret, name string) *v1.Secret {
	var labels map[string]string
	var data map[string][]byte
	var stringData map[string]string
	diagnostics.Append(secret.Labels.ElementsAs(ctx, &labels, true)...)
	diagnostics.Append(secret.StringData.ElementsAs(ctx, &stringData, true)...)
	diagnostics.Append(secret.Data.ElementsAs(ctx, &data, true)...)
	diagnostics.Append(secret.StringData.ElementsAs(ctx, &stringData, true)...)
	return &v1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
		Data:       data,
		StringData: stringData,
	}
}

func ToClusterAPIModel(ctx context.Context, diag diag.Diagnostics, cluster *types.Cluster) *v1alpha1.Cluster {
	var labels map[string]string
	var annotations map[string]string
	diag.Append(cluster.Labels.ElementsAs(ctx, &labels, true)...)
	diag.Append(cluster.Annotations.ElementsAs(ctx, &annotations, true)...)
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
			Data:            ToClusterDataAPIModel(cluster.Spec.Data),
		},
	}
}

func ToArgoCDAPIModel(ctx context.Context, diag diag.Diagnostics, cd *types.ArgoCD) *v1alpha1.ArgoCD {
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
				IpAllowList:                  ToIPAllowListAPIModel(cd.Spec.InstanceSpec.IpAllowList),
				Subdomain:                    cd.Spec.InstanceSpec.Subdomain.ValueString(),
				DeclarativeManagementEnabled: cd.Spec.InstanceSpec.DeclarativeManagementEnabled.ValueBool(),
				Extensions:                   ToExtensionsAPIModel(cd.Spec.InstanceSpec.Extensions),
				ClusterCustomizationDefaults: ToClusterCustomizationAPIModel(ctx, diag, cd.Spec.InstanceSpec.ClusterCustomizationDefaults),
				ImageUpdaterEnabled:          cd.Spec.InstanceSpec.ImageUpdaterEnabled.ValueBool(),
				BackendIpAllowListEnabled:    cd.Spec.InstanceSpec.BackendIpAllowListEnabled.ValueBool(),
				RepoServerDelegate:           ToRepoServerDelegateAPIModel(cd.Spec.InstanceSpec.RepoServerDelegate),
				AuditExtensionEnabled:        cd.Spec.InstanceSpec.AuditExtensionEnabled.ValueBool(),
				SyncHistoryExtensionEnabled:  cd.Spec.InstanceSpec.SyncHistoryExtensionEnabled.ValueBool(),
				ImageUpdaterDelegate:         ToImageUpdaterDelegateAPIModel(cd.Spec.InstanceSpec.ImageUpdaterDelegate),
				AppSetDelegate:               ToAppSetDelegateAPIModel(cd.Spec.InstanceSpec.AppSetDelegate),
			},
		},
	}
}

func ToClusterDataAPIModel(clusterData types.ClusterData) v1alpha1.ClusterData {
	raw := runtime.RawExtension{}
	if err := yaml.Unmarshal([]byte(clusterData.Kustomization.ValueString()), &raw); err != nil {
		fmt.Println("Failed to unmarshal YAML:", err)
	} else {
		fmt.Println("Successfully unmarshalled YAML into RawExtension")
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

func ToRepoServerDelegateAPIModel(repoServerDelegate *types.RepoServerDelegate) *v1alpha1.RepoServerDelegate {
	if repoServerDelegate == nil {
		return nil
	}
	return &v1alpha1.RepoServerDelegate{
		ControlPlane:   repoServerDelegate.ControlPlane.ValueBool(),
		ManagedCluster: ToManagedClusterAPIModel(repoServerDelegate.ManagedCluster),
	}
}

func ToImageUpdaterDelegateAPIModel(imageUpdaterDelegate *types.ImageUpdaterDelegate) *v1alpha1.ImageUpdaterDelegate {
	if imageUpdaterDelegate == nil {
		return nil
	}
	return &v1alpha1.ImageUpdaterDelegate{
		ControlPlane:   imageUpdaterDelegate.ControlPlane.ValueBool(),
		ManagedCluster: ToManagedClusterAPIModel(imageUpdaterDelegate.ManagedCluster),
	}
}

func ToAppSetDelegateAPIModel(appSetDelegate *types.AppSetDelegate) *v1alpha1.AppSetDelegate {
	if appSetDelegate == nil {
		return nil
	}
	return &v1alpha1.AppSetDelegate{
		ManagedCluster: ToManagedClusterAPIModel(appSetDelegate.ManagedCluster),
	}
}

func ToManagedClusterAPIModel(cluster *types.ManagedCluster) *v1alpha1.ManagedCluster {
	if cluster == nil {
		return nil
	}
	return &v1alpha1.ManagedCluster{
		ClusterName: cluster.ClusterName.ValueString(),
	}
}

func ToClusterCustomizationAPIModel(ctx context.Context, diag diag.Diagnostics, clusterCustomization tftypes.Object) *v1alpha1.ClusterCustomization {
	var customization types.ClusterCustomization
	diag.Append(clusterCustomization.As(ctx, &customization, basetypes.ObjectAsOptions{})...)
	raw := runtime.RawExtension{}
	if err := yaml.Unmarshal([]byte(customization.Kustomization.ValueString()), &raw); err != nil {
		fmt.Println("Failed to unmarshal YAML:", err)
	} else {
		fmt.Println("Successfully unmarshalled YAML into RawExtension")
	}
	return &v1alpha1.ClusterCustomization{
		AutoUpgradeDisabled: customization.AutoUpgradeDisabled.ValueBool(),
		Kustomization:       raw,
		AppReplication:      customization.AppReplication.ValueBool(),
		RedisTunneling:      customization.RedisTunneling.ValueBool(),
	}
}

func ToIPAllowListAPIModel(entries []*types.IPAllowListEntry) []*v1alpha1.IPAllowListEntry {
	var ipAllowList []*v1alpha1.IPAllowListEntry
	for _, entry := range entries {
		ipAllowList = append(ipAllowList, &v1alpha1.IPAllowListEntry{
			Ip:          entry.Ip.ValueString(),
			Description: entry.Description.ValueString(),
		})
	}
	return ipAllowList
}

func ToExtensionsAPIModel(entries []*types.ArgoCDExtensionInstallEntry) []*v1alpha1.ArgoCDExtensionInstallEntry {
	var extensions []*v1alpha1.ArgoCDExtensionInstallEntry
	for _, entry := range entries {
		extensions = append(extensions, &v1alpha1.ArgoCDExtensionInstallEntry{
			Id:      entry.Id.ValueString(),
			Version: entry.Version.ValueString(),
		})
	}
	return extensions
}
