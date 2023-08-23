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
	assert.Equal(t, reflect.TypeOf(types.Instance{}).NumField(), len(getAKPInstanceDataSourceAttributes()))
}

// If this test fails, a field has been added/removed to the ArgoCD related type.
// Update the schema attribute accordingly.
func TestNoNewArgoCDDataSourceFields(t *testing.T) {
	assert.Equal(t, reflect.TypeOf(types.ArgoCD{}).NumField(), len(getArgoCDDataSourceAttributes()))
	assert.Equal(t, reflect.TypeOf(types.ArgoCDSpec{}).NumField(), len(getArgoCDSpecDataSourceAttributes()))
	assert.Equal(t, reflect.TypeOf(types.InstanceSpec{}).NumField(), len(getInstanceSpecDataSourceAttributes()))
	assert.Equal(t, reflect.TypeOf(types.IPAllowListEntry{}).NumField(), len(getIPAllowListEntryDataSourceAttributes()))
	assert.Equal(t, reflect.TypeOf(types.ArgoCDExtensionInstallEntry{}).NumField(), len(getArgoCDExtensionInstallEntryDataSourceAttributes()))
	assert.Equal(t, reflect.TypeOf(types.ClusterCustomization{}).NumField(), len(getClusterCustomizationDataSourceAttributes()))
	assert.Equal(t, reflect.TypeOf(types.RepoServerDelegate{}).NumField(), len(getRepoServerDelegateDataSourceAttributes()))
	assert.Equal(t, reflect.TypeOf(types.ImageUpdaterDelegate{}).NumField(), len(getImageUpdaterDelegateDataSourceAttributes()))
	assert.Equal(t, reflect.TypeOf(types.AppSetDelegate{}).NumField(), len(getAppSetDelegateDataSourceAttributes()))
	assert.Equal(t, reflect.TypeOf(types.ManagedCluster{}).NumField(), len(getManagedClusterDataSourceAttributes()))
	assert.Equal(t, reflect.TypeOf(types.AppsetPolicy{}).NumField(), len(getAppsetPolicyDataSourceAttributes()))

}
