package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"
)

type mockInstance struct {
	startFunc func() error
	stopFunc  func() error
	starts    int
	stops     int
}

func (m *mockInstance) Start() error {
	m.starts++
	if m.startFunc != nil {
		return m.startFunc()
	}
	return nil
}

func (m *mockInstance) Stop() error {
	m.stops++
	if m.stopFunc != nil {
		return m.stopFunc()
	}
	return nil
}

func TestLauncherSuccessfulRun(t *testing.T) {
	t.Parallel()

	mockDB := &mockInstance{}
	var capturedPort uint32
	var capturedRuntimeDir string
	var capturedDSN string
	var gateInvoked bool
	var cleanedPaths []string

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	launcher := &Launcher{
		NewInstance: func(port uint32, runtimeDir string) (PostgresInstance, string, error) {
			capturedPort = port
			capturedRuntimeDir = runtimeDir
			capturedDSN = fmt.Sprintf("postgres://postgres:postgres@127.0.0.1:%d/postgres?sslmode=disable", port)
			return mockDB, capturedDSN, nil
		},
		RunGate: func(ctx context.Context, dsn string, out, err io.Writer) error {
			gateInvoked = true
			if dsn != capturedDSN {
				t.Errorf("expected dsn %q, got %q", capturedDSN, dsn)
			}
			return nil
		},
		MkdirTemp: func(dir, pattern string) (string, error) {
			return "/fake/tmp/embedded-postgres-test", nil
		},
		RemoveAll: func(path string) error {
			cleanedPaths = append(cleanedPaths, path)
			return nil
		},
		Stdout: stdout,
		Stderr: stderr,
	}

	err := launcher.Run(context.Background(), nil)
	if err != nil {
		t.Fatalf("expected nil error on successful run, got: %v", err)
	}

	if mockDB.starts != 1 {
		t.Errorf("expected 1 start call, got %d", mockDB.starts)
	}
	if mockDB.stops != 1 {
		t.Errorf("expected 1 stop call, got %d", mockDB.stops)
	}
	if !gateInvoked {
		t.Errorf("expected live gate to be invoked")
	}
	if capturedPort != uint32(DefaultPort) {
		t.Errorf("expected default port %d, got %d", DefaultPort, capturedPort)
	}
	if capturedRuntimeDir != "/fake/tmp/embedded-postgres-test" {
		t.Errorf("expected runtime dir /fake/tmp/embedded-postgres-test, got %s", capturedRuntimeDir)
	}
	if len(cleanedPaths) != 1 || cleanedPaths[0] != "/fake/tmp/embedded-postgres-test" {
		t.Errorf("expected runtime dir to be cleaned up, got %v", cleanedPaths)
	}

	outStr := stdout.String()
	if !strings.Contains(outStr, "starting ephemeral PostgreSQL 16 on 127.0.0.1:5433") {
		t.Errorf("stdout missing startup notice: %s", outStr)
	}
	if !strings.Contains(outStr, "ok (all live tests passed and child stopped)") {
		t.Errorf("stdout missing completion notice: %s", outStr)
	}
}

func TestLauncherStartupFailureCleansUpAndDoesNotRunGate(t *testing.T) {
	t.Parallel()

	mockDB := &mockInstance{
		startFunc: func() error {
			return errors.New("port 5433 already in use")
		},
	}
	var gateInvoked bool
	var cleanedPaths []string

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	launcher := &Launcher{
		NewInstance: func(port uint32, runtimeDir string) (PostgresInstance, string, error) {
			return mockDB, "postgres://postgres:postgres@127.0.0.1:5433/postgres?sslmode=disable", nil
		},
		RunGate: func(ctx context.Context, dsn string, out, err io.Writer) error {
			gateInvoked = true
			return nil
		},
		MkdirTemp: func(dir, pattern string) (string, error) {
			return "/fake/tmp/embedded-postgres-fail-start", nil
		},
		RemoveAll: func(path string) error {
			cleanedPaths = append(cleanedPaths, path)
			return nil
		},
		Stdout: stdout,
		Stderr: stderr,
	}

	err := launcher.Run(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error on startup failure, got nil")
	}
	if !strings.Contains(err.Error(), "start ephemeral postgres: port 5433 already in use") {
		t.Fatalf("unexpected error message: %v", err)
	}
	if gateInvoked {
		t.Error("gate runner must not be invoked when database startup fails")
	}
	if mockDB.stops != 0 {
		t.Errorf("stop should not be called if start failed, got %d stop calls", mockDB.stops)
	}
	if len(cleanedPaths) != 1 || cleanedPaths[0] != "/fake/tmp/embedded-postgres-fail-start" {
		t.Errorf("runtime directory must be cleaned up on start failure, got %v", cleanedPaths)
	}
}

func TestLauncherGateFailureStopsDBAndCleansUp(t *testing.T) {
	t.Parallel()

	mockDB := &mockInstance{}
	var cleanedPaths []string

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	launcher := &Launcher{
		NewInstance: func(port uint32, runtimeDir string) (PostgresInstance, string, error) {
			return mockDB, "postgres://postgres:postgres@127.0.0.1:5433/postgres?sslmode=disable", nil
		},
		RunGate: func(ctx context.Context, dsn string, out, err io.Writer) error {
			return errors.New("test failed: TestPostgresLiveMigrateApplyAndTimestamp")
		},
		MkdirTemp: func(dir, pattern string) (string, error) {
			return "/fake/tmp/embedded-postgres-gate-fail", nil
		},
		RemoveAll: func(path string) error {
			cleanedPaths = append(cleanedPaths, path)
			return nil
		},
		Stdout: stdout,
		Stderr: stderr,
	}

	err := launcher.Run(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error on gate failure, got nil")
	}
	if !strings.Contains(err.Error(), "live gate failed: test failed: TestPostgresLiveMigrateApplyAndTimestamp") {
		t.Fatalf("unexpected error message: %v", err)
	}
	if mockDB.starts != 1 {
		t.Errorf("expected 1 start call, got %d", mockDB.starts)
	}
	if mockDB.stops != 1 {
		t.Errorf("expected database to be stopped on gate failure, got %d stops", mockDB.stops)
	}
	if len(cleanedPaths) != 1 || cleanedPaths[0] != "/fake/tmp/embedded-postgres-gate-fail" {
		t.Errorf("runtime directory must be cleaned up on gate failure, got %v", cleanedPaths)
	}
}

func TestLauncherPortValidation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		args      []string
		wantError string
		wantPort  uint32
	}{
		{
			name:      "below minimum port",
			args:      []string{"-port", "80"},
			wantError: "invalid port 80: must be between 1024 and 65535",
		},
		{
			name:      "just below minimum port",
			args:      []string{"-port", "1023"},
			wantError: "invalid port 1023: must be between 1024 and 65535",
		},
		{
			name:      "above maximum port",
			args:      []string{"-port", "70000"},
			wantError: "invalid port 70000: must be between 1024 and 65535",
		},
		{
			name:     "valid custom port",
			args:     []string{"-port", "5434"},
			wantPort: 5434,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var capturedPort uint32
			mockDB := &mockInstance{}

			launcher := &Launcher{
				NewInstance: func(port uint32, runtimeDir string) (PostgresInstance, string, error) {
					capturedPort = port
					return mockDB, fmt.Sprintf("postgres://postgres:postgres@127.0.0.1:%d/postgres?sslmode=disable", port), nil
				},
				RunGate: func(ctx context.Context, dsn string, out, err io.Writer) error {
					return nil
				},
				MkdirTemp: func(dir, pattern string) (string, error) {
					return "/fake/tmp/port-test", nil
				},
				RemoveAll: func(path string) error {
					return nil
				},
				Stdout: io.Discard,
				Stderr: io.Discard,
			}

			err := launcher.Run(context.Background(), tc.args)
			if tc.wantError != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tc.wantError)
				}
				if !strings.Contains(err.Error(), tc.wantError) {
					t.Fatalf("expected error containing %q, got: %v", tc.wantError, err)
				}
			} else {
				if err != nil {
					t.Fatalf("expected nil error, got: %v", err)
				}
				if capturedPort != tc.wantPort {
					t.Errorf("expected port %d, got %d", tc.wantPort, capturedPort)
				}
			}
		})
	}
}

func TestLauncherRejectsPositionalArguments(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
	}{
		{
			name: "single bare argument",
			args: []string{"unexpected_arg"},
		},
		{
			name: "flag followed by positional argument",
			args: []string{"-port", "5433", "extra_arg"},
		},
		{
			name: "multiple positional arguments",
			args: []string{"foo", "bar"},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var tempDirCreated bool
			mockDB := &mockInstance{}

			launcher := &Launcher{
				NewInstance: func(port uint32, runtimeDir string) (PostgresInstance, string, error) {
					return mockDB, "", nil
				},
				RunGate: func(ctx context.Context, dsn string, out, err io.Writer) error {
					return nil
				},
				MkdirTemp: func(dir, pattern string) (string, error) {
					tempDirCreated = true
					return "/fake/tmp/positional-test", nil
				},
				RemoveAll: func(path string) error {
					return nil
				},
				Stdout: io.Discard,
				Stderr: io.Discard,
			}

			err := launcher.Run(context.Background(), tc.args)
			if err == nil {
				t.Fatal("expected error on positional arguments, got nil")
			}
			if !strings.Contains(err.Error(), "unexpected positional argument") {
				t.Fatalf("expected error mentioning positional argument, got: %v", err)
			}
			if tempDirCreated {
				t.Error("temp directory must not be created when positional arguments are present")
			}
			if mockDB.starts != 0 {
				t.Error("database must not be started when positional arguments are present")
			}
		})
	}
}

func TestLauncherEnvironmentIsolation(t *testing.T) {
	// Not t.Parallel because we inspect os.Getenv
	originalEnv := os.Getenv("TEST_DATABASE_URL")
	defer func() {
		if originalEnv == "" {
			_ = os.Unsetenv("TEST_DATABASE_URL")
		} else {
			_ = os.Setenv("TEST_DATABASE_URL", originalEnv)
		}
	}()
	_ = os.Setenv("TEST_DATABASE_URL", "original_value_must_not_change")

	mockDB := &mockInstance{}
	var childDSN string

	launcher := &Launcher{
		NewInstance: func(port uint32, runtimeDir string) (PostgresInstance, string, error) {
			dsn := fmt.Sprintf("postgres://postgres:postgres@127.0.0.1:%d/postgres?sslmode=disable", port)
			return mockDB, dsn, nil
		},
		RunGate: func(ctx context.Context, dsn string, out, err io.Writer) error {
			childDSN = dsn
			return nil
		},
		MkdirTemp: func(dir, pattern string) (string, error) {
			return "/fake/tmp/env-isolation-test", nil
		},
		RemoveAll: func(path string) error {
			return nil
		},
		Stdout: io.Discard,
		Stderr: io.Discard,
	}

	err := launcher.Run(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if childDSN != "postgres://postgres:postgres@127.0.0.1:5433/postgres?sslmode=disable" {
		t.Errorf("unexpected child DSN: %s", childDSN)
	}

	if parentVal := os.Getenv("TEST_DATABASE_URL"); parentVal != "original_value_must_not_change" {
		t.Errorf("parent process TEST_DATABASE_URL was mutated to: %q", parentVal)
	}
}

func TestBuildChildEnv(t *testing.T) {
	t.Parallel()

	baseEnv := []string{
		"PATH=/usr/bin:/bin",
		"TEST_DATABASE_URL=postgres://old_user:old_pass@localhost:5432/old_db",
		"test_database_url=postgres://lowercase@localhost:5432/db",
		"Test_Database_Url=postgres://mixedcase@localhost:5432/db",
		"APP_ENV=development",
		"OTHER_VAR=value=with=equals",
	}

	dsn := "postgres://postgres:postgres@127.0.0.1:5433/postgres?sslmode=disable"
	childEnv := BuildChildEnv(baseEnv, dsn)

	var foundCount int
	var foundVal string
	for _, env := range childEnv {
		key, val, found := strings.Cut(env, "=")
		if found && strings.EqualFold(key, "TEST_DATABASE_URL") {
			foundCount++
			foundVal = val
			if key != "TEST_DATABASE_URL" {
				t.Errorf("expected canonical key TEST_DATABASE_URL, got %s", key)
			}
		}
	}

	if foundCount != 1 {
		t.Fatalf("expected exactly 1 TEST_DATABASE_URL entry in child env, found %d", foundCount)
	}
	if foundVal != dsn {
		t.Fatalf("expected injected DSN %q, got %q", dsn, foundVal)
	}

	// Verify preservation of other variables
	var hasPath, hasAppEnv, hasOther bool
	for _, env := range childEnv {
		switch {
		case env == "PATH=/usr/bin:/bin":
			hasPath = true
		case env == "APP_ENV=development":
			hasAppEnv = true
		case env == "OTHER_VAR=value=with=equals":
			hasOther = true
		}
	}

	if !hasPath || !hasAppEnv || !hasOther {
		t.Errorf("expected unrelated environment variables to be preserved: path=%v appEnv=%v other=%v", hasPath, hasAppEnv, hasOther)
	}
}

func TestBuildGateCommandProperties(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	baseEnv := []string{"PATH=/bin", "TEST_DATABASE_URL=stale_value"}
	dsn := "postgres://postgres:postgres@127.0.0.1:5433/postgres?sslmode=disable"
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	cmd := buildGateCommand(ctx, baseEnv, dsn, stdout, stderr)

	expectedArgs := []string{"go", "run", "./server/tools/postgres-live-gate"}
	if !reflect.DeepEqual(cmd.Args, expectedArgs) {
		t.Errorf("expected fixed command args %v, got %v", expectedArgs, cmd.Args)
	}

	if cmd.Stdout != stdout {
		t.Error("cmd.Stdout was not wired to provided stdout")
	}
	if cmd.Stderr != stderr {
		t.Error("cmd.Stderr was not wired to provided stderr")
	}
	if cmd.Stdin != os.Stdin {
		t.Error("cmd.Stdin was not wired to os.Stdin")
	}

	// Check child environment contains the filtered DSN
	var dsnCount int
	for _, env := range cmd.Env {
		if strings.HasPrefix(env, "TEST_DATABASE_URL=") {
			dsnCount++
			if env != "TEST_DATABASE_URL="+dsn {
				t.Errorf("unexpected TEST_DATABASE_URL in cmd.Env: %s", env)
			}
		}
	}
	if dsnCount != 1 {
		t.Errorf("expected exactly 1 TEST_DATABASE_URL entry in cmd.Env, got %d", dsnCount)
	}
}

func TestLauncherStopWarningPreservesGateError(t *testing.T) {
	t.Parallel()

	mockDB := &mockInstance{
		stopFunc: func() error {
			return errors.New("cleanup warning: child process did not stop cleanly")
		},
	}

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	launcher := &Launcher{
		NewInstance: func(port uint32, runtimeDir string) (PostgresInstance, string, error) {
			return mockDB, "postgres://postgres:postgres@127.0.0.1:5433/postgres?sslmode=disable", nil
		},
		RunGate: func(ctx context.Context, dsn string, out, err io.Writer) error {
			return errors.New("gate failed")
		},
		MkdirTemp: func(dir, pattern string) (string, error) {
			return "/fake/tmp/stop-warn", nil
		},
		RemoveAll: func(path string) error {
			return nil
		},
		Stdout: stdout,
		Stderr: stderr,
	}

	err := launcher.Run(context.Background(), nil)
	if err == nil {
		t.Fatal("expected gate error, got nil")
	}
	if !strings.Contains(err.Error(), "live gate failed: gate failed") {
		t.Errorf("expected gate error to be preserved, got: %v", err)
	}
	if !strings.Contains(stderr.String(), "warning: failed to stop embedded postgres") {
		t.Errorf("expected stop warning in stderr, got: %s", stderr.String())
	}
}

func TestValidatePortBoundaries(t *testing.T) {
	t.Parallel()

	if err := ValidatePort(0); err == nil {
		t.Error("expected port 0 to be invalid")
	}
	if err := ValidatePort(80); err == nil {
		t.Error("expected port 80 to be invalid")
	}
	if err := ValidatePort(1023); err == nil {
		t.Error("expected port 1023 to be invalid")
	}
	if err := ValidatePort(1024); err != nil {
		t.Errorf("expected port 1024 to be valid, got: %v", err)
	}
	if err := ValidatePort(5432); err != nil {
		t.Errorf("expected port 5432 to be valid, got: %v", err)
	}
	if err := ValidatePort(5433); err != nil {
		t.Errorf("expected port 5433 to be valid, got: %v", err)
	}
	if err := ValidatePort(65535); err != nil {
		t.Errorf("expected port 65535 to be valid, got: %v", err)
	}
	if err := ValidatePort(65536); err == nil {
		t.Error("expected port 65536 to be invalid")
	}
}
