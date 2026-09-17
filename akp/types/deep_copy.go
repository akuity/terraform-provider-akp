package types

import "reflect"

// DeepCopyArgoCD creates a deep copy of an ArgoCD struct, ensuring all pointer
// and slice fields are fully independent. This is critical for BuildStateFromAPI
// where we need a plan reference that won't be corrupted when the state struct
// is modified in place.
func DeepCopyArgoCD(src *ArgoCD) *ArgoCD {
	if src == nil {
		return nil
	}
	dst := new(ArgoCD)
	deepCopyValue(reflect.ValueOf(dst).Elem(), reflect.ValueOf(src).Elem())
	return dst
}

func DeepCopyClusterSpec(src *ClusterSpec) *ClusterSpec {
	if src == nil {
		return nil
	}
	dst := new(ClusterSpec)
	deepCopyValue(reflect.ValueOf(dst).Elem(), reflect.ValueOf(src).Elem())
	return dst
}

func DeepCopyKargo(src *Kargo) *Kargo {
	if src == nil {
		return nil
	}
	dst := new(Kargo)
	deepCopyValue(reflect.ValueOf(dst).Elem(), reflect.ValueOf(src).Elem())
	return dst
}

func DeepCopyKargoAgentSpec(src *KargoAgentSpec) *KargoAgentSpec {
	if src == nil {
		return nil
	}
	dst := new(KargoAgentSpec)
	deepCopyValue(reflect.ValueOf(dst).Elem(), reflect.ValueOf(src).Elem())
	return dst
}

// deepCopyValue recursively deep-copies src into dst.
// Both must be settable reflect.Values of the same type.
func deepCopyValue(dst, src reflect.Value) {
	switch src.Kind() {
	case reflect.Ptr:
		if src.IsNil() {
			dst.Set(reflect.Zero(src.Type()))
			return
		}
		newPtr := reflect.New(src.Type().Elem())
		deepCopyValue(newPtr.Elem(), src.Elem())
		dst.Set(newPtr)

	case reflect.Struct:
		srcType := src.Type()
		// Check if this struct needs deep copy: only if it has exported
		// fields of reference types (Ptr, Slice, Map, or nested Struct with same).
		// TF framework value types (e.g. basetypes.StringValue) have only
		// unexported fields — copy them wholesale.
		if !structNeedsDeepCopy(srcType) {
			dst.Set(src)
			return
		}
		// First, copy everything wholesale (handles unexported fields correctly),
		// then selectively deep-copy exported reference-type fields.
		dst.Set(src)
		for i := 0; i < src.NumField(); i++ {
			field := srcType.Field(i)
			if !field.IsExported() {
				continue
			}
			srcField := src.Field(i)
			dstField := dst.Field(i)
			switch srcField.Kind() {
			case reflect.Ptr, reflect.Slice, reflect.Map:
				deepCopyValue(dstField, srcField)
			case reflect.Struct:
				deepCopyValue(dstField, srcField)
			}
		}

	case reflect.Slice:
		if src.IsNil() {
			dst.Set(reflect.Zero(src.Type()))
			return
		}
		newSlice := reflect.MakeSlice(src.Type(), src.Len(), src.Len())
		for i := 0; i < src.Len(); i++ {
			deepCopyValue(newSlice.Index(i), src.Index(i))
		}
		dst.Set(newSlice)

	case reflect.Map:
		if src.IsNil() {
			dst.Set(reflect.Zero(src.Type()))
			return
		}
		newMap := reflect.MakeMap(src.Type())
		for _, key := range src.MapKeys() {
			newVal := reflect.New(src.Type().Elem()).Elem()
			deepCopyValue(newVal, src.MapIndex(key))
			newMap.SetMapIndex(key, newVal)
		}
		dst.Set(newMap)

	default:
		// Value types (string, bool, int, float, TF framework types like types.String, etc.)
		// are safe to copy directly — they don't contain shared mutable state.
		dst.Set(src)
	}
}

// structNeedsDeepCopy returns true if the struct type has any exported fields
// that could contain shared mutable references (Ptr, Slice, Map, or nested
// Struct that itself needs deep copy). Returns false for leaf value types
// like basetypes.StringValue which only have unexported fields.
func structNeedsDeepCopy(t reflect.Type) bool {
	for field := range t.Fields() {
		if !field.IsExported() {
			continue
		}
		switch field.Type.Kind() {
		case reflect.Ptr, reflect.Slice, reflect.Map:
			return true
		case reflect.Struct:
			if structNeedsDeepCopy(field.Type) {
				return true
			}
		}
	}
	return false
}
