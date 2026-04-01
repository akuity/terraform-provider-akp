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
	assert.Equal(t, reflect.TypeOf(types.KargoInstanceDataSource{}).NumField(), len(getAKPKargoDataSourceAttributes()))
	assert.Equal(t, reflect.TypeOf(types.KargoDataSource{}).NumField(), len(getKargoDataSourceAttributes()))
	assert.Equal(t, reflect.TypeOf(types.KargoSpecDataSource{}).NumField(), len(getKargoSpecDataSourceAttributes()))
	assert.Equal(t, reflect.TypeOf(types.KargoIPAllowListEntry{}).NumField(), len(getKargoIPAllowListEntryDataSourceAttributes()))
	assert.Equal(t, reflect.TypeOf(types.KargoAgentCustomization{}).NumField(), len(getKargoAgentCustomizationDataSourceAttributes()))
	assert.Equal(t, reflect.TypeOf(types.KargoInstanceSpec{}).NumField(), len(getKargoInstanceSpecDataSourceAttributes()))
	assert.Equal(t, reflect.TypeOf(types.KargoOidcConfigDataSource{}).NumField(), len(getOIDCConfigDataSourceAttributes()))
	assert.Equal(t, reflect.TypeOf(types.GarbageCollectorConfig{}).NumField(), len(getGarbageCollectorConfigDataSourceAttributes()))
	assert.Equal(t, reflect.TypeOf(types.AkuityIntelligence{}).NumField(), len(getKargoAkuityIntelligenceDataSourceAttributes()))
	assert.Equal(t, reflect.TypeOf(types.KargoArgoCDUIConfig{}).NumField(), len(getKargoArgoCDUIConfigDataSourceAttributes()))
}
