package akp

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/akuity/terraform-provider-akp/akp/types"
)

func TestNoNewKargoAgentDataSourceFields(t *testing.T) {
	assert.Equal(t, reflect.TypeOf(types.KargoAgent{}).NumField(), len(getAKPKargoAgentDataSourceAttributes()))
	assert.Equal(t, reflect.TypeOf(types.KargoAgentSpec{}).NumField(), len(getAKPKargoAgentSpecDataSourceAttributes()))
	assert.Equal(t, reflect.TypeOf(types.KargoAgentData{}).NumField(), len(getAKPKargoAgentDataDataSourceAttributes()))
}
