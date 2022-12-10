<a href="https://terraform.io">
    <img src=".github/tf.png" alt="Terraform logo" title="Terraform" align="left" height="50" />
</a>

# Terraform Provider for Akuity Platform

With this provider you can manage Argo CD instances and clusters on [Akuity Platform](https://akuity.io/akuity-platform/).

* [Akuity Platform Docs](https://docs.akuity.io/)
* [Argo CD Docs](https://argo-cd.readthedocs.io/)
* [Akuity Platform Portal](https://akuity.cloud/)

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 1.0

## Typical use case
 Add a new cluster `test-cluster` to the existing Argo CD instance `manualy-created` and then install [the agent](https://docs.akuity.io/akuity-platform/agent) to the configured cluster.

1. Create an API key for your organization
   * Use `Admin` role for the key
1. Configure Environment variables
    ```shell
    export AKUITY_API_KEY_ID=<key-id>
    export AKUITY_API_KEY_SECRET=<key-secret>
    ```
1. Use this or similar configuration:
    ```hcl
    terraform {
        required_providers {
            akp = {
                source = "akuity/akp"
            }
            kubectl = {
                source  = "gavinbunney/kubectl"
                version = "~> 1.14"
            }
        }
    }

    provider "akp" {
        org_name = "<organization-name>"
    }

    provider "kubectl" {
        # Configure the kubectl provider
        # to connect to the existing kubernetes cluster
    }

    # Read Existing Argo CD Instance
    data "akp_instance" "existing" {
        name = "manualy-created"
    }

    # Add cluster to the existing instance
    resource "akp_cluster" "test" {
        name             = "test-cluster"
        description      = "Test Cluster 1"
        namespace        = "akuity"
        instance_id      = data.akp_instance.existing.id
    }

    # Split and install agent manifests
    data "kubectl_file_documents" "agent" {
        content = akp_cluster.test.manifests
    }

    # Create namespace first
    resource "kubectl_manifest" "agent_namespace" {
        yaml_body = lookup(data.kubectl_file_documents.agent.manifests, "/api/v1/namespaces/${akp_cluster.test.namespace}/namespaces/${akp_cluster.test.namespace}")
        wait      = true
    }

    # Create everything else
    resource "kubectl_manifest" "agent" {
        for_each  = data.kubectl_file_documents.agent.manifests
        yaml_body = each.value
        # Important!
        wait_for_rollout = false
        depends_on = [
            kubectl_manifest.agent_namespace
        ]
    }

    ```

1. First create the cluster using `-target`
    ```shell
    terraform apply -target akp_cluster.test
    ```
    This has to be done first, because terraform is unable to apply an unknown number of manifests in `for_each` loop, and the manifests only available *after* the cluster is created

1. Then you can apply the manifests
    ```shell
    terraform apply
    ```
