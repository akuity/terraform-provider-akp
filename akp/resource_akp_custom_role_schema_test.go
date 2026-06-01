//go:build !acc

package akp

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/akuity/terraform-provider-akp/akp/types"
)

// If this test fails, a field has been added/removed to types.CustomRole.
// Update the schema attribute accordingly.
func TestNoNewCustomRoleFields(t *testing.T) {
	assert.Equal(t, reflect.TypeFor[types.CustomRole]().NumField(), len(getCustomRoleAttributes()))
}
