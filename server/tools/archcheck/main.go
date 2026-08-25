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
	ScanRoot                  string
	ModuleRoot                string
	PlatformRoot              string
	DenyCrossModuleImports    bool
	DenyPlatformModuleImports bool
}

func main() {
	policy, err := loadImportPolicy(architecturePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "archcheck:", err)
		os.Exit(1)
	}
	violations, err := checkArchitecture(".", policy)
	if err != nil {
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
		if strings.Contains(raw[:len(raw)-len(strings.TrimLeft(raw, " \t"))], "\t") {
			return importPolicy{}, fmt.Errorf("%s:%d: tabs are not supported in enforcement.import_rules indentation", path, lineNumber)
		}
		indent := len(raw) - len(strings.TrimLeft(raw, " "))
		switch indent {
		case 0:
			inEnforcement = trimmed == "enforcement:"
			inImportRules = false
		case 2:
			if inEnforcement {
				inImportRules = trimmed == "import_rules:"
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

	required := []string{
		"scan_root",
		"module_root",
		"platform_root",
		"deny_cross_module_imports",
		"deny_platform_module_imports",
	}
	for _, key := range required {
		if _, exists := values[key]; !exists {
			return importPolicy{}, fmt.Errorf("%s: missing enforcement.import_rules.%s", path, key)
		}
	}
	if len(values) != len(required) {
		allowed := make(map[string]struct{}, len(required))
		for _, key := range required {
			allowed[key] = struct{}{}
		}
		for key := range values {
			if _, ok := allowed[key]; !ok {
				return importPolicy{}, fmt.Errorf("%s: unsupported enforcement.import_rules key %q", path, key)
			}
		}
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
	if !withinRoot(moduleRoot, scanRoot) {
		return importPolicy{}, fmt.Errorf("%s: module_root %q must be under scan_root %q", path, moduleRoot, scanRoot)
	}
	if !withinRoot(platformRoot, scanRoot) {
		return importPolicy{}, fmt.Errorf("%s: platform_root %q must be under scan_root %q", path, platformRoot, scanRoot)
	}

	denyCross, err := strconv.ParseBool(values["deny_cross_module_imports"])
	if err != nil {
		return importPolicy{}, fmt.Errorf("%s: deny_cross_module_imports must be true or false", path)
	}
	denyPlatform, err := strconv.ParseBool(values["deny_platform_module_imports"])
	if err != nil {
		return importPolicy{}, fmt.Errorf("%s: deny_platform_module_imports must be true or false", path)
	}

	return importPolicy{
		ScanRoot:                  scanRoot,
		ModuleRoot:                moduleRoot,
		PlatformRoot:              platformRoot,
		DenyCrossModuleImports:    denyCross,
		DenyPlatformModuleImports: denyPlatform,
	}, nil
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

func checkArchitecture(repositoryRoot string, policy importPolicy) ([]string, error) {
	var violations []string
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
		sourceModule := moduleFromPath(relativePath, policy.ModuleRoot)
		isPlatform := withinRoot(relativePath, policy.PlatformRoot)
		for _, spec := range file.Imports {
			importPath, _ := strconv.Unquote(spec.Path.Value)
			targetModule := moduleFromImport(importPath, policy.ModuleRoot)
			if policy.DenyCrossModuleImports && sourceModule != "" && targetModule != "" && sourceModule != targetModule {
				violations = append(violations, fmt.Sprintf("%s imports module %q directly (%s)", relativePath, targetModule, importPath))
			}
			if policy.DenyPlatformModuleImports && isPlatform && targetModule != "" {
				violations = append(violations, fmt.Sprintf("%s: platform must not import modules (%s)", relativePath, importPath))
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(violations)
	return violations, nil
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
