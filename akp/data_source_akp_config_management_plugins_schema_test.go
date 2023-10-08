//go:build !acc

package akp

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/akuity/terraform-provider-akp/akp/types"
)

// If this test fails, a field has been added/removed to the ConfigManagementPlugins related type.
// Update the schema attribute accordingly.
func TestNoNewConfigManagementPluginsFields(t *testing.T) {
	assert.Equal(t, reflect.TypeOf(types.ConfigManagementPlugins{}).NumField(), len(getAKPConfigManagementPluginsDataSourceAttributes()))
}
