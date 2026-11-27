resource "akp_kargo_agent" "federated" {
  instance_id = akp_kargo_instance.example.id
  name        = "federated-agent"
  namespace   = "akuity"
  spec = {
    data = {
      size          = "small"
      remote_argocd = akp_instance.argocd.id
      federation = {
        issuer                     = "https://oidc.eks.us-west-2.amazonaws.com/id/EXAMPLE"
        jwks_url                   = "https://oidc.eks.us-west-2.amazonaws.com/id/EXAMPLE/keys"
        promotion_service_accounts = ["kargo-promoter"]
      }
    }
  }
}
