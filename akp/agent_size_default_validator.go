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
	var size tftypes.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, v.defaultsPath.AtName("size"), &size)...)
	var autoscaler tftypes.Object
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, v.defaultsPath.AtName("autoscaler_config"), &autoscaler)...)
	// An unknown size is resolved at apply time; the control plane still rejects a bad combination.
	if resp.Diagnostics.HasError() || size.IsUnknown() || autoscaler.IsNull() || autoscaler.IsUnknown() || size.ValueString() == "auto" {
		return
	}
	got := size.ValueString()
	if got == "" {
		got = "unset"
	}
	resp.Diagnostics.AddAttributeError(
		v.defaultsPath.AtName("autoscaler_config"),
		`autoscaler_config requires size = "auto"`,
		fmt.Sprintf(`autoscaler_config is only stored for an "auto" agent-size default, but size is %s. `+
			`The control plane drops it otherwise, which would fail the apply with an inconsistent-result error. `+
			`Set size = "auto" or remove autoscaler_config.`, got),
	)
}

var _ resource.ConfigValidator = agentSizeDefaultValidator{}
