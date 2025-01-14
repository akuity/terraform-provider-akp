package types

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/pkg/errors"

	kargov1 "github.com/akuity/api-client-go/pkg/api/gen/kargo/v1"
	"github.com/akuity/terraform-provider-akp/akp/apis/v1alpha1"
	"github.com/akuity/terraform-provider-akp/akp/marshal"
)

type KargoInstance struct {
	ID    types.String `tfsdk:"id"`
	Name  types.String `tfsdk:"name"`
	Kargo *Kargo       `tfsdk:"kargo"`
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
	k.Kargo.Update(ctx, diagnostics, kargo)
	return nil
}
