package types

import (
	"bytes"
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"

	kargov1 "github.com/akuity/api-client-go/pkg/api/gen/kargo/v1"
	"github.com/akuity/terraform-provider-akp/akp/apis/v1alpha1"
)

var KargoAgentSizeString = map[kargov1.KargoAgentSize]string{
	kargov1.KargoAgentSize_KARGO_AGENT_SIZE_SMALL:       "small",
	kargov1.KargoAgentSize_KARGO_AGENT_SIZE_MEDIUM:      "medium",
	kargov1.KargoAgentSize_KARGO_AGENT_SIZE_LARGE:       "large",
	kargov1.KargoAgentSize_KARGO_AGENT_SIZE_UNSPECIFIED: "unspecified",
}

func (k *Kargo) Update(ctx context.Context, diagnostics *diag.Diagnostics, kargo *v1alpha1.Kargo) {
	var backendIpAllowListEnabled bool
	if kargo.Spec.KargoInstanceSpec.BackendIpAllowListEnabled != nil {
		backendIpAllowListEnabled = *kargo.Spec.KargoInstanceSpec.BackendIpAllowListEnabled
	}
	var acd *KargoAgentCustomization
	if kargo.Spec.KargoInstanceSpec.AgentCustomizationDefaults != nil {
		kacd := kargo.Spec.KargoInstanceSpec.AgentCustomizationDefaults
		var disabled bool
		if kacd.AutoUpgradeDisabled != nil {
			disabled = *kacd.AutoUpgradeDisabled
		}

		// If we have existing customization defaults, compare the normalized YAML
		if k.Spec.KargoInstanceSpec.AgentCustomizationDefaults != nil && len(kacd.Kustomization.Raw) > 0 {
			var newData, existingData map[string]interface{}
			existingYaml := []byte(k.Spec.KargoInstanceSpec.AgentCustomizationDefaults.Kustomization.ValueString())

			if err := yaml.Unmarshal(kacd.Kustomization.Raw, &newData); err == nil {
				if err := yaml.Unmarshal(existingYaml, &existingData); err == nil {
					newNormalized, err1 := yaml.Marshal(newData)
					existingNormalized, err2 := yaml.Marshal(existingData)
					if err1 == nil && err2 == nil && bytes.Equal(newNormalized, existingNormalized) {
						// If they're equal, use the existing customization
						acd = k.Spec.KargoInstanceSpec.AgentCustomizationDefaults
					}
				}
			}
		} else if k.Spec.KargoInstanceSpec.AgentCustomizationDefaults == nil && !disabled && len(kacd.Kustomization.Raw) == 0 {
			acd = nil
		} else {
			acd = toKargoAgentCustomizationTFModel(kargo.Spec.KargoInstanceSpec.AgentCustomizationDefaults, diagnostics)
		}
	}
	k.Spec = KargoSpec{
		Description: tftypes.StringValue(kargo.Spec.Description),
		Version:     tftypes.StringValue(kargo.Spec.Version),
		KargoInstanceSpec: KargoInstanceSpec{
			BackendIpAllowListEnabled:  tftypes.BoolValue(backendIpAllowListEnabled),
			IpAllowList:                toKargoIPAllowListTFModel(kargo.Spec.KargoInstanceSpec.IpAllowList),
			AgentCustomizationDefaults: acd,
			DefaultShardAgent:          tftypes.StringValue(kargo.Spec.KargoInstanceSpec.DefaultShardAgent),
			GlobalCredentialsNs:        toStringArrayTFModel(kargo.Spec.KargoInstanceSpec.GlobalCredentialsNs),
			GlobalServiceAccountNs:     toStringArrayTFModel(kargo.Spec.KargoInstanceSpec.GlobalServiceAccountNs),
		},
	}
}

func (k *Kargo) ToKargoAPIModel(ctx context.Context, diag *diag.Diagnostics, name string) *v1alpha1.Kargo {
	return &v1alpha1.Kargo{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Kargo",
			APIVersion: "kargo.akuity.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1alpha1.KargoSpec{
			Description: k.Spec.Description.ValueString(),
			Version:     k.Spec.Version.ValueString(),
			KargoInstanceSpec: v1alpha1.KargoInstanceSpec{
				BackendIpAllowListEnabled:  k.Spec.KargoInstanceSpec.BackendIpAllowListEnabled.ValueBoolPointer(),
				IpAllowList:                toKargoIpAllowListAPIModel(k.Spec.KargoInstanceSpec.IpAllowList),
				AgentCustomizationDefaults: toKargoAgentCustomizationAPIModel(k.Spec.KargoInstanceSpec.AgentCustomizationDefaults, diag),
				DefaultShardAgent:          k.Spec.KargoInstanceSpec.DefaultShardAgent.ValueString(),
				GlobalCredentialsNs:        toStringArrayAPIModel(k.Spec.KargoInstanceSpec.GlobalCredentialsNs),
				GlobalServiceAccountNs:     toStringArrayAPIModel(k.Spec.KargoInstanceSpec.GlobalServiceAccountNs),
			},
		},
	}
}

func toKargoIpAllowListAPIModel(ipAllowList []*KargoIPAllowListEntry) []*v1alpha1.KargoIPAllowListEntry {
	ipAllowListAPIModel := make([]*v1alpha1.KargoIPAllowListEntry, len(ipAllowList))
	for i, ipAllowListEntry := range ipAllowList {
		ipAllowListAPIModel[i] = &v1alpha1.KargoIPAllowListEntry{
			Ip:          ipAllowListEntry.Ip.ValueString(),
			Description: ipAllowListEntry.Description.ValueString(),
		}
	}
	return ipAllowListAPIModel
}

func toStringArrayAPIModel(strings []types.String) []string {
	ss := make([]string, len(strings))
	for i, s := range strings {
		ss[i] = s.ValueString()
	}
	return ss
}

func toKargoAgentCustomizationAPIModel(agentCustomizationDefaults *KargoAgentCustomization, diags *diag.Diagnostics) *v1alpha1.KargoAgentCustomization {
	if agentCustomizationDefaults == nil {
		return nil
	}
	var raw runtime.RawExtension
	if !agentCustomizationDefaults.Kustomization.IsNull() {
		if err := yaml.Unmarshal([]byte(agentCustomizationDefaults.Kustomization.ValueString()), &raw); err != nil {
			diags.AddError("failed unmarshal kustomization string to yaml", err.Error())
		}
	}
	return &v1alpha1.KargoAgentCustomization{
		AutoUpgradeDisabled: agentCustomizationDefaults.AutoUpgradeDisabled.ValueBoolPointer(),
		Kustomization:       raw,
	}
}

func toKargoIPAllowListTFModel(ipAllowList []*v1alpha1.KargoIPAllowListEntry) []*KargoIPAllowListEntry {
	if ipAllowList == nil {
		return nil
	}
	ipAllowListTF := make([]*KargoIPAllowListEntry, len(ipAllowList))
	for i, ipAllowListEntry := range ipAllowList {
		ipAllowListTF[i] = &KargoIPAllowListEntry{
			Ip:          tftypes.StringValue(ipAllowListEntry.Ip),
			Description: tftypes.StringValue(ipAllowListEntry.Description),
		}
	}
	return ipAllowListTF
}

func toKargoAgentCustomizationTFModel(agentCustomizationDefaults *v1alpha1.KargoAgentCustomization, diags *diag.Diagnostics) *KargoAgentCustomization {
	if agentCustomizationDefaults == nil {
		return nil
	}
	var autoUpgradeDisabled bool
	if agentCustomizationDefaults.AutoUpgradeDisabled != nil {
		autoUpgradeDisabled = *agentCustomizationDefaults.AutoUpgradeDisabled
	}
	var kustomization types.String
	if len(agentCustomizationDefaults.Kustomization.Raw) == 0 {
		kustomization = tftypes.StringNull()
	} else {
		yamlData, err := yaml.JSONToYAML(agentCustomizationDefaults.Kustomization.Raw)
		if err != nil {
			diags.AddError("failed to convert json to yaml", err.Error())
		}
		kustomization = tftypes.StringValue(string(yamlData))
	}
	return &KargoAgentCustomization{
		AutoUpgradeDisabled: tftypes.BoolValue(autoUpgradeDisabled),
		Kustomization:       kustomization,
	}
}

func toStringArrayTFModel(strings []string) []types.String {
	if len(strings) == 0 {
		return nil
	}
	nss := make([]types.String, len(strings))
	for i, s := range strings {
		nss[i] = types.StringValue(s)
	}
	return nss
}

func (ka *KargoAgent) Update(ctx context.Context, diagnostics *diag.Diagnostics, apiKargoAgent *kargov1.KargoAgent, plan *KargoAgent) {
	ka.ID = tftypes.StringValue(apiKargoAgent.GetId())
	ka.Name = tftypes.StringValue(apiKargoAgent.GetName())
	ka.Namespace = tftypes.StringValue(apiKargoAgent.GetData().GetNamespace())
	if ka.RemoveAgentResourcesOnDestroy.IsUnknown() || ka.RemoveAgentResourcesOnDestroy.IsNull() {
		ka.RemoveAgentResourcesOnDestroy = tftypes.BoolValue(true)
	}
	labels, d := tftypes.MapValueFrom(ctx, tftypes.StringType, apiKargoAgent.GetData().GetLabels())
	if d.HasError() {
		labels = tftypes.MapNull(tftypes.StringType)
	}
	diagnostics.Append(d...)
	annotations, d := tftypes.MapValueFrom(ctx, tftypes.StringType, apiKargoAgent.GetData().GetAnnotations())
	if d.HasError() {
		annotations = tftypes.MapNull(tftypes.StringType)
	}
	diagnostics.Append(d...)
	jsonData, err := apiKargoAgent.GetData().GetKustomization().MarshalJSON()
	if err != nil {
		diagnostics.AddError("getting kargo agent kustomization", fmt.Sprintf("%s", err.Error()))
	}
	yamlData, err := yaml.JSONToYAML(jsonData)
	if err != nil {
		diagnostics.AddError("getting kargo agent kustomization", fmt.Sprintf("%s", err.Error()))
	}

	kustomization := tftypes.StringValue(string(yamlData))
	if ka.Spec != nil {
		rawPlan := runtime.RawExtension{}
		old := ka.Spec.Data.Kustomization
		if err := yaml.Unmarshal([]byte(old.ValueString()), &rawPlan); err != nil {
			diagnostics.AddError("failed unmarshal kustomization string to yaml", err.Error())
		}

		oldYamlData, err := yaml.Marshal(&rawPlan)
		if err != nil {
			diagnostics.AddError("failed to convert json to yaml data", err.Error())
		}
		if bytes.Equal(oldYamlData, yamlData) {
			kustomization = old
		}
	}

	ka.Labels = labels
	ka.Annotations = annotations

	size := tftypes.StringValue(KargoAgentSizeString[apiKargoAgent.GetData().GetSize()])
	ka.Spec = &KargoAgentSpec{
		Description: tftypes.StringValue(apiKargoAgent.GetDescription()),
		Data: KargoAgentData{
			Size:                size,
			AutoUpgradeDisabled: tftypes.BoolValue(apiKargoAgent.GetData().GetAutoUpgradeDisabled()),
			TargetVersion:       tftypes.StringValue(apiKargoAgent.GetData().GetTargetVersion()),
			Kustomization:       kustomization,
			RemoteArgocd:        tftypes.StringValue(apiKargoAgent.GetData().GetRemoteArgocd()),
			AkuityManaged:       tftypes.BoolValue(apiKargoAgent.GetData().GetAkuityManaged()),
			ArgocdNamespace:     tftypes.StringValue(apiKargoAgent.GetData().GetArgocdNamespace()),
		},
	}
}

func (ka *KargoAgent) ToKargoAgentAPIModel(ctx context.Context, diagnostics *diag.Diagnostics) *v1alpha1.KargoAgent {
	var labels map[string]string
	var annotations map[string]string
	diagnostics.Append(ka.Labels.ElementsAs(ctx, &labels, true)...)
	diagnostics.Append(ka.Annotations.ElementsAs(ctx, &annotations, true)...)
	return &v1alpha1.KargoAgent{
		TypeMeta: metav1.TypeMeta{
			Kind:       "KargoAgent",
			APIVersion: "kargo.akuity.io/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        ka.Name.ValueString(),
			Namespace:   ka.Namespace.ValueString(),
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: v1alpha1.KargoAgentSpec{
			Description: ka.Spec.Description.ValueString(),
			Data:        toKargoAgentDataAPIModel(ctx, diagnostics, ka.Spec.Data),
		},
	}
}

func toKargoAgentDataAPIModel(ctx context.Context, diagnostics *diag.Diagnostics, data KargoAgentData) v1alpha1.KargoAgentData {
	var existingConfig map[string]any
	raw := runtime.RawExtension{}
	if data.Kustomization.ValueString() != "" {
		if err := yaml.Unmarshal([]byte(data.Kustomization.ValueString()), &raw); err != nil {
			diagnostics.AddError("failed unmarshal kustomization string to yaml", err.Error())
			return v1alpha1.KargoAgentData{}
		}
		if err := yaml.Unmarshal(raw.Raw, &existingConfig); err != nil {
			diagnostics.AddError("failed to parse existing kustomization", err.Error())
			return v1alpha1.KargoAgentData{}
		}
	}
	yamlData, err := yaml.Marshal(existingConfig)
	if err != nil {
		diagnostics.AddError("failed to convert json to yaml data", err.Error())
		return v1alpha1.KargoAgentData{}
	}
	if err = yaml.Unmarshal(yamlData, &raw); err != nil {
		diagnostics.AddError("failed to convert yaml to json data", err.Error())
		return v1alpha1.KargoAgentData{}
	}

	return v1alpha1.KargoAgentData{
		Size:                v1alpha1.KargoAgentSize(data.Size.ValueString()),
		AutoUpgradeDisabled: data.AutoUpgradeDisabled.ValueBoolPointer(),
		TargetVersion:       data.TargetVersion.ValueString(),
		Kustomization:       raw,
		RemoteArgocd:        data.RemoteArgocd.ValueString(),
		AkuityManaged:       data.AkuityManaged.ValueBoolPointer(),
		ArgocdNamespace:     data.ArgocdNamespace.ValueString(),
	}
}
