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

- [Terraform](https://www.terraform.io/downloads.html) >= 1.2

## Typical use case
 Add a new cluster `test-cluster` to the existing Argo CD instance `manualy-created` and install [the agent](https://docs.akuity.io/argo-cd/clusters/) to the configured cluster.

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
      version = "~> 0.7.0"
    }
  }
}

provider "akp" {
  org_name = "<organization-name>"
}

resource "akp_instance" "argocd" {
   name = "argocd"
   argocd = {
      "spec" = {
         "instance_spec" = {
            "declarative_management_enabled" = true
         }
         "version" = "v2.11.4"
      }
   }
}

resource "akp_cluster" "example" {
   instance_id = akp_instance.argocd.id
   kube_config = {
      "config_path" = "test.kubeconfig"
   }
   name      = "test-cluster"
   namespace = "test"
   spec = {
      data = {
         size = "small"
      }
   }
}
```
See more examples in [here](https://github.com/akuity/terraform-provider-akp/tree/main/examples).


## Upgrading
- [Upgrading to v0.5](./docs/guides/v0.5-upgrading.md)
- [Upgrading to v0.6](./docs/guides/v0.6-upgrading.md)
- [Upgrading to v0.7](./docs/guides/v0.7-upgrading.md)
