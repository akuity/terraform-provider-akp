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
  kargo_resources = [
    jsonencode(yamldecode(<<-YAML
      apiVersion: kargo.akuity.io/v1alpha1
      kind: Project
      metadata:
        name: kargo-demo
    YAML
    )),
    jsonencode(yamldecode(<<-YAML
      apiVersion: kargo.akuity.io/v1alpha1
      kind: Warehouse
      metadata:
        name: kargo-demo
        namespace: kargo-demo
      spec:
        subscriptions:
        - image:
            repoURL: public.ecr.aws/nginx/nginx
            semverConstraint: ^1.26.0
            discoveryLimit: 5
    YAML
    )),
    jsonencode(yamldecode(<<-YAML
      apiVersion: kargo.akuity.io/v1alpha1
      kind: PromotionTask
      metadata:
        name: demo-promo-process
        namespace: kargo-demo
      spec:
        vars:
        - name: gitopsRepo
          value: "https://hxp.github.com/test"
        - name: imageRepo
          value: public.ecr.aws/nginx/nginx
        steps:
        - uses: git-clone
          config:
            repoURL: \$${{ vars.gitopsRepo }}
            checkout:
            - branch: main
              path: ./src
            - branch: stage/\$${{ ctx.stage }}
              create: true
              path: ./out
        - uses: git-clear
          config:
            path: ./out
        - uses: kustomize-set-image
          as: update-image
          config:
            path: ./src/base
            images:
            - image: \$${{ vars.imageRepo }}
              tag: \$${{ imageFrom(vars.imageRepo).Tag }}
        - uses: kustomize-build
          config:
            path: ./src/stages/\$${{ ctx.stage }}
            outPath: ./out
        - uses: git-commit
          as: commit
          config:
            path: ./out
            messageFromSteps:
            - update-image
    YAML
    )),
    jsonencode(yamldecode(<<-YAML
      apiVersion: kargo.akuity.io/v1alpha1
      kind: Stage
      metadata:
        name: test
        namespace: kargo-demo
      spec:
        requestedFreight:
        - origin:
            kind: Warehouse
            name: kargo-demo
          sources:
            direct: true
        promotionTemplate:
          spec:
            steps:
            - task:
                name: demo-promo-process
              as: promo-process
    YAML
    ))
  ]
}
