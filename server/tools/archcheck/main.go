package main

import (
	"bufio"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const architecturePath = "architecture.yaml"

type importPolicy struct {
	Version      string
	ScanRoot     string
	ModuleRoot   string
	PlatformRoot string
}

type coverage struct {
	GoFiles       int
	ModuleFiles   int
	PlatformFiles int
}

func main() {
	policy, err := loadImportPolicy(architecturePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "archcheck:", err)
		os.Exit(1)
	}
	violations, coverage, err := checkArchitecture(".", policy)
	if err != nil {
		fmt.Fprintln(os.Stderr, "archcheck:", err)
		os.Exit(1)
	}
	if err := validateCoverage(coverage); err != nil {
		fmt.Fprintln(os.Stderr, "archcheck:", err)
		os.Exit(1)
	}
	if len(violations) > 0 {
		for _, violation := range violations {
			fmt.Fprintln(os.Stderr, "ARCH VIOLATION:", violation)
		}
		os.Exit(1)
	}
	fmt.Println("archcheck: ok")
}

func loadImportPolicy(path string) (importPolicy, error) {
	file, err := os.Open(path)
	if err != nil {
		return importPolicy{}, fmt.Errorf("read %s: %w", path, err)
	}
	defer file.Close()

	values := map[string]string{}
	inEnforcement := false
	inImportRules := false
	scanner := bufio.NewScanner(file)
	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		raw := strings.TrimRight(scanner.Text(), "\r")
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		indentPrefix := raw[:len(raw)-len(strings.TrimLeft(raw, " \t"))]
		if strings.Contains(indentPrefix, "\t") {
			return importPolicy{}, fmt.Errorf("%s:%d: tabs are not supported in enforcement indentation", path, lineNumber)
		}
		indent := len(raw) - len(strings.TrimLeft(raw, " "))
		switch indent {
		case 0:
			inEnforcement = trimmed == "enforcement:"
			inImportRules = false
		case 2:
			if inEnforcement {
				if trimmed == "import_rules:" {
					inImportRules = true
					continue
				}
				inImportRules = false
				key, value, ok := strings.Cut(trimmed, ":")
				if !ok || strings.TrimSpace(value) == "" {
					return importPolicy{}, fmt.Errorf("%s:%d: enforcement entry must be key: value", path, lineNumber)
				}
				key = strings.TrimSpace(key)
				if key != "version" {
					return importPolicy{}, fmt.Errorf("%s:%d: unsupported enforcement key %q", path, lineNumber, key)
				}
				if _, exists := values["version"]; exists {
					return importPolicy{}, fmt.Errorf("%s:%d: duplicate enforcement.version", path, lineNumber)
				}
				values["version"] = strings.TrimSpace(value)
			}
		case 4:
			if !inEnforcement || !inImportRules {
				continue
			}
			key, value, ok := strings.Cut(trimmed, ":")
			if !ok || strings.TrimSpace(value) == "" {
				return importPolicy{}, fmt.Errorf("%s:%d: enforcement.import_rules entry must be key: value", path, lineNumber)
			}
			key = strings.TrimSpace(key)
			value = strings.TrimSpace(value)
			if _, exists := values[key]; exists {
				return importPolicy{}, fmt.Errorf("%s:%d: duplicate enforcement.import_rules key %q", path, lineNumber, key)
			}
			values[key] = value
		}
	}
	if err := scanner.Err(); err != nil {
		return importPolicy{}, fmt.Errorf("scan %s: %w", path, err)
	}

	required := []string{"version", "scan_root", "module_root", "platform_root", "cross_module_imports", "platform_to_modules"}
	allowed := make(map[string]struct{}, len(required))
	for _, key := range required {
		allowed[key] = struct{}{}
		if _, exists := values[key]; !exists {
			return importPolicy{}, fmt.Errorf("%s: missing enforcement.%s", path, key)
		}
	}
	for key := range values {
		if _, ok := allowed[key]; !ok {
			return importPolicy{}, fmt.Errorf("%s: unsupported enforcement.import_rules key %q", path, key)
		}
	}
	if values["version"] != "1" {
		return importPolicy{}, fmt.Errorf("%s: enforcement.version must be 1", path)
	}
	if values["cross_module_imports"] != "deny" {
		return importPolicy{}, fmt.Errorf("%s: cross_module_imports must be deny", path)
	}
	if values["platform_to_modules"] != "deny" {
		return importPolicy{}, fmt.Errorf("%s: platform_to_modules must be deny", path)
	}

	scanRoot, err := cleanRelativeRoot(values["scan_root"])
	if err != nil {
		return importPolicy{}, fmt.Errorf("%s: scan_root: %w", path, err)
	}
	moduleRoot, err := cleanRelativeRoot(values["module_root"])
	if err != nil {
		return importPolicy{}, fmt.Errorf("%s: module_root: %w", path, err)
	}
	platformRoot, err := cleanRelativeRoot(values["platform_root"])
	if err != nil {
		return importPolicy{}, fmt.Errorf("%s: platform_root: %w", path, err)
	}
	if !strictlyWithinRoot(moduleRoot, scanRoot) {
		return importPolicy{}, fmt.Errorf("%s: module_root %q must be a strict descendant of scan_root %q", path, moduleRoot, scanRoot)
	}
	if !strictlyWithinRoot(platformRoot, scanRoot) {
		return importPolicy{}, fmt.Errorf("%s: platform_root %q must be a strict descendant of scan_root %q", path, platformRoot, scanRoot)
	}
	if moduleRoot == platformRoot {
		return importPolicy{}, fmt.Errorf("%s: module_root and platform_root must be distinct", path)
	}

	return importPolicy{Version: "1", ScanRoot: scanRoot, ModuleRoot: moduleRoot, PlatformRoot: platformRoot}, nil
}

func cleanRelativeRoot(value string) (string, error) {
	value = filepath.ToSlash(strings.TrimSpace(value))
	if value == "" || filepath.IsAbs(filepath.FromSlash(value)) || strings.ContainsAny(value, "*?[") {
		return "", fmt.Errorf("must be a non-empty relative directory without globs")
	}
	cleaned := filepath.ToSlash(filepath.Clean(filepath.FromSlash(value)))
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", fmt.Errorf("must stay inside the repository")
	}
	return strings.TrimSuffix(cleaned, "/"), nil
}

func checkArchitecture(repositoryRoot string, policy importPolicy) ([]string, coverage, error) {
	roots := []struct {
		label string
		root  string
	}{
		{"scan_root", policy.ScanRoot},
		{"module_root", policy.ModuleRoot},
		{"platform_root", policy.PlatformRoot},
	}
	for _, item := range roots {
		info, err := os.Stat(filepath.Join(repositoryRoot, filepath.FromSlash(item.root)))
		if err != nil {
			return nil, coverage{}, fmt.Errorf("%s %q: %w", item.label, item.root, err)
		}
		if !info.IsDir() {
			return nil, coverage{}, fmt.Errorf("%s %q is not a directory", item.label, item.root)
		}
	}

	var violations []string
	var cov coverage
	scanRoot := filepath.Join(repositoryRoot, filepath.FromSlash(policy.ScanRoot))
	err := filepath.WalkDir(scanRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, content, parser.ImportsOnly)
		if err != nil {
			return err
		}
		relativePath, err := filepath.Rel(repositoryRoot, path)
		if err != nil {
			return err
		}
		relativePath = filepath.ToSlash(relativePath)
		cov.GoFiles++
		sourceModule := moduleFromPath(relativePath, policy.ModuleRoot)
		if sourceModule != "" {
			cov.ModuleFiles++
		}
		isPlatform := withinRoot(relativePath, policy.PlatformRoot)
		if isPlatform {
			cov.PlatformFiles++
		}
		for _, spec := range file.Imports {
			importPath, _ := strconv.Unquote(spec.Path.Value)
			targetModule := moduleFromImport(importPath, policy.ModuleRoot)
			if sourceModule != "" && targetModule != "" && sourceModule != targetModule {
				violations = append(violations, fmt.Sprintf("%s imports module %q directly (%s)", relativePath, targetModule, importPath))
			}
			if isPlatform && targetModule != "" {
				violations = append(violations, fmt.Sprintf("%s: platform must not import modules (%s)", relativePath, importPath))
			}
		}
		return nil
	})
	if err != nil {
		return nil, coverage{}, err
	}
	sort.Strings(violations)
	return violations, cov, nil
}

func validateCoverage(cov coverage) error {
	if cov.GoFiles == 0 {
		return fmt.Errorf("scan_root contains no non-test Go files")
	}
	if cov.ModuleFiles == 0 {
		return fmt.Errorf("module_root matched no non-test Go files")
	}
	if cov.PlatformFiles == 0 {
		return fmt.Errorf("platform_root matched no non-test Go files")
	}
	return nil
}

func moduleFromPath(path, moduleRoot string) string {
	path = filepath.ToSlash(path)
	moduleRoot = strings.TrimSuffix(filepath.ToSlash(moduleRoot), "/")
	prefix := moduleRoot + "/"
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	rest := strings.TrimPrefix(path, prefix)
	if rest == "" {
		return ""
	}
	if slash := strings.IndexByte(rest, '/'); slash >= 0 {
		return rest[:slash]
	}
	return ""
}

func moduleFromImport(importPath, moduleRoot string) string {
	marker := "/" + strings.Trim(filepath.ToSlash(moduleRoot), "/") + "/"
	index := strings.Index(importPath, marker)
	if index < 0 {
		return ""
	}
	rest := importPath[index+len(marker):]
	if slash := strings.IndexByte(rest, '/'); slash >= 0 {
		return rest[:slash]
	}
	return rest
}

func withinRoot(path, root string) bool {
	path = strings.TrimSuffix(filepath.ToSlash(path), "/")
	root = strings.TrimSuffix(filepath.ToSlash(root), "/")
	return path == root || strings.HasPrefix(path, root+"/")
}

func strictlyWithinRoot(path, root string) bool {
	path = strings.TrimSuffix(filepath.ToSlash(path), "/")
	root = strings.TrimSuffix(filepath.ToSlash(root), "/")
	return path != root && strings.HasPrefix(path, root+"/")
}
