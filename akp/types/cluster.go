package types

import (
	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type ProtoCluster struct {
	*argocdv1.Cluster
}

type AkpCluster struct {
	Id              types.String `tfsdk:"id"`
	InstanceId      types.String `tfsdk:"instance_id"`
	Name            types.String `tfsdk:"name"`
	Description     types.String `tfsdk:"description"`
	Namespace       types.String `tfsdk:"namespace"`
	NamespaceScoped types.Bool   `tfsdk:"namespace_scoped"`
	Manifests       types.String `tfsdk:"manifests"`
}

func (x *ProtoCluster) FromProto(instanceId string) *AkpCluster {
	return &AkpCluster{
		Id:              types.StringValue(x.Id),
		Name:            types.StringValue(x.Name),
		Description:     types.StringValue(x.Description),
		Namespace:       types.StringValue(x.Namespace),
		NamespaceScoped: types.BoolValue(x.NamespaceScoped),
		InstanceId:      types.StringValue(instanceId),
		Manifests:       types.StringNull(),
	}
}

func (x *AkpCluster) UpdateFromProto(protoCluster *argocdv1.Cluster) *AkpCluster {
	if protoCluster.Name != "" {
		x.Name = types.StringValue(protoCluster.Name)
	}
	if protoCluster.Namespace != "" {
		x.Namespace = types.StringValue(protoCluster.Namespace)
	}
	x.Description = types.StringValue(protoCluster.GetDescription())
	x.NamespaceScoped = types.BoolValue(protoCluster.GetNamespaceScoped())
	return x
}
