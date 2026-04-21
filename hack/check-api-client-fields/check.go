// Package main implements a CI check that ensures every field defined in
// akp/apis/v1alpha1/*.go is also present in the corresponding struct in the
// pinned github.com/akuity/api-client-go module. This guards against silent
// drift where a field is added to the auto-translated Terraform types but
// cannot actually be sent to the API because the api-client-go version hasn't
// been bumped to include it.
package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// structFields maps a Go struct type name to the set of its field names.
// The field set uses *normalized* keys (see normalizeName) so that initialism
// differences like ClientID vs ClientId are treated as the same field.
// The value is the original (non-normalized) field name for reporting.
type structFields map[string]map[string]string

// allowlist describes intentional exceptions to the field-compatibility check.
type allowlist struct {
	// Fields maps "<StructName>.<FieldName>" -> reason. A field on this list
	// is allowed to exist in v1alpha1 without appearing in api-client-go.
	Fields map[string]string `yaml:"fields"`
	// Structs maps "<StructName>" -> reason. A struct on this list is allowed
	// to exist in v1alpha1 without appearing in api-client-go; no field-level
	// check is performed for it.
	Structs map[string]string `yaml:"structs"`
}

// missingStruct is reported when a v1alpha1 struct has no counterpart in
// api-client-go and is not allowlisted.
type missingStruct struct {
	Name string
}

// missingField is reported when a field in v1alpha1 has no counterpart in the
// same-named api-client-go struct and is not allowlisted.
type missingField struct {
	Struct string
	Field  string
}

// findings bundles all mismatches discovered by a single run.
type findings struct {
	Structs []missingStruct
	Fields  []missingField
	// UnusedAllowlist entries are entries in the allowlist that no longer
	// correspond to a real v1alpha1 struct or field. We surface them so the
	// allowlist doesn't rot.
	UnusedAllowlistStructs []string
	UnusedAllowlistFields  []string
}

// ok reports whether the findings represent a clean (passing) check.
func (f findings) ok() bool {
	return len(f.Structs) == 0 && len(f.Fields) == 0 &&
		len(f.UnusedAllowlistStructs) == 0 && len(f.UnusedAllowlistFields) == 0
}

// normalizeName lowercases and strips underscores so that initialism and
// snake_case / camelCase differences map to the same key. Examples:
//
//	ClientID       -> clientid
//	ClientId       -> clientid
//	client_id      -> clientid
//	IssuerURL      -> issuerurl
//	K8SNamespaces  -> k8snamespaces
func normalizeName(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "_", "")
	return s
}

// parseStructsFromDir walks dir, parses every .go file (skipping _test.go),
// and returns all top-level struct types declared in it. Embedded fields and
// fields without names (anonymous structs with no declared name) are skipped;
// those are never the kind of drift we're checking for.
func parseStructsFromDir(dir string) (structFields, error) {
	result := make(structFields)
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("walk %s: %w", path, err)
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		if err := parseStructsFromFile(path, result); err != nil {
			return fmt.Errorf("parse %s: %w", path, err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// parseStructsFromFiles parses a specific set of files (useful for api-client-go
// where we only care about 2 specific .pb.go files, not a whole directory).
func parseStructsFromFiles(paths []string) (structFields, error) {
	result := make(structFields)
	for _, path := range paths {
		if err := parseStructsFromFile(path, result); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
	}
	return result, nil
}

func parseStructsFromFile(path string, out structFields) error {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return err
	}
	ast.Inspect(node, func(n ast.Node) bool {
		typeSpec, ok := n.(*ast.TypeSpec)
		if !ok {
			return true
		}
		structType, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			return true
		}
		fields := make(map[string]string)
		for _, field := range structType.Fields.List {
			// Skip embedded fields (no names) and blank fields.
			if len(field.Names) == 0 {
				continue
			}
			for _, name := range field.Names {
				// Skip unexported fields — protobuf generates a few
				// unexported housekeeping fields like `state`, `sizeCache`.
				if !name.IsExported() {
					continue
				}
				fields[normalizeName(name.Name)] = name.Name
			}
		}
		if len(fields) > 0 {
			out[typeSpec.Name.Name] = fields
		}
		return true
	})
	return nil
}

// compare evaluates v1alpha1 against api-client-go under the given allowlist
// and returns the findings. The function is pure: no I/O, fully table-testable.
func compare(v1 structFields, client structFields, allow allowlist) findings {
	var f findings

	// Track which allowlist entries we actually use so we can flag stale ones.
	usedStructs := make(map[string]bool, len(allow.Structs))
	usedFields := make(map[string]bool, len(allow.Fields))

	// Walk structs in deterministic order for stable output.
	structNames := make([]string, 0, len(v1))
	for name := range v1 {
		structNames = append(structNames, name)
	}
	sort.Strings(structNames)

	for _, name := range structNames {
		clientFields, exists := client[name]
		if _, allowed := allow.Structs[name]; allowed {
			// Allowlist entry is "used" only if api-client-go actually lacks
			// the struct. If api-client-go has it, the entry is stale and we
			// should still perform field-level checks.
			if !exists {
				usedStructs[name] = true
				continue
			}
			// Fall through to field-level checks below.
		} else if !exists {
			f.Structs = append(f.Structs, missingStruct{Name: name})
			continue
		}

		// Walk fields in deterministic order.
		v1Fields := v1[name]
		fieldKeys := make([]string, 0, len(v1Fields))
		for k := range v1Fields {
			fieldKeys = append(fieldKeys, k)
		}
		sort.Strings(fieldKeys)

		for _, normalized := range fieldKeys {
			original := v1Fields[normalized]
			fieldKey := name + "." + original
			_, present := clientFields[normalized]
			if _, allowed := allow.Fields[fieldKey]; allowed {
				// A field allowlist is only "used" if api-client-go truly
				// lacks the field. If it's present, the entry is stale.
				if !present {
					usedFields[fieldKey] = true
				}
				continue
			}
			if !present {
				f.Fields = append(f.Fields, missingField{Struct: name, Field: original})
			}
		}
	}

	// Report unused allowlist entries. A stale allowlist is bad: it hides new
	// drift. These are warnings, not hard errors — but we still fail CI on
	// them so they get cleaned up promptly.
	for name := range allow.Structs {
		if !usedStructs[name] {
			// It's unused if either: the v1alpha1 struct no longer exists, OR
			// the v1alpha1 struct now has a matching api-client-go struct.
			if _, inV1 := v1[name]; !inV1 {
				f.UnusedAllowlistStructs = append(f.UnusedAllowlistStructs, name)
				continue
			}
			if _, inClient := client[name]; inClient {
				f.UnusedAllowlistStructs = append(f.UnusedAllowlistStructs, name)
			}
		}
	}
	for key := range allow.Fields {
		if usedFields[key] {
			continue
		}
		structName, fieldName, ok := strings.Cut(key, ".")
		if !ok {
			f.UnusedAllowlistFields = append(f.UnusedAllowlistFields, key)
			continue
		}
		v1Fields, inV1 := v1[structName]
		if !inV1 {
			f.UnusedAllowlistFields = append(f.UnusedAllowlistFields, key)
			continue
		}
		_, hasField := v1Fields[normalizeName(fieldName)]
		if !hasField {
			f.UnusedAllowlistFields = append(f.UnusedAllowlistFields, key)
			continue
		}
		// v1alpha1 struct and field both exist. The entry is unused if
		// api-client-go now has the field (so the allowlist is no longer
		// masking anything).
		if clientFields, inClient := client[structName]; inClient {
			if _, has := clientFields[normalizeName(fieldName)]; has {
				f.UnusedAllowlistFields = append(f.UnusedAllowlistFields, key)
			}
		}
	}
	sort.Strings(f.UnusedAllowlistStructs)
	sort.Strings(f.UnusedAllowlistFields)
	return f
}

// loadAllowlist reads an allowlist YAML file. A missing file returns an empty
// (but non-nil) allowlist; any other error is returned as-is.
func loadAllowlist(path string) (allowlist, error) {
	var a allowlist
	a.Fields = map[string]string{}
	a.Structs = map[string]string{}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return a, nil
		}
		return a, fmt.Errorf("read allowlist %s: %w", path, err)
	}
	var parsed allowlist
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		return a, fmt.Errorf("parse allowlist %s: %w", path, err)
	}
	if parsed.Fields != nil {
		a.Fields = parsed.Fields
	}
	if parsed.Structs != nil {
		a.Structs = parsed.Structs
	}
	return a, nil
}
