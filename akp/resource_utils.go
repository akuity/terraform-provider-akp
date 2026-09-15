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
