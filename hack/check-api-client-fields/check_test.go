package main

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

func TestNormalizeName(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"ClientID", "clientid"},
		{"ClientId", "clientid"},
		{"client_id", "clientid"},
		{"IssuerURL", "issuerurl"},
		{"IssuerUrl", "issuerurl"},
		{"K8SNamespaces", "k8snamespaces"},
		{"k8s_namespaces", "k8snamespaces"},
		{"Fqdn", "fqdn"},
		{"", ""},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got := normalizeName(tc.in)
			if got != tc.want {
				t.Errorf("normalizeName(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestParseStructsFromDir(t *testing.T) {
	got, err := parseStructsFromDir("testdata/v1alpha1")
	if err != nil {
		t.Fatalf("parseStructsFromDir: %v", err)
	}
	// Foo has only "Spec" as a named field (TypeMeta/ObjectMeta are embedded).
	if _, ok := got["Foo"]; !ok {
		t.Fatalf("expected struct Foo to be parsed; got %v", structKeys(got))
	}
	if _, ok := got["Foo"]["spec"]; !ok {
		t.Errorf("expected Foo.spec; got %v", got["Foo"])
	}
	// FooSpec should have all its named fields, without the embedded metav1 types.
	fooSpec, ok := got["FooSpec"]
	if !ok {
		t.Fatalf("expected FooSpec to be parsed; got %v", structKeys(got))
	}
	for _, want := range []string{"name", "clientid", "issuerurl", "tfonlyfield", "blob", "enabled"} {
		if _, ok := fooSpec[want]; !ok {
			t.Errorf("FooSpec missing normalized field %q; have %v", want, fooSpec)
		}
	}
	// TerraformOnlyStruct should also be present.
	if _, ok := got["TerraformOnlyStruct"]; !ok {
		t.Errorf("expected TerraformOnlyStruct to be parsed; got %v", structKeys(got))
	}
}

func TestParseStructsFromFiles(t *testing.T) {
	// Pass the fixture files directly; parseStructsFromFiles is used by the
	// tool to parse a curated list of api-client-go .pb.go files rather than
	// a whole directory.
	got, err := parseStructsFromFiles([]string{
		"testdata/apiclient/types.go",
	})
	if err != nil {
		t.Fatalf("parseStructsFromFiles: %v", err)
	}
	if _, ok := got["FooSpec"]; !ok {
		t.Fatalf("expected FooSpec; got %v", structKeys(got))
	}
}

func TestParseStructsFromFilesBadPath(t *testing.T) {
	_, err := parseStructsFromFiles([]string{"testdata/does-not-exist.go"})
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestParseStructsIgnoresUnexportedAndEmbedded(t *testing.T) {
	got, err := parseStructsFromDir("testdata/apiclient")
	if err != nil {
		t.Fatalf("parseStructsFromDir: %v", err)
	}
	fooSpec, ok := got["FooSpec"]
	if !ok {
		t.Fatalf("expected FooSpec to be parsed; got %v", structKeys(got))
	}
	// The unexported protobuf housekeeping fields must not leak in.
	for _, bad := range []string{"state", "sizecache", "unknownfields"} {
		if _, present := fooSpec[bad]; present {
			t.Errorf("unexported field %q should be ignored; got %v", bad, fooSpec)
		}
	}
	// Normalized client fields should match regardless of initialism differences.
	for _, want := range []string{"name", "clientid", "issuerurl", "blob", "enabled"} {
		if _, ok := fooSpec[want]; !ok {
			t.Errorf("expected normalized field %q; got %v", want, fooSpec)
		}
	}
}

func TestCompareAllMatch(t *testing.T) {
	v1 := structFields{
		"FooSpec": {"name": "Name", "clientid": "ClientID"},
	}
	client := structFields{
		"FooSpec": {"name": "Name", "clientid": "ClientId"},
	}
	got := compare(v1, client, allowlist{})
	if !got.ok() {
		t.Errorf("expected clean findings; got %+v", got)
	}
}

func TestCompareMissingField(t *testing.T) {
	v1 := structFields{
		"FooSpec": {"name": "Name", "newfield": "NewField"},
	}
	client := structFields{
		"FooSpec": {"name": "Name"},
	}
	got := compare(v1, client, allowlist{})
	want := []missingField{{Struct: "FooSpec", Field: "NewField"}}
	if !reflect.DeepEqual(got.Fields, want) {
		t.Errorf("fields: got %+v, want %+v", got.Fields, want)
	}
	if len(got.Structs) != 0 {
		t.Errorf("unexpected missing structs: %+v", got.Structs)
	}
}

func TestCompareMissingStruct(t *testing.T) {
	v1 := structFields{
		"OnlyHere": {"x": "X"},
	}
	client := structFields{}
	got := compare(v1, client, allowlist{})
	want := []missingStruct{{Name: "OnlyHere"}}
	if !reflect.DeepEqual(got.Structs, want) {
		t.Errorf("structs: got %+v, want %+v", got.Structs, want)
	}
}

func TestCompareAllowlistedField(t *testing.T) {
	v1 := structFields{
		"FooSpec": {"name": "Name", "tfonlyfield": "TfOnlyField"},
	}
	client := structFields{
		"FooSpec": {"name": "Name"},
	}
	allow := allowlist{
		Fields: map[string]string{"FooSpec.TfOnlyField": "terraform-only"},
	}
	got := compare(v1, client, allow)
	if !got.ok() {
		t.Errorf("expected clean findings with allowlist; got %+v", got)
	}
}

func TestCompareAllowlistedStruct(t *testing.T) {
	v1 := structFields{
		"TerraformOnlyStruct": {"only": "Only"},
	}
	client := structFields{}
	allow := allowlist{
		Structs: map[string]string{"TerraformOnlyStruct": "terraform-only wrapper"},
	}
	got := compare(v1, client, allow)
	if !got.ok() {
		t.Errorf("expected clean findings with struct allowlist; got %+v", got)
	}
}

func TestCompareUnusedAllowlistEntries(t *testing.T) {
	v1 := structFields{
		"FooSpec": {"name": "Name"},
	}
	client := structFields{
		"FooSpec": {"name": "Name"},
	}
	allow := allowlist{
		Fields: map[string]string{
			"FooSpec.Name":       "stale: api-client-go now has this field",
			"FooSpec.GoneField":  "stale: v1alpha1 no longer has this field",
			"OldStruct.X":        "stale: v1alpha1 no longer has OldStruct",
		},
		Structs: map[string]string{
			"FooSpec":       "stale: api-client-go now has matching struct",
			"NeverExisted":  "stale: never in v1alpha1",
		},
	}
	got := compare(v1, client, allow)

	wantStructs := []string{"FooSpec", "NeverExisted"}
	sort.Strings(got.UnusedAllowlistStructs)
	if !reflect.DeepEqual(got.UnusedAllowlistStructs, wantStructs) {
		t.Errorf("unused structs: got %+v, want %+v", got.UnusedAllowlistStructs, wantStructs)
	}
	wantFields := []string{"FooSpec.GoneField", "FooSpec.Name", "OldStruct.X"}
	sort.Strings(got.UnusedAllowlistFields)
	if !reflect.DeepEqual(got.UnusedAllowlistFields, wantFields) {
		t.Errorf("unused fields: got %+v, want %+v", got.UnusedAllowlistFields, wantFields)
	}
}

func TestCompareFixtureEndToEnd(t *testing.T) {
	v1, err := parseStructsFromDir("testdata/v1alpha1")
	if err != nil {
		t.Fatalf("parse v1alpha1: %v", err)
	}
	client, err := parseStructsFromDir("testdata/apiclient")
	if err != nil {
		t.Fatalf("parse apiclient: %v", err)
	}

	// Without any allowlist, we expect:
	//   - Foo struct missing in apiclient
	//   - TerraformOnlyStruct missing in apiclient
	//   - FooSpec.TfOnlyField missing
	got := compare(v1, client, allowlist{})
	wantStructs := []string{"Foo", "TerraformOnlyStruct"}
	gotStructs := make([]string, 0, len(got.Structs))
	for _, s := range got.Structs {
		gotStructs = append(gotStructs, s.Name)
	}
	sort.Strings(gotStructs)
	if !reflect.DeepEqual(gotStructs, wantStructs) {
		t.Errorf("missing structs: got %v, want %v", gotStructs, wantStructs)
	}
	wantFields := []string{"FooSpec.TfOnlyField"}
	gotFields := make([]string, 0, len(got.Fields))
	for _, f := range got.Fields {
		gotFields = append(gotFields, f.Struct+"."+f.Field)
	}
	sort.Strings(gotFields)
	if !reflect.DeepEqual(gotFields, wantFields) {
		t.Errorf("missing fields: got %v, want %v", gotFields, wantFields)
	}

	// With an allowlist covering each, the check should pass.
	allow := allowlist{
		Structs: map[string]string{
			"Foo":                 "k8s wrapper type",
			"TerraformOnlyStruct": "terraform-only",
		},
		Fields: map[string]string{
			"FooSpec.TfOnlyField": "terraform-only",
		},
	}
	if clean := compare(v1, client, allow); !clean.ok() {
		t.Errorf("expected clean with full allowlist; got %+v", clean)
	}
}

func TestLoadAllowlistMissingFile(t *testing.T) {
	a, err := loadAllowlist(filepath.Join(t.TempDir(), "does-not-exist.yaml"))
	if err != nil {
		t.Fatalf("expected nil error for missing file; got %v", err)
	}
	if len(a.Fields) != 0 || len(a.Structs) != 0 {
		t.Errorf("expected empty allowlist for missing file; got %+v", a)
	}
}

func TestLoadAllowlistRoundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "allowlist.yaml")
	content := []byte(`fields:
  FooSpec.TfOnlyField: "terraform-only"
structs:
  TerraformOnlyStruct: "terraform-only wrapper"
`)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	a, err := loadAllowlist(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if a.Fields["FooSpec.TfOnlyField"] != "terraform-only" {
		t.Errorf("fields: got %+v", a.Fields)
	}
	if a.Structs["TerraformOnlyStruct"] != "terraform-only wrapper" {
		t.Errorf("structs: got %+v", a.Structs)
	}
}

func structKeys(s structFields) []string {
	keys := make([]string, 0, len(s))
	for k := range s {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
