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
func TestNoNewAKPInstanceFields(t *testing.T) {
	assert.Equal(t, reflect.TypeFor[types.Instance]().NumField(), len(getAKPInstanceAttributes()))
}

// If this test fails, a field has been added/removed to the ArgoCD related type.
// Update the schema attribute accordingly.
func TestNoNewArgoCDFields(t *testing.T) {
	assert.Equal(t, reflect.TypeFor[types.ArgoCD]().NumField(), len(getArgoCDAttributes()))
	assert.Equal(t, reflect.TypeFor[types.ArgoCDSpec]().NumField(), len(getArgoCDSpecAttributes()))
	assert.Equal(t, reflect.TypeFor[types.InstanceSpec]().NumField(), len(getInstanceSpecAttributes()))
	assert.Equal(t, reflect.TypeFor[types.ArgoCDExtensionInstallEntry]().NumField(), len(getArgoCDExtensionInstallEntryAttributes()))
	assert.Equal(t, reflect.TypeFor[types.ClusterCustomization]().NumField(), len(getClusterCustomizationAttributes()))
	assert.Equal(t, reflect.TypeFor[types.RepoServerDelegate]().NumField(), len(getRepoServerDelegateAttributes()))
	assert.Equal(t, reflect.TypeFor[types.ImageUpdaterDelegate]().NumField(), len(getImageUpdaterDelegateAttributes()))
	assert.Equal(t, reflect.TypeFor[types.AppSetDelegate]().NumField(), len(getAppSetDelegateAttributes()))
	assert.Equal(t, reflect.TypeFor[types.ManagedCluster]().NumField(), len(getManagedClusterAttributes()))
	assert.Equal(t, reflect.TypeFor[types.AppsetPolicy]().NumField(), len(getAppsetPolicyAttributes()))
	assert.Equal(t, reflect.TypeFor[types.HostAliases]().NumField(), len(getHostAliasAttributes()))
	assert.Equal(t, reflect.TypeFor[types.AppsetPlugins]().NumField(), len(getAppsetPluginsAttributes()))
}

func TestNoNewManifestGenerationFields(t *testing.T) {
	assert.Equal(t, reflect.TypeFor[types.ManifestGeneration]().NumField(), len(getManifestGenerationAttributes()))
	assert.Equal(t, reflect.TypeFor[types.ConfigManagementToolVersions]().NumField(), len(getConfigManagementToolVersionsAttributes()))
}

// If this test fails, a field has been added/removed to the ConfigManagementPlugin related type.
// Update the schema attribute accordingly.
func TestNoNewConfigManagementPluginFields(t *testing.T) {
	assert.Equal(t, reflect.TypeFor[types.ConfigManagementPlugin]().NumField(), len(getAKPConfigManagementPluginAttributes()))
	assert.Equal(t, reflect.TypeFor[types.PluginSpec]().NumField(), len(getPluginSpecAttributes()))
	assert.Equal(t, reflect.TypeFor[types.Command]().NumField(), len(getCommandAttributes()))
	assert.Equal(t, reflect.TypeFor[types.Discover]().NumField(), len(getDiscoverAttributes()))
	assert.Equal(t, reflect.TypeFor[types.Find]().NumField(), len(getFindAttributes()))
	assert.Equal(t, reflect.TypeFor[types.Parameters]().NumField(), len(getParametersAttributes()))
	assert.Equal(t, reflect.TypeFor[types.Dynamic]().NumField(), len(getDynamicAttributes()))
	assert.Equal(t, reflect.TypeFor[types.ParameterAnnouncement]().NumField(), len(getParameterAnnouncementAttributes()))
}

// If this test fails, a field has been added/removed to the AI/KubeVision related types.
// Update the schema attribute accordingly.
func TestNoNewAIConfigFields(t *testing.T) {
	assert.Equal(t, reflect.TypeFor[types.KubeVisionConfig]().NumField(), len(getKubeVisionConfigAttributes()))
	assert.Equal(t, reflect.TypeFor[types.CveScanConfig]().NumField(), len(getCveScanConfigAttributes()))
	assert.Equal(t, reflect.TypeFor[types.AIConfig]().NumField(), len(getAIConfigAttributes()))
	assert.Equal(t, reflect.TypeFor[types.Runbook]().NumField(), len(getRunbookAttributes()))
	assert.Equal(t, reflect.TypeFor[types.RunbookRepo]().NumField(), len(getRunbookRepoAttributes()))
	assert.Equal(t, reflect.TypeFor[types.TargetSelector]().NumField(), len(getTargetSelectorAttributes()))
	assert.Equal(t, reflect.TypeFor[types.IncidentsConfig]().NumField(), len(getIncidentsConfigAttributes()))
	assert.Equal(t, reflect.TypeFor[types.IncidentWebhookConfig]().NumField(), len(getIncidentWebhookConfigAttributes()))
	assert.Equal(t, reflect.TypeFor[types.IncidentsGroupingConfig]().NumField(), len(getIncidentsGroupingConfigAttributes()))
	assert.Equal(t, reflect.TypeFor[types.IncidentInvestigationApprovalConfig]().NumField(), len(getIncidentInvestigationApprovalConfigAttributes()))
	assert.Equal(t, reflect.TypeFor[types.IncidentInvestigationApprovalScope]().NumField(), len(getIncidentInvestigationApprovalScopeAttributes()))
	assert.Equal(t, reflect.TypeFor[types.AdditionalAttributeRule]().NumField(), len(getAdditionalAttributeRuleAttributes()))
}
