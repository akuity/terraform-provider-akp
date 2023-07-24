package akp

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/akuity/terraform-provider-akp/akp/types"
)

// If this test fails, a field has been added/removed to the Clusters related type.
// Update the schema attribute accordingly.
func TestNoNewClustersDataSourceFields(t *testing.T) {
	assert.Equal(t, reflect.TypeOf(types.Clusters{}).NumField(), len(getAKPClustersDataSourceAttributes()))
}
