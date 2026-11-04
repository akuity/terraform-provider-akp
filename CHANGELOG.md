# Release notes

## Unreleased

### Breaking changes

- Removed the deprecated `argocd.spec.instance_spec.assistant_extension_enabled` attribute from the `akp_instance` resource and data source. Configurations that still set or reference it will fail to plan. Remove those settings and references before upgrading the provider; the updated provider migrates existing state automatically without recreating instances. See the [migration guide](docs/guides/legacy-assistant-removal.md).
- Before upgrading Akuity Platform, remove the setting even if the provider version remains pinned. Older providers such as `v0.9.21` read the retired field as `false`, so an explicit `true` configuration can cause repeated plan differences or an inconsistent-result error after apply. Default `false` values sent by older providers remain accepted.
