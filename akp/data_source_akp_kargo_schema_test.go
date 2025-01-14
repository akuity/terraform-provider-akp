//go:build !acc

package akp

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/akuity/terraform-provider-akp/akp/types"
)

// If this test fails, a field has been added/removed to the AKP Instance type.
// Update the schema attribute accordingly.
func TestNoNewKargoDataSourceFields(t *testing.T) {
	assert.Equal(t, reflect.TypeOf(types.KargoInstance{}).NumField(), len(getAKPKargoDataSourceAttributes()))
	assert.Equal(t, reflect.TypeOf(types.Kargo{}).NumField(), len(getKargoDataSourceAttributes()))
	assert.Equal(t, reflect.TypeOf(types.KargoSpec{}).NumField(), len(getKargoSpecDataSourceAttributes()))
	assert.Equal(t, reflect.TypeOf(types.KargoIPAllowListEntry{}).NumField(), len(getKargoIPAllowListEntryDataSourceAttributes()))
	assert.Equal(t, reflect.TypeOf(types.KargoAgentCustomization{}).NumField(), len(getKargoAgentCustomizationDataSourceAttributes()))
}
