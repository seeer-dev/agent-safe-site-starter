package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var (
	changeIDPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	commitPattern   = regexp.MustCompile(`^[0-9a-f]{40}$`)
)

type Scope struct {
	Task         string   `json:"task"`
	AllowedPaths []string `json:"allowed_paths"`
}

type SelectedControl struct {
	ChangeID           string   `json:"change_id"`
	Revision           int      `json:"revision"`
	Status             string   `json:"status"`
	RepositoryBaseline string   `json:"repository_baseline"`
	AppliesTo          []string `json:"applies_to"`
}

func main() {
	if err := run(os.Stdout, os.Stderr); err != nil {
		os.Exit(1)
	}
}

func run(stdout, stderr io.Writer) error {
	changeID := strings.TrimSpace(os.Getenv("SCOPE_CHANGE_ID"))
	if changeID != "" {
		return runSelectedChange(changeID, stdout, stderr)
	}
	return runLegacyScope(stdout, stderr)
}

func runLegacyScope(stdout, stderr io.Writer) error {
	body, err := os.ReadFile(".ai/scope.json")
	if os.IsNotExist(err) {
		fmt.Fprintln(stdout, "scopecheck: skipped (.ai/scope.json not present)")
		return nil
	}
	if err != nil {
		return fatal(stderr, fmt.Errorf("read .ai/scope.json: %w", err))
	}
	var scope Scope
	if err := json.Unmarshal(body, &scope); err != nil {
		return fatal(stderr, fmt.Errorf("parse .ai/scope.json: %w", err))
	}
	if len(scope.AllowedPaths) == 0 {
		return fatal(stderr, fmt.Errorf("scope has no allowed_paths"))
	}
	changed, err := changedFiles()
	if err != nil {
		return fatal(stderr, err)
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
		fmt.Fprintln(stderr, "scopecheck: files outside allowed scope:")
		for _, path := range outside {
			fmt.Fprintln(stderr, " -", path)
		}
		return errors.New("files outside allowed scope")
	}
	fmt.Fprintf(stdout, "scopecheck: ok (%d changed file(s))\n", len(changed))
	return nil
}

func runSelectedChange(changeID string, stdout, stderr io.Writer) error {
	if !changeIDPattern.MatchString(changeID) {
		return fatal(stderr, fmt.Errorf("invalid SCOPE_CHANGE_ID %q: must be lowercase hyphen-case", changeID))
	}

	if strings.TrimSpace(os.Getenv("CI")) != "" {
		return fatal(stderr, fmt.Errorf("SCOPE_CHANGE_ID cannot be used in CI; CI requires full-diff verification"))
	}

	isLinked, err := isLinkedWorktree()
	if err != nil {
		return fatal(stderr, err)
	}
	if !isLinked {
		return fatal(stderr, fmt.Errorf("SCOPE_CHANGE_ID requires an isolated linked worktree; current execution is in primary worktree"))
	}

	controlPath := filepath.Join("specs", "changes", changeID, "control.json")
	body, err := os.ReadFile(controlPath)
	if err != nil {
		return fatal(stderr, fmt.Errorf("read controlled change %s: %w", changeID, err))
	}
	var ctrl SelectedControl
	if err := json.Unmarshal(body, &ctrl); err != nil {
		return fatal(stderr, fmt.Errorf("parse %s: %w", controlPath, err))
	}

	if ctrl.ChangeID != changeID {
		return fatal(stderr, fmt.Errorf("control.json change_id %q does not match SCOPE_CHANGE_ID %q", ctrl.ChangeID, changeID))
	}
	if ctrl.Status != "Applying" && ctrl.Status != "Verifying" {
		return fatal(stderr, fmt.Errorf("controlled change %s has status %q; selected validation requires Applying or Verifying", changeID, ctrl.Status))
	}
	if len(ctrl.AppliesTo) == 0 {
		return fatal(stderr, fmt.Errorf("controlled change %s has no applies_to patterns", changeID))
	}
	if !commitPattern.MatchString(ctrl.RepositoryBaseline) {
		return fatal(stderr, fmt.Errorf("controlled change %s has invalid repository_baseline %q: must be 40-character hex commit", changeID, ctrl.RepositoryBaseline))
	}

	if _, err := gitOutput("merge-base", "--is-ancestor", ctrl.RepositoryBaseline, "HEAD"); err != nil {
		return fatal(stderr, fmt.Errorf("repository_baseline %s is not an ancestor of HEAD: %w", ctrl.RepositoryBaseline, err))
	}

	topLevel, err := gitOutput("rev-parse", "--show-toplevel")
	if err != nil {
		return fatal(stderr, fmt.Errorf("get repository top-level: %w", err))
	}
	if err := checkWorktreeOverlap(strings.TrimSpace(topLevel), ctrl.AppliesTo, changeID); err != nil {
		return fatal(stderr, err)
	}

	changed, err := selectedChangedFiles(ctrl.RepositoryBaseline)
	if err != nil {
		return fatal(stderr, err)
	}

	var outside []string
	for _, path := range changed {
		if !allowed(path, ctrl.AppliesTo) {
			outside = append(outside, path)
		}
	}

	if len(outside) > 0 {
		fmt.Fprintf(stderr, "scopecheck: files outside allowed scope for %s:\n", changeID)
		for _, path := range outside {
			fmt.Fprintf(stderr, " - %s\n", path)
		}
		return errors.New("files outside allowed scope")
	}

	fmt.Fprintf(stdout, "scopecheck: ok (%d changed file(s) in selected change %s)\n", len(changed), changeID)
	return nil
}

func isLinkedWorktree() (bool, error) {
	gitDirOut, err := gitOutput("rev-parse", "--git-dir")
	if err != nil {
		return false, fmt.Errorf("inspect git-dir: %w", err)
	}
	gitCommonDirOut, err := gitOutput("rev-parse", "--git-common-dir")
	if err != nil {
		return false, fmt.Errorf("inspect git-common-dir: %w", err)
	}

	gitDir, err := canonicalPath(strings.TrimSpace(gitDirOut))
	if err != nil {
		return false, fmt.Errorf("canonicalize git-dir: %w", err)
	}
	gitCommonDir, err := canonicalPath(strings.TrimSpace(gitCommonDirOut))
	if err != nil {
		return false, fmt.Errorf("canonicalize git-common-dir: %w", err)
	}

	return gitDir != gitCommonDir, nil
}

func checkWorktreeOverlap(currentWorktree string, appliesTo []string, changeID string) error {
	out, err := gitOutput("worktree", "list", "--porcelain")
	if err != nil {
		return fmt.Errorf("list worktrees: %w", err)
	}

	worktrees := parseWorktreeList(out)
	currCanon, err := canonicalPath(currentWorktree)
	if err != nil {
		return fmt.Errorf("canonicalize current worktree path %s: %w", currentWorktree, err)
	}

	for _, wt := range worktrees {
		wtCanon, err := canonicalPath(wt)
		if err != nil {
			return fmt.Errorf("canonicalize worktree path %s: %w", wt, err)
		}
		if wtCanon == currCanon {
			continue
		}

		otherDirty := map[string]struct{}{}

		diffOut, err := gitOutput("-C", wt, "diff", "--name-only", "HEAD")
		if err != nil {
			return fmt.Errorf("inspect diff in worktree %s: %w", wt, err)
		}
		addLines(otherDirty, diffOut)

		untrackedOut, err := gitOutput("-C", wt, "ls-files", "--others", "--exclude-standard")
		if err != nil {
			return fmt.Errorf("inspect untracked in worktree %s: %w", wt, err)
		}
		addLines(otherDirty, untrackedOut)

		for path := range otherDirty {
			if allowed(path, appliesTo) {
				return fmt.Errorf("concurrent ownership conflict: path %q in worktree %s overlaps with selected change %s applies_to", path, wt, changeID)
			}
		}
	}
	return nil
}

func parseWorktreeList(body string) []string {
	var worktrees []string
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "worktree ") {
			wt := strings.TrimSpace(strings.TrimPrefix(line, "worktree "))
			if wt != "" {
				worktrees = append(worktrees, wt)
			}
		}
	}
	return worktrees
}

var resolveCanonicalPath = defaultCanonicalPath

func defaultCanonicalPath(p string) (string, error) {
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	eval, err := filepath.EvalSymlinks(abs)
	if err == nil {
		abs = eval
	}
	return filepath.Clean(filepath.ToSlash(abs)), nil
}

func canonicalPath(p string) (string, error) {
	return resolveCanonicalPath(p)
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
	if out, err := runGitCommand(args...); err == nil {
		addLines(set, out)
	}
	if out, err := runGitCommand("ls-files", "--others", "--exclude-standard"); err == nil {
		addLines(set, out)
	}
	paths := make([]string, 0, len(set))
	for path := range set {
		paths = append(paths, filepath.ToSlash(path))
	}
	sort.Strings(paths)
	return paths, nil
}

func selectedChangedFiles(baseline string) ([]string, error) {
	set := map[string]struct{}{}

	out, err := gitOutput("diff", "--name-only", baseline+"...HEAD")
	if err != nil {
		return nil, err
	}
	addLines(set, out)

	out, err = gitOutput("diff", "--name-only", "HEAD")
	if err != nil {
		return nil, err
	}
	addLines(set, out)

	out, err = gitOutput("ls-files", "--others", "--exclude-standard")
	if err != nil {
		return nil, err
	}
	addLines(set, out)

	paths := make([]string, 0, len(set))
	for path := range set {
		paths = append(paths, filepath.ToSlash(path))
	}
	sort.Strings(paths)
	return paths, nil
}

var runGitCommand = func(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return string(out), nil
}

func gitOutput(args ...string) (string, error) {
	return runGitCommand(args...)
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

func fatal(stderr io.Writer, err error) error {
	fmt.Fprintln(stderr, "scopecheck:", err)
	return err
}
