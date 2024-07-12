resource "akp_instance" "argocd" {
  name = "argocd"
  argocd = {
    "spec" = {
      "instance_spec" = {
        "declarative_management_enabled" = true
      }
      "version" = "v2.11.4"
    }
  }
  config_management_plugins = {
    "kasane" = {
      image   = "gcr.io/kasaneapp/kasane"
      enabled = false
      spec = {
        init = {
          command = [
            "kasane",
            "update"
          ]
        }
        generate = {
          command = [
            "kasane",
            "show"
          ]
        }
      }
    }
    "tanka" = {
      enabled = true
      image   = "grafana/tanka:0.25.0"
      spec = {
        discover = {
          file_name = "jsonnetfile.json"
        }
        generate = {
          args = [
            "tk show environments/$PARAM_ENV --dangerous-allow-redirect",
          ]
          command = [
            "sh",
            "-c",
          ]
        }
        init = {
          command = [
            "jb",
            "update",
          ]
        }
        parameters = {
          static = [
            {
              name     = "env"
              required = true
              string   = "default"
            },
          ]
        }
        preserve_file_mode = false
        version            = "v1.0"
      }
    },
  }
}
