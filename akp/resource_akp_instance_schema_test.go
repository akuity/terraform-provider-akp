//go:build !acc

package akp

import (
	"context"
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"

	"github.com/akuity/terraform-provider-akp/akp/types"
)

func TestLabelSelectorValidator(t *testing.T) {
	testCases := map[string]struct {
		value       tftypes.String
		errExpected bool
	}{
		"null is skipped":    {value: tftypes.StringNull()},
		"unknown is skipped": {value: tftypes.StringUnknown()},
		"empty selector":     {value: tftypes.StringValue("")},
		"equality selector":  {value: tftypes.StringValue("env=prod")},
		"set selector":       {value: tftypes.StringValue("env in (prod,staging),!legacy")},
		"invalid selector":   {value: tftypes.StringValue("env=("), errExpected: true},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			resp := &validator.StringResponse{}
			labelSelectorValidator{}.ValidateString(context.Background(), validator.StringRequest{
				Path:        path.Root("cluster_selector"),
				ConfigValue: tc.value,
			}, resp)
			assert.Equal(t, tc.errExpected, resp.Diagnostics.HasError(), "%v", resp.Diagnostics)
		})
	}
}

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

// If this test fails, a field has been added/removed to the secret-sync related types.
// Update the schema attribute accordingly.
func TestNoNewSecretsManagementFields(t *testing.T) {
	assert.Equal(t, reflect.TypeFor[types.SecretsManagementConfig]().NumField(), len(getSecretsManagementConfigAttributes()))
	assert.Equal(t, reflect.TypeFor[types.ClusterSecretMapping]().NumField(), len(getClusterSecretMappingAttributes()))
	assert.Equal(t, reflect.TypeFor[types.ObjectSelector]().NumField(), len(getObjectSelectorAttributes()))
	assert.Equal(t, reflect.TypeFor[types.LabelSelectorRequirement]().NumField(), len(getLabelSelectorRequirementAttributes()))
}

func TestNoNewManagedSecretFields(t *testing.T) {
	assert.Equal(t, reflect.TypeFor[types.ManagedSecret]().NumField(), len(getManagedSecretAttributes()))
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
