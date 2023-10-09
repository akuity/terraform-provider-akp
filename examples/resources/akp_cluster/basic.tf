resource "akp_cluster" "my-cluster" {
  instance_id = akp_instance.argocd.id
  kube_config = {
    host                   = "https://${cluster.my-cluster.endpoint}"
    token                  = var.my_token
    client_certificate     = "${base64decode(cluster.my-cluster.master_auth.0.client_certificate)}"
    client_key             = "${base64decode(cluster.my-cluster.master_auth.0.client_key)}"
    cluster_ca_certificate = "${base64decode(cluster.my-cluster.master_auth.0.cluster_ca_certificate)}"
  }
  name      = "my-cluster"
  namespace = "akuity"
  spec = {
    data = {
      size = "small"
    }
  }
}
