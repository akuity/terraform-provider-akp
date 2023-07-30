//go:build !acc

package akp

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/akuity/terraform-provider-akp/akp/types"
)

// If this test fails, a field has been added/removed to the Cluster related type.
// Update the schema attribute accordingly.
func TestNoNewClusterDataSourceFields(t *testing.T) {
	assert.Equal(t, reflect.TypeOf(types.Cluster{}).NumField(), len(getAKPClusterDataSourceAttributes()))
	assert.Equal(t, reflect.TypeOf(types.ClusterSpec{}).NumField(), len(getClusterSpecDataSourceAttributes()))
	assert.Equal(t, reflect.TypeOf(types.ClusterData{}).NumField(), len(getClusterDataDataSourceAttributes()))
}
