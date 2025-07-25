//go:build !acc

package akp

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/stretchr/testify/assert"

	"github.com/akuity/terraform-provider-akp/akp/types"
)

// If this test fails, a field has been added/removed to the Cluster related type.
// Update the schema attribute accordingly.
func TestNoNewClusterResourceFields(t *testing.T) {
	assert.Equal(t, reflect.TypeOf(types.Cluster{}).NumField(), len(types.ClusterResourceSchema.Attributes))
	assert.Equal(t, reflect.TypeOf(types.ClusterSpec{}).NumField(), len(types.ClusterResourceSchema.Attributes["spec"].(schema.SingleNestedAttribute).Attributes))
	assert.Equal(t, reflect.TypeOf(types.ClusterData{}).NumField(), len(types.ClusterResourceSchema.Attributes["spec"].(schema.SingleNestedAttribute).Attributes["data"].(schema.SingleNestedAttribute).Attributes))
}
