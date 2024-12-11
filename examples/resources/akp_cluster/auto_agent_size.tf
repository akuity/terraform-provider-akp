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
      size = "auto"
      # auto_upgrade_disabled  can be set to true if you want to enable auto scaling of the agent size for the cluster
      auto_upgrade_disabled = false
      auto_agent_size_config = {
        application_controller = {
          resource_maximum = {
            cpu = "3"
            mem = "2Gi"
          },
          resource_minimum = {
            cpu = "250m",
            mem = "1Gi"
          }
        },
        repo_server = {
          replica_maximum = 3,
          # minimum number of replicas should be set to 1
          replica_minimum = 1,
          resource_maximum = {
            cpu = "3"
            mem = "2.00Gi"
          },
          resource_minimum = {
            cpu = "250m",
            mem = "256Mi"
          }
        }
      }
    }
  }

}
