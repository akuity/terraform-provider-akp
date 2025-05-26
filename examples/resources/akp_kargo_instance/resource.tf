resource "akp_kargo_instance" "example" {
  name      = "test"
  workspace = "kargo-workspace"
  kargo_cm = {
    adminAccountEnabled  = "true"
    adminAccountTokenTtl = "24h"
  }
  kargo_secret = {
    adminAccountPasswordHash = "$2a$10$wThs/VVwx5Tbygkk5Rzbv.V8hR8JYYmRdBiGjue9pd0YcEXl7.Kn."
  }
  kargo = {
    spec = {
      description = "test-description"
      version     = "v1.4.3"
      // only set one of fqdn and subdomain
      fqdn      = "fqdn.example.com"
      subdomain = ""
      oidc_config = {
        enabled     = true
        dex_enabled = false
        # client_id should be set only if dex_enabled is false
        client_id = "test-client-id"
        # client_secret should be set only if dex_enabled is false
        cli_client_id = "test-cli-client-id"
        # issuer_url should be set only if dex_enabled is false
        issuer_url = "https://test.com"
        # additional_scopes should be set only if dex_enabled is false
        additional_scopes = ["test-scope"]
        # dex_secret should be set only if dex_enabled is false
        dex_secret = {
          name = "test-secret"
        }
        # dex_config should be set only if dex_enabled is true, and if dex is set, then oidc related fields should not be set
        dex_config = ""
        admin_account = {
          claims = {
            groups = {
              values = ["admin-group@example.com"]
            }
            email = {
              values = ["admin@example.com"]
            }
            sub = {
              values = ["admin-sub@example.com"]
            }
          }
        }
        viewer_account = {
          claims = {
            groups = {
              values = ["viewer-group@example.com"]
            }
            email = {
              values = ["viewer@example.com"]
            }
            sub = {
              values = ["viewer-sub@example.com"]
            }
          }
        }
      }
      kargo_instance_spec = {
        backend_ip_allow_list_enabled = true
        ip_allow_list = [
          {
            ip          = "88.88.88.88"
            description = "test-description"
          }
        ]
        agent_customization_defaults = {
          auto_upgrade_disabled = true
          kustomization         = <<-EOT
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
images:
  - name: ghcr.io/akuity/kargo
    newName: quay.io/akuityy/kargo
  - name: quay.io/akuityio/argo-rollouts
    newName: quay.io/akuityy/argo-rollouts
  - name: quay.io/akuity/agent
    newName: quay.io/akuityy/agent
EOT
        }
        default_shard_agent       = "test"
        global_credentials_ns     = ["test1", "test2"]
        global_service_account_ns = ["test3", "test4"]
      }
    }
  }
  kargo_resources = local.kargo_resources
}

# Choose a directory that contains Kargo resource manifests.
# For example, here we have kargo.yaml in the kargo-manifests directory, and the data is like:
# ---------------------------------------------
# apiVersion: kargo.akuity.io/v1alpha1
# kind: Project
# metadata:
#   name: kargo-demo
# ---
# apiVersion: kargo.akuity.io/v1alpha1
# kind: Warehouse
# metadata:
#   name: kargo-demo
#   namespace: kargo-demo
# spec:
#   subscriptions:
#   - image:
#       repoURL: public.ecr.aws/nginx/nginx
#       semverConstraint: ^1.28.0
#       discoveryLimit: 5
# ---
# ...
# ---------------------------------------------
#
# The following expression can parse the provided YAMLs into JSON strings for the provider to be validated and applied correctly.
# Remember to put the parsed kargo resources into `akp_kargo_instance.kargo_resources` field.
locals {
  yaml_files = fileset("${path.module}/kargo-manifests", "*.yaml")

  kargo_resources = merge([
    for file_name in local.yaml_files : {
      for idx, resource_yaml in split("\n---\n", file("${path.module}/kargo-manifests/${file_name}")) :
      "${yamldecode(resource_yaml).apiVersion}/${yamldecode(resource_yaml).kind}/${try(yamldecode(resource_yaml).metadata.namespace, "")}/${yamldecode(resource_yaml).metadata.name}" => jsonencode(yamldecode(resource_yaml))
      if trimspace(resource_yaml) != ""
    }
  ]...)
}
