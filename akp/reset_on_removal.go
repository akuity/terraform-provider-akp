package akp

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	tfgotypes "github.com/hashicorp/terraform-plugin-go/tftypes"

	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	kargov1 "github.com/akuity/api-client-go/pkg/api/gen/kargo/v1"
)

// Description sentences for the custom_ca_bundle attributes ResetOnRemoval
// covers. They describe editing configuration, so toDataSourceAttribute drops
// them from the derived data-source schemas. The provider cannot tell a value
// removed from configuration from one that was never in it, so a bundle set
// outside Terraform is reset just the same.
const (
	clusterCABundleRemovalNote = " Omitting it from your configuration resets a non-empty bundle, including one set outside " +
		"Terraform, to the one a new cluster would inherit from the instance-level default, which is the empty bundle " +
		"when the instance defines none. A cluster with no bundle keeps none, even once the instance defines a default."
	kargoAgentCABundleRemovalNote = " Omitting it from your configuration resets a non-empty bundle, including one set outside " +
		"Terraform, to the one a new agent would inherit from the instance-level default, which is the empty bundle " +
		"when the instance defines none. An agent with no bundle keeps none, even once the instance defines a default."
	instanceCABundleRemovalNote = " Omitting it from your configuration clears the default, including one set outside " +
		"Terraform; clusters that inherited the old default are reset on their next plan."
	kargoInstanceCABundleRemovalNote = " Omitting it from your configuration clears the default, including one set outside " +
		"Terraform; agents that inherited the old default are reset on their next plan."
)

// resourceOnlyNotes lists the description suffixes that only make sense on a
// resource schema.
var resourceOnlyNotes = []string{
	clusterCABundleRemovalNote,
	kargoAgentCABundleRemovalNote,
	instanceCABundleRemovalNote,
	kargoInstanceCABundleRemovalNote,
}

// resetOnRemoval declares an Optional+Computed string attribute that goes back
// to the value a freshly created resource would receive once the practitioner
// deletes it from configuration.
//
// An attribute plan modifier cannot do this. The framework marks a Computed
// attribute with a null configuration unknown, and the UseStateForUnknown
// modifier these attributes carry refills it from prior state, silently
// dropping the removal (issue #12590). Nor can the reset be unconditional:
// prior state may hold a default the child inherited from its instance at
// create time rather than a value the practitioner wrote, and telling those
// apart costs an API call — hence ModifyPlan.
//
// That default is read from the live parent, so changing a parent's default and
// letting a child follow it converges over two applies; the child's plan still
// sees the old default. A provider gets no view of another resource's planned
// value, so that is inherent rather than a choice.
type resetOnRemoval struct {
	// attribute locates the value in the resource schema.
	attribute path.Path
	// inherited resolves what a freshly created resource would receive, from an
	// existing one's prior state. Nil means there is nothing to inherit, so
	// removal resets to the protobuf default ("").
	inherited func(ctx context.Context, cli *AkpCli, state tfsdk.State) (string, error)
}

// ModifyPlan resets attributes the configuration no longer sets. It runs after
// the schema's attribute plan modifiers, so it sees — and can correct — the
// value UseStateForUnknown put back.
func (r *GenericResource[Plan]) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// A create has no prior value to reset, a destroy no plan to write to.
	if len(r.ResetOnRemoval) == 0 || req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}
	// Unconfigured during `terraform validate`, where nothing can be resolved.
	if r.akpCli == nil {
		return
	}

	ctx = r.AuthCtx(ctx)
	for _, entry := range r.ResetOnRemoval {
		var configValue, stateValue tftypes.String
		resp.Diagnostics.Append(req.Config.GetAttribute(ctx, entry.attribute, &configValue)...)
		resp.Diagnostics.Append(req.State.GetAttribute(ctx, entry.attribute, &stateValue)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if !removalPending(configValue, stateValue) || configUnknown(req.Config, entry.attribute) {
			continue
		}

		inherited := ""
		if entry.inherited != nil {
			resolved, err := entry.inherited(ctx, r.akpCli, req.State)
			if err != nil {
				resp.Diagnostics.AddError(
					"Client Error",
					fmt.Sprintf(
						"%s was removed from the configuration, but the value a new %s would inherit could not be "+
							"resolved, so the plan cannot say what it becomes: %s",
						entry.attribute, r.TypeNameSuffix, err,
					),
				)
				return
			}
			inherited = resolved
		}

		value, ok := resetTo(stateValue, inherited)
		if !ok {
			continue
		}
		resp.Diagnostics.Append(resp.Plan.SetAttribute(ctx, entry.attribute, value)...)
	}
}

// removalPending reports whether the configuration dropped an attribute that
// still holds a value. Checked before resolving the inherited value, so an
// unchanged configuration costs no API call. A null configuration value can
// also mean "not resolvable yet"; configUnknown tells the two apart.
func removalPending(configValue, stateValue tftypes.String) bool {
	if !configValue.IsNull() {
		return false
	}
	if stateValue.IsNull() || stateValue.IsUnknown() {
		return false
	}
	return stateValue.ValueString() != ""
}

// resetTo reports the value to plan for a removed attribute. ok is false when
// prior state already holds what a fresh resource would inherit: that value
// came from the control plane, not from the configuration just edited.
func resetTo(stateValue tftypes.String, inherited string) (string, bool) {
	if stateValue.ValueString() == inherited {
		return "", false
	}
	return inherited, true
}

// configUnknown reports whether the configuration cannot yet say what the
// attribute holds. GetAttribute cannot answer that: an attribute under an
// unknown object reads back as null, which is what a removal looks like, so
// the ancestors are walked in the raw configuration instead. The walk stops at
// the first unknown or null ancestor and returns it; a null one is a deleted
// block, which is a removal.
// https://github.com/hashicorp/terraform-plugin-framework/issues/186
func configUnknown(config tfsdk.Config, attribute path.Path) bool {
	steps := tfgotypes.NewAttributePath()
	for _, step := range attribute.Steps() {
		name, ok := step.(path.PathStepAttributeName)
		if !ok {
			return true // ResetOnRemoval only names attributes; do not guess
		}
		steps = steps.WithAttributeName(string(name))
	}
	reached, _, _ := tfgotypes.WalkAttributePath(config.Raw, steps)
	value, ok := reached.(tfgotypes.Value)
	return !ok || !value.IsKnown()
}

// kargoAgentInheritedCustomCABundle returns the CA bundle a new agent of this
// agent's Kargo instance would be given.
func kargoAgentInheritedCustomCABundle(ctx context.Context, cli *AkpCli, state tfsdk.State) (string, error) {
	instanceID, err := resourceInstanceID(ctx, state)
	if err != nil {
		return "", err
	}
	resp, err := retryWithBackoff(ctx, func(ctx context.Context) (*kargov1.GetKargoInstanceResponse, error) {
		return getKargoInstanceByIdentity(ctx, cli.KargoCli, cli.OrgId, instanceID, "")
	}, "GetKargoInstance")
	if err != nil {
		return "", err
	}
	return resp.GetInstance().GetSpec().GetAgentCustomizationDefaults().GetCustomCaBundle(), nil
}

// clusterInheritedCustomCABundle returns the CA bundle a new cluster of this
// cluster's Argo CD instance would be given.
func clusterInheritedCustomCABundle(ctx context.Context, cli *AkpCli, state tfsdk.State) (string, error) {
	instanceID, err := resourceInstanceID(ctx, state)
	if err != nil {
		return "", err
	}
	resp, err := retryWithBackoff(ctx, func(ctx context.Context) (*argocdv1.GetInstanceResponse, error) {
		return cli.Cli.GetInstance(ctx, instanceGetRequest(cli.OrgId, instanceID, ""))
	}, "GetInstance")
	if err != nil {
		return "", err
	}
	return resp.GetInstance().GetSpec().GetClusterCustomizationDefaults().GetCustomCaBundle(), nil
}

// resourceInstanceID reads the parent instance ID from a child resource's
// prior state, which always carries the resolved ID for a resource that
// already exists.
func resourceInstanceID(ctx context.Context, state tfsdk.State) (string, error) {
	var instanceID tftypes.String
	if diags := state.GetAttribute(ctx, path.Root("instance_id"), &instanceID); diags.HasError() {
		return "", fmt.Errorf("unable to read instance_id from state: %v", diags.Errors())
	}
	if instanceID.ValueString() == "" {
		return "", fmt.Errorf("instance_id is not set in state")
	}
	return instanceID.ValueString(), nil
}
