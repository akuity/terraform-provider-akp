package types

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/structpb"

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
	return nil
}
