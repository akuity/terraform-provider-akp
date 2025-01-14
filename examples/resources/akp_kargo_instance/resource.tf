resource "akp_kargo_instance" "example" {
  name = "test"
  kargo = {
    spec = {
      description = "test-description"
      version     = "v1.1.1"
      kargo_instance_spec = {
        backend_ip_allow_list_enabled = true
        ip_allow_list = [
          {
            ip          = "66.66.66.66"
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
}
