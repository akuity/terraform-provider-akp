//go:build !unit

package akp

import (
	"testing"
)

func TestAccAll(t *testing.T) {
	t.Cleanup(func() {
		if t.Failed() {
			cleanupTestKargoInstance()
			cleanupTestInstance()
		}
	})

	t.Run("Resources", func(t *testing.T) {
		t.Run("Cluster_Basic", func(t *testing.T) { runClusterResource(t) })
		t.Run("Cluster_IPv6", func(t *testing.T) { runClusterResourceIPv6(t) })
		t.Run("Cluster_ArgoCDNotifications", func(t *testing.T) { runClusterResourceArgoCDNotifications(t) })
		t.Run("Cluster_CustomAgentSize", func(t *testing.T) { runClusterResourceCustomAgentSize(t) })
		t.Run("Cluster_ManagedCluster", func(t *testing.T) { runClusterResourceManagedCluster(t) })
		t.Run("Cluster_Features", func(t *testing.T) { runClusterResourceFeatures(t) })
		t.Run("Cluster_ReapplyManifests", func(t *testing.T) { runClusterResourceReapplyManifests(t) })
		t.Run("Cluster_NamespaceScoped", func(t *testing.T) { runClusterResourceNamespaceScoped(t) })
		t.Run("Cluster_Project", func(t *testing.T) { runClusterResourceProject(t) })
		t.Run("Cluster_AutoAgentSizeConsistency", func(t *testing.T) { runClusterResourceAutoAgentSizeConsistency(t) })
		t.Run("Cluster_AutoAgentSizeCreateWithConfig", func(t *testing.T) { runClusterResourceAutoAgentSizeCreateWithConfig(t) })
		t.Run("Cluster_Kubeconfig", func(t *testing.T) { runClusterResourceKubeconfig(t) })
		t.Run("Cluster_CustomAgentSizeWithKustomization", func(t *testing.T) { runClusterResourceCustomAgentSizeWithKustomization(t) })
		t.Run("Cluster_CustomAgentSizeKustomizationOnly", func(t *testing.T) { runClusterResourceCustomAgentSizeKustomizationOnly(t) })
		t.Run("Cluster_CustomAgentSizeTransitions", func(t *testing.T) { runClusterResourceCustomAgentSizeTransitions(t) })
		t.Run("Cluster_ValidationError", func(t *testing.T) { runClusterResourceValidationError(t) })
		t.Run("Cluster_MergeData", func(t *testing.T) { runClusterResourceMergeData(t) })
		t.Run("Cluster_CustomAgentSizeInconsistency", func(t *testing.T) { runCluster_CustomAgentSizeInconsistency(t) })
		t.Run("Cluster_NamespaceScopedMissingField", func(t *testing.T) { runCluster_NamespaceScopedMissingField(t) })
		t.Run("Cluster_NamespaceScopedImportHydration", func(t *testing.T) { runCluster_NamespaceScopedImportHydration(t) })
		t.Run("Cluster_ServerSideDiff", func(t *testing.T) { runClusterResourceServerSideDiff(t) })
		t.Run("Cluster_MaintenanceMode", func(t *testing.T) { runClusterResourceMaintenanceMode(t) })
		t.Run("Cluster_FeatureToggleTransitions", func(t *testing.T) { runClusterResourceFeatureToggleTransitions(t) })
		t.Run("Cluster_IdempotentReapply", func(t *testing.T) { runClusterResourceIdempotentReapply(t) })
		t.Run("Cluster_MaintenanceModeTransitions", func(t *testing.T) { runClusterResourceMaintenanceModeTransitions(t) })
		t.Run("Cluster_MinimalNestedImport", func(t *testing.T) { runCluster_MinimalNestedImport(t) })
		t.Run("Cluster_PartialCustomSizeImport", func(t *testing.T) { runCluster_PartialCustomSizeImport(t) })
		t.Run("Cluster_PartialNotificationsImport", func(t *testing.T) { runCluster_PartialNotificationsImport(t) })

		t.Run("IPAllowList_Single", func(t *testing.T) { runInstanceIPAllowListResource(t) })
		t.Run("IPAllowList_MultipleResources", func(t *testing.T) { runInstanceIPAllowListResource_MultipleResources(t) })
		t.Run("IPAllowList_DuplicateIP", func(t *testing.T) { runInstanceIPAllowListResource_DuplicateIP(t) })
		t.Run("IPAllowList_DuplicateInSameResource", func(t *testing.T) { runInstanceIPAllowListResource_DuplicateInSameResource(t) })
		t.Run("IPAllowList_IPv6", func(t *testing.T) { runInstanceIPAllowListResource_IPv6(t) })
		t.Run("IPAllowList_NoDescription", func(t *testing.T) { runInstanceIPAllowListResource_NoDescription(t) })
		t.Run("IPAllowList_PreservesInstanceSettings", func(t *testing.T) { runInstanceIPAllowListResource_PreservesInstanceSettings(t) })
		t.Run("IPAllowList_LargeScale", func(t *testing.T) { runInstanceIPAllowListResource_LargeScale(t) })

		t.Run("KargoAgent_Basic", func(t *testing.T) { runKargoAgentResource(t) })
		t.Run("KargoAgent_RemoteArgoCD", func(t *testing.T) { runKargoAgentResourceRemoteArgoCD(t) })
		t.Run("KargoAgent_CustomNamespace", func(t *testing.T) { runKargoAgentResourceCustomNamespace(t) })
		t.Run("KargoAgent_ReapplyManifests", func(t *testing.T) { runKargoAgentResourceReapplyManifests(t) })
		t.Run("KargoAgent_TargetVersion", func(t *testing.T) { runKargoAgentResourceTargetVersion(t) })
		t.Run("KargoAgent_Kubeconfig", func(t *testing.T) { runKargoAgentResourceKubeconfig(t) })
		t.Run("KargoAgent_AllowedJobSA", func(t *testing.T) { runKargoAgentResourceAllowedJobSA(t) })
		t.Run("KargoAgent_MaintenanceMode", func(t *testing.T) { runKargoAgentResourceMaintenanceMode(t) })
		t.Run("KargoAgent_MaintenanceModeTransitions", func(t *testing.T) { runKargoAgentResourceMaintenanceModeTransitions(t) })
		t.Run("KargoAgent_AllowedJobSATransitions", func(t *testing.T) { runKargoAgentResourceAllowedJobSATransitions(t) })
		t.Run("KargoAgent_IdempotentReapply", func(t *testing.T) { runKargoAgentResourceIdempotentReapply(t) })
		t.Run("KargoAgent_MinimalNestedImport", func(t *testing.T) { runKargoAgent_MinimalNestedImport(t) })
		t.Run("KargoAgent_PartialDataImport", func(t *testing.T) { runKargoAgent_PartialDataImport(t) })
		t.Run("KargoAgent_PodInheritMetadata", func(t *testing.T) { runKargoAgentResourcePodInheritMetadata(t) })
		t.Run("KargoAgent_Autosize", func(t *testing.T) { runKargoAgentResourceAutosize(t) })
		t.Run("KargoAgent_DefaultShardDeleteRejected", func(t *testing.T) { runKargoAgent_DefaultShardDeleteRejected(t) })

		t.Run("KargoDefaultShardAgent", func(t *testing.T) { runKargoDefaultShardAgentResource(t) })
	})

	t.Run("InstanceConfigs", func(t *testing.T) {
		t.Run("ArgoCD", func(t *testing.T) {
			t.Parallel()
			t.Run("Configs", func(t *testing.T) { runInstanceConfigTests(t) })
			t.Run("NestedOptionalObjectStability", func(t *testing.T) { runInstance_NestedOptionalObjectStability(t) })
			t.Run("RBACChangeWithCombinedCustomizations", func(t *testing.T) { runInstance_RBACChangeWithCombinedCustomizations(t) })
			t.Run("MinimalSpecImport", func(t *testing.T) { runInstance_MinimalSpecImport(t) })
			t.Run("PartialInstanceSpecImport", func(t *testing.T) { runInstance_PartialInstanceSpecImport(t) })
		})

		t.Run("Kargo", func(t *testing.T) {
			t.Parallel()
			t.Run("Configs", func(t *testing.T) { runKargoConfigTests(t) })
			t.Run("NestedOptionalObjectStability", func(t *testing.T) { runKargo_NestedOptionalObjectStability(t) })
			t.Run("MinimalSpecImport", func(t *testing.T) { runKargo_MinimalSpecImport(t) })
			t.Run("PartialOIDCImport", func(t *testing.T) { runKargo_PartialOIDCImport(t) })
			t.Run("PartialKargoInstanceSpecImport", func(t *testing.T) { runKargo_PartialKargoInstanceSpecImport(t) })
		})
	})
}
