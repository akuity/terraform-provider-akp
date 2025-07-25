//go:build !acc

package akp

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/stretchr/testify/assert"

	"github.com/akuity/terraform-provider-akp/akp/types"
)

// If this test fails, a field has been added/removed to the Cluster related type.
// Update the schema attribute accordingly.
func TestNoNewClusterDataSourceFields(t *testing.T) {
	assert.Equal(t, reflect.TypeOf(types.Cluster{}).NumField(), len(types.ClusterDataSourceSchema.Attributes))
	assert.Equal(t, reflect.TypeOf(types.ClusterSpec{}).NumField(), len(types.ClusterDataSourceSchema.Attributes["spec"].(schema.SingleNestedAttribute).Attributes))
	assert.Equal(t, reflect.TypeOf(types.ClusterData{}).NumField(), len(types.ClusterDataSourceSchema.Attributes["spec"].(schema.SingleNestedAttribute).Attributes["data"].(schema.SingleNestedAttribute).Attributes))
}
