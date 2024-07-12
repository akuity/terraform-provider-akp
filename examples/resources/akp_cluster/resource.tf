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
      size                  = "small"
      auto_upgrade_disabled = true
      target_version        = "0.4.0"
      managed_cluster_config = {
        secret_key  = "secret"
        secret_name = "secret-name"
      }
      eks_addon_enabled           = true
      datadog_annotations_enabled = true
      kustomization               = <<EOF
  apiVersion: kustomize.config.k8s.io/v1beta1
  kind: Kustomization
  patches:
    - patch: |-
        apiVersion: apps/v1
        kind: Deployment
        metadata:
          name: argocd-repo-server
        spec:
          template:
            spec:
              containers:
              - name: argocd-repo-server
                resources:
                  limits:
                    memory: 2Gi
                  requests:
                    cpu: 750m
                    memory: 1Gi
      target:
        kind: Deployment
        name: argocd-repo-server
            EOF
    }
  }

  kube_config = {
    config_path = "test.kubeconfig"
    token       = "YOUR TOKEN"
  }

  # When using a Kubernetes token retrieved from a Terraform provider (e.g. aws_eks_cluster_auth or google_client_config) in the above `kube_config`,
  # the token value may change over time. This will cause Terraform to detect a diff in the `token` on each plan and apply.
  # To prevent constant changes, you can add the `token` field path to the `lifecycle` block's `ignore_changes` list:
  #  https://developer.hashicorp.com/terraform/language/meta-arguments/lifecycle#ignore_changes
  lifecycle {
    ignore_changes = [
      kube_config.token,
    ]
  }
}
