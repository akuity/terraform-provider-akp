package akp

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	tftypes "github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/akuity/api-client-go/pkg/api/exportfilter"
	"github.com/akuity/terraform-provider-akp/akp/types"
)

var resourceFilterPath = path.Root("resource_filter")

// resourceFilterAttribute is the resource_filter input of a data source whose objects land in
// resourcesAttr. resourceType is the type syncResources gets ("ArgoCD" or "Kargo"), examples
// shows how to write entries, and fetched says what a platform that supports the filter stops
// listing.
func resourceFilterAttribute(resourceType, resourcesAttr, examples, fetched string) schema.SetAttribute {
	return schema.SetAttribute{
		MarkdownDescription: fmt.Sprintf("Limits `%[1]s` to these objects. Write each entry the way `%[1]s` keys an object, `<apiVersion>/<kind>/<namespace>/<name>` "+
			"with an empty namespace for cluster-scoped kinds, for example %[2]s. "+
			"Objects are matched on API group, kind, namespace and name; the version is not compared. "+
			"Omit the attribute to get every object; an empty set gives an empty map. Named objects that don't exist are left out without an error. "+
			"Platforms that support the filter read only the named objects instead of listing %[3]s; older platforms export everything and the provider drops the rest. "+
			"Admission policies are always exported in full and filtered by the provider. Other attributes are not filtered. "+
			"The entries must fit in 4 KiB once URL-encoded; split longer lists across several data sources.", resourcesAttr, examples, fetched),
		Optional:    true,
		ElementType: tftypes.StringType,
		Validators:  []validator.Set{resourceFilterValidator{resourceType: resourceType}},
	}
}

// kubeVersion matches Kubernetes API versions such as v1, v1alpha1 and v2beta3. Requiring one
// rejects entries written without a version, whose first part would otherwise pass for one.
var kubeVersion = regexp.MustCompile(`^v[0-9]+((alpha|beta)[0-9]+)?$`)

// parseResourceKey parses a resource_filter entry, written like an argocd_resources or
// kargo_resources key: "<apiVersion>/<kind>/<namespace>/<name>".
func parseResourceKey(resourceType, key string) (exportfilter.Entry, error) {
	invalid := fmt.Errorf("%q is not <apiVersion>/<kind>/<namespace>/<name>", key)
	parts := strings.Split(key, "/")
	var group string
	switch {
	case len(parts) == 5 && parts[0] != "":
		group, parts = parts[0], parts[1:]
	case len(parts) != 4: // a core-group apiVersion is just the version
		return exportfilter.Entry{}, invalid
	}
	if !kubeVersion.MatchString(parts[0]) {
		return exportfilter.Entry{}, invalid
	}
	kind, name := parts[1], parts[3]
	namespace := types.DataSourceKeyNamespace(resourceType, kind, parts[2])
	return exportfilter.Parse(exportfilter.Entry{Group: group, Kind: kind, Namespace: namespace, Name: name}.String())
}

// parseResourceFilter reads a resource_filter value. A null value gives a nil filter, which
// keeps every object. unknown reports an unknown set or entry, which must never be read as an
// empty filter.
func parseResourceFilter(resourceType string, value tftypes.Set) (filter *types.ResourceFilter, unknown bool, diags diag.Diagnostics) {
	if value.IsUnknown() {
		return nil, true, nil
	}
	if value.IsNull() {
		return nil, false, nil
	}
	var entries []exportfilter.Entry
	for _, elem := range value.Elements() {
		s, ok := elem.(tftypes.String)
		if !ok || s.IsNull() {
			diags.AddAttributeError(resourceFilterPath, "Invalid resource_filter entry", "Entries must not be null.")
			continue
		}
		if s.IsUnknown() {
			unknown = true
			continue
		}
		e, err := parseResourceKey(resourceType, s.ValueString())
		if err != nil {
			diags.AddAttributeError(resourceFilterPath.AtSetValue(s), "Invalid resource_filter entry", err.Error())
			continue
		}
		entries = append(entries, e)
	}
	if unknown || diags.HasError() {
		return nil, unknown, diags
	}
	filter = types.NewResourceFilter(entries)
	if size := exportFilterQueryBytes(filter.Refs()); size > maxExportFilterQueryBytes {
		diags.AddAttributeError(resourceFilterPath, "resource_filter is too long",
			fmt.Sprintf("The entries take %d bytes in the export request URL, over the %d-byte limit. Split them across several data sources.", size, maxExportFilterQueryBytes))
		return nil, false, diags
	}
	return filter, false, nil
}

// readResourceFilter parses resource_filter for a data source read. Terraform defers a read
// until its configuration is known, so an unknown value here is an error.
func readResourceFilter(resourceType string, value tftypes.Set, diags *diag.Diagnostics) *types.ResourceFilter {
	filter, unknown, parseDiags := parseResourceFilter(resourceType, value)
	diags.Append(parseDiags...)
	if unknown {
		diags.AddAttributeError(resourceFilterPath, "resource_filter is not known", "The data source can't be read until every resource_filter entry is known.")
	}
	return filter
}

// resourceFilterValidator reports bad entries during validation. Unknown entries are checked
// when the data source is read.
type resourceFilterValidator struct {
	resourceType string
}

func (resourceFilterValidator) Description(context.Context) string {
	return "entries are <apiVersion>/<kind>/<namespace>/<name> and fit in 4 KiB once URL-encoded"
}

func (v resourceFilterValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v resourceFilterValidator) ValidateSet(_ context.Context, req validator.SetRequest, resp *validator.SetResponse) {
	_, _, diags := parseResourceFilter(v.resourceType, req.ConfigValue)
	resp.Diagnostics.Append(diags...)
}
