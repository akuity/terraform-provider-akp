---
page_title: "Using Terraform AKP Provider to Manage Config Management Plugins"
subcategory: ""
description: Using Terraform AKP Provider to Manage Config Management Plugins
---
# Using Terraform AKP Provider to Manage Config Management Plugins
Version 0.6.2 of the AKP provider for Terraform introduces the ability to manage [Config Management Plugins (CMPs) v2](https://argo-cd.readthedocs.io/en/stable/operator-manual/config-management-plugins/) in your Akuity Platform Argo CD instance. This guide will help with that process.

## Provider Version Configuration

Managing CMPs requires version 0.6.2 or later of the AKP provider.

If you are using v0.5 or earlier version of AKP provider, please follow the [v0.5](v0.5-upgrading.md) and [v0.6 upgrade guide](v0.6-upgrading.md) to upgrade to v0.6 version.

If you are using v0.6.0 or v0.6.1 of AKP provider, you can update the version constraints in your Terraform configuration and run `terraform init -upgrade` to upgrade to v0.6.2.

## Take over the previously created CMPs from AKP UI
!> If you have any CMPs created previously in AKP UI, this step is mandatory. Make sure to add them to your terraform configuration before `terraform apply`, otherwise, CMPs created from UI will be pruned.

To do that, you can run `terraform refresh` to refresh state to have the UI-created CMPs in the `terraform.tfstate` file.

Then, you can do `terraform show` and copy the `config_management_plugins` block in `akp_instance` resource to the same `akp_instance` resource in your terraform configuration.

Finally, run `terraform plan` to make sure there is no changes to be applied.

## Add new CMPs

If you want to use AKP provider to manage CMPs, you need to add `config_management_plugins` block to your `akp_instance` resource. Please check the [`akp_instance` resource](../resources/instance.md) for CMP schema and examples.

## Delete CMPs
You can delete CMPs by removing them from the `config_management_plugins` block in your `akp_instance` resource.

!> Note that removing the entire `config_management_plugins` block from `akp_instance` resource will prune all the CMPs in your AKP Argo CD instance.

Make sure to review the changes before applying them.