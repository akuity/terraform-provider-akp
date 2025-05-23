resource "akp_cluster" "my-cluster" {
  instance_id = akp_instance.argocd.id
  kube_config = {
    host                   = "https://${cluster.my-cluster.endpoint}"
    cluster_ca_certificate = "${base64decode(cluster.my-cluster.master_auth.0.cluster_ca_certificate)}"
    // No need to hardcode a token!
    exec = {
      api_version = "client.authentication.k8s.io/v1"
      args        = ["eks", "get-token", "--cluster-name", "some-cluster"]
      command     = "aws"
      env = {
        AWS_REGION = "us-west-2"
      }
    }
  }
  name      = "my-cluster"
  namespace = "akuity"
  spec = {
    data = {
      size = "small"
    }
  }
}
