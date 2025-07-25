//go:build !acc

package akp

import (
	"reflect"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/stretchr/testify/assert"

	"github.com/akuity/terraform-provider-akp/akp/types"
)

func TestNoNewKargoAgentDataSourceFields(t *testing.T) {
	assert.Equal(t, reflect.TypeOf(types.KargoAgent{}).NumField(), len(types.KargoAgentDataSourceSchema.Attributes))
	assert.Equal(t, reflect.TypeOf(types.KargoAgentSpec{}).NumField(), len(types.KargoAgentDataSourceSchema.Attributes["spec"].(schema.SingleNestedAttribute).Attributes))
	assert.Equal(t, reflect.TypeOf(types.KargoAgentData{}).NumField(), len(types.KargoAgentDataSourceSchema.Attributes["spec"].(schema.SingleNestedAttribute).Attributes["data"].(schema.SingleNestedAttribute).Attributes))
}
