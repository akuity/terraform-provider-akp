resource "akp_cluster" "federated" {
  instance_id = akp_instance.argocd.id
  name        = "federated-cluster"
  namespace   = "akuity"
  spec = {
    data = {
      size = "small"
      federation = {
        issuer = "https://oidc.eks.us-west-2.amazonaws.com/id/EXAMPLE"
        # Or paste the cluster's keys instead:
        # jwks = jsonencode({ keys = [...] })
        jwks_url = "https://oidc.eks.us-west-2.amazonaws.com/id/EXAMPLE/keys"
      }
    }
  }
}
