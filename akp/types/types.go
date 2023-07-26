package types

import (
	"bytes"
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"

	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	"github.com/akuity/terraform-provider-akp/akp/apis/v1alpha1"
)

var (
	clusterCustomizationAttrTypes = map[string]attr.Type{
		"auto_upgrade_disabled": tftypes.BoolType,
		"kustomization":         tftypes.StringType,
		"app_replication":       tftypes.BoolType,
		"redis_tunneling":       tftypes.BoolType,
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
			},
		},
	}
}

func (c *Cluster) Update(ctx context.Context, diagnostics *diag.Diagnostics, apiCluster *argocdv1.Cluster) {
	c.ID = tftypes.StringValue(apiCluster.GetId())
	c.Name = tftypes.StringValue(apiCluster.GetName())
	c.Namespace = tftypes.StringValue(apiCluster.GetNamespace())
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
	diagnostics.Append(clusterCustomization.As(ctx, &customization, basetypes.ObjectAsOptions{})...)
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
	c := ClusterCustomization{
		AutoUpgradeDisabled: tftypes.BoolValue(autoUpgradeDisabled),
		Kustomization:       tftypes.StringValue(string(yamlData)),
		AppReplication:      tftypes.BoolValue(appReplication),
		RedisTunneling:      tftypes.BoolValue(redisTunneling),
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
