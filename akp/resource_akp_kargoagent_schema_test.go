//go:build !acc

package akp

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/akuity/terraform-provider-akp/akp/types"
)

// If this test fails, a field has been added/removed to the Kargo Agent related type.
// Update the schema attribute accordingly.
func TestNoNewKargoAgentFields(t *testing.T) {
	assert.Equal(t, reflect.TypeOf(types.KargoAgent{}).NumField(), len(getAKPKargoAgentResourceAttributes()))
	assert.Equal(t, reflect.TypeOf(types.KargoAgentSpec{}).NumField(), len(getAKPKargoAgentSpecAttributes()))
	assert.Equal(t, reflect.TypeOf(types.KargoAgentCustomization{}).NumField(), len(getKargoAgentCustomizationAttributes()))
}
