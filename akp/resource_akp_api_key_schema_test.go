//go:build !acc

package akp

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/akuity/terraform-provider-akp/akp/types"
)

// If this test fails, a field has been added/removed to one of the API key
// related types. Update the schema attribute accordingly.
func TestNoNewApiKeyFields(t *testing.T) {
	assert.Equal(t, reflect.TypeFor[types.ApiKey]().NumField(), len(getApiKeyAttributes()))
	assert.Equal(t, reflect.TypeFor[types.ApiKeyPermissions]().NumField(), len(getApiKeyPermissionsAttributes()))
}
