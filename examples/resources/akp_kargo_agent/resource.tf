resource "akp_kargo_agent" "example-agent" {
  instance_id = akp_kargo_instance.example.id
  name        = "test-agent"
  labels = {
    "app" = "kargo"
  }
  annotations = {
    "app" = "kargo"
  }
  spec = {
    description = "test-description"
    data = {
      target_version        = "0.5.52"
      size                  = "small"
      auto_upgrade_disabled = true
      RemoteArgocd          = "test-argocd"
      AkuityManaged         = true
      ArgocdNamespace       = "test-argocd-namespace"
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
