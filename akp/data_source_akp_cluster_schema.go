package akp

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func (d *AkpClusterDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Gets information about a cluster by its name and Argo CD instance ID",
		Attributes:          getAKPClusterDataSourceAttributes(),
	}
}

func getAKPClusterDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"instance_id": schema.StringAttribute{
			MarkdownDescription: "Argo CD instance ID",
			Required:            true,
		},
		"id": schema.StringAttribute{
			MarkdownDescription: "Cluster ID",
			Computed:            true,
		},
		"name": schema.StringAttribute{
			Required:            true,
			MarkdownDescription: "Cluster name",
		},
		"namespace": schema.StringAttribute{
			MarkdownDescription: "Agent installation namespace",
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
			MarkdownDescription: "Cluster spec",
			Computed:            true,
			Attributes:          getClusterSpecDataSourceAttributes(),
		},
		"kube_config": schema.SingleNestedAttribute{
			MarkdownDescription: "Kubernetes connection settings. If configured, terraform will try to connect to the cluster and install the agent",
			Computed:            true,
			Attributes:          getKubeconfigDataSourceAttributes(),
		},
		"remove_agent_resources_on_destroy": schema.BoolAttribute{
			MarkdownDescription: "Remove agent Kubernetes resources from the managed cluster when destroying cluster",
			Computed:            true,
		},
	}
}

func getClusterSpecDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"description": schema.StringAttribute{
			MarkdownDescription: "Cluster description",
			Computed:            true,
		},
		"namespace_scoped": schema.BoolAttribute{
			MarkdownDescription: "If the agent is namespace scoped",
			Computed:            true,
		},
		"data": schema.SingleNestedAttribute{
			MarkdownDescription: "Cluster data",
			Computed:            true,
			Attributes:          getClusterDataDataSourceAttributes(),
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
			MarkdownDescription: "Disables agents auto upgrade. On resource update terraform will try to update the agent if this is set to `true`. Otherwise agent will update itself automatically",
			Computed:            true,
		},
		"kustomization": schema.StringAttribute{
			MarkdownDescription: "Kustomize configuration that will be applied to generated agent installation manifests",
			Computed:            true,
		},
		"app_replication": schema.BoolAttribute{
			MarkdownDescription: "Enables Argo CD state replication to the managed cluster that allows disconnecting the cluster from Akuity Platform without losing core Argocd features",
			Computed:            true,
		},
		"target_version": schema.StringAttribute{
			MarkdownDescription: "The version of the agent to install on your cluster",
			Computed:            true,
		},
		"redis_tunneling": schema.BoolAttribute{
			MarkdownDescription: "Enables the ability to connect to Redis over a web-socket tunnel that allows using Akuity agent behind HTTPS proxy",
			Computed:            true,
		},
		"datadog_annotations_enabled": schema.BoolAttribute{
			MarkdownDescription: "Enable Datadog metrics collection of Application Controller and Repo Server. Make sure that you install Datadog agent in cluster.",
			Computed:            true,
		},
		"eks_addon_enabled": schema.BoolAttribute{
			MarkdownDescription: "Enable this if you are installing this cluster on EKS.",
			Computed:            true,
		},
		"managed_cluster_config": schema.SingleNestedAttribute{
			MarkdownDescription: "The config to access managed Kubernetes cluster. By default agent is using \"in-cluster\" config.",
			Computed:            true,
			Attributes:          getManagedClusterConfigDataSourceAttributes(),
		},
		"multi_cluster_k8s_dashboard_enabled": schema.BoolAttribute{
			MarkdownDescription: "Enable the KubeVision feature on the managed cluster",
			Computed:            true,
		},
		"autoscaler_config": schema.SingleNestedAttribute{
			MarkdownDescription: "Autoscaler config for auto agent size",
			Computed:            true,
			Attributes:          getAutoScalerConfigDataSourceAttributes(),
		},
		"custom_agent_size_config": schema.SingleNestedAttribute{
			MarkdownDescription: "Custom agent size config",
			Computed:            true,
			Attributes:          getAutoScalerConfigDataSourceAttributes(),
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

func getManagedClusterConfigDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"secret_name": schema.StringAttribute{
			Description: "The name of the secret for the managed cluster config",
			Computed:    true,
		},
		"secret_key": schema.StringAttribute{
			Description: "The key in the secret for the managed cluster config",
			Computed:    true,
		},
	}
}

func getAutoScalerConfigDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"application_controller": schema.SingleNestedAttribute{
			Description: "Application Controller auto scaling config",
			Computed:    true,
			Attributes:  getAppControllerAutoScalingConfigDataSourceAttributes(),
		},
		"repo_server": schema.SingleNestedAttribute{
			Description: "Repo Server auto scaling config",
			Computed:    true,
			Attributes:  getRepoServerAutoScalingConfigDataSourceAttributes(),
		},
	}
}

func getAppControllerAutoScalingConfigDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"resource_minimum": schema.SingleNestedAttribute{
			Description: "Resource minimum",
			Computed:    true,
			Attributes:  getResourcesDataSourceAttributes(),
		},
		"resource_maximum": schema.SingleNestedAttribute{
			Description: "Resource maximum",
			Computed:    true,
			Attributes:  getResourcesDataSourceAttributes(),
		},
	}
}

func getRepoServerAutoScalingConfigDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"resource_minimum": schema.SingleNestedAttribute{
			Description: "Resource minimum",
			Computed:    true,
			Attributes:  getResourcesDataSourceAttributes(),
		},
		"resource_maximum": schema.SingleNestedAttribute{
			Description: "Resource maximum",
			Computed:    true,
			Attributes:  getResourcesDataSourceAttributes(),
		},
		"replica_maximum": schema.Int64Attribute{
			Description: "Replica maximum",
			Computed:    true,
		},
		"replica_minimum": schema.Int64Attribute{
			Description: "Replica minimum",
			Computed:    true,
		},
	}
}

func getResourcesDataSourceAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"mem": schema.StringAttribute{
			Description: "Memory",
			Computed:    true,
		},
		"cpu": schema.StringAttribute{
			Description: "CPU",
			Computed:    true,
		},
	}
}
