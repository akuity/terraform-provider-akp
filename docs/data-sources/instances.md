---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "akp_instances Data Source - akp"
subcategory: ""
description: |-
  List all Argo CD instances
---

# akp_instances (Data Source)

List all Argo CD instances

## Example Usage

```terraform
data "akp_instances" "all" {}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Read-Only

- `id` (String) The ID of this resource.
- `instances` (Attributes List) List of Argo CD instances for organization (see [below for nested schema](#nestedatt--instances))

<a id="nestedatt--instances"></a>
### Nested Schema for `instances`

Read-Only:

- `description` (String) Instance Description
- `hostname` (String) Instance Hostname
- `id` (String) Instance ID
- `name` (String) Instance Name
- `version` (String) Argo CD Version


