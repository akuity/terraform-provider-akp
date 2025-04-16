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
	assert.Equal(t, reflect.TypeOf(types.Instance{}).NumField(), len(getAKPInstanceAttributes()))
}

// If this test fails, a field has been added/removed to the ArgoCD related type.
// Update the schema attribute accordingly.
func TestNoNewArgoCDFields(t *testing.T) {
	assert.Equal(t, reflect.TypeOf(types.ArgoCD{}).NumField(), len(getArgoCDAttributes()))
	assert.Equal(t, reflect.TypeOf(types.ArgoCDSpec{}).NumField(), len(getArgoCDSpecAttributes()))
	assert.Equal(t, reflect.TypeOf(types.InstanceSpec{}).NumField(), len(getInstanceSpecAttributes()))
	assert.Equal(t, reflect.TypeOf(types.IPAllowListEntry{}).NumField(), len(getIPAllowListEntryAttributes()))
	assert.Equal(t, reflect.TypeOf(types.ArgoCDExtensionInstallEntry{}).NumField(), len(getArgoCDExtensionInstallEntryAttributes()))
	assert.Equal(t, reflect.TypeOf(types.ClusterCustomization{}).NumField(), len(getClusterCustomizationAttributes()))
	assert.Equal(t, reflect.TypeOf(types.RepoServerDelegate{}).NumField(), len(getRepoServerDelegateAttributes()))
	assert.Equal(t, reflect.TypeOf(types.ImageUpdaterDelegate{}).NumField(), len(getImageUpdaterDelegateAttributes()))
	assert.Equal(t, reflect.TypeOf(types.AppSetDelegate{}).NumField(), len(getAppSetDelegateAttributes()))
	assert.Equal(t, reflect.TypeOf(types.ManagedCluster{}).NumField(), len(getManagedClusterAttributes()))
	assert.Equal(t, reflect.TypeOf(types.AppsetPolicy{}).NumField(), len(getAppsetPolicyAttributes()))
	assert.Equal(t, reflect.TypeOf(types.HostAliases{}).NumField(), len(getAppsetPolicyAttributes()))
	assert.Equal(t, reflect.TypeOf(types.AppsetPlugins{}).NumField(), len(getAppsetPluginsAttributes()))
}

// If this test fails, a field has been added/removed to the ConfigManagementPlugin related type.
// Update the schema attribute accordingly.
func TestNoNewConfigManagementPluginFields(t *testing.T) {
	assert.Equal(t, reflect.TypeOf(types.ConfigManagementPlugin{}).NumField(), len(getAKPConfigManagementPluginAttributes()))
	assert.Equal(t, reflect.TypeOf(types.PluginSpec{}).NumField(), len(getPluginSpecAttributes()))
	assert.Equal(t, reflect.TypeOf(types.Command{}).NumField(), len(getCommandAttributes()))
	assert.Equal(t, reflect.TypeOf(types.Discover{}).NumField(), len(getDiscoverAttributes()))
	assert.Equal(t, reflect.TypeOf(types.Find{}).NumField(), len(getFindAttributes()))
	assert.Equal(t, reflect.TypeOf(types.Parameters{}).NumField(), len(getParametersAttributes()))
	assert.Equal(t, reflect.TypeOf(types.Dynamic{}).NumField(), len(getDynamicAttributes()))
	assert.Equal(t, reflect.TypeOf(types.ParameterAnnouncement{}).NumField(), len(getParameterAnnouncementAttributes()))
}
