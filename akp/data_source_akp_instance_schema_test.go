//go:build !acc

package akp

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/akuity/terraform-provider-akp/akp/types"
)

// If this test fails, a field has been added/removed to the AKP Instance type.
// Update the schema attribute accordingly.
func TestNoNewAKPInstanceDataSourceFields(t *testing.T) {
	assert.Equal(t, reflect.TypeFor[types.InstanceDataSource]().NumField(), len(getAKPInstanceDataSourceAttributes()))
}

// If this test fails, a field has been added/removed to the ArgoCD related type.
// Update the schema attribute accordingly.
func TestNoNewArgoCDDataSourceFields(t *testing.T) {
	assert.Equal(t, reflect.TypeFor[types.ArgoCDDataSource]().NumField(), len(getArgoCDDataSourceAttributes()))
	assert.Equal(t, reflect.TypeFor[types.ArgoCDSpecDataSource]().NumField(), len(getArgoCDSpecDataSourceAttributes()))
	assert.Equal(t, reflect.TypeFor[types.InstanceSpecDataSource]().NumField(), len(getInstanceSpecDataSourceAttributes()))
	assert.Equal(t, reflect.TypeFor[types.ArgoCDExtensionInstallEntry]().NumField(), len(getArgoCDExtensionInstallEntryDataSourceAttributes()))
	assert.Equal(t, reflect.TypeFor[types.ClusterCustomization]().NumField(), len(getClusterCustomizationDataSourceAttributes()))
	assert.Equal(t, reflect.TypeFor[types.RepoServerDelegate]().NumField(), len(getRepoServerDelegateDataSourceAttributes()))
	assert.Equal(t, reflect.TypeFor[types.ImageUpdaterDelegate]().NumField(), len(getImageUpdaterDelegateDataSourceAttributes()))
	assert.Equal(t, reflect.TypeFor[types.AppSetDelegate]().NumField(), len(getAppSetDelegateDataSourceAttributes()))
	assert.Equal(t, reflect.TypeFor[types.ManagedCluster]().NumField(), len(getManagedClusterDataSourceAttributes()))
	assert.Equal(t, reflect.TypeFor[types.AppsetPolicy]().NumField(), len(getAppsetPolicyDataSourceAttributes()))
	assert.Equal(t, reflect.TypeFor[types.HostAliases]().NumField(), len(getHostAliasesDataSourceAttributes()))
	assert.Equal(t, reflect.TypeFor[types.AppsetPlugins]().NumField(), len(getAppsetPluginsDataSourceAttributes()))
}

func TestNoNewManifestGenerationDataSourceFields(t *testing.T) {
	assert.Equal(t, reflect.TypeFor[types.ManifestGeneration]().NumField(), len(getManifestGenerationDataSourceAttributes()))
	assert.Equal(t, reflect.TypeFor[types.ConfigManagementToolVersions]().NumField(), len(getConfigManagementToolVersionsDataSourceAttributes()))
}

// If this test fails, a field has been added/removed to the ConfigManagementPlugin related type.
// Update the schema attribute accordingly.
func TestNoNewConfigManagementPluginDataSourceFields(t *testing.T) {
	assert.Equal(t, reflect.TypeFor[types.ConfigManagementPlugin]().NumField(), len(getAKPConfigManagementPluginDataSourceAttributes()))
	assert.Equal(t, reflect.TypeFor[types.PluginSpec]().NumField(), len(getPluginSpecDataSourceAttributes()))
	assert.Equal(t, reflect.TypeFor[types.Command]().NumField(), len(getCommandDataSourceAttributes()))
	assert.Equal(t, reflect.TypeFor[types.Discover]().NumField(), len(getDiscoverDataSourceAttributes()))
	assert.Equal(t, reflect.TypeFor[types.Find]().NumField(), len(getFindDataSourceAttributes()))
	assert.Equal(t, reflect.TypeFor[types.Parameters]().NumField(), len(getParametersDataSourceAttributes()))
	assert.Equal(t, reflect.TypeFor[types.Dynamic]().NumField(), len(getDynamicDataSourceAttributes()))
	assert.Equal(t, reflect.TypeFor[types.ParameterAnnouncement]().NumField(), len(getParameterAnnouncementDataSourceAttributes()))
}

// If this test fails, a field has been added/removed to the AI/KubeVision related types.
// Update the schema attribute accordingly.
func TestNoNewAIConfigDataSourceFields(t *testing.T) {
	assert.Equal(t, reflect.TypeFor[types.KubeVisionConfig]().NumField(), len(getKubeVisionConfigDataSourceAttributes()))
	assert.Equal(t, reflect.TypeFor[types.CveScanConfig]().NumField(), len(getCveScanConfigDataSourceAttributes()))
	assert.Equal(t, reflect.TypeFor[types.AIConfig]().NumField(), len(getAIConfigDataSourceAttributes()))
	assert.Equal(t, reflect.TypeFor[types.Runbook]().NumField(), len(getRunbookDataSourceAttributes()))
	assert.Equal(t, reflect.TypeFor[types.RunbookRepo]().NumField(), len(getRunbookRepoDataSourceAttributes()))
	assert.Equal(t, reflect.TypeFor[types.TargetSelector]().NumField(), len(getTargetSelectorDataSourceAttributes()))
	assert.Equal(t, reflect.TypeFor[types.IncidentsConfig]().NumField(), len(getIncidentsConfigDataSourceAttributes()))
	assert.Equal(t, reflect.TypeFor[types.IncidentWebhookConfig]().NumField(), len(getIncidentWebhookConfigDataSourceAttributes()))
	assert.Equal(t, reflect.TypeFor[types.IncidentsGroupingConfig]().NumField(), len(getIncidentsGroupingConfigDataSourceAttributes()))
	assert.Equal(t, reflect.TypeFor[types.IncidentInvestigationApprovalConfig]().NumField(), len(getIncidentInvestigationApprovalConfigDataSourceAttributes()))
	assert.Equal(t, reflect.TypeFor[types.IncidentInvestigationApprovalScope]().NumField(), len(getIncidentInvestigationApprovalScopeDataSourceAttributes()))
	assert.Equal(t, reflect.TypeFor[types.AdditionalAttributeRule]().NumField(), len(getAdditionalAttributeRuleDataSourceAttributes()))
}
