package types

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ConfigMap struct {
	ObjectMeta `json:"metadata" tfsdk:"metadata"`
	Data       types.Map `json:"data,omitempty" tfsdk:"data"`
}
