package main

import (
	"errors"
	"strings"
	"testing"
)

func TestGatePassesExactInventory(t *testing.T) {
	t.Parallel()

	jsonStream := `
{"Time":"2026-08-14T20:00:00Z","Action":"run","Package":"github.com/example/ai-site-starter/server/internal/migrate","Test":"TestPostgresLiveMigrateApplyAndTimestamp"}
{"Time":"2026-08-14T20:00:01Z","Action":"pass","Package":"github.com/example/ai-site-starter/server/internal/migrate","Test":"TestPostgresLiveMigrateApplyAndTimestamp"}
{"Time":"2026-08-14T20:00:02Z","Action":"run","Package":"github.com/example/ai-site-starter/server/internal/modules/staff","Test":"TestPostgresLiveStaffLockActiveOwners"}
{"Time":"2026-08-14T20:00:03Z","Action":"pass","Package":"github.com/example/ai-site-starter/server/internal/modules/staff","Test":"TestPostgresLiveStaffLockActiveOwners"}
{"Time":"2026-08-14T20:00:04Z","Action":"run","Package":"github.com/example/ai-site-starter/server/internal/modules/media","Test":"TestPostgresLiveMediaLifecycleAndLocks"}
{"Time":"2026-08-14T20:00:05Z","Action":"pass","Package":"github.com/example/ai-site-starter/server/internal/modules/media","Test":"TestPostgresLiveMediaLifecycleAndLocks"}
`
	err := ValidateTestEvents(strings.NewReader(jsonStream), RequiredLiveTests)
	if err != nil {
		t.Fatalf("expected nil error on valid exact pass stream, got: %v", err)
	}
}

func TestGateRejectsMissingTest(t *testing.T) {
	t.Parallel()

	// Missing media test
	jsonStream := `
{"Time":"2026-08-14T20:00:00Z","Action":"run","Package":"github.com/example/ai-site-starter/server/internal/migrate","Test":"TestPostgresLiveMigrateApplyAndTimestamp"}
{"Time":"2026-08-14T20:00:01Z","Action":"pass","Package":"github.com/example/ai-site-starter/server/internal/migrate","Test":"TestPostgresLiveMigrateApplyAndTimestamp"}
{"Time":"2026-08-14T20:00:02Z","Action":"run","Package":"github.com/example/ai-site-starter/server/internal/modules/staff","Test":"TestPostgresLiveStaffLockActiveOwners"}
{"Time":"2026-08-14T20:00:03Z","Action":"pass","Package":"github.com/example/ai-site-starter/server/internal/modules/staff","Test":"TestPostgresLiveStaffLockActiveOwners"}
`
	err := ValidateTestEvents(strings.NewReader(jsonStream), RequiredLiveTests)
	if err == nil {
		t.Fatal("expected error on missing test, got nil")
	}
	if !strings.Contains(err.Error(), "TestPostgresLiveMediaLifecycleAndLocks") {
		t.Fatalf("expected error to name missing media test, got: %v", err)
	}
}

func TestGateRejectsSkippedTest(t *testing.T) {
	t.Parallel()

	// Staff test was skipped
	jsonStream := `
{"Time":"2026-08-14T20:00:00Z","Action":"run","Package":"github.com/example/ai-site-starter/server/internal/migrate","Test":"TestPostgresLiveMigrateApplyAndTimestamp"}
{"Time":"2026-08-14T20:00:01Z","Action":"pass","Package":"github.com/example/ai-site-starter/server/internal/migrate","Test":"TestPostgresLiveMigrateApplyAndTimestamp"}
{"Time":"2026-08-14T20:00:02Z","Action":"run","Package":"github.com/example/ai-site-starter/server/internal/modules/staff","Test":"TestPostgresLiveStaffLockActiveOwners"}
{"Time":"2026-08-14T20:00:03Z","Action":"skip","Package":"github.com/example/ai-site-starter/server/internal/modules/staff","Test":"TestPostgresLiveStaffLockActiveOwners"}
{"Time":"2026-08-14T20:00:04Z","Action":"run","Package":"github.com/example/ai-site-starter/server/internal/modules/media","Test":"TestPostgresLiveMediaLifecycleAndLocks"}
{"Time":"2026-08-14T20:00:05Z","Action":"pass","Package":"github.com/example/ai-site-starter/server/internal/modules/media","Test":"TestPostgresLiveMediaLifecycleAndLocks"}
`
	err := ValidateTestEvents(strings.NewReader(jsonStream), RequiredLiveTests)
	if err == nil {
		t.Fatal("expected error on skipped test, got nil")
	}
	if !strings.Contains(err.Error(), "was skipped") || !strings.Contains(err.Error(), "TestPostgresLiveStaffLockActiveOwners") {
		t.Fatalf("expected error to name skipped staff test, got: %v", err)
	}
}

func TestGateRejectsFailedTest(t *testing.T) {
	t.Parallel()

	// Migration test failed
	jsonStream := `
{"Time":"2026-08-14T20:00:00Z","Action":"run","Package":"github.com/example/ai-site-starter/server/internal/migrate","Test":"TestPostgresLiveMigrateApplyAndTimestamp"}
{"Time":"2026-08-14T20:00:01Z","Action":"fail","Package":"github.com/example/ai-site-starter/server/internal/migrate","Test":"TestPostgresLiveMigrateApplyAndTimestamp"}
{"Time":"2026-08-14T20:00:02Z","Action":"run","Package":"github.com/example/ai-site-starter/server/internal/modules/staff","Test":"TestPostgresLiveStaffLockActiveOwners"}
{"Time":"2026-08-14T20:00:03Z","Action":"pass","Package":"github.com/example/ai-site-starter/server/internal/modules/staff","Test":"TestPostgresLiveStaffLockActiveOwners"}
{"Time":"2026-08-14T20:00:04Z","Action":"run","Package":"github.com/example/ai-site-starter/server/internal/modules/media","Test":"TestPostgresLiveMediaLifecycleAndLocks"}
{"Time":"2026-08-14T20:00:05Z","Action":"pass","Package":"github.com/example/ai-site-starter/server/internal/modules/media","Test":"TestPostgresLiveMediaLifecycleAndLocks"}
`
	err := ValidateTestEvents(strings.NewReader(jsonStream), RequiredLiveTests)
	if err == nil {
		t.Fatal("expected error on failed test, got nil")
	}
	if !strings.Contains(err.Error(), "test failed") || !strings.Contains(err.Error(), "TestPostgresLiveMigrateApplyAndTimestamp") {
		t.Fatalf("expected error to name failed migration test, got: %v", err)
	}
}

func TestGateRejectsDuplicateRunWithoutPass(t *testing.T) {
	t.Parallel()

	// Migration test has duplicate "run" actions but NO "pass" action
	jsonStream := `
{"Time":"2026-08-14T20:00:00Z","Action":"run","Package":"github.com/example/ai-site-starter/server/internal/migrate","Test":"TestPostgresLiveMigrateApplyAndTimestamp"}
{"Time":"2026-08-14T20:00:01Z","Action":"run","Package":"github.com/example/ai-site-starter/server/internal/migrate","Test":"TestPostgresLiveMigrateApplyAndTimestamp"}
{"Time":"2026-08-14T20:00:02Z","Action":"run","Package":"github.com/example/ai-site-starter/server/internal/modules/staff","Test":"TestPostgresLiveStaffLockActiveOwners"}
{"Time":"2026-08-14T20:00:03Z","Action":"pass","Package":"github.com/example/ai-site-starter/server/internal/modules/staff","Test":"TestPostgresLiveStaffLockActiveOwners"}
{"Time":"2026-08-14T20:00:04Z","Action":"run","Package":"github.com/example/ai-site-starter/server/internal/modules/media","Test":"TestPostgresLiveMediaLifecycleAndLocks"}
{"Time":"2026-08-14T20:00:05Z","Action":"pass","Package":"github.com/example/ai-site-starter/server/internal/modules/media","Test":"TestPostgresLiveMediaLifecycleAndLocks"}
`
	err := ValidateTestEvents(strings.NewReader(jsonStream), RequiredLiveTests)
	if err == nil {
		t.Fatal("expected error on duplicate run without pass, got nil")
	}
	if !strings.Contains(err.Error(), "missing") || !strings.Contains(err.Error(), "TestPostgresLiveMigrateApplyAndTimestamp") {
		t.Fatalf("expected error to contain 'missing' and name TestPostgresLiveMigrateApplyAndTimestamp, got: %v", err)
	}
}

func TestGateFailsOnNonZeroExitEvenWithAllPasses(t *testing.T) {
	t.Parallel()

	err := EvaluateGateResult(nil, errors.New("exit status 1"))
	if err == nil {
		t.Fatal("expected non-nil error when process exit is non-zero, got nil")
	}
	if !strings.Contains(err.Error(), "process exited") {
		t.Fatalf("expected error message to contain 'process exited', got: %v", err)
	}

	if err := EvaluateGateResult(nil, nil); err != nil {
		t.Fatalf("expected nil error when validation and process succeed, got: %v", err)
	}
}
