package akp

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func (a *AkpKargoAgentDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Gets information about a cluster by its name and Argo CD instance ID",
		Attributes:          getAKPKargoAgentDataSourceAttributes(),
	}
}

func getAKPKargoAgentDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			MarkdownDescription: "The ID of the Kargo agent",
			Computed:            true,
		},
		"instance_id": schema.StringAttribute{
			MarkdownDescription: "The ID of the Kargo instance",
			Computed:            true,
		},
		"name": schema.StringAttribute{
			MarkdownDescription: "The name of the Kargo agent",
			Required:            true,
		},
		"namespace": schema.StringAttribute{
			MarkdownDescription: "The namespace of the Kargo agent",
			Computed:            true,
		},
		"labels": schema.MapAttribute{
			ElementType:         types.StringType,
			MarkdownDescription: "The labels of the Kargo agent",
			Computed:            true,
		},
		"annotations": schema.MapAttribute{
			ElementType:         types.StringType,
			MarkdownDescription: "The annotations of the Kargo agent",
			Computed:            true,
		},
		"spec": schema.SingleNestedAttribute{
			MarkdownDescription: "The spec of the Kargo agent",
			Computed:            true,
			Attributes:          getAKPKargoAgentSpecDataSourceAttributes(),
		},
		"kube_config": schema.SingleNestedAttribute{
			MarkdownDescription: "The kubeconfig of the Kargo agent",
			Computed:            true,
			Attributes:          getKubeconfigDataSourceAttributes(),
		},
		"remove_agent_resources_on_destroy": schema.BoolAttribute{
			MarkdownDescription: "Whether to remove agent resources on destroy",
			Computed:            true,
		},
	}
}

func getAKPKargoAgentSpecDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"description": schema.StringAttribute{
			MarkdownDescription: "The description of the Kargo agent",
			Computed:            true,
		},
		"data": schema.SingleNestedAttribute{
			MarkdownDescription: "The data of the Kargo agent",
			Computed:            true,
			Attributes:          getAKPKargoAgentDataDataSourceAttributes(),
		},
	}
}

func getAKPKargoAgentDataDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"size": schema.StringAttribute{
			MarkdownDescription: "The size of the Kargo agent",
			Computed:            true,
		},
		"auto_upgrade_disabled": schema.BoolAttribute{
			MarkdownDescription: "Whether auto upgrade is disabled",
			Computed:            true,
		},
		"target_version": schema.StringAttribute{
			MarkdownDescription: "The target version of the Kargo agent",
			Computed:            true,
		},
		"kustomization": schema.StringAttribute{
			MarkdownDescription: "Kustomize configuration that will be applied to generated Kargo agent installation manifests",
			Computed:            true,
		},
		"remote_argocd": schema.StringAttribute{
			MarkdownDescription: "The ID of the remote Argo CD instance",
			Computed:            true,
		},
		"akuity_managed": schema.BoolAttribute{
			MarkdownDescription: "Whether the Kargo agent is managed by Akuity",
			Computed:            true,
		},
		"argocd_namespace": schema.StringAttribute{
			MarkdownDescription: "The namespace of the Argo CD instance",
			Computed:            true,
		},
	}
}
