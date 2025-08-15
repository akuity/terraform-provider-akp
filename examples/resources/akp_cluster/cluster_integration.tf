data "akp_instance" "example-argo" {
  name = "test-argo"
}

data "akp_kargo_instance" "example-kargo" {
  name = "test-kargo"
}

resource "akp_cluster" "example-cluster-integration" {
  instance_id = data.akp_instance.example-argo.id
  name        = "test-cluster-integration"
  namespace   = "test"
  spec = {
    data = {
      size = "small"
      direct_cluster_spec = {
        cluster_type      = "kargo"
        kargo_instance_id = data.akp_kargo_instance.example-kargo.id
      }
    }
  }
}