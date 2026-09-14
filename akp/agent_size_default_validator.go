package akp

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
)

// agentSizeDefaultValidator rejects an instance-level agent-size default that
// carries scaling limits for a size that does not use them.
//
// The control plane keeps `autoscaler_config` only for an `auto` default and
// drops it otherwise. Post-apply state is rebuilt from ExportInstance, so a
// config that sets it against another size fails with the opaque "Provider
// produced inconsistent result after apply: .autoscaler_config: was
// cty.ObjectVal{...}, but now null". Catching it at plan time names the actual
// problem instead.
type agentSizeDefaultValidator struct {
	// defaultsPath locates the customization defaults object, which differs
	// between the Argo CD instance and the Kargo instance schemas.
	defaultsPath path.Path
	// autoSize is the size value the scaling limits belong to.
	autoSize string
}

func (v agentSizeDefaultValidator) Description(context.Context) string {
	return "Validates that an agent-size default's scaling limits match the selected size"
}

func (v agentSizeDefaultValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v agentSizeDefaultValidator) ValidateResource(
	ctx context.Context,
	req resource.ValidateConfigRequest,
	resp *resource.ValidateConfigResponse,
) {
	var defaults tftypes.Object
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, v.defaultsPath, &defaults)...)
	if resp.Diagnostics.HasError() || defaults.IsNull() || defaults.IsUnknown() {
		return
	}

	var size tftypes.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, v.defaultsPath.AtName("size"), &size)...)
	if resp.Diagnostics.HasError() || size.IsUnknown() {
		return
	}

	// A null size means no default at all, which cannot carry either override.
	sizeValue := ""
	if !size.IsNull() {
		sizeValue = size.ValueString()
	}

	check := func(attribute, requiredSize string) {
		var override tftypes.Object
		resp.Diagnostics.Append(req.Config.GetAttribute(ctx, v.defaultsPath.AtName(attribute), &override)...)
		if resp.Diagnostics.HasError() || override.IsNull() || override.IsUnknown() {
			return
		}
		if sizeValue == requiredSize {
			return
		}
		got := sizeValue
		if got == "" {
			got = "unset"
		}
		resp.Diagnostics.AddAttributeError(
			v.defaultsPath.AtName(attribute),
			fmt.Sprintf("%s requires size = %q", attribute, requiredSize),
			fmt.Sprintf(
				"%s is only stored for a %q agent-size default, but size is %s. "+
					"The control plane drops it otherwise, which would fail the apply with an "+
					"inconsistent-result error. Set size = %q or remove %s.",
				attribute, requiredSize, got, requiredSize, attribute,
			),
		)
	}

	check("autoscaler_config", v.autoSize)
}

var _ resource.ConfigValidator = agentSizeDefaultValidator{}
