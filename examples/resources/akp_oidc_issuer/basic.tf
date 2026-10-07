// GitHub Actions: keys come from the issuer's discovery document. Tokens
// must carry the server-assigned audience (see `default_audience`), which
// the workflow requests from GitHub's token endpoint.
resource "akp_oidc_issuer" "github_actions" {
  name       = "github-actions"
  issuer_url = "https://token.actions.githubusercontent.com"
  key_source = "discovery"
}

// Microsoft Entra cannot mint a custom audience, so the application's own
// audience is listed instead of the default.
resource "akp_oidc_issuer" "entra" {
  name              = "entra"
  issuer_url        = "https://login.microsoftonline.com/00000000-0000-0000-0000-000000000000/v2.0"
  key_source        = "discovery"
  audiences         = ["api://akuity-platform"]
  token_ttl_seconds = 900
}

// A private Kubernetes cluster whose API server is not reachable from the
// platform: paste its JWKS instead of fetching it.
resource "akp_oidc_issuer" "k8s" {
  name        = "prod-cluster"
  issuer_url  = "https://kubernetes.default.svc.cluster.local"
  key_source  = "static_jwks"
  static_jwks = file("${path.module}/prod-cluster-jwks.json")
}
