resource "akp_kargo_agent" "example-agent" {
  instance_id = akp_kargo_instance.example.id
  name        = "test-agent"
  spec = {
    data = {
      size = "small"
      # How the agent is reached: "public" (internet) or "private" (AWS PrivateLink).
      connectivity = "public"
      # PEM bundle of CA certificates the agent workloads must trust in addition to
      # the system roots (e.g. a TLS-intercepting proxy CA).
      custom_ca_bundle = "-----BEGIN CERTIFICATE-----\nMIIB...\n-----END CERTIFICATE-----\n"
    }
  }
}

resource "akp_kargo_agent" "example-agent-autosize" {
  instance_id = akp_kargo_instance.example.id
  name        = "test-agent-autosize"
  spec = {
    data = {
      size = "auto"
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
