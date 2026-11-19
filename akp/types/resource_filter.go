package types

import (
	"slices"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/akuity/api-client-go/pkg/api/exportfilter"
)

// ResourceFilter is a data source's resource_filter: the objects to keep in argocd_resources or
// kargo_resources. A nil *ResourceFilter keeps every object.
type ResourceFilter struct {
	entries map[exportfilter.Entry]struct{}
}

func NewResourceFilter(entries []exportfilter.Entry) *ResourceFilter {
	f := &ResourceFilter{entries: make(map[exportfilter.Entry]struct{}, len(entries))}
	for _, e := range entries {
		f.entries[e] = struct{}{}
	}
	return f
}

// Refs returns the entries, sorted, as the export request sends them.
func (f *ResourceFilter) Refs() []string {
	refs := make([]string, 0, len(f.entries))
	for e := range f.entries {
		refs = append(refs, e.String())
	}
	slices.Sort(refs)
	return refs
}

// keeps matches obj on its real identity, not on its argocd_resources key, which drops the
// argocd namespace.
func (f *ResourceFilter) keeps(obj map[string]any) bool {
	if f == nil {
		return true
	}
	u := unstructured.Unstructured{Object: obj}
	_, ok := f.entries[exportfilter.Entry{
		Group:     u.GroupVersionKind().Group,
		Kind:      u.GetKind(),
		Namespace: u.GetNamespace(),
		Name:      u.GetName(),
	}]
	return ok
}
