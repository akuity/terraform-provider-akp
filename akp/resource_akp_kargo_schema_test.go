//go:build !acc

package akp

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/akuity/terraform-provider-akp/akp/types"
)

// If this test fails, a field has been added/removed to the AKP Kargo Instance type.
// Update the schema attribute accordingly.
func TestNoNewKargoFields(t *testing.T) {
	assert.Equal(t, reflect.TypeOf(types.Kargo{}).NumField(), len(getKargoAttributes()))
	assert.Equal(t, reflect.TypeOf(types.KargoSpec{}).NumField(), len(getKargoSpecAttributes()))
	assert.Equal(t, reflect.TypeOf(types.KargoIPAllowListEntry{}).NumField(), len(getKargoIPAllowListEntryAttributes()))
	assert.Equal(t, reflect.TypeOf(types.KargoAgentCustomization{}).NumField(), len(getKargoAgentCustomizationAttributes()))
}
