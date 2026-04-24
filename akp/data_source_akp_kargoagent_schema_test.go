//go:build !acc

package akp

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/akuity/terraform-provider-akp/akp/types"
)

func TestNoNewKargoAgentDataSourceFields(t *testing.T) {
	assert.Equal(t, reflect.TypeFor[types.KargoAgent]().NumField(), len(getAKPKargoAgentDataSourceAttributes()))
	assert.Equal(t, reflect.TypeFor[types.KargoAgentSpec]().NumField(), len(getAKPKargoAgentSpecDataSourceAttributes()))
	assert.Equal(t, reflect.TypeFor[types.KargoAgentData]().NumField(), len(getAKPKargoAgentDataDataSourceAttributes()))
	assert.Equal(t, reflect.TypeFor[types.KargoAutoscalerConfig]().NumField(), len(getKargoAutoscalerConfigDataSourceAttributes()))
	assert.Equal(t, reflect.TypeFor[types.KargoControllerAutoScalingConfig]().NumField(), len(getKargoControllerAutoScalingConfigDataSourceAttributes()))
	assert.Equal(t, reflect.TypeFor[types.KargoResources]().NumField(), len(getKargoResourcesDataSourceAttributes()))
}
