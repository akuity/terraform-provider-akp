package types

import (
	"github.com/hashicorp/terraform-plugin-framework/attr"
)

type PlanModifierKind int

const (
	// PlanModNone indicates no plan modifier is set.
	PlanModNone PlanModifierKind = iota
	// PlanModUseStateForUnknown is the standard UseStateForUnknown modifier.
	PlanModUseStateForUnknown
	// PlanModUseStateForNullUnknown is the custom modifier that preserves state for null/unknown.
	PlanModUseStateForNullUnknown
	// PlanModIgnoreWhenNotConfigured preserves state when config is null.
	PlanModIgnoreWhenNotConfigured
)

var (
	RegisteredObjectAttrTypes = map[string]map[string]attr.Type{}
	RegisteredSensitiveFields = map[string]bool{}
	RegisteredComputedFields  = map[string]bool{}
	RegisteredPlanModifiers   = map[string]PlanModifierKind{}
)

func RegisterSchemaMetadata(
	objectAttrTypes map[string]map[string]attr.Type,
	sensitiveFields map[string]bool,
	computedFields map[string]bool,
	planModifiers map[string]PlanModifierKind,
) {
	for k, v := range objectAttrTypes {
		RegisteredObjectAttrTypes[k] = v
	}
	for k, v := range sensitiveFields {
		RegisteredSensitiveFields[k] = v
	}
	for k, v := range computedFields {
		RegisteredComputedFields[k] = v
	}
	for k, v := range planModifiers {
		RegisteredPlanModifiers[k] = v
	}
}
