package akp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"google.golang.org/protobuf/types/known/structpb"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// resourceGroupAppender is a function type that appends a resource to a request
type resourceGroupAppender[T any] func(req T, item *structpb.Struct)

// resourceValidator is a function type that validates a resource
type resourceValidator func(un *unstructured.Unstructured) error

// processResources processes a map of resources and appends them to the request
func processResources[T any](
	ctx context.Context,
	diagnostics *diag.Diagnostics,
	resources types.Map,
	resourceGroups map[string]struct {
		appendFunc resourceGroupAppender[T]
	},
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
	for _, strItem := range stringItems {
		if strItem.IsNull() || strItem.IsUnknown() {
			continue
		}
		var objMap map[string]any
		if err := json.Unmarshal([]byte(strItem.ValueString()), &objMap); err != nil {
			continue
		}
		resourceItems = append(resourceItems, unstructured.Unstructured{Object: objMap})
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

		resourceGroups[resourceItem.GetKind()].appendFunc(req, resourceStructPb)
	}
}

// validateResource validates a resource with the given API version and resource groups
func validateResource[T any](un *unstructured.Unstructured, apiVersion string, resourceGroups map[string]struct {
	appendFunc resourceGroupAppender[T]
},
) error {
	if un == nil {
		return errors.New("unstructured is nil")
	}

	if un.GetAPIVersion() != apiVersion {
		return errors.New("unsupported apiVersion")
	}

	if _, ok := resourceGroups[un.GetKind()]; !ok {
		return errors.New("unsupported kind")
	}

	if un.GetName() == "" {
		return errors.New("name is required")
	}

	return nil
}
