package akp

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"

	listplanmodifier2 "github.com/akuity/terraform-provider-akp/akp/modifiers/list"
	mapplanmodifier2 "github.com/akuity/terraform-provider-akp/akp/modifiers/map"
	objectplanmodifier2 "github.com/akuity/terraform-provider-akp/akp/modifiers/object"
	tfakptypes "github.com/akuity/terraform-provider-akp/akp/types"
)

// SchemaMetadata holds extracted metadata from a TF schema tree.
type SchemaMetadata struct {
	// ObjectAttrTypes maps dot-separated tfsdk paths to their attr.Type maps
	// for SingleNestedAttribute fields.
	ObjectAttrTypes map[string]map[string]attr.Type
	SensitiveFields map[string]bool
	ComputedFields  map[string]bool
	PlanModifiers   map[string]tfakptypes.PlanModifierKind
}

func buildSchemaMetadata(attrs map[string]schema.Attribute) *SchemaMetadata {
	meta := &SchemaMetadata{
		ObjectAttrTypes: make(map[string]map[string]attr.Type),
		SensitiveFields: make(map[string]bool),
		ComputedFields:  make(map[string]bool),
		PlanModifiers:   make(map[string]tfakptypes.PlanModifierKind),
	}
	walkAttributes("", attrs, meta)
	return meta
}

func walkAttributes(prefix string, attrs map[string]schema.Attribute, meta *SchemaMetadata) {
	for name, a := range attrs {
		path := name
		if prefix != "" {
			path = prefix + "." + name
		}
		if a.IsComputed() {
			meta.ComputedFields[path] = true
		}
		if a.IsSensitive() {
			meta.SensitiveFields[path] = true
		}
		if mod := detectPlanModifier(a); mod != tfakptypes.PlanModNone {
			meta.PlanModifiers[path] = mod
		}
		if nested, ok := a.(schema.SingleNestedAttribute); ok {
			childAttrs := nested.Attributes
			if len(childAttrs) > 0 {
				meta.ObjectAttrTypes[path] = schemaAttrsToAttrTypes(childAttrs)
				walkAttributes(path, childAttrs, meta)
			}
		}
		if nested, ok := a.(schema.ListNestedAttribute); ok {
			childAttrs := nested.NestedObject.Attributes
			if len(childAttrs) > 0 {
				walkAttributes(path, childAttrs, meta)
			}
		}
		if nested, ok := a.(schema.MapNestedAttribute); ok {
			childAttrs := nested.NestedObject.Attributes
			if len(childAttrs) > 0 {
				meta.ObjectAttrTypes[path] = schemaAttrsToAttrTypes(childAttrs)
				walkAttributes(path, childAttrs, meta)
			}
		}
	}
}

func schemaAttrsToAttrTypes(attrs map[string]schema.Attribute) map[string]attr.Type {
	result := make(map[string]attr.Type, len(attrs))
	for name, a := range attrs {
		result[name] = a.GetType()
	}
	return result
}

func detectPlanModifier(a schema.Attribute) tfakptypes.PlanModifierKind {
	switch t := a.(type) {
	case schema.ListNestedAttribute:
		return detectListPlanModifier(t.PlanModifiers)
	case schema.ListAttribute:
		return detectListPlanModifier(t.PlanModifiers)
	case schema.StringAttribute:
		for _, m := range t.PlanModifiers {
			if isUseStateForUnknownDesc(m.Description(context.TODO())) {
				return tfakptypes.PlanModUseStateForUnknown
			}
		}
	case schema.BoolAttribute:
		for _, m := range t.PlanModifiers {
			if isUseStateForUnknownDesc(m.Description(context.TODO())) {
				return tfakptypes.PlanModUseStateForUnknown
			}
		}
	case schema.Int64Attribute:
		for _, m := range t.PlanModifiers {
			if isUseStateForUnknownDesc(m.Description(context.TODO())) {
				return tfakptypes.PlanModUseStateForUnknown
			}
		}
	case schema.Float64Attribute:
		for _, m := range t.PlanModifiers {
			if isUseStateForUnknownDesc(m.Description(context.TODO())) {
				return tfakptypes.PlanModUseStateForUnknown
			}
		}
	case schema.MapAttribute:
		return detectMapPlanModifier(t.PlanModifiers)
	case schema.SingleNestedAttribute:
		for _, m := range t.PlanModifiers {
			if _, ok := m.(objectplanmodifier2.UseStateForNullUnknownModifier); ok {
				return tfakptypes.PlanModUseStateForNullUnknown
			}
			if isUseStateForUnknownDesc(m.Description(context.TODO())) {
				return tfakptypes.PlanModUseStateForUnknown
			}
		}
	}
	return tfakptypes.PlanModNone
}

func detectListPlanModifier(modifiers []planmodifier.List) tfakptypes.PlanModifierKind {
	for _, m := range modifiers {
		if _, ok := m.(listplanmodifier2.IgnoreWhenNotConfiguredModifier); ok {
			return tfakptypes.PlanModIgnoreWhenNotConfigured
		}
		if isUseStateForUnknownDesc(m.Description(context.TODO())) {
			return tfakptypes.PlanModUseStateForUnknown
		}
	}
	return tfakptypes.PlanModNone
}

func detectMapPlanModifier(modifiers []planmodifier.Map) tfakptypes.PlanModifierKind {
	for _, m := range modifiers {
		if _, ok := m.(mapplanmodifier2.UseStateForNullUnknownModifier); ok {
			return tfakptypes.PlanModUseStateForNullUnknown
		}
		if isUseStateForUnknownDesc(m.Description(context.TODO())) {
			return tfakptypes.PlanModUseStateForUnknown
		}
	}
	return tfakptypes.PlanModNone
}

func isUseStateForUnknownDesc(desc string) bool {
	return desc == "Once set, the value of this attribute in state will not change."
}

func init() {
	for _, attrs := range []map[string]schema.Attribute{
		getAKPInstanceAttributes(),
		getAKPClusterAttributes(),
		getAKPKargoAgentResourceAttributes(),
		getAKPKargoInstanceAttributes(),
	} {
		meta := buildSchemaMetadata(attrs)
		tfakptypes.RegisterSchemaMetadata(
			meta.ObjectAttrTypes,
			meta.SensitiveFields,
			meta.ComputedFields,
			meta.PlanModifiers,
		)
	}
}
