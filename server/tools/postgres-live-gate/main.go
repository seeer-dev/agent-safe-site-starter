package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
)

// RequiredLiveTests maps each package to its exact required live test names.
var RequiredLiveTests = map[string][]string{
	"github.com/example/ai-site-starter/server/internal/migrate": {
		"TestPostgresLiveMigrateApplyAndTimestamp",
	},
	"github.com/example/ai-site-starter/server/internal/modules/staff": {
		"TestPostgresLiveStaffLockActiveOwners",
	},
	"github.com/example/ai-site-starter/server/internal/modules/media": {
		"TestPostgresLiveMediaLifecycleAndLocks",
	},
}

// TestEvent represents a line emitted by `go test -json`.
type TestEvent struct {
	Action  string  `json:"Action"`
	Package string  `json:"Package"`
	Test    string  `json:"Test"`
	Output  string  `json:"Output"`
	Elapsed float64 `json:"Elapsed"`
}

func main() {
	log.SetFlags(0)
	if err := RunLiveGate(os.Stdout, os.Stderr); err != nil {
		log.Fatalf("postgres-live-gate failed: %v", err)
	}
	fmt.Println("postgres-live-gate: ok (all required live PostgreSQL tests passed)")
}

// RunLiveGate runs `go test -json -run '^TestPostgresLive' ./server/...` and verifies
// that every required test executed and passed without skip or failure.
func RunLiveGate(stdout, stderr io.Writer) error {
	cmd := exec.Command("go", "test", "-json", "-run", "^TestPostgresLive", "./server/...")
	cmd.Env = os.Environ()

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("create stdout pipe: %w", err)
	}
	cmd.Stderr = stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start go test: %w", err)
	}

	// Tee stdout to stdout while parsing events
	tee := io.TeeReader(stdoutPipe, stdout)
	valErr := ValidateTestEvents(tee, RequiredLiveTests)

	cmdErr := cmd.Wait()
	return EvaluateGateResult(valErr, cmdErr)
}

// EvaluateGateResult determines the final gate error based on event validation and process exit results.
func EvaluateGateResult(valErr, cmdErr error) error {
	if cmdErr != nil {
		if valErr != nil {
			return fmt.Errorf("tests failed (%v) and process exited: %w", valErr, cmdErr)
		}
		return fmt.Errorf("test process exited with error: %w", cmdErr)
	}
	if valErr != nil {
		return valErr
	}
	return nil
}

// ValidateTestEvents parses `go test -json` stream and asserts that every required test
// passed and no required test was skipped, missing, or failed.
func ValidateTestEvents(r io.Reader, required map[string][]string) error {
	scanner := bufio.NewScanner(r)

	// Map of package -> test -> final status (pass/fail/skip)
	passed := make(map[string]map[string]bool)
	skipped := make(map[string]map[string]bool)
	failed := make(map[string]map[string]bool)

	for pkg := range required {
		passed[pkg] = make(map[string]bool)
		skipped[pkg] = make(map[string]bool)
		failed[pkg] = make(map[string]bool)
	}

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var event TestEvent
		if err := json.Unmarshal(line, &event); err != nil {
			// Non-JSON output line, ignore or continue
			continue
		}

		if event.Test == "" || event.Package == "" {
			continue
		}

		if _, isRequiredPkg := required[event.Package]; !isRequiredPkg {
			continue
		}

		switch event.Action {
		case "pass":
			passed[event.Package][event.Test] = true
		case "skip":
			skipped[event.Package][event.Test] = true
		case "fail":
			failed[event.Package][event.Test] = true
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan test events: %w", err)
	}

	var errs []string
	for pkg, tests := range required {
		for _, testName := range tests {
			if failed[pkg][testName] {
				errs = append(errs, fmt.Sprintf("test failed: %s %s", pkg, testName))
			} else if skipped[pkg][testName] {
				errs = append(errs, fmt.Sprintf("test was skipped (must run and pass on live PostgreSQL): %s %s", pkg, testName))
			} else if !passed[pkg][testName] {
				errs = append(errs, fmt.Sprintf("required test missing from execution stream: %s %s", pkg, testName))
			}
		}
	}

	if len(errs) > 0 {
		return errors.New(fmt.Sprintf("%d test validation error(s):\n - %s", len(errs), join(errs, "\n - ")))
	}

	return nil
}

func join(items []string, sep string) string {
	if len(items) == 0 {
		return ""
	}
	res := items[0]
	for _, item := range items[1:] {
		res += sep + item
	}
	return res
}
