data "akp_instance" "example" {
  name = "test"
}

resource "akp_cluster" "example" {
  instance_id = data.akp_instance.example.id
  name        = "test-cluster"
  namespace   = "test"
  labels = {
    test-label = true
  }
  annotations = {
    test-annotation = false
  }
  spec = {
    namespace_scoped = true
    description      = "test-description"
    data = {
      size                  = "custom"
      auto_upgrade_disabled = false
      custom_agent_size_config = {
        application_controller = {
          cpu    = "1000m"
          memory = "2Gi"
        }
        repo_server = {
          replica = 3,
          cpu     = "1000m"
          memory  = "2Gi"
        }
      }
    }
  }
}
