//go:build !acc

package akp

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/stretchr/testify/assert"

	"github.com/akuity/terraform-provider-akp/akp/types"
)

// If this test fails, a field has been added/removed to the Kargo Agent related type.
// Update the schema attribute accordingly.
func TestNoNewKargoAgentFields(t *testing.T) {
	assert.Equal(t, reflect.TypeOf(types.KargoAgent{}).NumField(), len(types.KargoAgentResourceSchema.Attributes))
	assert.Equal(t, reflect.TypeOf(types.KargoAgentSpec{}).NumField(), len(types.KargoAgentResourceSchema.Attributes["spec"].(schema.SingleNestedAttribute).Attributes))
	assert.Equal(t, reflect.TypeOf(types.KargoAgentData{}).NumField(), len(types.KargoAgentResourceSchema.Attributes["spec"].(schema.SingleNestedAttribute).Attributes["data"].(schema.SingleNestedAttribute).Attributes))
}
