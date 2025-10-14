package types

import (
	"bytes"
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
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

type AgentMaps struct {
	NameToID map[string]string
	IDToName map[string]string
}

func (k *Kargo) Update(ctx context.Context, diagnostics *diag.Diagnostics, kargo *v1alpha1.Kargo, agentMaps *AgentMaps) {
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

	defaultShardAgent := types.StringValue(kargo.Spec.KargoInstanceSpec.DefaultShardAgent)
	if agentMaps != nil && !k.Spec.KargoInstanceSpec.DefaultShardAgent.IsNull() && !k.Spec.KargoInstanceSpec.DefaultShardAgent.IsUnknown() {
		userInput := k.Spec.KargoInstanceSpec.DefaultShardAgent.ValueString()
		apiValue := kargo.Spec.KargoInstanceSpec.DefaultShardAgent
		if agentMaps.IDToName[apiValue] == userInput || apiValue == userInput {
			defaultShardAgent = k.Spec.KargoInstanceSpec.DefaultShardAgent
		}
	}

	k.Spec = KargoSpec{
		Description: types.StringValue(kargo.Spec.Description),
		Version:     types.StringValue(kargo.Spec.Version),
		KargoInstanceSpec: KargoInstanceSpec{
			BackendIpAllowListEnabled:  types.BoolValue(backendIpAllowListEnabled),
			IpAllowList:                toKargoIPAllowListTFModel(kargo.Spec.KargoInstanceSpec.IpAllowList),
			AgentCustomizationDefaults: acd,
			DefaultShardAgent:          defaultShardAgent,
			GlobalCredentialsNs:        toStringArrayTFModel(kargo.Spec.KargoInstanceSpec.GlobalCredentialsNs),
			GlobalServiceAccountNs:     toStringArrayTFModel(kargo.Spec.KargoInstanceSpec.GlobalServiceAccountNs),
			AkuityIntelligence:         toKargoAkuityIntelligenceTFModel(kargo.Spec.KargoInstanceSpec.AkuityIntelligence, k),
		},
		Fqdn:       types.StringValue(kargo.Spec.Fqdn),
		Subdomain:  types.StringValue(kargo.Spec.Subdomain),
		OidcConfig: k.toKargoOidcConfigTFModel(ctx, kargo.Spec.OidcConfig),
	}
	if kargo.Spec.KargoInstanceSpec.GcConfig != nil {
		k.Spec.KargoInstanceSpec.GcConfig = &GarbageCollectorConfig{
			MaxRetainedFreight:      types.Int64Value(int64(kargo.Spec.KargoInstanceSpec.GcConfig.MaxRetainedFreight)),
			MaxRetainedPromotions:   types.Int64Value(int64(kargo.Spec.KargoInstanceSpec.GcConfig.MaxRetainedPromotions)),
			MinFreightDeletionAge:   types.Int64Value(int64(kargo.Spec.KargoInstanceSpec.GcConfig.MinFreightDeletionAge)),
			MinPromotionDeletionAge: types.Int64Value(int64(kargo.Spec.KargoInstanceSpec.GcConfig.MinPromotionDeletionAge)),
		}
	}
}

func (k *Kargo) ToKargoAPIModel(ctx context.Context, diag *diag.Diagnostics, name string, agentMaps *AgentMaps) *v1alpha1.Kargo {
	subdomain := k.Spec.Subdomain.ValueString()
	fqdn := k.Spec.Fqdn.ValueString()
	if subdomain != "" && fqdn != "" {
		diag.AddError("subdomain and fqdn cannot be set at the same time", "subdomain and fqdn are mutually exclusive")
		return nil
	}
	var gcConfig *v1alpha1.GarbageCollectorConfig
	if k.Spec.KargoInstanceSpec.GcConfig != nil {
		gcConfig = &v1alpha1.GarbageCollectorConfig{
			MaxRetainedFreight:      uint32(k.Spec.KargoInstanceSpec.GcConfig.MaxRetainedFreight.ValueInt64()),
			MaxRetainedPromotions:   uint32(k.Spec.KargoInstanceSpec.GcConfig.MaxRetainedPromotions.ValueInt64()),
			MinFreightDeletionAge:   uint32(k.Spec.KargoInstanceSpec.GcConfig.MinFreightDeletionAge.ValueInt64()),
			MinPromotionDeletionAge: uint32(k.Spec.KargoInstanceSpec.GcConfig.MinPromotionDeletionAge.ValueInt64()),
		}
	}

	defaultShardAgent := k.Spec.KargoInstanceSpec.DefaultShardAgent.ValueString()
	if agentMaps != nil && defaultShardAgent != "" {
		if agentID, ok := agentMaps.NameToID[defaultShardAgent]; ok {
			defaultShardAgent = agentID
		}
	}

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
				BackendIpAllowListEnabled:  toBoolPointer(k.Spec.KargoInstanceSpec.BackendIpAllowListEnabled),
				IpAllowList:                toKargoIpAllowListAPIModel(k.Spec.KargoInstanceSpec.IpAllowList),
				AgentCustomizationDefaults: toKargoAgentCustomizationAPIModel(k.Spec.KargoInstanceSpec.AgentCustomizationDefaults, diag),
				DefaultShardAgent:          defaultShardAgent,
				GlobalCredentialsNs:        toStringArrayAPIModel(k.Spec.KargoInstanceSpec.GlobalCredentialsNs),
				GlobalServiceAccountNs:     toStringArrayAPIModel(k.Spec.KargoInstanceSpec.GlobalServiceAccountNs),
				AkuityIntelligence:         toKargoAkuityIntelligenceAPIModel(k.Spec.KargoInstanceSpec.AkuityIntelligence),
				GcConfig:                   gcConfig,
			},
			Fqdn:       fqdn,
			Subdomain:  subdomain,
			OidcConfig: toKargoOidcConfigAPIModel(ctx, diag, k.Spec.OidcConfig),
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
		AutoUpgradeDisabled: toBoolPointer(agentCustomizationDefaults.AutoUpgradeDisabled),
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
			Ip:          types.StringValue(ipAllowListEntry.Ip),
			Description: types.StringValue(ipAllowListEntry.Description),
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
		kustomization = types.StringNull()
	} else {
		yamlData, err := yaml.JSONToYAML(agentCustomizationDefaults.Kustomization.Raw)
		if err != nil {
			diags.AddError("failed to convert json to yaml", err.Error())
		}
		kustomization = types.StringValue(string(yamlData))
	}
	return &KargoAgentCustomization{
		AutoUpgradeDisabled: types.BoolValue(autoUpgradeDisabled),
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
	ka.ID = types.StringValue(apiKargoAgent.GetId())
	ka.Name = types.StringValue(apiKargoAgent.GetName())
	ka.Namespace = types.StringValue(apiKargoAgent.GetData().GetNamespace())
	if ka.RemoveAgentResourcesOnDestroy.IsUnknown() || ka.RemoveAgentResourcesOnDestroy.IsNull() {
		ka.RemoveAgentResourcesOnDestroy = types.BoolValue(true)
	}
	if ka.ReapplyManifestsOnUpdate.IsUnknown() || ka.ReapplyManifestsOnUpdate.IsNull() {
		ka.ReapplyManifestsOnUpdate = types.BoolValue(false)
	} else if plan != nil {
		ka.ReapplyManifestsOnUpdate = plan.ReapplyManifestsOnUpdate
	}
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
	jsonData, err := apiKargoAgent.GetData().GetKustomization().MarshalJSON()
	if err != nil {
		diagnostics.AddError("getting kargo agent kustomization", err.Error())
	}
	yamlData, err := yaml.JSONToYAML(jsonData)
	if err != nil {
		diagnostics.AddError("getting kargo agent kustomization", err.Error())
	}

	kustomization := types.StringValue(string(yamlData))
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

	argocdNs := apiKargoAgent.GetData().GetArgocdNamespace()
	if !apiKargoAgent.GetData().GetAkuityManaged() && plan != nil && plan.Spec != nil {
		argocdNs = plan.Spec.Data.ArgocdNamespace.ValueString()
	}

	size := types.StringValue(KargoAgentSizeString[apiKargoAgent.GetData().GetSize()])
	ka.Spec = &KargoAgentSpec{
		Description: types.StringValue(apiKargoAgent.GetDescription()),
		Data: KargoAgentData{
			Size:                size,
			AutoUpgradeDisabled: types.BoolValue(apiKargoAgent.GetData().GetAutoUpgradeDisabled()),
			TargetVersion:       types.StringValue(apiKargoAgent.GetData().GetTargetVersion()),
			Kustomization:       kustomization,
			RemoteArgocd:        types.StringValue(apiKargoAgent.GetData().GetRemoteArgocd()),
			AkuityManaged:       types.BoolValue(apiKargoAgent.GetData().GetAkuityManaged()),
			ArgocdNamespace:     types.StringValue(argocdNs),
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

func toKargoAgentDataAPIModel(_ context.Context, diagnostics *diag.Diagnostics, data KargoAgentData) v1alpha1.KargoAgentData {
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
		AutoUpgradeDisabled: toBoolPointer(data.AutoUpgradeDisabled),
		TargetVersion:       data.TargetVersion.ValueString(),
		Kustomization:       raw,
		RemoteArgocd:        data.RemoteArgocd.ValueString(),
		AkuityManaged:       data.AkuityManaged.ValueBool(),
		ArgocdNamespace:     data.ArgocdNamespace.ValueString(),
	}
}

func toKargoOidcConfigAPIModel(ctx context.Context, diag *diag.Diagnostics, oidcConfig *KargoOidcConfig) *v1alpha1.KargoOidcConfig {
	if oidcConfig == nil {
		return nil
	}
	additionalScopes := []string{}
	for _, scope := range oidcConfig.AdditionalScopes {
		additionalScopes = append(additionalScopes, scope.ValueString())
	}
	return &v1alpha1.KargoOidcConfig{
		Enabled:               toBoolPointer(oidcConfig.Enabled),
		DexEnabled:            toBoolPointer(oidcConfig.DexEnabled),
		DexConfig:             oidcConfig.DexConfig.ValueString(),
		DexConfigSecret:       toKargoDexConfigSecretAPIModel(ctx, oidcConfig.DexConfigSecret),
		IssuerURL:             oidcConfig.IssuerURL.ValueString(),
		ClientID:              oidcConfig.ClientID.ValueString(),
		CliClientID:           oidcConfig.CliClientID.ValueString(),
		AdminAccount:          toKargoPredefinedAccountAPIModel(ctx, diag, oidcConfig.AdminAccount),
		ViewerAccount:         toKargoPredefinedAccountAPIModel(ctx, diag, oidcConfig.ViewerAccount),
		UserAccount:           toKargoPredefinedAccountAPIModel(ctx, diag, oidcConfig.UserAccount),
		ProjectCreatorAccount: toKargoPredefinedAccountAPIModel(ctx, diag, oidcConfig.ProjectCreatorAccount),
		AdditionalScopes:      additionalScopes,
	}
}

func toKargoDexConfigSecretAPIModel(ctx context.Context, secret types.Map) map[string]v1alpha1.Value {
	if secret.IsNull() {
		return nil
	}

	cfg := map[string]v1alpha1.Value{}
	data := map[string]string{}
	if err := secret.ElementsAs(ctx, &data, true); err != nil {
		return nil
	}
	for k, v := range data {
		cfg[k] = v1alpha1.Value{Value: &v}
	}
	return cfg
}

func toKargoPredefinedAccountAPIModel(_ context.Context, diag *diag.Diagnostics, accounts types.Object) v1alpha1.KargoPredefinedAccountData {
	result := v1alpha1.KargoPredefinedAccountData{
		Claims: make(map[string]v1alpha1.KargoPredefinedAccountClaimValue),
	}

	if accounts.IsNull() {
		return result
	}

	attrs := accounts.Attributes()
	claims, ok := attrs["claims"]
	if !ok {
		return result
	}

	claimsMap, ok := claims.(types.Map)
	if !ok {
		diag.AddError("invalid claims type", "claims must be a map")
		return result
	}

	elements := claimsMap.Elements()
	for key, value := range elements {
		claimObj, ok := value.(types.Object)
		if !ok {
			diag.AddError("invalid claim type", fmt.Sprintf("claim %s must be an object", key))
			continue
		}

		claimAttrs := claimObj.Attributes()
		valuesList, ok := claimAttrs["values"].(types.List)
		if !ok {
			continue
		}

		var stringValues []string
		for _, v := range valuesList.Elements() {
			stringValues = append(stringValues, v.(basetypes.StringValue).ValueString())
		}

		result.Claims[key] = v1alpha1.KargoPredefinedAccountClaimValue{
			Values: stringValues,
		}
	}

	return result
}

func (k *Kargo) toKargoOidcConfigTFModel(ctx context.Context, oidcConfig *v1alpha1.KargoOidcConfig) *KargoOidcConfig {
	if oidcConfig == nil || oidcConfig.Enabled == nil || !*oidcConfig.Enabled {
		return nil
	}

	additionalScopes := make([]types.String, len(oidcConfig.AdditionalScopes))
	for i, scope := range oidcConfig.AdditionalScopes {
		additionalScopes[i] = types.StringValue(scope)
	}
	if len(additionalScopes) == 0 {
		additionalScopes = nil
	}

	return &KargoOidcConfig{
		Enabled:               types.BoolPointerValue(oidcConfig.Enabled),
		DexEnabled:            types.BoolPointerValue(oidcConfig.DexEnabled),
		DexConfig:             types.StringValue(oidcConfig.DexConfig),
		DexConfigSecret:       toKargoDexConfigSecretTFModel(ctx, oidcConfig.DexConfigSecret),
		IssuerURL:             types.StringValue(oidcConfig.IssuerURL),
		ClientID:              types.StringValue(oidcConfig.ClientID),
		CliClientID:           types.StringValue(oidcConfig.CliClientID),
		AdminAccount:          k.toKargoPredefinedAccountTFModel(oidcConfig.AdminAccount, adminAccount),
		ViewerAccount:         k.toKargoPredefinedAccountTFModel(oidcConfig.ViewerAccount, viewerAccount),
		UserAccount:           k.toKargoPredefinedAccountTFModel(oidcConfig.UserAccount, viewerAccount),
		ProjectCreatorAccount: k.toKargoPredefinedAccountTFModel(oidcConfig.ProjectCreatorAccount, viewerAccount),
		AdditionalScopes:      additionalScopes,
	}
}

func toKargoDexConfigSecretTFModel(ctx context.Context, secret map[string]v1alpha1.Value) types.Map {
	if secret == nil {
		return types.MapNull(types.StringType)
	}

	secretData := make(map[string]string)
	for k, v := range secret {
		if v.Value == nil {
			continue
		}
		secretData[k] = *v.Value
	}
	mapVal, _ := types.MapValueFrom(ctx, types.StringType, secretData)
	return mapVal
}

type PredefinedAccountType int

const (
	adminAccount PredefinedAccountType = iota
	viewerAccount
)

func (k *Kargo) toKargoPredefinedAccountTFModel(account v1alpha1.KargoPredefinedAccountData, accountType PredefinedAccountType) types.Object {
	objectType := map[string]attr.Type{
		"claims": types.MapType{
			ElemType: types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"values": types.ListType{
						ElemType: types.StringType,
					},
				},
			},
		},
	}

	if len(account.Claims) == 0 {
		if accountType == adminAccount {
			return k.Spec.OidcConfig.AdminAccount
		} else {
			return k.Spec.OidcConfig.ViewerAccount
		}
	}

	claimsMap := make(map[string]attr.Value)
	for claimKey, claimValue := range account.Claims {
		valuesList, _ := types.ListValueFrom(context.Background(), types.StringType, claimValue.Values)

		claimObject := types.ObjectValueMust(
			map[string]attr.Type{
				"values": types.ListType{
					ElemType: types.StringType,
				},
			},
			map[string]attr.Value{
				"values": valuesList,
			},
		)

		claimsMap[claimKey] = claimObject
	}

	claimsAttr, _ := types.MapValueFrom(context.Background(),
		types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"values": types.ListType{
					ElemType: types.StringType,
				},
			},
		},
		claimsMap,
	)

	return types.ObjectValueMust(
		objectType,
		map[string]attr.Value{
			"claims": claimsAttr,
		},
	)
}

func toKargoAkuityIntelligenceTFModel(intelligence *v1alpha1.AkuityIntelligence, plan *Kargo) *AkuityIntelligence {
	if plan != nil {
		if plan.Spec.KargoInstanceSpec.AkuityIntelligence == nil {
			return nil
		}
	}

	if intelligence == nil {
		return nil
	}
	return &AkuityIntelligence{
		AiSupportEngineerEnabled: types.BoolValue(intelligence.AiSupportEngineerEnabled != nil && *intelligence.AiSupportEngineerEnabled),
		Enabled:                  types.BoolValue(intelligence.Enabled != nil && *intelligence.Enabled),
		AllowedUsernames:         convertSlice(intelligence.AllowedUsernames, func(s string) types.String { return types.StringValue(s) }),
		AllowedGroups:            convertSlice(intelligence.AllowedGroups, func(s string) types.String { return types.StringValue(s) }),
		ModelVersion:             types.StringValue(intelligence.ModelVersion),
	}
}

func toKargoAkuityIntelligenceAPIModel(intelligence *AkuityIntelligence) *v1alpha1.AkuityIntelligence {
	if intelligence == nil {
		return nil
	}
	tflog.Warn(context.Background(), fmt.Sprintf("hanxiaop model %+v", intelligence))
	return &v1alpha1.AkuityIntelligence{
		AiSupportEngineerEnabled: toBoolPointer(intelligence.AiSupportEngineerEnabled),
		Enabled:                  toBoolPointer(intelligence.Enabled),
		AllowedUsernames:         convertSlice(intelligence.AllowedUsernames, tfStringToString),
		AllowedGroups:            convertSlice(intelligence.AllowedGroups, tfStringToString),
		ModelVersion:             intelligence.ModelVersion.ValueString(),
	}
}
