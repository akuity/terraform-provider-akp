resource "akp_kargo_agent" "example-agent" {
  instance_id = akp_kargo_instance.example.id
  name        = "test-agent"
  namespace   = "test-namespace"
  labels = {
    "app" = "kargo"
  }
  annotations = {
    "app" = "kargo"
  }
  spec = {
    description = "test-description"
    data = {
      target_version = "0.5.53"
      size           = "medium"
      // Set this to false if the agent is self-hosted, and this should not be changed anymore once it is set.
      akuity_managed = false
      # this needs to be the ArgoCD instance ID, and once it is set, it should not be changed.
      remote_argocd = ""
      # this can be configured in self-hosted mode, if the remote argocd is not provided, and if this is provided, the remote argocd will be ignored.
      argocd_namespace = "argocd"
      # configure this based on the situation of self-hosted or not.
      auto_upgrade_disabled = true
      kustomization         = <<-EOT
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
images:
  - name: ghcr.io/akuity/kargo
    newName: quay.io/akuity/kargo
  - name: quay.io/akuityio/argo-rollouts
    newName: quay.io/akuity/argo-rollouts
  - name: quay.io/akuity/agent
    newName: quay.io/akuity/agent
EOT
    }
  }
}
