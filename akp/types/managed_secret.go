package types

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
)

type ManagedSecret struct {
	Labels          types.Map    `tfsdk:"labels"`
	AllowedClusters types.List   `tfsdk:"allowed_clusters"`
	ClusterSelector types.String `tfsdk:"cluster_selector"`
	Data            types.Map    `tfsdk:"data"`
	DataVersion     types.String `tfsdk:"data_version"`
}

type ManagedSecretUpsert struct {
	Secret    *argocdv1.ManagedSecret
	Data      map[string]string
	ClearData bool
}

func ToManagedSecretUpsertAPIModel(ctx context.Context, diagnostics *diag.Diagnostics, name string, secret *ManagedSecret) *ManagedSecretUpsert {
	var labels map[string]string
	if !secret.Labels.IsNull() {
		diagnostics.Append(secret.Labels.ElementsAs(ctx, &labels, true)...)
	}

	var allowedClusters []string
	if !secret.AllowedClusters.IsNull() {
		diagnostics.Append(secret.AllowedClusters.ElementsAs(ctx, &allowedClusters, true)...)
	}

	var clusterSelector *argocdv1.ObjectSelector
	if selector := secret.ClusterSelector.ValueString(); selector != "" {
		var err error
		clusterSelector, err = objectSelectorFromString(selector)
		if err != nil {
			diagnostics.AddError("Client Error", "Unable to parse managed secret cluster selector. "+err.Error())
			return nil
		}
	}

	var data map[string]string
	clearData := false
	if !secret.Data.IsNull() {
		diagnostics.Append(secret.Data.ElementsAs(ctx, &data, true)...)
		clearData = len(data) == 0
	}

	return &ManagedSecretUpsert{
		Secret: &argocdv1.ManagedSecret{
			Name:            name,
			Labels:          labels,
			AllowedClusters: allowedClusters,
			ClusterSelector: clusterSelector,
		},
		Data:      data,
		ClearData: clearData,
	}
}

func objectSelectorFromString(selector string) (*argocdv1.ObjectSelector, error) {
	parsed, err := metav1.ParseToLabelSelector(selector)
	if err != nil {
		return nil, err
	}
	objectSelector := &argocdv1.ObjectSelector{
		MatchLabels:      parsed.MatchLabels,
		MatchExpressions: make([]*argocdv1.LabelSelectorRequirement, 0, len(parsed.MatchExpressions)),
	}
	for i := range parsed.MatchExpressions {
		expression := parsed.MatchExpressions[i]
		operator := string(expression.Operator)
		objectSelector.MatchExpressions = append(objectSelector.MatchExpressions, &argocdv1.LabelSelectorRequirement{
			Key:      &expression.Key,
			Operator: &operator,
			Values:   expression.Values,
		})
	}
	return objectSelector, nil
}

func ToManagedSecretsTFModel(ctx context.Context, diagnostics *diag.Diagnostics, prior map[string]*ManagedSecret, apiSecrets []*argocdv1.ManagedSecret) map[string]*ManagedSecret {
	if prior == nil {
		return nil
	}
	actual := make(map[string]*argocdv1.ManagedSecret, len(apiSecrets))
	for _, secret := range apiSecrets {
		if secret != nil {
			actual[secret.GetName()] = secret
		}
	}
	result := make(map[string]*ManagedSecret, len(prior))
	for name, old := range prior {
		secret, ok := actual[name]
		if !ok || old == nil {
			continue
		}
		labels := types.MapNull(types.StringType)
		if len(secret.GetLabels()) > 0 || !old.Labels.IsNull() {
			apiLabels := secret.GetLabels()
			if apiLabels == nil {
				apiLabels = map[string]string{}
			}
			var d diag.Diagnostics
			labels, d = types.MapValueFrom(ctx, types.StringType, apiLabels)
			diagnostics.Append(d...)
		}
		allowedClusters := types.ListNull(types.StringType)
		if len(secret.GetAllowedClusters()) > 0 || !old.AllowedClusters.IsNull() {
			apiAllowedClusters := secret.GetAllowedClusters()
			if apiAllowedClusters == nil {
				apiAllowedClusters = []string{}
			}
			var d diag.Diagnostics
			allowedClusters, d = types.ListValueFrom(ctx, types.StringType, apiAllowedClusters)
			diagnostics.Append(d...)
		}
		clusterSelector := types.StringNull()
		if secret.GetClusterSelector() != nil {
			selector, err := managedSecretSelectorString(secret.GetClusterSelector())
			if err != nil {
				diagnostics.AddError("Client Error", "Unable to convert managed secret cluster selector. "+err.Error())
				continue
			}
			clusterSelector = types.StringValue(selector)
		} else if !old.ClusterSelector.IsNull() {
			clusterSelector = types.StringValue("")
		}
		result[name] = &ManagedSecret{
			Labels:          labels,
			AllowedClusters: allowedClusters,
			ClusterSelector: clusterSelector,
			Data:            types.MapNull(types.StringType),
			DataVersion:     old.DataVersion,
		}
	}
	return result
}

func ToManagedSecretsDataSourceModel(ctx context.Context, diagnostics *diag.Diagnostics, apiSecrets []*argocdv1.ManagedSecret) map[string]*ManagedSecretDataSource {
	result := make(map[string]*ManagedSecretDataSource, len(apiSecrets))
	for _, secret := range apiSecrets {
		if secret == nil {
			continue
		}
		apiLabels := secret.GetLabels()
		if apiLabels == nil {
			apiLabels = map[string]string{}
		}
		labels, d := types.MapValueFrom(ctx, types.StringType, apiLabels)
		diagnostics.Append(d...)
		allowedClusters := secret.GetAllowedClusters()
		if allowedClusters == nil {
			allowedClusters = []string{}
		}
		allowed, d := types.ListValueFrom(ctx, types.StringType, allowedClusters)
		diagnostics.Append(d...)
		secretKeys := secret.GetSecretKeys()
		if secretKeys == nil {
			secretKeys = []string{}
		}
		keys, d := types.ListValueFrom(ctx, types.StringType, secretKeys)
		diagnostics.Append(d...)
		clusterSelector := types.StringValue("")
		if secret.GetClusterSelector() != nil {
			selector, err := managedSecretSelectorString(secret.GetClusterSelector())
			if err != nil {
				diagnostics.AddError("Client Error", "Unable to convert managed secret cluster selector. "+err.Error())
				continue
			}
			clusterSelector = types.StringValue(selector)
		}
		result[secret.GetName()] = &ManagedSecretDataSource{
			Labels:          labels,
			AllowedClusters: allowed,
			ClusterSelector: clusterSelector,
			SecretKeys:      keys,
		}
	}
	return result
}

func managedSecretSelectorString(selector *argocdv1.ObjectSelector) (string, error) {
	requirements := make([]metav1.LabelSelectorRequirement, 0, len(selector.GetMatchExpressions()))
	for _, expression := range selector.GetMatchExpressions() {
		requirements = append(requirements, metav1.LabelSelectorRequirement{
			Key:      expression.GetKey(),
			Operator: metav1.LabelSelectorOperator(expression.GetOperator()),
			Values:   expression.GetValues(),
		})
	}
	parsed, err := metav1.LabelSelectorAsSelector(&metav1.LabelSelector{
		MatchLabels:      selector.GetMatchLabels(),
		MatchExpressions: requirements,
	})
	if err != nil {
		return "", err
	}
	return parsed.String(), nil
}
