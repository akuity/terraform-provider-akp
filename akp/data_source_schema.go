package akp

import (
	"fmt"
	"slices"

	ds "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	rs "github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// toDataSourceAttributes derives a data-source schema from a resource schema:
// every attribute becomes Computed except the lookup keys named in required.
func toDataSourceAttributes(attrs map[string]rs.Attribute, required ...string) map[string]ds.Attribute {
	out := make(map[string]ds.Attribute, len(attrs))
	for name, a := range attrs {
		out[name] = toDataSourceAttribute(a, slices.Contains(required, name))
	}
	return out
}

func toDataSourceAttribute(a rs.Attribute, required bool) ds.Attribute {
	switch t := a.(type) {
	case rs.StringAttribute:
		return ds.StringAttribute{Required: required, Computed: !required, Sensitive: t.Sensitive, Description: t.Description, MarkdownDescription: t.MarkdownDescription, DeprecationMessage: t.DeprecationMessage}
	case rs.BoolAttribute:
		return ds.BoolAttribute{Computed: true, Sensitive: t.Sensitive, Description: t.Description, MarkdownDescription: t.MarkdownDescription, DeprecationMessage: t.DeprecationMessage}
	case rs.Int64Attribute:
		return ds.Int64Attribute{Computed: true, Sensitive: t.Sensitive, Description: t.Description, MarkdownDescription: t.MarkdownDescription, DeprecationMessage: t.DeprecationMessage}
	case rs.ListAttribute:
		return ds.ListAttribute{ElementType: t.ElementType, Computed: true, Sensitive: t.Sensitive, Description: t.Description, MarkdownDescription: t.MarkdownDescription, DeprecationMessage: t.DeprecationMessage}
	case rs.SetAttribute:
		return ds.SetAttribute{ElementType: t.ElementType, Computed: true, Sensitive: t.Sensitive, Description: t.Description, MarkdownDescription: t.MarkdownDescription, DeprecationMessage: t.DeprecationMessage}
	case rs.MapAttribute:
		return ds.MapAttribute{ElementType: t.ElementType, Computed: true, Sensitive: t.Sensitive, Description: t.Description, MarkdownDescription: t.MarkdownDescription, DeprecationMessage: t.DeprecationMessage}
	case rs.SingleNestedAttribute:
		return ds.SingleNestedAttribute{Attributes: toDataSourceAttributes(t.Attributes), Computed: true, Sensitive: t.Sensitive, Description: t.Description, MarkdownDescription: t.MarkdownDescription, DeprecationMessage: t.DeprecationMessage}
	case rs.ListNestedAttribute:
		return ds.ListNestedAttribute{NestedObject: ds.NestedAttributeObject{Attributes: toDataSourceAttributes(t.NestedObject.Attributes)}, Computed: true, Sensitive: t.Sensitive, Description: t.Description, MarkdownDescription: t.MarkdownDescription, DeprecationMessage: t.DeprecationMessage}
	case rs.MapNestedAttribute:
		return ds.MapNestedAttribute{NestedObject: ds.NestedAttributeObject{Attributes: toDataSourceAttributes(t.NestedObject.Attributes)}, Computed: true, Sensitive: t.Sensitive, Description: t.Description, MarkdownDescription: t.MarkdownDescription, DeprecationMessage: t.DeprecationMessage}
	}
	panic(fmt.Sprintf("toDataSourceAttribute: unsupported attribute type %T", a))
}
