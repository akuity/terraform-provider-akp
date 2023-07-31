terraform {
  required_providers {
    akp = {
      source = "akuity/akp"
    }
  }
}

provider "akp" {
  org_name = "test"
}

data "akp_instance" "example" {
  name = "test"
}

resource "akp_cluster" "example" {
  instance_id = data.akp_instance.example.id
  kube_config = {
    "config_path" = "test.kubeconfig"
  }
  name      = "test-cluster"
  namespace = "test"
  labels = {
    test-label = "true"
  }
  annotations = {
    test-annotation = "false"
  }
  spec = {
    namespace_scoped = true
    description      = "test-description"
    data = {
      size                  = "small"
      auto_upgrade_disabled = true
      target_version        = "0.4.0"
      kustomization         = <<EOF
  apiVersion: kustomize.config.k8s.io/v1beta1
  kind: Kustomization
  resources:
  - test.yaml
            EOF
    }
  }
}
