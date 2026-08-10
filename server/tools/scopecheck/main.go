package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

type Scope struct {
	Task         string   `json:"task"`
	AllowedPaths []string `json:"allowed_paths"`
}

func main() {
	body, err := os.ReadFile(".ai/scope.json")
	if os.IsNotExist(err) {
		fmt.Println("scopecheck: skipped (.ai/scope.json not present)")
		return
	}
	if err != nil {
		fatal(err)
	}
	var scope Scope
	if err := json.Unmarshal(body, &scope); err != nil {
		fatal(fmt.Errorf("parse .ai/scope.json: %w", err))
	}
	if len(scope.AllowedPaths) == 0 {
		fatal(fmt.Errorf("scope has no allowed_paths"))
	}
	changed, err := changedFiles()
	if err != nil {
		fatal(err)
	}
	var outside []string
	for _, path := range changed {
		if path == ".ai/scope.json" {
			continue
		}
		if !allowed(path, scope.AllowedPaths) {
			outside = append(outside, path)
		}
	}
	if len(outside) > 0 {
		fmt.Fprintln(os.Stderr, "scopecheck: files outside allowed scope:")
		for _, path := range outside {
			fmt.Fprintln(os.Stderr, " -", path)
		}
		os.Exit(1)
	}
	fmt.Printf("scopecheck: ok (%d changed file(s))\n", len(changed))
}

func changedFiles() ([]string, error) {
	set := map[string]struct{}{}
	base := strings.TrimSpace(os.Getenv("SCOPE_BASE_REF"))
	var args []string
	if base != "" {
		args = []string{"diff", "--name-only", base + "...HEAD"}
	} else {
		args = []string{"diff", "--name-only", "HEAD"}
	}
	if out, err := exec.Command("git", args...).Output(); err == nil {
		addLines(set, string(out))
	}
	if out, err := exec.Command("git", "ls-files", "--others", "--exclude-standard").Output(); err == nil {
		addLines(set, string(out))
	}
	paths := make([]string, 0, len(set))
	for path := range set {
		paths = append(paths, filepath.ToSlash(path))
	}
	sort.Strings(paths)
	return paths, nil
}

func addLines(set map[string]struct{}, body string) {
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			set[filepath.ToSlash(line)] = struct{}{}
		}
	}
}

func allowed(path string, patterns []string) bool {
	path = filepath.ToSlash(path)
	for _, pattern := range patterns {
		pattern = filepath.ToSlash(strings.TrimSpace(pattern))
		if strings.HasSuffix(pattern, "/**") {
			prefix := strings.TrimSuffix(pattern, "**")
			if strings.HasPrefix(path, prefix) {
				return true
			}
			continue
		}
		if ok, _ := filepath.Match(pattern, path); ok {
			return true
		}
	}
	return false
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "scopecheck:", err)
	os.Exit(1)
}
