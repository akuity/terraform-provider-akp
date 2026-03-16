resource "akp_kargo_instance" "kargo_instance" {
  name = "kargo-instance"
  kargo = {
    spec = {
      version             = "v1.9.3-ak.0"
      kargo_instance_spec = {}
    }
  }
}

resource "akp_kargo_agent" "kargo_agent" {
  instance_id = akp_kargo_instance.kargo_instance.id
  name        = "kargo-agent"
  spec = {
    data = {
      size = "small"
    }
  }
}

resource "akp_kargo_default_shard_agent" "default_shard_agent" {
  kargo_instance_id = akp_kargo_instance.kargo_instance.id
  agent_id          = akp_kargo_agent.kargo_agent.id
}
