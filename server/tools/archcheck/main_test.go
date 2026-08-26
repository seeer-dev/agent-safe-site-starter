package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadImportPolicy(t *testing.T) {
	path := filepath.Join(t.TempDir(), "architecture.yaml")
	writeTestFile(t, path, `enforcement:
  version: 1
  import_rules:
    scan_root: src
    module_root: src/features
    platform_root: src/platform
    cross_module_imports: deny
    platform_to_modules: deny
`)
	policy, err := loadImportPolicy(path)
	if err != nil {
		t.Fatalf("loadImportPolicy: %v", err)
	}
	if policy.ScanRoot != "src" || policy.ModuleRoot != "src/features" || policy.PlatformRoot != "src/platform" {
		t.Fatalf("unexpected policy: %+v", policy)
	}
}

func TestLoadImportPolicyFailsClosed(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		wantError string
	}{
		{"missing version", `enforcement:\n  import_rules:\n    scan_root: src\n    module_root: src/features\n    platform_root: src/platform\n    cross_module_imports: deny\n    platform_to_modules: deny\n`, "missing enforcement.version"},
		{"unsupported version", `enforcement:\n  version: 2\n  import_rules:\n    scan_root: src\n    module_root: src/features\n    platform_root: src/platform\n    cross_module_imports: deny\n    platform_to_modules: deny\n`, "version must be 1"},
		{"rule cannot be disabled", `enforcement:\n  version: 1\n  import_rules:\n    scan_root: src\n    module_root: src/features\n    platform_root: src/platform\n    cross_module_imports: allow\n    platform_to_modules: deny\n`, "cross_module_imports must be deny"},
		{"unsafe root", `enforcement:\n  version: 1\n  import_rules:\n    scan_root: src\n    module_root: ../features\n    platform_root: src/platform\n    cross_module_imports: deny\n    platform_to_modules: deny\n`, "must stay inside the repository"},
		{"module root equals scan root", `enforcement:\n  version: 1\n  import_rules:\n    scan_root: src\n    module_root: src\n    platform_root: src/platform\n    cross_module_imports: deny\n    platform_to_modules: deny\n`, "strict descendant"},
		{"roots overlap", `enforcement:\n  version: 1\n  import_rules:\n    scan_root: src\n    module_root: src/shared\n    platform_root: src/shared\n    cross_module_imports: deny\n    platform_to_modules: deny\n`, "must be distinct"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "architecture.yaml")
			writeTestFile(t, path, strings.ReplaceAll(test.body, `\n`, "\n"))
			_, err := loadImportPolicy(path)
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("expected %q, got %v", test.wantError, err)
			}
		})
	}
}

func TestCheckArchitectureUsesConfiguredRoots(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "src/features/orders/orders.go"), `package orders
import _ "example.test/src/features/media"
`)
	writeTestFile(t, filepath.Join(root, "src/features/media/media.go"), "package media\n")
	writeTestFile(t, filepath.Join(root, "src/platform/platform.go"), `package platform
import _ "example.test/src/features/media"
`)
	violations, cov, err := checkArchitecture(root, importPolicy{Version: "1", ScanRoot: "src", ModuleRoot: "src/features", PlatformRoot: "src/platform"})
	if err != nil {
		t.Fatalf("checkArchitecture: %v", err)
	}
	if err := validateCoverage(cov); err != nil {
		t.Fatalf("validateCoverage: %v", err)
	}
	if len(violations) != 2 {
		t.Fatalf("expected two violations, got %#v", violations)
	}
	if !strings.Contains(strings.Join(violations, "\n"), `imports module "media" directly`) || !strings.Contains(strings.Join(violations, "\n"), "platform must not import modules") {
		t.Fatalf("unexpected violations: %#v", violations)
	}
}

func TestCheckArchitectureFailsOnMissingRoots(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "src/ok.go"), "package src\n")
	_, _, err := checkArchitecture(root, importPolicy{Version: "1", ScanRoot: "src", ModuleRoot: "src/missing-modules", PlatformRoot: "src/missing-platform"})
	if err == nil || !strings.Contains(err.Error(), "module_root") {
		t.Fatalf("expected missing module root error, got %v", err)
	}
}

func TestCoverageFailsClosed(t *testing.T) {
	for _, test := range []struct {
		name string
		cov  coverage
		want string
	}{
		{"no go", coverage{}, "scan_root"},
		{"no modules", coverage{GoFiles: 1}, "module_root"},
		{"no platform", coverage{GoFiles: 2, ModuleFiles: 1}, "platform_root"},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := validateCoverage(test.cov)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected %q, got %v", test.want, err)
			}
		})
	}
}

func writeTestFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
