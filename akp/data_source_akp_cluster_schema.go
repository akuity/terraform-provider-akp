package akp

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func (d *AkpClusterDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Find a cluster by its name and Argo CD instance ID",
		Attributes:          getAKPClusterDataSourceAttributes(),
	}
}

func getAKPClusterDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id": schema.StringAttribute{
			Computed:            true,
			MarkdownDescription: "Cluster ID",
		},
		"instance_id": schema.StringAttribute{
			MarkdownDescription: "Argo CD Instance ID",
			Required:            true,
		},
		"name": schema.StringAttribute{
			Required:            true,
			MarkdownDescription: "Name",
		},
		"namespace": schema.StringAttribute{
			MarkdownDescription: "Agent Installation Namespace",
			Computed:            true,
		},
		"labels": schema.MapAttribute{
			ElementType:         types.StringType,
			MarkdownDescription: "Labels",
			Computed:            true,
		},
		"annotations": schema.MapAttribute{
			ElementType:         types.StringType,
			MarkdownDescription: "Annotations",
			Computed:            true,
		},
		"spec": schema.SingleNestedAttribute{
			Computed:   true,
			Attributes: getClusterSpecDataSourceAttributes(),
		},
		"kubeconfig": schema.SingleNestedAttribute{
			MarkdownDescription: "Kubernetes connection settings. If configured, terraform will try to connect to the cluster and install the agent",
			Computed:            true,
			Attributes:          getKubeconfigDataSourceAttributes(),
		},
		"manifests": schema.StringAttribute{
			MarkdownDescription: "Agent Installation Manifests",
			Computed:            true,
			Sensitive:           true,
		},
	}
}

func getClusterSpecDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"description": schema.StringAttribute{
			MarkdownDescription: "Cluster Description",
			Computed:            true,
		},
		"namespace_scoped": schema.BoolAttribute{
			MarkdownDescription: "Agent Namespace Scoped",
			Computed:            true,
		},
		"data": schema.SingleNestedAttribute{
			Computed:   true,
			Attributes: getClusterDataDataSourceAttributes(),
		},
	}
}

func getClusterDataDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"size": schema.StringAttribute{
			MarkdownDescription: "Cluster Size. One of `small`, `medium` or `large`",
			Computed:            true,
		},
		"auto_upgrade_disabled": schema.BoolAttribute{
			MarkdownDescription: "Disable Agents Auto Upgrade. On resource update terraform will try to update the agent if this is set to `true`. Otherwise agent will update itself automatically",
			Computed:            true,
		},
		"kustomization": schema.StringAttribute{
			Computed: true,
		},
		"app_replication": schema.BoolAttribute{
			Computed: true,
		},
		"target_version": schema.StringAttribute{
			MarkdownDescription: "Installed agent version",
			Computed:            true,
		},
		"redis_tunneling": schema.BoolAttribute{
			Computed: true,
		},
	}
}

func getKubeconfigDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"host": schema.StringAttribute{
			Computed:    true,
			Description: "The hostname (in form of URI) of Kubernetes master.",
		},
		"username": schema.StringAttribute{
			Computed:    true,
			Description: "The username to use for HTTP basic authentication when accessing the Kubernetes master endpoint.",
		},
		"password": schema.StringAttribute{
			Computed:    true,
			Sensitive:   true,
			Description: "The password to use for HTTP basic authentication when accessing the Kubernetes master endpoint.",
		},
		"insecure": schema.BoolAttribute{
			Computed:    true,
			Description: "Whether server should be accessed without verifying the TLS certificate.",
		},
		"client_certificate": schema.StringAttribute{
			Computed:    true,
			Description: "PEM-encoded client certificate for TLS authentication.",
		},
		"client_key": schema.StringAttribute{
			Computed:    true,
			Sensitive:   true,
			Description: "PEM-encoded client certificate key for TLS authentication.",
		},
		"cluster_ca_certificate": schema.StringAttribute{
			Computed:    true,
			Description: "PEM-encoded root certificates bundle for TLS authentication.",
		},
		"config_paths": schema.ListAttribute{
			ElementType: types.StringType,
			Computed:    true,
			Description: "A list of paths to kube config files.",
		},
		"config_path": schema.StringAttribute{
			Computed:    true,
			Description: "Path to the kube config file.",
		},
		"config_context": schema.StringAttribute{
			Computed:    true,
			Description: "Context name to load from the kube config file.",
		},
		"config_context_auth_info": schema.StringAttribute{
			Computed:    true,
			Description: "",
		},
		"config_context_cluster": schema.StringAttribute{
			Computed:    true,
			Description: "",
		},
		"token": schema.StringAttribute{
			Computed:    true,
			Sensitive:   true,
			Description: "Token to authenticate an service account",
		},
		"proxy_url": schema.StringAttribute{
			Computed:    true,
			Description: "URL to the proxy to be used for all API requests",
		},
	}
}
