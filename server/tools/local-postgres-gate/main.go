package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
)

const (
	DefaultPort uint = 5433
	MinPort     uint = 1024
	MaxPort     uint = 65535
)

// PostgresInstance abstracts database start and stop operations for testing.
type PostgresInstance interface {
	Start() error
	Stop() error
}

// InstanceFactory creates a PostgresInstance given the loopback port and runtime directory.
type InstanceFactory func(port uint32, runtimeDir string) (PostgresInstance, string, error)

// GateRunner invokes the downstream postgres-live-gate process with the provided DSN.
type GateRunner func(ctx context.Context, dsn string, stdout, stderr io.Writer) error

// Launcher coordinates the ephemeral PostgreSQL lifecycle and live gate execution.
type Launcher struct {
	NewInstance InstanceFactory
	RunGate     GateRunner
	MkdirTemp   func(dir, pattern string) (string, error)
	RemoveAll   func(path string) error
	Stdout      io.Writer
	Stderr      io.Writer
}

// DefaultLauncher constructs a production Launcher using real embedded PostgreSQL and exec.
func DefaultLauncher() *Launcher {
	return &Launcher{
		NewInstance: defaultNewInstance,
		RunGate:     defaultRunGate,
		MkdirTemp:   os.MkdirTemp,
		RemoveAll:   os.RemoveAll,
		Stdout:      os.Stdout,
		Stderr:      os.Stderr,
	}
}

func defaultNewInstance(port uint32, runtimeDir string) (PostgresInstance, string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		cacheDir = filepath.Join(os.TempDir(), "embedded-postgres-cache")
	}
	binariesCacheDir := filepath.Join(cacheDir, "embedded-postgres-binaries")

	cfg := embeddedpostgres.DefaultConfig().
		Version(embeddedpostgres.V16).
		Port(port).
		RuntimePath(runtimeDir).
		DataPath(filepath.Join(runtimeDir, "data")).
		BinariesPath(binariesCacheDir).
		Username("postgres").
		Password("postgres").
		Database("postgres").
		StartParameters(map[string]string{
			"listen_addresses": "127.0.0.1",
		})

	dsn := fmt.Sprintf("postgres://postgres:postgres@127.0.0.1:%d/postgres?sslmode=disable", port)
	db := embeddedpostgres.NewDatabase(cfg)
	return db, dsn, nil
}

// BuildChildEnv constructs an isolated environment slice for the child test gate.
// It removes any existing TEST_DATABASE_URL keys case-insensitively from baseEnv
// before appending exactly one generated TEST_DATABASE_URL entry.
func BuildChildEnv(baseEnv []string, dsn string) []string {
	result := make([]string, 0, len(baseEnv)+1)
	for _, env := range baseEnv {
		key, _, found := strings.Cut(env, "=")
		if found && strings.EqualFold(key, "TEST_DATABASE_URL") {
			continue
		}
		result = append(result, env)
	}
	result = append(result, "TEST_DATABASE_URL="+dsn)
	return result
}

// buildGateCommand constructs the fixed-argument exec.Cmd for the live gate child process.
func buildGateCommand(ctx context.Context, baseEnv []string, dsn string, stdout, stderr io.Writer) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "go", "run", "./server/tools/postgres-live-gate")
	cmd.Env = BuildChildEnv(baseEnv, dsn)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Stdin = os.Stdin
	return cmd
}

func defaultRunGate(ctx context.Context, dsn string, stdout, stderr io.Writer) error {
	cmd := buildGateCommand(ctx, os.Environ(), dsn, stdout, stderr)
	return cmd.Run()
}

// ValidatePort checks that the port is within the allowed non-privileged TCP range.
func ValidatePort(port uint) error {
	if port < MinPort || port > MaxPort {
		return fmt.Errorf("invalid port %d: must be between %d and %d", port, MinPort, MaxPort)
	}
	return nil
}

func main() {
	log.SetFlags(0)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	launcher := DefaultLauncher()
	if err := launcher.Run(ctx, os.Args[1:]); err != nil {
		log.Fatalf("local-postgres-gate: %v", err)
	}
}

// Run parses arguments, prepares the isolated runtime directory, starts ephemeral PostgreSQL 16,
// runs the live test gate with child-only TEST_DATABASE_URL, and cleans up all runtime paths.
func (l *Launcher) Run(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("local-postgres-gate", flag.ContinueOnError)
	fs.SetOutput(l.Stderr)
	var port uint
	fs.UintVar(&port, "port", DefaultPort, "loopback port for ephemeral PostgreSQL 16 (1024-65535)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if fs.NArg() != 0 {
		return fmt.Errorf("unexpected positional argument(s): %s; local-postgres-gate accepts only flags (e.g. -port)", strings.Join(fs.Args(), " "))
	}

	if err := ValidatePort(port); err != nil {
		return err
	}

	runtimeDir, err := l.MkdirTemp("", "embedded-postgres-*")
	if err != nil {
		return fmt.Errorf("create temporary runtime directory: %w", err)
	}
	defer func() {
		if cleanupErr := l.RemoveAll(runtimeDir); cleanupErr != nil {
			fmt.Fprintf(l.Stderr, "local-postgres-gate: warning: failed to cleanup runtime directory %s: %v\n", runtimeDir, cleanupErr)
		}
	}()

	db, dsn, err := l.NewInstance(uint32(port), runtimeDir)
	if err != nil {
		return fmt.Errorf("configure embedded postgres: %w", err)
	}

	fmt.Fprintf(l.Stdout, "local-postgres-gate: starting ephemeral PostgreSQL 16 on 127.0.0.1:%d...\n", port)
	if err := db.Start(); err != nil {
		return fmt.Errorf("start ephemeral postgres: %w", err)
	}
	defer func() {
		if stopErr := db.Stop(); stopErr != nil {
			fmt.Fprintf(l.Stderr, "local-postgres-gate: warning: failed to stop embedded postgres: %v\n", stopErr)
		}
	}()

	fmt.Fprintf(l.Stdout, "local-postgres-gate: postgres ready; invoking postgres-live-gate...\n")
	if err := l.RunGate(ctx, dsn, l.Stdout, l.Stderr); err != nil {
		return fmt.Errorf("live gate failed: %w", err)
	}

	fmt.Fprintf(l.Stdout, "local-postgres-gate: ok (all live tests passed and child stopped)\n")
	return nil
}
