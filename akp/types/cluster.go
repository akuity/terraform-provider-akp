package types

import (
	"context"

	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/diag"
)

type ProtoCluster struct {
	*argocdv1.Cluster
}

var (
	StringClusterSize = map[string]argocdv1.ClusterSize{
		"small": argocdv1.ClusterSize_CLUSTER_SIZE_SMALL,
		"medium": argocdv1.ClusterSize_CLUSTER_SIZE_MEDIUM,
		"large": argocdv1.ClusterSize_CLUSTER_SIZE_LARGE,
		"unspecified": argocdv1.ClusterSize_CLUSTER_SIZE_UNSPECIFIED,
	}
	ClusterSizeString = map[argocdv1.ClusterSize]string{
		argocdv1.ClusterSize_CLUSTER_SIZE_SMALL: "small",
		argocdv1.ClusterSize_CLUSTER_SIZE_MEDIUM: "medium",
		argocdv1.ClusterSize_CLUSTER_SIZE_LARGE: "large",
		argocdv1.ClusterSize_CLUSTER_SIZE_UNSPECIFIED: "unspecified",
	}
)

type AkpCluster struct {
	Id                          types.String `tfsdk:"id"`
	InstanceId                  types.String `tfsdk:"instance_id"`
	Name                        types.String `tfsdk:"name"`
	Description                 types.String `tfsdk:"description"`
	Namespace                   types.String `tfsdk:"namespace"`
	NamespaceScoped             types.Bool   `tfsdk:"namespace_scoped"`
	Size                        types.String `tfsdk:"size"`
	AutoUpgradeDisabled         types.Bool   `tfsdk:"auto_upgrade_disabled"`
	CustomImageRegistryArgoproj types.String `tfsdk:"custom_image_registry_argoproj"`
	CustomImageRegistryAkuity   types.String `tfsdk:"custom_image_registry_akuity"`
	Manifests                   types.String `tfsdk:"manifests"`
	Labels                      types.Map    `tfsdk:"labels"`
	Annotations                 types.Map    `tfsdk:"annotations"`
}

func (x *ProtoCluster) FromProto(instanceId string) (*AkpCluster, diag.Diagnostics) {
	diags := diag.Diagnostics{}
	labels, diag := types.MapValueFrom(context.Background(), types.StringType, x.Data.Labels)
	if diag.HasError() {
		labels = types.MapNull(types.StringType)
		diags = append(diags, diag.Errors()...)
	}
	annotations, diag := types.MapValueFrom(context.Background(), types.StringType, x.Data.Annotations)
	if diag.HasError() {
		annotations = types.MapNull(types.StringType)
		diags = append(diags, diag.Errors()...)
	}
	res := &AkpCluster{
		Id:                          types.StringValue(x.Id),
		Name:                        types.StringValue(x.Name),
		Description:                 types.StringValue(x.Description),
		Namespace:                   types.StringValue(x.Namespace),
		NamespaceScoped:             types.BoolValue(x.NamespaceScoped),
		InstanceId:                  types.StringValue(instanceId),
		Manifests:                   types.StringNull(),
		Size:                        types.StringValue(ClusterSizeString[x.Data.Size]),
		AutoUpgradeDisabled:         types.BoolValue(*x.Data.AutoUpgradeDisabled),
		CustomImageRegistryArgoproj: types.StringValue(*x.Data.CustomImageRegistryArgoproj),
		CustomImageRegistryAkuity:   types.StringValue(*x.Data.CustomImageRegistryAkuity),
		Labels:                      labels,
		Annotations:                 annotations,
	}
	return res, diags
}

func (x *AkpCluster) UpdateFromProto(protoCluster *argocdv1.Cluster) diag.Diagnostics {
	if protoCluster.Name != "" {
		x.Name = types.StringValue(protoCluster.Name)
	}
	if protoCluster.Namespace != "" {
		x.Namespace = types.StringValue(protoCluster.Namespace)
	}
	diags := diag.Diagnostics{}
	labels, diag := types.MapValueFrom(context.Background(), types.StringType, protoCluster.Data.Labels)
	if diag.HasError() {
		labels = types.MapNull(types.StringType)
		diags = append(diags, diag.Errors()...)
	}
	annotations, diag := types.MapValueFrom(context.Background(), types.StringType, protoCluster.Data.Annotations)
	if diag.HasError() {
		annotations = types.MapNull(types.StringType)
		diags = append(diags, diag.Errors()...)
	}
	x.Description = types.StringValue(protoCluster.GetDescription())
	x.NamespaceScoped = types.BoolValue(protoCluster.GetNamespaceScoped())
	x.Size = types.StringValue(ClusterSizeString[protoCluster.Data.Size])
	x.AutoUpgradeDisabled = types.BoolValue(*protoCluster.Data.AutoUpgradeDisabled)
	x.CustomImageRegistryArgoproj = types.StringValue(*protoCluster.Data.CustomImageRegistryArgoproj)
	x.CustomImageRegistryAkuity = types.StringValue(*protoCluster.Data.CustomImageRegistryAkuity)
	x.Annotations = annotations
	x.Labels = labels
	return diags
}
