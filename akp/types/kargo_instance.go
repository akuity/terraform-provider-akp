package types

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"google.golang.org/protobuf/types/known/structpb"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"

	kargov1 "github.com/akuity/api-client-go/pkg/api/gen/kargo/v1"
)

type KargoInstance struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Kargo          *Kargo       `tfsdk:"kargo"`
	KargoConfigMap types.Map    `tfsdk:"kargo_cm"`
	KargoSecret    types.Map    `tfsdk:"kargo_secret"`
	Workspace      types.String `tfsdk:"workspace"`
	KargoResources types.Map    `tfsdk:"kargo_resources"`
}

// Update refreshes k from an export. filter applies only to a data source's kargo_resources;
// resources pass nil.
func (k *KargoInstance) Update(ctx context.Context, diagnostics *diag.Diagnostics, exportResp *kargov1.ExportKargoInstanceResponse, isDataSource bool, filter *ResourceFilter) error {
	if k.Kargo == nil {
		k.Kargo = &Kargo{}
	}
	plan := DeepCopy(k.Kargo)
	apiMap := exportResp.GetKargo().AsMap()
	if isDataSource {
		diagnostics.Append(BuildStateFromAPI(ctx, apiMap, k.Kargo, nil, KargoReverseOverridesMap, KargoReverseRenamesMap, "kargo")...)
	} else {
		diagnostics.Append(BuildStateFromAPI(ctx, apiMap, k.Kargo, plan, KargoReverseOverridesMap, KargoReverseRenamesMap, "kargo")...)
		preserveKargoInstanceAutoscalerPlanValues(k.Kargo, plan)
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
		return fmt.Errorf("unable to convert ConfigMap to struct: %w", err)
	}
	k.KargoConfigMap = ToConfigMapTFModel(ctx, diagnostics, configMapStruct, k.KargoConfigMap)

	if err := k.syncKargoResources(ctx, exportResp, diagnostics, isDataSource, filter); err != nil {
		return err
	}

	return nil
}

func (k *KargoInstance) syncKargoResources(
	ctx context.Context,
	exportResp *kargov1.ExportKargoInstanceResponse,
	diagnostics *diag.Diagnostics,
	isDataSource bool,
	filter *ResourceFilter,
) error {
	appliedResources := slices.Concat(
		exportResp.AnalysisTemplates, exportResp.PromotionTasks, exportResp.ClusterPromotionTasks,
		exportResp.Projects, exportResp.ProjectConfigs, exportResp.MessageChannels, exportResp.ClusterMessageChannels,
		exportResp.EventRouters, exportResp.CustomPromotionSteps, exportResp.ClusterConfigs, exportResp.Warehouses,
		exportResp.Stages, exportResp.ServiceAccounts, exportResp.Roles, exportResp.RoleBindings, exportResp.Configmaps,
		exportResp.ValidatingAdmissionPolicies, exportResp.ValidatingAdmissionPolicyBindings,
	)

	newMap, err := syncResources(
		ctx,
		diagnostics,
		k.KargoResources,
		appliedResources,
		"Kargo",
		isDataSource,
		filter,
	)
	if err != nil {
		return err
	}
	k.KargoResources = newMap
	return nil
}

// extractResourceMetadata extracts metadata from a resource
func extractResourceMetadata(resource any) (key, kindStr string, err error) {
	if m, ok := resource.(map[string]any); ok {
		kindVal, _ := m["kind"].(string)
		apiVersionVal, _ := m["apiVersion"].(string)
		nameVal := ""
		namespaceVal := ""
		if metadataMap, okMeta := m["metadata"].(map[string]any); okMeta {
			nameVal, _ = metadataMap["name"].(string)
			if v, ok := metadataMap["namespace"]; ok {
				namespaceVal, _ = v.(string)
			}
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
	resources types.Map,
	exportedResources []*structpb.Struct,
	resourceType string,
	isDataSource bool,
	filter *ResourceFilter,
) (types.Map, error) {
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

	elementsToAdd := make(map[string]attr.Value)
	if isDataSource {
		// Data sources should expose API resources as canonical JSON strings.
		for _, obj := range exportedResourceMap {
			// The filter is applied here too: portals that predate it export everything, and
			// admission policies are always exported in full.
			objMap := obj.AsMap()
			if !filter.keeps(objMap) || shouldExcludeDefaultDataSourceResource(resourceType, objMap) {
				continue
			}
			key, err := normalizeDataSourceResourceKey(resourceType, obj.AsMap())
			if err != nil {
				diagnostics.AddError(
					"Exported Resource Metadata Error",
					fmt.Sprintf("Error normalizing exported %s resource key: %s. Resource: %v", resourceType, err.Error(), obj.AsMap()),
				)
				continue
			}
			data, err := json.Marshal(obj.AsMap())
			if err != nil {
				diagnostics.AddError(
					"Exported Resource Serialization Error",
					fmt.Sprintf("Error serializing exported %s resource %s: %s", resourceType, key, err.Error()),
				)
				continue
			}
			elementsToAdd[key] = types.StringValue(string(data))
		}
	} else {
		// For resources: only keep existing resources that are also in the exported map
		for key, attrVal := range resources.Elements() {
			if _, ok := exportedResourceMap[key]; ok {
				elementsToAdd[key] = attrVal
				continue
			}
			// Not in export; preserve if it's a Secret
			if strVal, ok := attrVal.(types.String); ok && !strVal.IsNull() && !strVal.IsUnknown() {
				var objMap map[string]any
				if err := json.Unmarshal([]byte(strVal.ValueString()), &objMap); err == nil {
					if _, kind, err := extractResourceMetadata(objMap); err == nil && kind == "Secret" {
						elementsToAdd[key] = attrVal
					}
				}
			}
		}
	}
	if len(elementsToAdd) == 0 {
		// An explicit filter always yields a map, so callers can index and count it.
		if filter != nil {
			return types.MapValueMust(types.StringType, elementsToAdd), nil
		}
		return resources, nil
	}

	newMap, mapDiags := types.MapValueFrom(ctx, types.StringType, elementsToAdd)
	diagnostics.Append(mapDiags...)

	if mapDiags.HasError() {
		return resources, errors.New(fmt.Sprintf("error creating updated %s Resources map", resourceType))
	}

	return newMap, nil
}

func shouldExcludeDefaultDataSourceResource(resourceType string, resourceMap map[string]any) bool {
	if resourceType != "ArgoCD" {
		return false
	}

	kind, _ := resourceMap["kind"].(string)
	if kind != "AppProject" {
		return false
	}
	metadata, ok := resourceMap["metadata"].(map[string]any)
	if !ok {
		return false
	}

	name, _ := metadata["name"].(string)
	namespace, _ := metadata["namespace"].(string)
	return name == "default" && namespace == "argocd"
}

func normalizeDataSourceResourceKey(resourceType string, resourceMap map[string]any) (string, error) {
	key, kind, err := extractResourceMetadata(resourceMap)
	if err != nil {
		return "", err
	}
	if resourceType != "ArgoCD" {
		return key, nil
	}

	metadata, _ := resourceMap["metadata"].(map[string]any)
	namespace, _ := metadata["namespace"].(string)
	if namespace != "argocd" || !slices.Contains(argoCDKindsKeyedWithoutNamespace, kind) {
		return key, nil
	}

	apiVersion, _ := resourceMap["apiVersion"].(string)
	name, _ := metadata["name"].(string)
	return fmt.Sprintf("%s/%s//%s", apiVersion, kind, name), nil
}

// argoCDKindsKeyedWithoutNamespace are the Argo CD kinds whose data source keys leave out the
// argocd namespace.
var argoCDKindsKeyedWithoutNamespace = []string{"Application", "ApplicationSet", "AppProject"}

// DataSourceKeyNamespace is the namespace of an object whose data source key has namespace:
// argocd for the Argo CD kinds keyed without it.
func DataSourceKeyNamespace(resourceType, kind, namespace string) string {
	if resourceType == "ArgoCD" && namespace == "" && slices.Contains(argoCDKindsKeyedWithoutNamespace, kind) {
		return "argocd"
	}
	return namespace
}
