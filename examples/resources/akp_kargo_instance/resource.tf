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

locals {
  yaml_files = fileset("${path.module}/kargo-manifests", "*.yaml")

  kargo_resources = flatten([
    for file_name in local.yaml_files : [
      for resource in split("\n---\n", file("${path.module}/kargo-manifests/${file_name}")) :
      jsonencode(yamldecode(resource))
    ]
  ])
}
