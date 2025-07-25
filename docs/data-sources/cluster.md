---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "akp_cluster Data Source - akp"
subcategory: ""
description: |-
  Gets information about a cluster by its name and Argo CD instance ID
---

# akp_cluster (Data Source)

Gets information about a cluster by its name and Argo CD instance ID

## Example Usage

```terraform
data "akp_instance" "example" {
  name = "test"
}

data "akp_cluster" "example" {
  instance_id = data.akp_instance.example.id
  name        = "test"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `instance_id` (String) Argo CD instance ID
- `name` (String) Cluster name

### Read-Only

- `annotations` (Map of String) Annotations
- `id` (String) Cluster ID
- `kube_config` (Attributes) Kubernetes connection settings. If configured, terraform will try to connect to the cluster and install the agent (see [below for nested schema](#nestedatt--kube_config))
- `labels` (Map of String) Labels
- `namespace` (String) Agent installation namespace
- `remove_agent_resources_on_destroy` (Boolean) Remove agent Kubernetes resources from the managed cluster when destroying cluster
- `spec` (Attributes) Cluster spec (see [below for nested schema](#nestedatt--spec))

<a id="nestedatt--kube_config"></a>
### Nested Schema for `kube_config`

Read-Only:

- `client_certificate` (String) PEM-encoded client certificate for TLS authentication.
- `client_key` (String, Sensitive) PEM-encoded client certificate key for TLS authentication.
- `cluster_ca_certificate` (String) PEM-encoded root certificates bundle for TLS authentication.
- `config_context` (String) Context name to load from the kube config file.
- `config_context_auth_info` (String)
- `config_context_cluster` (String)
- `config_path` (String) Path to the kube config file.
- `config_paths` (List of String) A list of paths to kube config files.
- `host` (String) The hostname (in form of URI) of Kubernetes master.
- `insecure` (Boolean) Whether server should be accessed without verifying the TLS certificate.
- `password` (String, Sensitive) The password to use for HTTP basic authentication when accessing the Kubernetes master endpoint.
- `proxy_url` (String) URL to the proxy to be used for all API requests
- `token` (String, Sensitive) Token to authenticate an service account
- `username` (String) The username to use for HTTP basic authentication when accessing the Kubernetes master endpoint.


<a id="nestedatt--spec"></a>
### Nested Schema for `spec`

Read-Only:

- `data` (Attributes) Cluster data (see [below for nested schema](#nestedatt--spec--data))
- `description` (String) Cluster description
- `namespace_scoped` (Boolean) If the agent is namespace scoped

<a id="nestedatt--spec--data"></a>
### Nested Schema for `spec.data`

Read-Only:

- `app_replication` (Boolean) Enables Argo CD state replication to the managed cluster that allows disconnecting the cluster from Akuity Platform without losing core Argocd features
- `argocd_notifications_settings` (Attributes) ArgoCD notifications settings (see [below for nested schema](#nestedatt--spec--data--argocd_notifications_settings))
- `auto_agent_size_config` (Attributes) Autoscaler config for auto agent size (see [below for nested schema](#nestedatt--spec--data--auto_agent_size_config))
- `auto_upgrade_disabled` (Boolean) Disables agents auto upgrade. On resource update terraform will try to update the agent if this is set to `true`. Otherwise agent will update itself automatically
- `compatibility` (Attributes) Cluster compatibility settings (see [below for nested schema](#nestedatt--spec--data--compatibility))
- `custom_agent_size_config` (Attributes) Custom agent size config (see [below for nested schema](#nestedatt--spec--data--custom_agent_size_config))
- `datadog_annotations_enabled` (Boolean) Enable Datadog metrics collection of Application Controller and Repo Server. Make sure that you install Datadog agent in cluster.
- `eks_addon_enabled` (Boolean) Enable this if you are installing this cluster on EKS.
- `kustomization` (String) Kustomize configuration that will be applied to generated agent installation manifests
- `managed_cluster_config` (Attributes) The config to access managed Kubernetes cluster. By default agent is using "in-cluster" config. (see [below for nested schema](#nestedatt--spec--data--managed_cluster_config))
- `multi_cluster_k8s_dashboard_enabled` (Boolean) Enable the KubeVision feature on the managed cluster
- `project` (String) Project name
- `redis_tunneling` (Boolean) Enables the ability to connect to Redis over a web-socket tunnel that allows using Akuity agent behind HTTPS proxy
- `size` (String) Cluster Size. One of `small`, `medium`, `large`, `custom` or `auto`
- `target_version` (String) The version of the agent to install on your cluster

<a id="nestedatt--spec--data--argocd_notifications_settings"></a>
### Nested Schema for `spec.data.argocd_notifications_settings`

Read-Only:

- `in_cluster_settings` (Boolean) Enable in-cluster settings for ArgoCD notifications


<a id="nestedatt--spec--data--auto_agent_size_config"></a>
### Nested Schema for `spec.data.auto_agent_size_config`

Read-Only:

- `application_controller` (Attributes) Application Controller auto scaling config (see [below for nested schema](#nestedatt--spec--data--auto_agent_size_config--application_controller))
- `repo_server` (Attributes) Repo Server auto scaling config (see [below for nested schema](#nestedatt--spec--data--auto_agent_size_config--repo_server))

<a id="nestedatt--spec--data--auto_agent_size_config--application_controller"></a>
### Nested Schema for `spec.data.auto_agent_size_config.application_controller`

Read-Only:

- `resource_maximum` (Attributes) Resource maximum (see [below for nested schema](#nestedatt--spec--data--auto_agent_size_config--application_controller--resource_maximum))
- `resource_minimum` (Attributes) Resource minimum (see [below for nested schema](#nestedatt--spec--data--auto_agent_size_config--application_controller--resource_minimum))

<a id="nestedatt--spec--data--auto_agent_size_config--application_controller--resource_maximum"></a>
### Nested Schema for `spec.data.auto_agent_size_config.application_controller.resource_maximum`

Read-Only:

- `cpu` (String) CPU
- `memory` (String) Memory


<a id="nestedatt--spec--data--auto_agent_size_config--application_controller--resource_minimum"></a>
### Nested Schema for `spec.data.auto_agent_size_config.application_controller.resource_minimum`

Read-Only:

- `cpu` (String) CPU
- `memory` (String) Memory



<a id="nestedatt--spec--data--auto_agent_size_config--repo_server"></a>
### Nested Schema for `spec.data.auto_agent_size_config.repo_server`

Read-Only:

- `replicas_maximum` (Number) Replica maximum
- `replicas_minimum` (Number) Replica minimum
- `resource_maximum` (Attributes) Resource maximum (see [below for nested schema](#nestedatt--spec--data--auto_agent_size_config--repo_server--resource_maximum))
- `resource_minimum` (Attributes) Resource minimum (see [below for nested schema](#nestedatt--spec--data--auto_agent_size_config--repo_server--resource_minimum))

<a id="nestedatt--spec--data--auto_agent_size_config--repo_server--resource_maximum"></a>
### Nested Schema for `spec.data.auto_agent_size_config.repo_server.resource_maximum`

Read-Only:

- `cpu` (String) CPU
- `memory` (String) Memory


<a id="nestedatt--spec--data--auto_agent_size_config--repo_server--resource_minimum"></a>
### Nested Schema for `spec.data.auto_agent_size_config.repo_server.resource_minimum`

Read-Only:

- `cpu` (String) CPU
- `memory` (String) Memory




<a id="nestedatt--spec--data--compatibility"></a>
### Nested Schema for `spec.data.compatibility`

Read-Only:

- `ipv6_only` (Boolean) IPv6 only configuration


<a id="nestedatt--spec--data--custom_agent_size_config"></a>
### Nested Schema for `spec.data.custom_agent_size_config`

Read-Only:

- `application_controller` (Attributes) Application Controller custom agent size config (see [below for nested schema](#nestedatt--spec--data--custom_agent_size_config--application_controller))
- `repo_server` (Attributes) Repo Server custom agent size config (see [below for nested schema](#nestedatt--spec--data--custom_agent_size_config--repo_server))

<a id="nestedatt--spec--data--custom_agent_size_config--application_controller"></a>
### Nested Schema for `spec.data.custom_agent_size_config.application_controller`

Read-Only:

- `cpu` (String) CPU
- `memory` (String) Memory


<a id="nestedatt--spec--data--custom_agent_size_config--repo_server"></a>
### Nested Schema for `spec.data.custom_agent_size_config.repo_server`

Read-Only:

- `cpu` (String) CPU
- `memory` (String) Memory
- `replicas` (Number) Replica



<a id="nestedatt--spec--data--managed_cluster_config"></a>
### Nested Schema for `spec.data.managed_cluster_config`

Read-Only:

- `secret_key` (String) The key in the secret for the managed cluster config
- `secret_name` (String) The name of the secret for the managed cluster config
