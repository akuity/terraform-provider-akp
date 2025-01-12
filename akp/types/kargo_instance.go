package types

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	tftypes "github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"

	kargov1 "github.com/akuity/api-client-go/pkg/api/gen/kargo/v1"
	"github.com/akuity/terraform-provider-akp/akp/apis/v1alpha1"
	"github.com/akuity/terraform-provider-akp/akp/marshal"
)

type KargoInstance struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Kargo       *Kargo       `tfsdk:"kargo"`
	WorkspaceId types.String `tfsdk:"workspace_id"`
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

func (k *Kargo) Update(ctx context.Context, diagnostics *diag.Diagnostics, kargo *v1alpha1.Kargo) {
	var backendIpAllowListEnabled bool
	if kargo.Spec.KargoInstanceSpec.BackendIpAllowListEnabled != nil {
		backendIpAllowListEnabled = *kargo.Spec.KargoInstanceSpec.BackendIpAllowListEnabled
	}
	k.Spec = KargoSpec{
		Description: tftypes.StringValue(kargo.Spec.Description),
		Version:     tftypes.StringValue(kargo.Spec.Version),
		KargoInstanceSpec: KargoInstanceSpec{
			BackendIpAllowListEnabled:  tftypes.BoolValue(backendIpAllowListEnabled),
			IpAllowList:                toKargoIPAllowListTFModel(kargo.Spec.KargoInstanceSpec.IpAllowList),
			AgentCustomizationDefaults: toKargoAgentCustomizationTFModel(kargo.Spec.KargoInstanceSpec.AgentCustomizationDefaults, diagnostics),
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
	raw := runtime.RawExtension{}
	if err := yaml.Unmarshal([]byte(agentCustomizationDefaults.Kustomization.ValueString()), &raw); err != nil {
		diags.AddError("failed unmarshal kustomization string to yaml", err.Error())
	}
	return &v1alpha1.KargoAgentCustomization{
		AutoUpgradeDisabled: agentCustomizationDefaults.AutoUpgradeDisabled.ValueBoolPointer(),
		Kustomization:       raw,
	}
}

func toKargoIPAllowListTFModel(ipAllowList []*v1alpha1.KargoIPAllowListEntry) []*KargoIPAllowListEntry {
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
	yamlData, err := yaml.JSONToYAML(agentCustomizationDefaults.Kustomization.Raw)
	if err != nil {
		diags.AddError("failed to convert json to yaml", err.Error())
	}
	return &KargoAgentCustomization{
		AutoUpgradeDisabled: tftypes.BoolValue(autoUpgradeDisabled),
		Kustomization:       tftypes.StringValue(string(yamlData)),
	}
}

func toStringArrayTFModel(strings []string) []types.String {
	nss := make([]types.String, len(strings))
	for i, s := range strings {
		nss[i] = types.StringValue(s)
	}
	return nss
}
