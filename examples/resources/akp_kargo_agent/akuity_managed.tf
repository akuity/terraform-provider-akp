resource "akp_kargo_agent" "example-agent" {
  instance_id = akp_kargo_instance.example.id
  name        = "test-agent"
  namespace   = "test-namespace"
  workspace   = "kargo-workspace"
  labels = {
    "app" = "kargo"
  }
  annotations = {
    "app" = "kargo"
  }
  spec = {
    description = "test-description"
    data = {
      size = "medium"
      # Set this to false if the agent is self-hosted, and this should not be changed anymore once it is set.
      akuity_managed = true
      # this needs to be the ArgoCD instance ID, and once it is set, it should not be changed.
      remote_argocd = "<your_argocd_instance_id>" # Replace with your actual ArgoCD instance ID
    }
  }
}
