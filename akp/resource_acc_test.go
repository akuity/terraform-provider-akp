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
		t.Run("Cluster_Basic", runClusterResource)
		t.Run("Cluster_IPv6", runClusterResourceIPv6)
		t.Run("Cluster_ArgoCDNotifications", runClusterResourceArgoCDNotifications)
		t.Run("Cluster_CustomAgentSize", runClusterResourceCustomAgentSize)
		t.Run("Cluster_ManagedCluster", runClusterResourceManagedCluster)
		t.Run("Cluster_Features", runClusterResourceFeatures)
		t.Run("Cluster_ReapplyManifests", runClusterResourceReapplyManifests)
		t.Run("Cluster_NamespaceScoped", runClusterResourceNamespaceScoped)
		t.Run("Cluster_Project", runClusterResourceProject)
		t.Run("Cluster_AutoAgentSizeConsistency", runClusterResourceAutoAgentSizeConsistency)
		t.Run("Cluster_AutoAgentSizeCreateWithConfig", runClusterResourceAutoAgentSizeCreateWithConfig)
		t.Run("Cluster_Kubeconfig", runClusterResourceKubeconfig)
		t.Run("Cluster_CustomAgentSizeWithKustomization", runClusterResourceCustomAgentSizeWithKustomization)
		t.Run("Cluster_CustomAgentSizeKustomizationOnly", runClusterResourceCustomAgentSizeKustomizationOnly)
		t.Run("Cluster_CustomAgentSizeTransitions", runClusterResourceCustomAgentSizeTransitions)
		t.Run("Cluster_ValidationError", runClusterResourceValidationError)
		t.Run("Cluster_MergeData", runClusterResourceMergeData)
		t.Run("Cluster_CustomAgentSizeInconsistency", runCluster_CustomAgentSizeInconsistency)
		t.Run("Cluster_NamespaceScopedMissingField", runCluster_NamespaceScopedMissingField)
		t.Run("Cluster_NamespaceScopedImportHydration", runCluster_NamespaceScopedImportHydration)
		t.Run("Cluster_ServerSideDiff", runClusterResourceServerSideDiff)
		t.Run("Cluster_MaintenanceMode", runClusterResourceMaintenanceMode)
		t.Run("Cluster_FeatureToggleTransitions", runClusterResourceFeatureToggleTransitions)
		t.Run("Cluster_IdempotentReapply", runClusterResourceIdempotentReapply)
		t.Run("Cluster_MaintenanceModeTransitions", runClusterResourceMaintenanceModeTransitions)
		t.Run("Cluster_MinimalNestedImport", runCluster_MinimalNestedImport)
		t.Run("Cluster_PartialCustomSizeImport", runCluster_PartialCustomSizeImport)
		t.Run("Cluster_PartialNotificationsImport", runCluster_PartialNotificationsImport)

		t.Run("IPAllowList_Single", runInstanceIPAllowListResource)
		t.Run("IPAllowList_MultipleResources", runInstanceIPAllowListResource_MultipleResources)
		t.Run("IPAllowList_DuplicateIP", runInstanceIPAllowListResource_DuplicateIP)
		t.Run("IPAllowList_DuplicateInSameResource", runInstanceIPAllowListResource_DuplicateInSameResource)
		t.Run("IPAllowList_IPv6", runInstanceIPAllowListResource_IPv6)
		t.Run("IPAllowList_NoDescription", runInstanceIPAllowListResource_NoDescription)
		t.Run("IPAllowList_PreservesInstanceSettings", runInstanceIPAllowListResource_PreservesInstanceSettings)
		t.Run("IPAllowList_LargeScale", runInstanceIPAllowListResource_LargeScale)

		t.Run("KargoAgent_Basic", runKargoAgentResource)
		t.Run("KargoAgent_RemoteArgoCD", runKargoAgentResourceRemoteArgoCD)
		t.Run("KargoAgent_CustomNamespace", runKargoAgentResourceCustomNamespace)
		t.Run("KargoAgent_ReapplyManifests", runKargoAgentResourceReapplyManifests)
		t.Run("KargoAgent_TargetVersion", runKargoAgentResourceTargetVersion)
		t.Run("KargoAgent_Kubeconfig", runKargoAgentResourceKubeconfig)
		t.Run("KargoAgent_AllowedJobSA", runKargoAgentResourceAllowedJobSA)
		t.Run("KargoAgent_MaintenanceMode", runKargoAgentResourceMaintenanceMode)
		t.Run("KargoAgent_MaintenanceModeTransitions", runKargoAgentResourceMaintenanceModeTransitions)
		t.Run("KargoAgent_AllowedJobSATransitions", runKargoAgentResourceAllowedJobSATransitions)
		t.Run("KargoAgent_IdempotentReapply", runKargoAgentResourceIdempotentReapply)
		t.Run("KargoAgent_MinimalNestedImport", runKargoAgent_MinimalNestedImport)
		t.Run("KargoAgent_PartialDataImport", runKargoAgent_PartialDataImport)
		t.Run("KargoAgent_PodInheritMetadata", runKargoAgentResourcePodInheritMetadata)

		t.Run("KargoDefaultShardAgent", runKargoDefaultShardAgentResource)
	})

	t.Run("InstanceConfigs", func(t *testing.T) {
		t.Run("ArgoCD", func(t *testing.T) {
			t.Parallel()
			t.Run("Configs", runInstanceConfigTests)
			t.Run("NestedOptionalObjectStability", runInstance_NestedOptionalObjectStability)
			t.Run("MinimalSpecImport", runInstance_MinimalSpecImport)
			t.Run("PartialInstanceSpecImport", runInstance_PartialInstanceSpecImport)
		})

		t.Run("Kargo", func(t *testing.T) {
			t.Parallel()
			t.Run("Configs", runKargoConfigTests)
			t.Run("NestedOptionalObjectStability", runKargo_NestedOptionalObjectStability)
			t.Run("MinimalSpecImport", runKargo_MinimalSpecImport)
			t.Run("PartialOIDCImport", runKargo_PartialOIDCImport)
			t.Run("PartialKargoInstanceSpecImport", runKargo_PartialKargoInstanceSpecImport)
		})
	})
}
