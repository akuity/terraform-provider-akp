package types

import (
	"context"
	"fmt"

	argocdv1 "github.com/akuity/api-client-go/pkg/api/gen/argocd/v1"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	StringClusterSize = map[string]argocdv1.ClusterSize{
		"small":       argocdv1.ClusterSize_CLUSTER_SIZE_SMALL,
		"medium":      argocdv1.ClusterSize_CLUSTER_SIZE_MEDIUM,
		"large":       argocdv1.ClusterSize_CLUSTER_SIZE_LARGE,
		"unspecified": argocdv1.ClusterSize_CLUSTER_SIZE_UNSPECIFIED,
	}
	ClusterSizeString = map[argocdv1.ClusterSize]string{
		argocdv1.ClusterSize_CLUSTER_SIZE_SMALL:       "small",
		argocdv1.ClusterSize_CLUSTER_SIZE_MEDIUM:      "medium",
		argocdv1.ClusterSize_CLUSTER_SIZE_LARGE:       "large",
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
	Manifests                   types.String `tfsdk:"manifests"`
	Labels                      types.Map    `tfsdk:"labels"`
	Annotations                 types.Map    `tfsdk:"annotations"`
	AgentVersion                types.String `tfsdk:"agent_version"`
}

type AkpClusterKube struct {
	Id                          types.String `tfsdk:"id"`
	InstanceId                  types.String `tfsdk:"instance_id"`
	Name                        types.String `tfsdk:"name"`
	Description                 types.String `tfsdk:"description"`
	Namespace                   types.String `tfsdk:"namespace"`
	NamespaceScoped             types.Bool   `tfsdk:"namespace_scoped"`
	Size                        types.String `tfsdk:"size"`
	AutoUpgradeDisabled         types.Bool   `tfsdk:"auto_upgrade_disabled"`
	Manifests                   types.String `tfsdk:"manifests"`
	Labels                      types.Map    `tfsdk:"labels"`
	Annotations                 types.Map    `tfsdk:"annotations"`
	AgentVersion                types.String `tfsdk:"agent_version"`
	KubeConfig                  types.Object `tfsdk:"kube_config"`
}

func (x *AkpClusterKube) Update(p *AkpCluster) error {
	x.Id = p.Id
	x.Name = p.Name
	x.Description = p.Description
	x.Namespace = p.Namespace
	x.NamespaceScoped = p.NamespaceScoped
	x.Size = p.Size
	x.AutoUpgradeDisabled = p.AutoUpgradeDisabled
	// x.Manifests = p.Manifests
	x.Labels = p.Labels
	x.Annotations = p.Annotations
	x.AgentVersion = p.AgentVersion
	return nil
}

func (x *AkpCluster) UpdateCluster(p *argocdv1.Cluster) diag.Diagnostics {
	diags := diag.Diagnostics{}
	labels, diag := types.MapValueFrom(context.Background(), types.StringType, p.Data.Labels)
	if diag.HasError() {
		labels = types.MapNull(types.StringType)
		diags = append(diags, diag.Errors()...)
	}
	x.Labels = labels
	annotations, diag := types.MapValueFrom(context.Background(), types.StringType, p.Data.Annotations)
	if diag.HasError() {
		annotations = types.MapNull(types.StringType)
		diags = append(diags, diag.Errors()...)
	}
	x.Annotations = annotations
	if p.AgentState != nil {
		x.AgentVersion = types.StringValue(p.AgentState.Version)
	} else {
		x.AgentVersion = types.StringNull()
	}
	x.Id = types.StringValue(p.Id)
	x.Name = types.StringValue(p.GetName())
	x.Description = types.StringValue(p.GetDescription())
	x.Namespace = types.StringValue(p.GetNamespace())
	x.NamespaceScoped = types.BoolValue(p.GetNamespaceScoped())
	x.Size = types.StringValue(ClusterSizeString[p.Data.Size])
	x.AutoUpgradeDisabled = types.BoolValue(*p.Data.AutoUpgradeDisabled)
	return diags
}

func (x *AkpClusterKube) UpdateCluster(p *argocdv1.Cluster) diag.Diagnostics {
	akpCluster := &AkpCluster{}
	diag := akpCluster.UpdateCluster(p)
	x.Update(akpCluster)
	return diag
}

func (x *AkpCluster) UpdateManifests(ctx context.Context, client argocdv1.ArgoCDServiceGatewayClient, orgId string) diag.Diagnostics {
	diags := diag.Diagnostics{}
	apiReq := &argocdv1.GetInstanceClusterManifestsRequest{
		OrganizationId: orgId,
		InstanceId:     x.InstanceId.ValueString(),
		Id:             x.Id.ValueString(),
	}
	tflog.Debug(ctx, fmt.Sprintf("apiReq: %s", apiReq))
	apiResp, err := client.GetInstanceClusterManifests(ctx, apiReq)
	if err != nil {
		diags.AddError("Akuity API error", fmt.Sprintf("Unable to download manifests: %s", err))
		return diags
	}
	tflog.Debug(ctx, fmt.Sprintf("apiResp: %s", apiResp))
	x.Manifests = types.StringValue(string(apiResp.GetData()))
	return diags
}

func (x *AkpClusterKube) UpdateManifests(ctx context.Context, client argocdv1.ArgoCDServiceGatewayClient, orgId string) diag.Diagnostics {
	akpCluster := &AkpCluster{
		Id:         x.Id,
		InstanceId: x.InstanceId,
	}
	diag := akpCluster.UpdateManifests(ctx, client, orgId)
	x.Manifests = akpCluster.Manifests
	return diag
}
