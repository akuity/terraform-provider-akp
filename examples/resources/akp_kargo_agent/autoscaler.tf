resource "akp_kargo_agent" "example-agent-autosize" {
  instance_id = akp_kargo_instance.example.id
  name        = "test-agent-autosize"
  namespace   = "test-namespace"
  spec = {
    data = {
      size          = "auto"
      remote_argocd = ""
      autoscaler_config = {
        kargo_controller = {
          resource_minimum = {
            mem = "1Gi"
            cpu = "500m"
          }
          resource_maximum = {
            mem = "4Gi"
            cpu = "2000m"
          }
        }
      }
    }
  }
}
