package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type testRepo struct {
	primaryRoot string
	baseline    string
}

func initTestRepo(t *testing.T) *testRepo {
	t.Helper()
	root := t.TempDir()
	primary := filepath.Join(root, "primary")
	if err := os.MkdirAll(primary, 0o755); err != nil {
		t.Fatal(err)
	}

	runGit(t, primary, "init")
	runGit(t, primary, "config", "user.name", "test")
	runGit(t, primary, "config", "user.email", "test@example.com")
	runGit(t, primary, "config", "commit.gpgsign", "false")

	// Create initial baseline commit
	baseFile := filepath.Join(primary, "base.txt")
	if err := os.WriteFile(baseFile, []byte("baseline\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, primary, "add", "base.txt")
	runGit(t, primary, "commit", "-m", "initial baseline")

	baselineOut := strings.TrimSpace(runGit(t, primary, "rev-parse", "HEAD"))
	if len(baselineOut) != 40 {
		t.Fatalf("expected 40-char SHA, got: %s", baselineOut)
	}

	return &testRepo{
		primaryRoot: primary,
		baseline:    baselineOut,
	}
}

func (r *testRepo) addLinkedWorktree(t *testing.T, branchName, dirName string) string {
	t.Helper()
	wtPath := filepath.Join(filepath.Dir(r.primaryRoot), dirName)
	runGit(t, r.primaryRoot, "worktree", "add", "-b", branchName, wtPath, r.baseline)
	return wtPath
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s in %s failed: %v\nOutput: %s", strings.Join(args, " "), dir, err, string(out))
	}
	return string(out)
}

func writeControlFile(t *testing.T, dir, changeID, status, baseline string, appliesTo []string) {
	t.Helper()
	controlDir := filepath.Join(dir, "specs", "changes", changeID)
	if err := os.MkdirAll(controlDir, 0o755); err != nil {
		t.Fatal(err)
	}
	ctrl := SelectedControl{
		ChangeID:           changeID,
		Revision:           1,
		Status:             status,
		RepositoryBaseline: baseline,
		AppliesTo:          appliesTo,
	}
	body, err := json.MarshalIndent(ctrl, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(controlDir, "control.json"), append(body, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}

func runScopecheckInDir(t *testing.T, dir string, envMap map[string]string) (int, string, string) {
	t.Helper()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Chdir(oldWd)
	}()

	// Save and restore env
	var envKeysToClear []string
	for k, v := range envMap {
		oldVal, exists := os.LookupEnv(k)
		if exists {
			defer os.Setenv(k, oldVal)
		} else {
			envKeysToClear = append(envKeysToClear, k)
		}
		os.Setenv(k, v)
	}
	defer func() {
		for _, k := range envKeysToClear {
			_ = os.Unsetenv(k)
		}
	}()

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	runErr := run(stdout, stderr)
	exitCode := 0
	if runErr != nil {
		exitCode = 1
	}

	return exitCode, stdout.String(), stderr.String()
}

func TestSelectedChangeIgnoresDirtyPrimaryWorktree(t *testing.T) {
	repo := initTestRepo(t)
	wt := repo.addLinkedWorktree(t, "feat-selected-pass", "wt-selected-pass")

	// Dirty the primary worktree with foreign tracked/untracked changes outside the change scope
	if err := os.WriteFile(filepath.Join(repo.primaryRoot, "primary-untracked.txt"), []byte("foreign\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo.primaryRoot, "base.txt"), []byte("foreign modification\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// In linked worktree, write valid control for change-01
	appliesTo := []string{"allowed.txt", "specs/changes/change-01/**"}
	writeControlFile(t, wt, "change-01", "Applying", repo.baseline, appliesTo)
	if err := os.WriteFile(filepath.Join(wt, "allowed.txt"), []byte("allowed content\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	exitCode, stdout, stderr := runScopecheckInDir(t, wt, map[string]string{
		"SCOPE_CHANGE_ID": "change-01",
		"CI":              "",
	})

	if exitCode != 0 {
		t.Fatalf("expected scopecheck exit 0, got %d.\nStderr: %s\nStdout: %s", exitCode, stderr, stdout)
	}
	if !strings.Contains(stdout, "scopecheck: ok") || !strings.Contains(stdout, "change-01") {
		t.Fatalf("unexpected stdout: %s", stdout)
	}
	if strings.Contains(stdout, "primary-untracked") || strings.Contains(stderr, "primary-untracked") {
		t.Fatalf("primary worktree dirt leaked into output:\nStdout: %s\nStderr: %s", stdout, stderr)
	}
}

func TestSelectedChangeRejectsOutsidePath(t *testing.T) {
	repo := initTestRepo(t)
	wt := repo.addLinkedWorktree(t, "feat-selected-fail", "wt-selected-fail")

	appliesTo := []string{"allowed.txt", "specs/changes/change-outside/**"}
	writeControlFile(t, wt, "change-outside", "Applying", repo.baseline, appliesTo)
	if err := os.WriteFile(filepath.Join(wt, "allowed.txt"), []byte("allowed content\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wt, "unauthorized.txt"), []byte("forbidden content\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	exitCode, stdout, stderr := runScopecheckInDir(t, wt, map[string]string{
		"SCOPE_CHANGE_ID": "change-outside",
		"CI":              "",
	})

	if exitCode == 0 {
		t.Fatalf("expected scopecheck failure, got exit 0.\nStdout: %s\nStderr: %s", stdout, stderr)
	}
	if !strings.Contains(stderr, "files outside allowed scope for change-outside") || !strings.Contains(stderr, "unauthorized.txt") {
		t.Fatalf("expected stderr to name unauthorized.txt, got:\n%s", stderr)
	}
}

func TestSelectedChangeRejectsPrimaryWorktree(t *testing.T) {
	repo := initTestRepo(t)

	appliesTo := []string{"base.txt", "specs/changes/change-primary/**"}
	writeControlFile(t, repo.primaryRoot, "change-primary", "Applying", repo.baseline, appliesTo)

	exitCode, _, stderr := runScopecheckInDir(t, repo.primaryRoot, map[string]string{
		"SCOPE_CHANGE_ID": "change-primary",
		"CI":              "",
	})

	if exitCode == 0 {
		t.Fatal("expected failure when running in primary worktree with SCOPE_CHANGE_ID")
	}
	if !strings.Contains(stderr, "requires an isolated linked worktree") {
		t.Fatalf("expected primary worktree rejection error, got:\n%s", stderr)
	}
}

func TestSelectedChangeRejectsCI(t *testing.T) {
	repo := initTestRepo(t)
	wt := repo.addLinkedWorktree(t, "feat-ci", "wt-ci")

	appliesTo := []string{"specs/changes/change-ci/**"}
	writeControlFile(t, wt, "change-ci", "Applying", repo.baseline, appliesTo)

	exitCode, _, stderr := runScopecheckInDir(t, wt, map[string]string{
		"SCOPE_CHANGE_ID": "change-ci",
		"CI":              "true",
	})

	if exitCode == 0 {
		t.Fatal("expected failure when running with SCOPE_CHANGE_ID in CI")
	}
	if !strings.Contains(stderr, "cannot be used in CI") {
		t.Fatalf("expected CI rejection message, got:\n%s", stderr)
	}
}

func TestSelectedChangeRejectsInvalidControlAndBaseline(t *testing.T) {
	repo := initTestRepo(t)
	wt := repo.addLinkedWorktree(t, "feat-invalid", "wt-invalid")

	t.Run("invalid change id format", func(t *testing.T) {
		exitCode, _, stderr := runScopecheckInDir(t, wt, map[string]string{
			"SCOPE_CHANGE_ID": "Bad_ID_Uppercase",
			"CI":              "",
		})
		if exitCode == 0 || !strings.Contains(stderr, "must be lowercase hyphen-case") {
			t.Fatalf("expected invalid ID format error, got: %s", stderr)
		}
	})

	t.Run("missing control json", func(t *testing.T) {
		exitCode, _, stderr := runScopecheckInDir(t, wt, map[string]string{
			"SCOPE_CHANGE_ID": "nonexistent-change",
			"CI":              "",
		})
		if exitCode == 0 || !strings.Contains(stderr, "read controlled change nonexistent-change") {
			t.Fatalf("expected missing control file error, got: %s", stderr)
		}
	})

	t.Run("malformed control json", func(t *testing.T) {
		dir := filepath.Join(wt, "specs", "changes", "malformed-json")
		_ = os.MkdirAll(dir, 0o755)
		_ = os.WriteFile(filepath.Join(dir, "control.json"), []byte("{not json"), 0o644)

		exitCode, _, stderr := runScopecheckInDir(t, wt, map[string]string{
			"SCOPE_CHANGE_ID": "malformed-json",
			"CI":              "",
		})
		if exitCode == 0 || !strings.Contains(stderr, "parse") {
			t.Fatalf("expected malformed JSON error, got: %s", stderr)
		}
	})

	t.Run("invalid status Draft", func(t *testing.T) {
		writeControlFile(t, wt, "draft-change", "Draft", repo.baseline, []string{"foo.go"})
		exitCode, _, stderr := runScopecheckInDir(t, wt, map[string]string{
			"SCOPE_CHANGE_ID": "draft-change",
			"CI":              "",
		})
		if exitCode == 0 || !strings.Contains(stderr, "selected validation requires Applying or Verifying") {
			t.Fatalf("expected status error, got: %s", stderr)
		}
	})

	t.Run("empty applies to", func(t *testing.T) {
		writeControlFile(t, wt, "empty-scope", "Applying", repo.baseline, []string{})
		exitCode, _, stderr := runScopecheckInDir(t, wt, map[string]string{
			"SCOPE_CHANGE_ID": "empty-scope",
			"CI":              "",
		})
		if exitCode == 0 || !strings.Contains(stderr, "has no applies_to patterns") {
			t.Fatalf("expected empty scope error, got: %s", stderr)
		}
	})

	t.Run("invalid baseline format", func(t *testing.T) {
		writeControlFile(t, wt, "bad-sha", "Applying", "not-a-40-hex-sha", []string{"foo.go"})
		exitCode, _, stderr := runScopecheckInDir(t, wt, map[string]string{
			"SCOPE_CHANGE_ID": "bad-sha",
			"CI":              "",
		})
		if exitCode == 0 || !strings.Contains(stderr, "must be 40-character hex commit") {
			t.Fatalf("expected 40-hex error, got: %s", stderr)
		}
	})

	t.Run("non-ancestor baseline", func(t *testing.T) {
		// Create orphan branch commit
		runGit(t, repo.primaryRoot, "checkout", "--orphan", "orphan-branch")
		_ = os.WriteFile(filepath.Join(repo.primaryRoot, "orphan.txt"), []byte("orphan"), 0o644)
		runGit(t, repo.primaryRoot, "add", "orphan.txt")
		runGit(t, repo.primaryRoot, "commit", "-m", "orphan commit")
		orphanSHA := strings.TrimSpace(runGit(t, repo.primaryRoot, "rev-parse", "HEAD"))
		runGit(t, repo.primaryRoot, "checkout", "master")

		writeControlFile(t, wt, "orphan-change", "Applying", orphanSHA, []string{"foo.go"})
		exitCode, _, stderr := runScopecheckInDir(t, wt, map[string]string{
			"SCOPE_CHANGE_ID": "orphan-change",
			"CI":              "",
		})
		if exitCode == 0 || !strings.Contains(stderr, "not an ancestor of HEAD") {
			t.Fatalf("expected non-ancestor error, got: %s", stderr)
		}
	})
}

func TestSelectedChangeRejectsDirtyPathInOtherWorktree(t *testing.T) {
	repo := initTestRepo(t)
	wtA := repo.addLinkedWorktree(t, "feat-worktree-a", "wt-worktree-a")
	wtB := repo.addLinkedWorktree(t, "feat-worktree-b", "wt-worktree-b")

	// Worktree A edits shared.go
	if err := os.WriteFile(filepath.Join(wtA, "shared.go"), []byte("package main\n// edited in A\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Worktree B is running change-b which has shared.go in applies_to
	appliesToB := []string{"shared.go", "specs/changes/change-b/**"}
	writeControlFile(t, wtB, "change-b", "Applying", repo.baseline, appliesToB)

	exitCode, _, stderr := runScopecheckInDir(t, wtB, map[string]string{
		"SCOPE_CHANGE_ID": "change-b",
		"CI":              "",
	})

	if exitCode == 0 {
		t.Fatal("expected scopecheck in worktree B to fail due to overlap in worktree A")
	}
	if !strings.Contains(stderr, "concurrent ownership conflict") || !strings.Contains(stderr, "shared.go") {
		t.Fatalf("expected overlap error naming shared.go, got:\n%s", stderr)
	}
}

func TestSelectedChangeTrackedStagedUnstagedDeletedAndUntracked(t *testing.T) {
	repo := initTestRepo(t)
	wt := repo.addLinkedWorktree(t, "feat-lifecycle", "wt-lifecycle")

	appliesTo := []string{
		"base.txt",
		"staged.txt",
		"unstaged.txt",
		"deleted.txt",
		"untracked.txt",
		"specs/changes/change-lifecycle/**",
	}
	writeControlFile(t, wt, "change-lifecycle", "Applying", repo.baseline, appliesTo)

	// Create and commit deleted.txt first so it can be deleted
	if err := os.WriteFile(filepath.Join(wt, "deleted.txt"), []byte("to delete\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, wt, "add", "deleted.txt")
	runGit(t, wt, "commit", "-m", "commit deleted.txt")

	// Delete deleted.txt
	_ = os.Remove(filepath.Join(wt, "deleted.txt"))

	// Create and stage staged.txt
	if err := os.WriteFile(filepath.Join(wt, "staged.txt"), []byte("staged\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, wt, "add", "staged.txt")

	// Create unstaged modification to base.txt
	if err := os.WriteFile(filepath.Join(wt, "base.txt"), []byte("modified base\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create untracked.txt
	if err := os.WriteFile(filepath.Join(wt, "untracked.txt"), []byte("untracked\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	exitCode, stdout, stderr := runScopecheckInDir(t, wt, map[string]string{
		"SCOPE_CHANGE_ID": "change-lifecycle",
		"CI":              "",
	})

	if exitCode != 0 {
		t.Fatalf("expected all allowed lifecycle changes to pass, got exit %d.\nStderr: %s\nStdout: %s", exitCode, stderr, stdout)
	}
	if !strings.Contains(stdout, "scopecheck: ok") {
		t.Fatalf("unexpected stdout: %s", stdout)
	}
}

func TestLegacyScopeMissingSkips(t *testing.T) {
	repo := initTestRepo(t)

	exitCode, stdout, stderr := runScopecheckInDir(t, repo.primaryRoot, map[string]string{
		"SCOPE_CHANGE_ID": "",
		"CI":              "",
	})

	if exitCode != 0 {
		t.Fatalf("expected legacy missing scope to exit 0, got %d. Stderr: %s", exitCode, stderr)
	}
	if !strings.Contains(stdout, "scopecheck: skipped (.ai/scope.json not present)") {
		t.Fatalf("unexpected stdout: %s", stdout)
	}
}

func TestLegacyScopeAllowsConfiguredPaths(t *testing.T) {
	repo := initTestRepo(t)

	aiDir := filepath.Join(repo.primaryRoot, ".ai")
	_ = os.MkdirAll(aiDir, 0o755)
	scope := Scope{
		Task:         "legacy task",
		AllowedPaths: []string{"base.txt", "new.txt"},
	}
	body, _ := json.Marshal(scope)
	_ = os.WriteFile(filepath.Join(aiDir, "scope.json"), body, 0o644)

	_ = os.WriteFile(filepath.Join(repo.primaryRoot, "base.txt"), []byte("changed"), 0o644)
	_ = os.WriteFile(filepath.Join(repo.primaryRoot, "new.txt"), []byte("new"), 0o644)

	exitCode, stdout, stderr := runScopecheckInDir(t, repo.primaryRoot, map[string]string{
		"SCOPE_CHANGE_ID": "",
		"CI":              "",
	})

	if exitCode != 0 {
		t.Fatalf("expected legacy allowed paths to pass, got %d.\nStderr: %s\nStdout: %s", exitCode, stderr, stdout)
	}
	if !strings.Contains(stdout, "scopecheck: ok") {
		t.Fatalf("unexpected stdout: %s", stdout)
	}
}

func TestLegacyScopeRejectsOutsideAndUntrackedPaths(t *testing.T) {
	repo := initTestRepo(t)

	aiDir := filepath.Join(repo.primaryRoot, ".ai")
	_ = os.MkdirAll(aiDir, 0o755)
	scope := Scope{
		Task:         "legacy task",
		AllowedPaths: []string{"base.txt"},
	}
	body, _ := json.Marshal(scope)
	_ = os.WriteFile(filepath.Join(aiDir, "scope.json"), body, 0o644)

	_ = os.WriteFile(filepath.Join(repo.primaryRoot, "forbidden.txt"), []byte("forbidden"), 0o644)

	exitCode, _, stderr := runScopecheckInDir(t, repo.primaryRoot, map[string]string{
		"SCOPE_CHANGE_ID": "",
		"CI":              "",
	})

	if exitCode == 0 {
		t.Fatal("expected legacy outside path to fail")
	}
	if !strings.Contains(stderr, "files outside allowed scope") || !strings.Contains(stderr, "forbidden.txt") {
		t.Fatalf("expected stderr to name forbidden.txt, got:\n%s", stderr)
	}
}

func TestLegacyScopeBaseRef(t *testing.T) {
	repo := initTestRepo(t)

	// Commit a new file on branch
	_ = os.WriteFile(filepath.Join(repo.primaryRoot, "branch.txt"), []byte("branch"), 0o644)
	runGit(t, repo.primaryRoot, "add", "branch.txt")
	runGit(t, repo.primaryRoot, "commit", "-m", "branch commit")

	aiDir := filepath.Join(repo.primaryRoot, ".ai")
	_ = os.MkdirAll(aiDir, 0o755)
	scope := Scope{
		Task:         "legacy baseref task",
		AllowedPaths: []string{"branch.txt"},
	}
	body, _ := json.Marshal(scope)
	_ = os.WriteFile(filepath.Join(aiDir, "scope.json"), body, 0o644)

	exitCode, stdout, stderr := runScopecheckInDir(t, repo.primaryRoot, map[string]string{
		"SCOPE_CHANGE_ID": "",
		"SCOPE_BASE_REF":  repo.baseline,
		"CI":              "",
	})

	if exitCode != 0 {
		t.Fatalf("expected legacy SCOPE_BASE_REF to pass, got %d.\nStderr: %s\nStdout: %s", exitCode, stderr, stdout)
	}
	if !strings.Contains(stdout, "scopecheck: ok") {
		t.Fatalf("unexpected stdout: %s", stdout)
	}
}

func TestSelectedChangeGitFailureFailClosed(t *testing.T) {
	repo := initTestRepo(t)
	wt := repo.addLinkedWorktree(t, "feat-git-fail", "wt-git-fail")
	appliesTo := []string{"specs/changes/change-fail/**"}
	writeControlFile(t, wt, "change-fail", "Applying", repo.baseline, appliesTo)

	t.Run("git discovery rev-parse git-dir failure fails closed", func(t *testing.T) {
		orig := runGitCommand
		defer func() { runGitCommand = orig }()

		runGitCommand = func(args ...string) (string, error) {
			if len(args) >= 2 && args[0] == "rev-parse" && args[1] == "--git-dir" {
				return "", fmt.Errorf("git rev-parse --git-dir: exit status 128: fatal: simulated git discovery failure")
			}
			return orig(args...)
		}

		exitCode, stdout, stderr := runScopecheckInDir(t, wt, map[string]string{
			"SCOPE_CHANGE_ID": "change-fail",
			"CI":              "",
		})

		if exitCode == 0 {
			t.Fatalf("expected scopecheck to fail when git-dir discovery fails, got exit 0.\nStdout: %s\nStderr: %s", stdout, stderr)
		}
		if !strings.Contains(stderr, "inspect git-dir") || !strings.Contains(stderr, "simulated git discovery failure") {
			t.Fatalf("expected clear diagnostic in stderr, got:\n%s", stderr)
		}
	})

	t.Run("git worktree list failure fails closed", func(t *testing.T) {
		orig := runGitCommand
		defer func() { runGitCommand = orig }()

		runGitCommand = func(args ...string) (string, error) {
			if len(args) >= 2 && args[0] == "worktree" && args[1] == "list" {
				return "", fmt.Errorf("git worktree list --porcelain: exit status 128: fatal: simulated worktree list error")
			}
			return orig(args...)
		}

		exitCode, stdout, stderr := runScopecheckInDir(t, wt, map[string]string{
			"SCOPE_CHANGE_ID": "change-fail",
			"CI":              "",
		})

		if exitCode == 0 {
			t.Fatalf("expected scopecheck to fail when worktree listing fails, got exit 0.\nStdout: %s\nStderr: %s", stdout, stderr)
		}
		if !strings.Contains(stderr, "list worktrees") || !strings.Contains(stderr, "simulated worktree list error") {
			t.Fatalf("expected clear diagnostic in stderr, got:\n%s", stderr)
		}
	})

	t.Run("git selected diff failure fails closed", func(t *testing.T) {
		orig := runGitCommand
		defer func() { runGitCommand = orig }()

		runGitCommand = func(args ...string) (string, error) {
			if len(args) >= 2 && args[0] == "diff" && args[1] == "--name-only" {
				return "", fmt.Errorf("git diff --name-only: exit status 128: fatal: simulated diff error")
			}
			return orig(args...)
		}

		exitCode, stdout, stderr := runScopecheckInDir(t, wt, map[string]string{
			"SCOPE_CHANGE_ID": "change-fail",
			"CI":              "",
		})

		if exitCode == 0 {
			t.Fatalf("expected scopecheck to fail when git diff fails, got exit 0.\nStdout: %s\nStderr: %s", stdout, stderr)
		}
		if !strings.Contains(stderr, "git diff") || !strings.Contains(stderr, "simulated diff error") {
			t.Fatalf("expected clear diagnostic in stderr, got:\n%s", stderr)
		}
	})

	t.Run("git ls-files untracked failure fails closed", func(t *testing.T) {
		orig := runGitCommand
		defer func() { runGitCommand = orig }()

		runGitCommand = func(args ...string) (string, error) {
			if len(args) >= 1 && args[0] == "ls-files" {
				return "", fmt.Errorf("git ls-files: exit status 128: fatal: simulated ls-files error")
			}
			return orig(args...)
		}

		exitCode, stdout, stderr := runScopecheckInDir(t, wt, map[string]string{
			"SCOPE_CHANGE_ID": "change-fail",
			"CI":              "",
		})

		if exitCode == 0 {
			t.Fatalf("expected scopecheck to fail when git ls-files fails, got exit 0.\nStdout: %s\nStderr: %s", stdout, stderr)
		}
		if !strings.Contains(stderr, "git ls-files") || !strings.Contains(stderr, "simulated ls-files error") {
			t.Fatalf("expected clear diagnostic in stderr, got:\n%s", stderr)
		}
	})
}

func TestLegacyScopeGitFailureIgnored(t *testing.T) {
	repo := initTestRepo(t)

	aiDir := filepath.Join(repo.primaryRoot, ".ai")
	_ = os.MkdirAll(aiDir, 0o755)
	scope := Scope{
		Task:         "legacy git failure tolerance",
		AllowedPaths: []string{"untracked-allowed.txt"},
	}
	body, _ := json.Marshal(scope)
	_ = os.WriteFile(filepath.Join(aiDir, "scope.json"), body, 0o644)

	// Create an untracked allowed file
	_ = os.WriteFile(filepath.Join(repo.primaryRoot, "untracked-allowed.txt"), []byte("allowed\n"), 0o644)

	t.Run("diff failure ignored and validation proceeds with ls-files", func(t *testing.T) {
		orig := runGitCommand
		defer func() { runGitCommand = orig }()

		runGitCommand = func(args ...string) (string, error) {
			if len(args) >= 1 && args[0] == "diff" {
				return "", fmt.Errorf("git diff: exit status 128: simulated diff failure in legacy mode")
			}
			return orig(args...)
		}

		exitCode, stdout, stderr := runScopecheckInDir(t, repo.primaryRoot, map[string]string{
			"SCOPE_CHANGE_ID": "",
			"CI":              "",
		})

		if exitCode != 0 {
			t.Fatalf("expected legacy scopecheck to ignore git diff failure and exit 0, got %d.\nStderr: %s\nStdout: %s", exitCode, stderr, stdout)
		}
		if !strings.Contains(stdout, "scopecheck: ok") {
			t.Fatalf("expected legacy scopecheck to succeed, got stdout: %s", stdout)
		}
	})

	t.Run("ls-files failure ignored and validation proceeds with diff", func(t *testing.T) {
		orig := runGitCommand
		defer func() { runGitCommand = orig }()

		// Create a tracked modification to base.txt and allow base.txt
		_ = os.WriteFile(filepath.Join(repo.primaryRoot, "base.txt"), []byte("modified\n"), 0o644)
		_ = os.Remove(filepath.Join(repo.primaryRoot, "untracked-allowed.txt"))
		scope.AllowedPaths = []string{"base.txt"}
		newBody, _ := json.Marshal(scope)
		_ = os.WriteFile(filepath.Join(aiDir, "scope.json"), newBody, 0o644)

		runGitCommand = func(args ...string) (string, error) {
			if len(args) >= 1 && args[0] == "ls-files" {
				return "", fmt.Errorf("git ls-files: exit status 128: simulated ls-files failure in legacy mode")
			}
			return orig(args...)
		}

		exitCode, stdout, stderr := runScopecheckInDir(t, repo.primaryRoot, map[string]string{
			"SCOPE_CHANGE_ID": "",
			"CI":              "",
		})

		if exitCode != 0 {
			t.Fatalf("expected legacy scopecheck to ignore git ls-files failure and exit 0, got %d.\nStderr: %s\nStdout: %s", exitCode, stderr, stdout)
		}
		if !strings.Contains(stdout, "scopecheck: ok") {
			t.Fatalf("expected legacy scopecheck to succeed, got stdout: %s", stdout)
		}
	})
}

func TestSelectedChangePathResolutionFailClosed(t *testing.T) {
	repo := initTestRepo(t)
	wt := repo.addLinkedWorktree(t, "feat-path-fail", "wt-path-fail")
	appliesTo := []string{"specs/changes/change-path-fail/**"}
	writeControlFile(t, wt, "change-path-fail", "Applying", repo.baseline, appliesTo)

	t.Run("current worktree path canonicalization failure fails closed", func(t *testing.T) {
		orig := resolveCanonicalPath
		defer func() { resolveCanonicalPath = orig }()

		resolveCanonicalPath = func(p string) (string, error) {
			// Fail only for current worktree root path in checkWorktreeOverlap, not .git / gitDir
			if !strings.Contains(p, ".git") && strings.HasSuffix(filepath.ToSlash(p), "wt-path-fail") {
				return "", fmt.Errorf("simulated symlink resolution error for current worktree")
			}
			return orig(p)
		}

		exitCode, stdout, stderr := runScopecheckInDir(t, wt, map[string]string{
			"SCOPE_CHANGE_ID": "change-path-fail",
			"CI":              "",
		})

		if exitCode == 0 {
			t.Fatalf("expected scopecheck to fail when current worktree path canonicalization fails, got exit 0.\nStdout: %s\nStderr: %s", stdout, stderr)
		}
		if !strings.Contains(stderr, "canonicalize current worktree path") || !strings.Contains(stderr, "simulated symlink resolution error") {
			t.Fatalf("expected clear diagnostic in stderr, got:\n%s", stderr)
		}
	})

	t.Run("other worktree path canonicalization failure fails closed", func(t *testing.T) {
		wtOther := repo.addLinkedWorktree(t, "feat-path-other", "wt-path-other")
		_ = wtOther

		orig := resolveCanonicalPath
		defer func() { resolveCanonicalPath = orig }()

		resolveCanonicalPath = func(p string) (string, error) {
			if strings.Contains(p, "wt-path-other") {
				return "", fmt.Errorf("simulated symlink resolution error for other worktree")
			}
			return orig(p)
		}

		exitCode, stdout, stderr := runScopecheckInDir(t, wt, map[string]string{
			"SCOPE_CHANGE_ID": "change-path-fail",
			"CI":              "",
		})

		if exitCode == 0 {
			t.Fatalf("expected scopecheck to fail when other worktree path canonicalization fails, got exit 0.\nStdout: %s\nStderr: %s", stdout, stderr)
		}
		if !strings.Contains(stderr, "canonicalize worktree path") || !strings.Contains(stderr, "simulated symlink resolution error") {
			t.Fatalf("expected clear diagnostic in stderr, got:\n%s", stderr)
		}
	})
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "architecture.yaml")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find repository root containing architecture.yaml")
		}
		dir = parent
	}
}

func TestScopeWorkflowDocumentsIsolationContract(t *testing.T) {
	root := findRepoRoot(t)

	type docContract struct {
		relPath        string
		uniqueMarker   string
		requiredShared []string
	}

	contracts := []docContract{
		{
			relPath:      "AGENTS.md",
			uniqueMarker: "For parallel delegated packets or dirty-primary working trees, route work through selected linked-worktree mode",
			requiredShared: []string{
				".ai/scope.json",
				"selected linked-worktree mode",
				"applies_to",
				"SCOPE_CHANGE_ID",
				"forbidden in CI",
				"prune Git metadata",
			},
		},
		{
			relPath:      "README.md",
			uniqueMarker: "CI requires full-diff verification against the PR base commit",
			requiredShared: []string{
				"Legacy mode",
				"Selected linked-worktree mode",
				"SCOPE_CHANGE_ID",
				"forbidden in CI",
				"prune Git metadata",
			},
		},
		{
			relPath:      "skills/expand-implementation/references/handoff-protocol.md",
			uniqueMarker: "Active Ownership Preflight",
			requiredShared: []string{
				"Repository Baseline",
				"Linked-Worktree Root",
				"git -C $repositoryRoot worktree add --detach",
				"git -C $repositoryRoot worktree prune",
				"SCOPE_CHANGE_ID",
				"forbidden in CI",
			},
		},
		{
			relPath:      "skills/site/SKILL.md",
			uniqueMarker: "6. For non-trivial edits in a clean/single-task worktree",
			requiredShared: []string{
				"clean/single-task worktree",
				"selected linked-worktree mode",
				"applies_to",
				"SCOPE_CHANGE_ID",
				"forbidden in CI",
				"prune Git metadata",
			},
		},
		{
			relPath:      "skills/site/references/change-safety.md",
			uniqueMarker: "Establish the appropriate scope gate for the working tree",
			requiredShared: []string{
				"clean/single-task working tree",
				"SCOPE_CHANGE_ID",
				"applies_to",
				"forbidden in CI",
				"prune Git metadata",
			},
		},
		{
			relPath:      "skills/site/references/delivery-lifecycle.md",
			uniqueMarker: "Establish the slice scope gate:",
			requiredShared: []string{
				"clean/single-task worktree",
				"SCOPE_CHANGE_ID",
				"applies_to",
				"forbidden in CI",
				"prune Git metadata",
			},
		},
		{
			relPath:      "skills/site/references/integration-planning.md",
			uniqueMarker: "plan for isolated OS-temporary linked worktrees",
			requiredShared: []string{
				"clean/single-task working trees",
				"SCOPE_CHANGE_ID",
				"applies_to",
				"forbidden in CI",
				"prune Git metadata",
			},
		},
		{
			relPath:      "skills/site/references/user-walkthrough.md",
			uniqueMarker: "before runtime UX acceptance",
			requiredShared: []string{
				"selected linked-worktree mode",
				"SCOPE_CHANGE_ID",
				"forbidden in CI",
				"prune Git metadata",
			},
		},
		{
			relPath:      "workflows/safe-change.md",
			uniqueMarker: "- parallel/dirty worktrees: preflight ownership (stop on overlap) -> OS-temp linked worktree -> $env:SCOPE_CHANGE_ID='<id>'",
			requiredShared: []string{
				"clean/single-task",
				"SCOPE_CHANGE_ID",
				"forbidden in CI",
				"prune Git metadata",
			},
		},
	}

	// Read all files
	docContents := make(map[string]string, len(contracts))
	for _, c := range contracts {
		fullPath := filepath.Join(root, filepath.FromSlash(c.relPath))
		contentBytes, err := os.ReadFile(fullPath)
		if err != nil {
			t.Fatalf("failed to read governance file %s: %v", c.relPath, err)
		}
		docContents[c.relPath] = string(contentBytes)
	}

	// Check each document against its unique and shared obligations
	for _, c := range contracts {
		c := c
		t.Run(c.relPath, func(t *testing.T) {
			content := docContents[c.relPath]

			// Assert unique marker is present in this file
			if !strings.Contains(content, c.uniqueMarker) {
				t.Errorf("governance file %s missing required unique clause marker: %q", c.relPath, c.uniqueMarker)
			}

			// Assert unique marker is NOT present in any other governance file
			for otherPath, otherContent := range docContents {
				if otherPath == c.relPath {
					continue
				}
				if strings.Contains(otherContent, c.uniqueMarker) {
					t.Errorf("unique clause marker %q for %s was unexpectedly found in %s", c.uniqueMarker, c.relPath, otherPath)
				}
			}

			// Assert all required shared markers are present in this file
			for _, marker := range c.requiredShared {
				if !strings.Contains(content, marker) {
					t.Errorf("governance file %s missing required shared clause marker: %q", c.relPath, marker)
				}
			}
		})
	}
}
