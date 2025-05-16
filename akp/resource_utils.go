package akp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"google.golang.org/protobuf/types/known/structpb"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// ResourceGroupAppender is a function type that appends a resource to a request
type ResourceGroupAppender func(req interface{}, item *structpb.Struct)

// ResourceValidator is a function type that validates a resource
type ResourceValidator func(un *unstructured.Unstructured) error

// ProcessResources processes a list of resources and appends them to the request
func ProcessResources(
	ctx context.Context,
	diagnostics *diag.Diagnostics,
	resources types.List,
	resourceGroups map[string]struct {
		appendFunc ResourceGroupAppender
	},
	validateFunc ResourceValidator,
	req interface{},
	resourceType string,
) {
	if resources.IsUnknown() {
		return
	}

	var stringItems []types.String
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

// InstanceBuilder is a function type that converts an instance to its API model
type InstanceBuilder[T any] func(ctx context.Context, diag *diag.Diagnostics, name string) T

// SpecProcessor is a function type that processes the spec map
type SpecProcessor func(spec map[string]any, apiModel interface{})

// BuildInstance is a generic function that builds an instance struct
func BuildInstance[T any](
	ctx context.Context,
	diagnostics *diag.Diagnostics,
	instance interface{},
	name string,
	toAPIModel InstanceBuilder[T],
	processSpec SpecProcessor,
) *structpb.Struct {
	apiModel := toAPIModel(ctx, diagnostics, name)
	jsonBytes, err := json.Marshal(apiModel)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to marshal instance. %s", err))
		return nil
	}

	var rawMap map[string]any
	if err = json.Unmarshal(jsonBytes, &rawMap); err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to unmarshal instance. %s", err))
		return nil
	}

	if spec, ok := rawMap["spec"].(map[string]any); ok {
		processSpec(spec, apiModel)
	}

	s, err := structpb.NewStruct(rawMap)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create instance struct. %s", err))
		return nil
	}
	return s
}
