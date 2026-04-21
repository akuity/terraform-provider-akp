package types

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	kargov1 "github.com/akuity/api-client-go/pkg/api/gen/kargo/v1"
	"github.com/akuity/terraform-provider-akp/akp/marshal"
)

var (
	kargoAgentSizeProtoToTF = map[string]string{
		"KARGO_AGENT_SIZE_SMALL":       "small",
		"KARGO_AGENT_SIZE_MEDIUM":      "medium",
		"KARGO_AGENT_SIZE_LARGE":       "large",
		"KARGO_AGENT_SIZE_AUTO":        "auto",
		"KARGO_AGENT_SIZE_UNSPECIFIED": "unspecified",
	}
	KargoOverridesMap = overrideMap{
		"spec.fqdn": alwaysIncludeString(),
		"spec.kargo_instance_spec.agent_customization_defaults.kustomization": yamlStringToObject(),
		"spec.oidc_config.dex_config_secret":                                  mapStringToValueObject(),
		"data.kustomization":                                                  yamlStringToObject(),
		"data.maintenance_mode_expiry":                                        suppressEmptyString(),
	}

	KargoRenamesMap = renameMap{}

	// KargoReverseOverridesMap defines custom API→TF conversion logic for Kargo resources.
	KargoReverseOverridesMap = reverseOverrideMap{
		// Kustomization fields are objects in the API but YAML strings in TF
		"spec.kargo_instance_spec.agent_customization_defaults.kustomization": ObjectToYAMLString(),
		"data.kustomization": ObjectToYAMLString(),
		// dex_config_secret wraps string values in {value: "..."}  objects
		"spec.oidc_config.dex_config_secret": ValueObjectToMapString(),
		// TF-only fields for KargoAgent
		"remove_agent_resources_on_destroy": TFOnlyField(types.BoolValue(true)),
		"reapply_manifests_on_update":       TFOnlyField(types.BoolValue(false)),
		// Enum fields: protojson outputs proto names, TF expects lowercase
		"data.size": ProtoEnumToLowerString(kargoAgentSizeProtoToTF),
	}

	// KargoReverseRenamesMap maps tfsdk tags to API camelCase keys for the reverse direction.
	KargoReverseRenamesMap = KargoRenamesMap
)

type AgentMaps struct {
	NameToID map[string]string
	IDToName map[string]string
}

func (ka *KargoAgent) Update(ctx context.Context, diagnostics *diag.Diagnostics, apiKargoAgent *kargov1.KargoAgent, plan *KargoAgent) {
	ka.ID = types.StringValue(apiKargoAgent.GetId())
	ka.Name = types.StringValue(apiKargoAgent.GetName())
	ka.Namespace = types.StringValue(apiKargoAgent.GetData().GetNamespace())

	labels, d := types.MapValueFrom(ctx, types.StringType, apiKargoAgent.GetData().GetLabels())
	if d.HasError() {
		labels = types.MapNull(types.StringType)
	}
	diagnostics.Append(d...)
	annotations, d := types.MapValueFrom(ctx, types.StringType, apiKargoAgent.GetData().GetAnnotations())
	if d.HasError() {
		annotations = types.MapNull(types.StringType)
	}
	diagnostics.Append(d...)
	ka.Labels = labels
	ka.Annotations = annotations

	if ka.RemoveAgentResourcesOnDestroy.IsUnknown() || ka.RemoveAgentResourcesOnDestroy.IsNull() {
		ka.RemoveAgentResourcesOnDestroy = types.BoolValue(true)
	}
	if ka.ReapplyManifestsOnUpdate.IsUnknown() || ka.ReapplyManifestsOnUpdate.IsNull() {
		ka.ReapplyManifestsOnUpdate = types.BoolValue(false)
	}

	if ka.Spec == nil {
		ka.Spec = &KargoAgentSpec{}
	}
	var planSpec *KargoAgentSpec
	if plan != nil && plan.Spec != nil {
		planSpec = DeepCopyKargoAgentSpec(plan.Spec)
	}

	apiMap, err := marshal.ProtoToMap(apiKargoAgent)
	if err != nil {
		diagnostics.AddError("failed to marshal kargo agent proto to map", err.Error())
		return
	}
	var planArg *KargoAgentSpec
	if plan != nil {
		planArg = planSpec
	}
	diagnostics.Append(BuildStateFromAPI(ctx, apiMap, ka.Spec, planArg, KargoReverseOverridesMap, KargoReverseRenamesMap, "spec")...)

	if !apiKargoAgent.GetData().GetAkuityManaged() &&
		apiKargoAgent.GetData().GetRemoteArgocd() == "" &&
		planSpec != nil &&
		!planSpec.Data.ArgocdNamespace.IsUnknown() {
		ka.Spec.Data.ArgocdNamespace = planSpec.Data.ArgocdNamespace
	}

	preserveKargoAutoscalerConfigPlanValues(ka.Spec.Data.AutoscalerConfig, planSpec)

	defaultNullOrUnknownString(&ka.Spec.Data.ArgocdNamespace)
	defaultNullOrUnknownString(&ka.Spec.Data.Kustomization)
	defaultNullOrUnknownString(&ka.Spec.Data.MaintenanceModeExpiry)
	defaultNullOrUnknownBool(&ka.Spec.Data.AkuityManaged)
	defaultNullOrUnknownBool(&ka.Spec.Data.MaintenanceMode)
	NormalizeKargoAgentReadStateForRefresh(ka)
}

// preserveKargoAutoscalerConfigPlanValues keeps the user's original resource
// quantity strings (e.g. "1Gi", "500m") when the API returns a normalized
// equivalent (e.g. "1.00Gi", "500m"), preventing perpetual plan diffs.
func preserveKargoAutoscalerConfigPlanValues(state *KargoAutoscalerConfig, plan *KargoAgentSpec) {
	if state == nil || state.KargoController == nil {
		return
	}
	if plan == nil || plan.Data.AutoscalerConfig == nil || plan.Data.AutoscalerConfig.KargoController == nil {
		return
	}
	ctrl := state.KargoController
	planCtrl := plan.Data.AutoscalerConfig.KargoController

	if ctrl.ResourceMinimum != nil && planCtrl.ResourceMinimum != nil {
		if areResourcesEquivalent(planCtrl.ResourceMinimum.Mem.ValueString(), ctrl.ResourceMinimum.Mem.ValueString()) {
			ctrl.ResourceMinimum.Mem = planCtrl.ResourceMinimum.Mem
		}
		if areResourcesEquivalent(planCtrl.ResourceMinimum.Cpu.ValueString(), ctrl.ResourceMinimum.Cpu.ValueString()) {
			ctrl.ResourceMinimum.Cpu = planCtrl.ResourceMinimum.Cpu
		}
	}
	if ctrl.ResourceMaximum != nil && planCtrl.ResourceMaximum != nil {
		if areResourcesEquivalent(planCtrl.ResourceMaximum.Mem.ValueString(), ctrl.ResourceMaximum.Mem.ValueString()) {
			ctrl.ResourceMaximum.Mem = planCtrl.ResourceMaximum.Mem
		}
		if areResourcesEquivalent(planCtrl.ResourceMaximum.Cpu.ValueString(), ctrl.ResourceMaximum.Cpu.ValueString()) {
			ctrl.ResourceMaximum.Cpu = planCtrl.ResourceMaximum.Cpu
		}
	}
}

func defaultNullOrUnknownString(field *types.String) {
	if field == nil || (!field.IsNull() && !field.IsUnknown()) {
		return
	}
	*field = types.StringValue("")
}

func defaultNullOrUnknownBool(field *types.Bool) {
	if field == nil || (!field.IsNull() && !field.IsUnknown()) {
		return
	}
	*field = types.BoolValue(false)
}

func NormalizeKargoAgentReadStateForRefresh(agent *KargoAgent) {
	if agent == nil || agent.Spec == nil {
		return
	}

	// The control plane clears argocd_namespace when an agent targets a remote
	// Argo CD or is Akuity-managed. Refresh should mirror that normalized state.
	if agent.Spec.Data.AkuityManaged.ValueBool() || agent.Spec.Data.RemoteArgocd.ValueString() != "" {
		agent.Spec.Data.ArgocdNamespace = types.StringValue("")
	}

	// The control plane only persists an expiry while maintenance mode is enabled.
	if !agent.Spec.Data.MaintenanceMode.ValueBool() {
		agent.Spec.Data.MaintenanceModeExpiry = types.StringValue("")
	}
}
