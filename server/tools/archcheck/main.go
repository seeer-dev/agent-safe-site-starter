package main

import (
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const moduleImportMarker = "/server/internal/modules/"

func main() {
	var violations []string
	err := filepath.WalkDir("server/internal", func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
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
		sourceModule := moduleFromPath(path)
		isPlatform := strings.Contains(filepath.ToSlash(path), "/server/internal/platform/")
		for _, spec := range file.Imports {
			imp, _ := strconv.Unquote(spec.Path.Value)
			targetModule := moduleFromImport(imp)
			if sourceModule != "" && targetModule != "" && sourceModule != targetModule {
				violations = append(violations, fmt.Sprintf("%s imports module %q directly (%s)", path, targetModule, imp))
			}
			if isPlatform && targetModule != "" {
				violations = append(violations, fmt.Sprintf("%s: platform must not import modules (%s)", path, imp))
			}
		}
		return nil
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "archcheck:", err)
		os.Exit(1)
	}
	if len(violations) > 0 {
		for _, v := range violations {
			fmt.Fprintln(os.Stderr, "ARCH VIOLATION:", v)
		}
		os.Exit(1)
	}
	fmt.Println("archcheck: ok")
}

func moduleFromPath(path string) string {
	parts := strings.Split(filepath.ToSlash(path), "/")
	for i := 0; i+1 < len(parts); i++ {
		if parts[i] == "modules" {
			return parts[i+1]
		}
	}
	return ""
}

func moduleFromImport(imp string) string {
	i := strings.Index(imp, moduleImportMarker)
	if i < 0 {
		return ""
	}
	rest := imp[i+len(moduleImportMarker):]
	if j := strings.IndexByte(rest, '/'); j >= 0 {
		return rest[:j]
	}
	return rest
}
