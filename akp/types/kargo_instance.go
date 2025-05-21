package types

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/structpb"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	kargov1 "github.com/akuity/api-client-go/pkg/api/gen/kargo/v1"
	"github.com/akuity/terraform-provider-akp/akp/apis/v1alpha1"
	"github.com/akuity/terraform-provider-akp/akp/marshal"
)

type KargoInstance struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Kargo          *Kargo       `tfsdk:"kargo"`
	KargoConfigMap types.Map    `tfsdk:"kargo_cm"`
	KargoSecret    types.Map    `tfsdk:"kargo_secret"`
	Workspace      types.String `tfsdk:"workspace"`
	KargoResources types.List   `tfsdk:"kargo_resources"`
}

func (k *KargoInstance) Update(ctx context.Context, diagnostics *diag.Diagnostics, exportResp *kargov1.ExportKargoInstanceResponse) error {
	var kargo *v1alpha1.Kargo
	err := marshal.RemarshalTo(exportResp.GetKargo().AsMap(), &kargo)
	if err != nil {
		return errors.Wrap(err, "Unable to get Kargo instance")
	}
	if k.Kargo == nil {
		k.Kargo = &Kargo{}
	}

	// Convert ConfigMap values, ensuring booleans are converted to strings
	configMap := exportResp.GetKargoConfigmap().AsMap()
	if !k.KargoConfigMap.IsNull() {
		existingConfigMap := k.KargoConfigMap.Elements()
		for key, value := range existingConfigMap {
			if _, exists := configMap[key]; !exists {
				if strVal, ok := value.(types.String); ok {
					configMap[key] = strVal.ValueString()
				}
			}
		}
	}
	for k, v := range configMap {
		switch val := v.(type) {
		case bool:
			configMap[k] = fmt.Sprintf("%t", val)
		}
	}
	configMapStruct, err := structpb.NewStruct(configMap)
	if err != nil {
		return errors.Wrap(err, "Unable to convert ConfigMap to struct")
	}
	k.KargoConfigMap = ToConfigMapTFModel(ctx, diagnostics, configMapStruct, k.KargoConfigMap)
	k.Kargo.Update(ctx, diagnostics, kargo)

	if err := k.syncKargoResources(ctx, exportResp, diagnostics); err != nil {
		return err
	}

	return nil
}

func (k *KargoInstance) syncKargoResources(
	ctx context.Context,
	exportResp *kargov1.ExportKargoInstanceResponse,
	diagnostics *diag.Diagnostics,
) error {
	appliedResources := make([]*structpb.Struct, 0)
	appliedResources = append(appliedResources, exportResp.AnalysisTemplates...)
	appliedResources = append(appliedResources, exportResp.PromotionTasks...)
	appliedResources = append(appliedResources, exportResp.ClusterPromotionTasks...)
	appliedResources = append(appliedResources, exportResp.Projects...)
	appliedResources = append(appliedResources, exportResp.Warehouses...)
	appliedResources = append(appliedResources, exportResp.Stages...)

	newList, err := syncResources(
		ctx,
		diagnostics,
		k.KargoResources,
		appliedResources,
		"Kargo",
	)
	if err != nil {
		return err
	}
	k.KargoResources = newList
	return nil
}

// extractResourceMetadata extracts metadata from a resource
func extractResourceMetadata(resource any) (key string, kindStr string, err error) {
	if m, ok := resource.(map[string]any); ok {
		kindVal, _ := m["kind"].(string)
		apiVersionVal, _ := m["apiVersion"].(string)
		nameVal := ""
		namespaceVal := ""
		if metadataMap, okMeta := m["metadata"].(map[string]any); okMeta {
			nameVal, _ = metadataMap["name"].(string)
			namespaceVal, _ = metadataMap["namespace"].(string)
		}
		if kindVal != "" && nameVal != "" {
			return fmt.Sprintf("%s/%s/%s/%s", apiVersionVal, kindVal, namespaceVal, nameVal), kindVal, nil
		}
	}

	return "", "", fmt.Errorf("extractResourceMetadata: unsupported type %T or insufficient data to form key/kind", resource)
}

// syncResources synchronizes resources between the current state and the exported state
func syncResources(
	ctx context.Context,
	diagnostics *diag.Diagnostics,
	resources types.List,
	exportedResources []*structpb.Struct,
	resourceType string,
) (types.List, error) {
	if resources.IsUnknown() {
		return resources, nil
	}

	exportedResourceMap := make(map[string]*structpb.Struct)
	for _, resStruct := range exportedResources {
		var unstrObj unstructured.Unstructured
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(resStruct.AsMap(), &unstrObj); err != nil {
			diagnostics.AddError(
				"Exported Resource Conversion Error",
				fmt.Sprintf("Error converting exported resource to unstructured: %s. Resource: %v", err.Error(), resStruct),
			)
			continue
		}
		key, _, err := extractResourceMetadata(unstrObj.Object)
		if err != nil {
			diagnostics.AddError(
				"Exported Resource Metadata Error",
				fmt.Sprintf("Error extracting metadata from exported resource: %s. Resource: %v", err.Error(), unstrObj.Object),
			)
			continue
		}
		exportedResourceMap[key] = resStruct
	}

	if diagnostics.HasError() {
		return resources, errors.New("error processing resources from export response, cannot reliably sync")
	}

	elementsToAdd := make([]attr.Value, 0)
	if len(resources.Elements()) == 0 {
		for _, obj := range exportedResourceMap {
			elementsToAdd = append(elementsToAdd, types.StringValue(obj.String()))
		}
	} else {
		for _, attrVal := range resources.Elements() {
			resourceStrVal, ok := attrVal.(types.String)
			if !ok {
				continue
			}

			var objMap map[string]any
			if err := json.Unmarshal([]byte(resourceStrVal.ValueString()), &objMap); err != nil {
				continue
			}

			unObj := unstructured.Unstructured{Object: objMap}
			key, _, err := extractResourceMetadata(unObj.Object)
			if err != nil {
				continue
			}

			if _, ok := exportedResourceMap[key]; !ok {
				continue
			}

			elementsToAdd = append(elementsToAdd, attrVal)
		}
	}

	newList, listDiags := types.ListValueFrom(ctx, types.StringType, elementsToAdd)
	diagnostics.Append(listDiags...)

	if listDiags.HasError() {
		return resources, errors.New(fmt.Sprintf("error creating updated %s Resources list", resourceType))
	}

	return newList, nil
}
