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
