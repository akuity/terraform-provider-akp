---
page_title: "Legacy AI Assistant removal"
subcategory: ""
description: Preparing Terraform configurations for removal of the legacy AI Assistant.
---

# Legacy AI Assistant Removal

## Unreleased: Breaking Change

The deprecated `argocd.spec.instance_spec.assistant_extension_enabled` attribute has been removed from the `akp_instance` resource and data source. The legacy AI Assistant was replaced by AI Agents, and this attribute no longer enables a feature.

## Before upgrading Akuity Platform

Remove `assistant_extension_enabled` from your resource configuration before upgrading the platform to the release that removes the legacy assistant, even if you keep your current provider version. Remove assignments to `true`, `false` or `null`.

The platform temporarily accepts and ignores the deprecated field in requests so older providers can still send their default `false` value. However, the field is no longer returned by instance export. Older providers such as `v0.9.21` turn that missing field into `false` during refresh and after apply. If your configuration explicitly sets it to `true`, this can cause repeated plan differences or an inconsistent-result error after apply. Accepting the request alone does not make that configuration compatible.

For example, remove this line from `argocd.spec.instance_spec`:

```terraform
assistant_extension_enabled = true
```

If the platform has already been upgraded, remove the setting before your next plan or apply. With the attribute omitted from configuration, older providers can accept the refreshed `false` value because the attribute is computed. You do not need to edit state or recreate the instance.

## Before upgrading the Terraform provider

Configurations that still set `assistant_extension_enabled` will fail to plan after upgrading to the provider release containing this change. Remove expressions that read the attribute from resources or data sources as well.

The updated provider drops the retired attribute when Terraform upgrades the existing state. You do not need to edit state, re-import an instance, or recreate an instance. Other instance settings remain unchanged.

After updating your configuration and provider version constraint, run `terraform init -upgrade` and review `terraform plan` before applying.

The request compatibility retained by the platform does not restore the legacy assistant.
