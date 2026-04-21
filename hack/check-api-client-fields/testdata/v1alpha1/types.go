// Package v1alpha1 is a fixture for the check-api-client-fields tests.
// It mimics the real akp/apis/v1alpha1 package: auto-generated Go structs with
// metav1 embeds, camelCase JSON tags, and initialism variants.
package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type Foo struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec FooSpec `json:"spec,omitempty"`
}

type FooSpec struct {
	Name        string               `json:"name,omitempty"`
	ClientID    string               `json:"clientId,omitempty"`
	IssuerURL   string               `json:"issuerUrl,omitempty"`
	TfOnlyField string               `json:"tfOnlyField,omitempty"`
	Blob        runtime.RawExtension `json:"blob,omitempty"`
	Enabled     *bool                `json:"enabled,omitempty"`
}

type TerraformOnlyStruct struct {
	Only string `json:"only,omitempty"`
}
