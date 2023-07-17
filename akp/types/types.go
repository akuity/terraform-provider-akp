package types

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ClusterObjectMeta struct {
	Name        types.String ` tfsdk:"name" json:"name,omitempty"`
	Namespace   types.String ` tfsdk:"namespace" json:"namespace,omitempty"`
	Labels      types.Map    `tfsdk:"labels" json:"labels,omitempty"`
	Annotations types.Map    `tfsdk:"annotations" json:"annotations,omitempty"`
}

type ObjectMeta struct {
	Name types.String ` tfsdk:"name" json:"name,omitempty"`
}

type SecretObjectMeta struct {
	Name   types.String ` tfsdk:"name" json:"name,omitempty"`
	Labels types.Map    `tfsdk:"labels" json:"labels,omitempty"`
}
