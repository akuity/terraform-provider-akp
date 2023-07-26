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
        version = "~> 0.4"
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

## Creating an Argo CD instance with Terraform

``` hcl
resource "akp_instance" "example" {
  name        = "tf-example"
  version     = "v2.6.0"
  description = "An example of terraform automation for managing Akuity Platform resources"
  web_terminal = {
    enabled = true
  }
  kustomize = {
    build_options = "--enable-helm"
  }
  secrets = {
    sso_secret = {
      value = "secret"
    }
  }
  image_updater = {
    secrets = {
      docker_json = {
        value = "secret"
      }
    }
    registries = {
      docker = {
        prefix      = "docker.io"
        api_url     = "https://registry-1.docker.io"
        credentials = "secret:argocd/argocd-image-updater-secret#docker_json"
      }
    }
  }
  declarative_management_enabled = true
  image_updater_enabled          = true
}
```

See more examples in [terraform-akp-example](https://github.com/akuity/terraform-akp-example) repo

## Migration
### Cluster
| Previous Field          | Current Field                     |
|-------------------------|-----------------------------------|
| `agent_version`         | `spec.data.target_version`        |
| `auto_upgrade_disabled` | `spec.data.auto_upgrade_disabled` |
| `description`           | `spec.description`                |
| `kube_config`           | `kubeconfig`                      |
| `namespace_scoped`      | `spec.namespace_scoped`           |
| `size`                  | `spec.data.size`                  |

### Instance
