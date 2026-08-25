package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadImportPolicy(t *testing.T) {
	path := filepath.Join(t.TempDir(), "architecture.yaml")
	writeTestFile(t, path, `version: 1
name: test

enforcement:
  import_rules:
    scan_root: src
    module_root: src/features
    platform_root: src/platform
    deny_cross_module_imports: true
    deny_platform_module_imports: false
`)

	policy, err := loadImportPolicy(path)
	if err != nil {
		t.Fatalf("loadImportPolicy: %v", err)
	}
	if policy.ScanRoot != "src" || policy.ModuleRoot != "src/features" || policy.PlatformRoot != "src/platform" {
		t.Fatalf("unexpected roots: %+v", policy)
	}
	if !policy.DenyCrossModuleImports || policy.DenyPlatformModuleImports {
		t.Fatalf("unexpected flags: %+v", policy)
	}
}

func TestLoadImportPolicyFailsClosedOnInvalidPolicy(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		wantError string
	}{
		{
			name: "missing required rule",
			body: `enforcement:
  import_rules:
    scan_root: server/internal
    module_root: server/internal/modules
    platform_root: server/internal/platform
    deny_cross_module_imports: true
`,
			wantError: "deny_platform_module_imports",
		},
		{
			name: "unsupported rule",
			body: `enforcement:
  import_rules:
    scan_root: server/internal
    module_root: server/internal/modules
    platform_root: server/internal/platform
    deny_cross_module_imports: true
    deny_platform_module_imports: true
    allow_hidden_plugins: true
`,
			wantError: "unsupported enforcement.import_rules key",
		},
		{
			name: "unsafe root",
			body: `enforcement:
  import_rules:
    scan_root: server/internal
    module_root: ../modules
    platform_root: server/internal/platform
    deny_cross_module_imports: true
    deny_platform_module_imports: true
`,
			wantError: "must stay inside the repository",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "architecture.yaml")
			writeTestFile(t, path, test.body)

			_, err := loadImportPolicy(path)
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("expected error containing %q, got %v", test.wantError, err)
			}
		})
	}
}

func TestCheckArchitectureUsesConfiguredModuleRoot(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "src/features/orders/orders.go"), `package orders
import _ "example.test/src/features/media"
`)
	writeTestFile(t, filepath.Join(root, "src/features/media/media.go"), "package media\n")
	writeTestFile(t, filepath.Join(root, "src/platform/platform.go"), "package platform\n")

	violations, err := checkArchitecture(root, importPolicy{
		ScanRoot:                  "src",
		ModuleRoot:                "src/features",
		PlatformRoot:              "src/platform",
		DenyCrossModuleImports:    true,
		DenyPlatformModuleImports: true,
	})
	if err != nil {
		t.Fatalf("checkArchitecture: %v", err)
	}
	if len(violations) != 1 || !strings.Contains(violations[0], `imports module "media" directly`) {
		t.Fatalf("expected configured-root violation, got %#v", violations)
	}
}

func TestCheckArchitectureHonorsDisabledCrossModuleRule(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "src/features/orders/orders.go"), `package orders
import _ "example.test/src/features/media"
`)
	writeTestFile(t, filepath.Join(root, "src/features/media/media.go"), "package media\n")
	writeTestFile(t, filepath.Join(root, "src/platform/platform.go"), "package platform\n")

	violations, err := checkArchitecture(root, importPolicy{
		ScanRoot:                  "src",
		ModuleRoot:                "src/features",
		PlatformRoot:              "src/platform",
		DenyCrossModuleImports:    false,
		DenyPlatformModuleImports: true,
	})
	if err != nil {
		t.Fatalf("checkArchitecture: %v", err)
	}
	if len(violations) != 0 {
		t.Fatalf("expected manifest policy to disable cross-module rule, got %#v", violations)
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
