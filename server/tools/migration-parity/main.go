package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

func main() {
	violations, count, err := checkParity("db/migrations/sqlite", "db/migrations/postgres")
	if err != nil {
		fmt.Fprintf(os.Stderr, "migration-parity: %v\n", err)
		os.Exit(1)
	}
	if len(violations) > 0 {
		fmt.Fprintln(os.Stderr, "migration-parity: violations:")
		for _, v := range violations {
			fmt.Fprintf(os.Stderr, " - %s\n", v)
		}
		os.Exit(1)
	}
	fmt.Printf("migration-parity: ok (%d migration(s) matching)\n", count)
}

func checkParity(sqliteDir, postgresDir string) ([]string, int, error) {
	sqliteFiles, err := listSQLFiles(sqliteDir)
	if err != nil {
		return nil, 0, fmt.Errorf("read sqlite migrations: %w", err)
	}
	postgresFiles, err := listSQLFiles(postgresDir)
	if err != nil {
		return nil, 0, fmt.Errorf("read postgres migrations: %w", err)
	}

	sqliteSet := make(map[string]struct{}, len(sqliteFiles))
	for _, f := range sqliteFiles {
		sqliteSet[f] = struct{}{}
	}

	postgresSet := make(map[string]struct{}, len(postgresFiles))
	for _, f := range postgresFiles {
		postgresSet[f] = struct{}{}
	}

	var violations []string
	for _, f := range sqliteFiles {
		if _, exists := postgresSet[f]; !exists {
			violations = append(violations, fmt.Sprintf("migration %q exists in SQLite (%s) but missing in PostgreSQL (%s)", f, sqliteDir, postgresDir))
		}
	}
	for _, f := range postgresFiles {
		if _, exists := sqliteSet[f]; !exists {
			violations = append(violations, fmt.Sprintf("migration %q exists in PostgreSQL (%s) but missing in SQLite (%s)", f, postgresDir, sqliteDir))
		}
	}
	sort.Strings(violations)

	if len(violations) == 0 {
		return nil, len(sqliteFiles), nil
	}
	return violations, 0, nil
}

func listSQLFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}
