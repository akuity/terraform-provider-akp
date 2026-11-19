# Release notes

## Unreleased

### Features

- The `akp_instance` and `akp_kargo_instance` data sources take an optional `resource_filter`: a set of keys, written like the keys of `argocd_resources` or `kargo_resources` (`<apiVersion>/<kind>/<namespace>/<name>`), that limits the map to the named objects. Platforms that support it then read only those objects instead of listing every Application, ApplicationSet, AppProject, or Kargo object; older platforms still export everything and the provider drops the rest. Without `resource_filter` the data sources behave as before, and an empty set gives an empty map. An Application, ApplicationSet, or AppProject in the `argocd` namespace can be named with or without that namespace, so both the resource's keys and the data source's own keys work. The API version is not compared. Other attributes are not filtered. An entry that isn't valid, or a set that takes more than 4 KiB in the request URL, fails the read rather than falling back to a full export.

### Improvements

- `akp_instance` and `akp_kargo_instance` reads now ask the platform only for the objects in `argocd_resources` or `kargo_resources`, instead of exporting every Application, ApplicationSet, AppProject, or Kargo object on the instance. Plan and refresh get much cheaper on large instances. A full export, filtered by the provider as before, is still used when the list of objects would make the request URL longer than 4 KiB, and against platforms that predate the filter.

### Breaking changes

- Removed the deprecated `argocd.spec.instance_spec.assistant_extension_enabled` attribute from the `akp_instance` resource and data source. Configurations that still set or reference it will fail to plan. Remove those settings and references before upgrading the provider; the updated provider migrates existing state automatically without recreating instances. See the [migration guide](docs/guides/legacy-assistant-removal.md).
- Before upgrading Akuity Platform, remove the setting even if the provider version remains pinned. Older providers such as `v0.9.21` read the retired field as `false`, so an explicit `true` configuration can cause repeated plan differences or an inconsistent-result error after apply. Default `false` values sent by older providers remain accepted.
