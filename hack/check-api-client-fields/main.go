package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// CLI flags. All have sane defaults relative to the project root, so the
// common local invocation is just `go run ./hack/check-api-client-fields`.
type options struct {
	v1alpha1Dir   string
	apiClientPkgs stringList
	allowlistPath string
	modulePath    string
	projectRoot   string
}

// stringList is a repeatable string flag (e.g. -pkg a -pkg b).
type stringList []string

func (s *stringList) String() string     { return strings.Join(*s, ",") }
func (s *stringList) Set(v string) error { *s = append(*s, v); return nil }

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "check-api-client-fields: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string, stdout *os.File) error {
	opts, err := parseArgs(args)
	if err != nil {
		return err
	}

	// Resolve project root so every path below is absolute and errors are
	// easy to copy-paste.
	root, err := resolveProjectRoot(opts.projectRoot)
	if err != nil {
		return fmt.Errorf("resolve project root: %w", err)
	}

	v1Dir := opts.v1alpha1Dir
	if !filepath.IsAbs(v1Dir) {
		v1Dir = filepath.Join(root, v1Dir)
	}
	allowPath := opts.allowlistPath
	if !filepath.IsAbs(allowPath) {
		allowPath = filepath.Join(root, allowPath)
	}

	// Resolve the on-disk location of the pinned api-client-go module using
	// `go list -m`. This avoids hard-coding $GOMODCACHE semantics and works
	// identically locally and in CI.
	apiClientDir, apiClientVersion, err := resolveModuleDir(root, opts.modulePath)
	if err != nil {
		return fmt.Errorf("locate module %s: %w", opts.modulePath, err)
	}

	fmt.Fprintf(stdout, "check-api-client-fields\n")
	fmt.Fprintf(stdout, "  project root      : %s\n", root)
	fmt.Fprintf(stdout, "  v1alpha1 dir      : %s\n", v1Dir)
	fmt.Fprintf(stdout, "  api-client-go     : %s@%s\n", opts.modulePath, apiClientVersion)
	fmt.Fprintf(stdout, "  api-client-go dir : %s\n", apiClientDir)
	fmt.Fprintf(stdout, "  allowlist         : %s\n", allowPath)
	fmt.Fprintln(stdout)

	// Parse v1alpha1.
	v1Structs, err := parseStructsFromDir(v1Dir)
	if err != nil {
		return fmt.Errorf("parse v1alpha1: %w", err)
	}
	fmt.Fprintf(stdout, "parsed %d structs from %s\n", len(v1Structs), relOrAbs(root, v1Dir))

	// Parse api-client-go. Resolve any -pkg entries against the module root.
	apiFiles, err := resolveApiClientFiles(apiClientDir, opts.apiClientPkgs)
	if err != nil {
		return fmt.Errorf("resolve api-client-go files: %w", err)
	}
	apiStructs, err := parseStructsFromFiles(apiFiles)
	if err != nil {
		return fmt.Errorf("parse api-client-go: %w", err)
	}
	fmt.Fprintf(stdout, "parsed %d structs from %d api-client-go files\n", len(apiStructs), len(apiFiles))
	for _, f := range apiFiles {
		fmt.Fprintf(stdout, "    %s\n", relOrAbs(apiClientDir, f))
	}

	// Load allowlist.
	allow, err := loadAllowlist(allowPath)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "allowlist: %d field entries, %d struct entries\n\n",
		len(allow.Fields), len(allow.Structs))

	// Run the check.
	result := compare(v1Structs, apiStructs, allow)
	printReport(stdout, result, allow)

	if !result.ok() {
		return fmt.Errorf("check failed: %d missing struct(s), %d missing field(s), %d stale allowlist entry/entries",
			len(result.Structs), len(result.Fields),
			len(result.UnusedAllowlistStructs)+len(result.UnusedAllowlistFields))
	}
	fmt.Fprintln(stdout, "OK: all v1alpha1 fields are present in api-client-go (or allowlisted)")
	return nil
}

func parseArgs(args []string) (options, error) {
	var opts options
	fs := flag.NewFlagSet("check-api-client-fields", flag.ContinueOnError)
	fs.StringVar(&opts.v1alpha1Dir, "v1alpha1", "akp/apis/v1alpha1",
		"directory containing v1alpha1 Go files (relative to project root or absolute)")
	fs.StringVar(&opts.allowlistPath, "allowlist", "hack/check-api-client-fields/allowlist.yaml",
		"path to the allowlist YAML file")
	fs.StringVar(&opts.modulePath, "module", "github.com/akuity/api-client-go",
		"Go module path of the API client")
	fs.StringVar(&opts.projectRoot, "project-root", "",
		"project root (defaults to the closest ancestor containing go.mod)")
	fs.Var(&opts.apiClientPkgs, "pkg",
		"api-client-go subpackage containing protobuf Go types; may be repeated. "+
			"Default: pkg/api/gen/argocd/v1 and pkg/api/gen/kargo/v1")
	if err := fs.Parse(args); err != nil {
		return opts, err
	}
	if len(opts.apiClientPkgs) == 0 {
		opts.apiClientPkgs = []string{"pkg/api/gen/argocd/v1", "pkg/api/gen/kargo/v1"}
	}
	return opts, nil
}

// resolveProjectRoot walks up from cwd (or the override) until it finds go.mod.
func resolveProjectRoot(override string) (string, error) {
	start := override
	if start == "" {
		var err error
		start, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}
	start, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	dir := start
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no go.mod found at or above %s", start)
		}
		dir = parent
	}
}

// resolveModuleDir invokes `go list -m -f {{.Dir}}|{{.Version}} <module>` from
// the project root and returns the module's on-disk directory plus version.
// Using `go list` means we respect replace directives and local GOMODCACHE.
func resolveModuleDir(projectRoot, module string) (dir, version string, err error) {
	cmd := exec.Command("go", "list", "-m", "-f", "{{.Dir}}|{{.Version}}", module)
	cmd.Dir = projectRoot
	var out, errOut bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	if err := cmd.Run(); err != nil {
		return "", "", fmt.Errorf("go list -m %s: %w (stderr: %s)", module, err, strings.TrimSpace(errOut.String()))
	}
	line := strings.TrimSpace(out.String())
	parts := strings.SplitN(line, "|", 2)
	if len(parts) != 2 || parts[0] == "" {
		return "", "", fmt.Errorf("unexpected output from go list: %q", line)
	}
	return parts[0], parts[1], nil
}

// resolveApiClientFiles collects all non-test .go files under each given
// package directory (relative to moduleDir). We intentionally use a flat file
// list rather than parsing by package so that generated "_grpc.pb.go" files
// don't add noise (they define transport-layer types like *Client interfaces,
// not schema types).
func resolveApiClientFiles(moduleDir string, packages []string) ([]string, error) {
	var files []string
	for _, pkg := range packages {
		pkgDir := filepath.Join(moduleDir, pkg)
		entries, err := os.ReadDir(pkgDir)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", pkgDir, err)
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			if !strings.HasSuffix(name, ".pb.go") {
				continue
			}
			// Skip grpc and gateway scaffolding: they don't define the
			// message types we want to compare against.
			if strings.HasSuffix(name, "_grpc.pb.go") || strings.HasSuffix(name, ".pb.gw.go") {
				continue
			}
			files = append(files, filepath.Join(pkgDir, name))
		}
	}
	sort.Strings(files)
	if len(files) == 0 {
		return nil, fmt.Errorf("no .pb.go files found in packages %v under %s", packages, moduleDir)
	}
	return files, nil
}

// printReport emits a human-readable, actionable report of all findings.
// Format is designed to be easy to read in CI logs: one line per issue, with
// a short header per category and a hint at the bottom.
func printReport(w *os.File, f findings, allow allowlist) {
	if f.ok() {
		return
	}
	fmt.Fprintln(w, "==========================================================")
	fmt.Fprintln(w, "check-api-client-fields: FAILED")
	fmt.Fprintln(w, "==========================================================")

	if len(f.Structs) > 0 {
		fmt.Fprintf(w, "\nStructs in v1alpha1 with no matching type in api-client-go (%d):\n", len(f.Structs))
		for _, s := range f.Structs {
			fmt.Fprintf(w, "  - %s\n", s.Name)
		}
		fmt.Fprintln(w, "\n  If this struct is intentionally terraform-only, add it to")
		fmt.Fprintln(w, "  allowlist.yaml under `structs:` with a reason. Otherwise, bump")
		fmt.Fprintln(w, "  the github.com/akuity/api-client-go version so the new type is")
		fmt.Fprintln(w, "  available before merging the v1alpha1 change.")
	}

	if len(f.Fields) > 0 {
		// Group missing fields by struct for readability.
		byStruct := map[string][]string{}
		for _, m := range f.Fields {
			byStruct[m.Struct] = append(byStruct[m.Struct], m.Field)
		}
		keys := make([]string, 0, len(byStruct))
		for k := range byStruct {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		total := len(f.Fields)
		fmt.Fprintf(w, "\nFields in v1alpha1 not present in api-client-go (%d field(s) across %d struct(s)):\n",
			total, len(keys))
		for _, name := range keys {
			fields := byStruct[name]
			sort.Strings(fields)
			fmt.Fprintf(w, "  %s:\n", name)
			for _, field := range fields {
				fmt.Fprintf(w, "    - %s.%s\n", name, field)
			}
		}
		fmt.Fprintln(w, "\n  If any of these fields are intentionally terraform-only (for")
		fmt.Fprintln(w, "  example an HCL-level workspace selector), add them to")
		fmt.Fprintln(w, "  allowlist.yaml under `fields:` with a one-line reason, e.g.")
		fmt.Fprintln(w, "    fields:")
		fmt.Fprintln(w, "      StructName.FieldName: \"reason this field is terraform-only\"")
		fmt.Fprintln(w, "  Otherwise, bump the api-client-go dependency before merging.")
	}

	if len(f.UnusedAllowlistStructs) > 0 || len(f.UnusedAllowlistFields) > 0 {
		fmt.Fprintf(w, "\nStale allowlist entries (%d). Please remove them so the check\n",
			len(f.UnusedAllowlistStructs)+len(f.UnusedAllowlistFields))
		fmt.Fprintln(w, "does not hide future drift:")
		for _, s := range f.UnusedAllowlistStructs {
			fmt.Fprintf(w, "  - structs.%s  # %s\n", s, reasonFor(s, allow.Structs))
		}
		for _, s := range f.UnusedAllowlistFields {
			fmt.Fprintf(w, "  - fields.%s  # %s\n", s, reasonFor(s, allow.Fields))
		}
	}

	fmt.Fprintln(w, "\n==========================================================")
}

func reasonFor(key string, m map[string]string) string {
	if m == nil {
		return "(no reason recorded)"
	}
	reason, ok := m[key]
	if !ok || reason == "" {
		return "(no reason recorded)"
	}
	return reason
}

// relOrAbs returns path relative to base when possible, for shorter logs.
func relOrAbs(base, path string) string {
	rel, err := filepath.Rel(base, path)
	if err != nil {
		return path
	}
	if strings.HasPrefix(rel, "..") {
		return path
	}
	return rel
}
