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
}

// If this test fails, a field has been added/removed to the ConfigMap type.
// Update the schema attribute accordingly.
func TestNoNewConfigMapFields(t *testing.T) {
	assert.Equal(t, reflect.TypeOf(types.ConfigMap{}).NumField(), len(getConfigMapAttributes()))
}

// If this test fails, a field has been added/removed to the Secret type.
// Update the schema attribute accordingly.
func TestNoNewSecretFields(t *testing.T) {
	assert.Equal(t, reflect.TypeOf(types.Secret{}).NumField(), len(getSecretAttributes()))
}
