//go:build !acc

package akp

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/akuity/terraform-provider-akp/akp/types"
)

// If this test fails, a field has been added/removed to the ConfigManagementPlugin related type.
// Update the schema attribute accordingly.
func TestNoNewConfigManagementPluginDataSourceFields(t *testing.T) {
	assert.Equal(t, reflect.TypeOf(types.ConfigManagementPlugin{}).NumField(), len(getAKPConfigManagementPluginDataSourceAttributes()))
	assert.Equal(t, reflect.TypeOf(types.PluginSpec{}).NumField(), len(getPluginSpecDataSourceAttributes()))
	assert.Equal(t, reflect.TypeOf(types.Command{}).NumField(), len(getCommandDataSourceAttributes()))
	assert.Equal(t, reflect.TypeOf(types.Discover{}).NumField(), len(getDiscoverDataSourceAttributes()))
	assert.Equal(t, reflect.TypeOf(types.Find{}).NumField(), len(getFindDataSourceAttributes()))
	assert.Equal(t, reflect.TypeOf(types.Parameters{}).NumField(), len(getParametersDataSourceAttributes()))
	assert.Equal(t, reflect.TypeOf(types.Dynamic{}).NumField(), len(getDynamicDataSourceAttributes()))
	assert.Equal(t, reflect.TypeOf(types.ParameterAnnouncement{}).NumField(), len(getParameterAnnouncementDataSourceAttributes()))
}
