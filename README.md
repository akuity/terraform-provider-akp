<a href="https://terraform.io">
    <img src=".github/tf.png" alt="Terraform logo" title="Terraform" align="left" height="50" />
</a>

# Terraform Provider for Akuity Platform
[![Tests](https://github.com/akuity/terraform-provider-akp/actions/workflows/test.yml/badge.svg?branch=main)](https://github.com/akuity/terraform-provider-akp/actions/workflows/test.yml)

With this provider you can manage Argo CD instances and clusters on [Akuity Platform](https://akuity.io/akuity-platform/).

* [Akuity Platform Docs](https://docs.akuity.io/)
* [Argo CD Docs](https://argo-cd.readthedocs.io/)
* [Akuity Platform Portal](https://akuity.cloud/)

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 1.0

## Typical use case
 Add a new cluster `test-cluster` to the existing Argo CD instance `manualy-created` and install [the agent](https://docs.akuity.io/akuity-platform/agent) to the configured cluster.

1. Create an API key for your organization
   * Use `Admin` role for the key
2. Configure Environment variables
    ```shell
    export AKUITY_API_KEY_ID=<key-id>
    export AKUITY_API_KEY_SECRET=<key-secret>
    ```
3. Use this or similar configuration:
    ```hcl
    terraform {
        required_providers {
            akp = {
                source = "akuity/akp"
                version = "~> 0.2"
            }
        }
    }

    provider "akp" {
        org_name = "<organization-name>"
    }

    # Read the existing Argo CD Instance
    data "akp_instance" "existing" {
        name = "manualy-created"
    }

    # Add cluster to the existing instance and install the agent
    resource "akp_cluster" "test" {
        name             = "test-cluster"
        description      = "Test Cluster 1"
        size             = "small"
        namespace        = "akuity"
        instance_id      = data.akp_instance.existing.id
        kube_config      = {
            # Configuration similar to `kubernetes` provider
        }
    }


    ```
