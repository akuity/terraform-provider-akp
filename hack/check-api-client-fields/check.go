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
	"maps"
	"os"
	"path/filepath"
	"slices"
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

	// Sort names so diagnostics remain deterministic.
	for _, name := range slices.Sorted(maps.Keys(v1)) {
		clientFields, exists := client[name]
		if !exists {
			if _, allowed := allow.Structs[name]; !allowed {
				f.Structs = append(f.Structs, missingStruct{Name: name})
			}
			continue
		}
		for _, normalized := range slices.Sorted(maps.Keys(v1[name])) {
			original := v1[name][normalized]
			_, present := clientFields[normalized]
			_, allowed := allow.Fields[name+"."+original]
			if !present && !allowed {
				f.Fields = append(f.Fields, missingField{Struct: name, Field: original})
			}
		}
	}

	// An exception is stale when its source disappears or the client catches up.
	for name := range allow.Structs {
		_, inV1 := v1[name]
		_, inClient := client[name]
		if !inV1 || inClient {
			f.UnusedAllowlistStructs = append(f.UnusedAllowlistStructs, name)
		}
	}
	for key := range allow.Fields {
		structName, fieldName, ok := strings.Cut(key, ".")
		normalized := normalizeName(fieldName)
		_, inV1 := v1[structName][normalized]
		_, inClient := client[structName][normalized]
		if !ok || !inV1 || inClient {
			f.UnusedAllowlistFields = append(f.UnusedAllowlistFields, key)
		}
	}

	sort.Strings(f.UnusedAllowlistStructs)
	sort.Strings(f.UnusedAllowlistFields)
	return f
}

// loadAllowlist reads an allowlist YAML file. A missing file returns an empty
// allowlist; other read and parse errors are returned.
func loadAllowlist(path string) (allowlist, error) {
	var a allowlist
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return a, nil
		}
		return a, fmt.Errorf("read allowlist %s: %w", path, err)
	}
	if err := yaml.Unmarshal(data, &a); err != nil {
		return a, fmt.Errorf("parse allowlist %s: %w", path, err)
	}
	return a, nil
}
