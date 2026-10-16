package akp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"google.golang.org/protobuf/types/known/structpb"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// resourceGroup maps a manifest kind to the API group it belongs to and the
// request slice that carries it.
type resourceGroup[Req any] struct {
	group string
	slice func(Req) *[]*structpb.Struct
}

// resourceValidator is a function type that validates a resource
type resourceValidator func(un *unstructured.Unstructured) error

// processResources processes a map of resources and appends them to the request
func processResources[T any](
	ctx context.Context,
	diagnostics *diag.Diagnostics,
	resources types.Map,
	resourceGroups map[string]resourceGroup[T],
	validateFunc resourceValidator,
	req T,
	resourceType string,
) {
	if resources.IsUnknown() {
		return
	}

	stringItems := make(map[string]types.String)
	diags := resources.ElementsAs(ctx, &stringItems, false)
	diagnostics.Append(diags...)
	if diagnostics.HasError() {
		return
	}

	resourceItems := make([]unstructured.Unstructured, 0, len(stringItems))
	for key, strItem := range stringItems {
		if strItem.IsNull() || strItem.IsUnknown() {
			continue
		}
		var objMap map[string]any
		if err := json.Unmarshal([]byte(strItem.ValueString()), &objMap); err != nil {
			diagnostics.AddError(
				fmt.Sprintf("Invalid %s Resource JSON, the input resource should be JSON format and will not be applied", resourceType),
				fmt.Sprintf("Failed to parse JSON for resource key '%s': %s\nResource content: %s", key, err.Error(), strItem.ValueString()),
			)
			continue
		}
		resourceItems = append(resourceItems, unstructured.Unstructured{Object: objMap})
	}
	if diagnostics.HasError() {
		return
	}

	for i, resourceItem := range resourceItems {
		if err := validateFunc(&resourceItem); err != nil {
			diagnostics.AddError(fmt.Sprintf("Invalid %s Resource %d", resourceType, i), err.Error())
			continue
		}

		resourceStructPb, err := structpb.NewStruct(resourceItem.Object)
		if err != nil {
			diagnostics.AddError(fmt.Sprintf("%s Resource Conversion Error", resourceType), fmt.Sprintf("Failed to convert resource %s (%s) to StructPb: %s", resourceItem.GetName(), resourceItem.GetKind(), err.Error()))
			continue
		}

		slice := resourceGroups[resourceItem.GetKind()].slice(req)
		*slice = append(*slice, resourceStructPb)
	}
}

// validateResource checks that the resource is a named object of a supported group/kind.
func validateResource[T any](un *unstructured.Unstructured, resourceGroups map[string]resourceGroup[T]) error {
	if un == nil {
		return errors.New("unstructured is nil")
	}
	if g, ok := resourceGroups[un.GetKind()]; !ok || g.group != un.GroupVersionKind().Group {
		return errors.New("unsupported kind")
	}
	if un.GetName() == "" {
		return errors.New("name is required")
	}
	return nil
}

func handleReadResourceError(ctx context.Context, resp *resource.ReadResponse, err error) {
	if isGoneErr(err) {
		resp.State.RemoveResource(ctx)
	} else {
		resp.Diagnostics.AddError("Client Error", err.Error())
	}
}

// pruneNormalizedEmptyFields drops spec.data keys whose value is "": the control
// plane normalizes them away instead of persisting an empty string, and sending
// "" back fails the apply (an empty maintenanceModeExpiry does not parse as a
// timestamp).
func pruneNormalizedEmptyFields(rawMap map[string]any, keys ...string) {
	dataMap, _ := rawMap["data"].(map[string]any)
	for _, key := range keys {
		if value, ok := dataMap[key].(string); ok && value == "" {
			delete(dataMap, key)
		}
	}
}

// admissionPolicyAPIVersion is the only apiVersion accepted for admission
// policy manifests in argocd_resources / kargo_resources.
const admissionPolicyAPIVersion = "admissionregistration.k8s.io/v1"

// admissionPolicyKinds are the manifests in argocd_resources / kargo_resources
// that the instance stores as its admission policies.
var admissionPolicyKinds = map[string]bool{
	"ValidatingAdmissionPolicy":        true,
	"ValidatingAdmissionPolicyBinding": true,
}

// hasAdmissionPolicyResources reports whether a resources map carries any
// admission policy manifest. Keys are user-chosen, so the manifests themselves
// are inspected.
func hasAdmissionPolicyResources(resources types.Map) bool {
	if resources.IsNull() || resources.IsUnknown() {
		return false
	}
	for _, element := range resources.Elements() {
		value, ok := element.(types.String)
		if !ok || value.IsNull() || value.IsUnknown() {
			continue
		}
		var manifest struct {
			APIVersion string `json:"apiVersion"`
			Kind       string `json:"kind"`
		}
		if err := json.Unmarshal([]byte(value.ValueString()), &manifest); err != nil {
			continue
		}
		if manifest.APIVersion == admissionPolicyAPIVersion && admissionPolicyKinds[manifest.Kind] {
			return true
		}
	}
	return false
}

type managesAdmissionPoliciesKey struct{}

// withManagedAdmissionPolicies records that the prior state managed admission
// policies, so the apply request built further down replaces the stored set
// even when the plan removed the last one.
func withManagedAdmissionPolicies(ctx context.Context, managed bool) context.Context {
	return context.WithValue(ctx, managesAdmissionPoliciesKey{}, managed)
}

// managesAdmissionPolicies reports whether Terraform owns the instance's
// admission policies: the plan or the prior state carries any. Only then does
// the apply replace the stored set, so policies managed in the portal UI are
// left alone for configurations that never declare one.
func managesAdmissionPolicies(ctx context.Context, plan types.Map) bool {
	if hasAdmissionPolicyResources(plan) {
		return true
	}
	managed, _ := ctx.Value(managesAdmissionPoliciesKey{}).(bool)
	return managed
}
