package main

import (
	"strings"
	"testing"
)

func TestVerifyStepsIncludeEssentialGates(t *testing.T) {
	t.Parallel()

	stepStrings := make([]string, len(defaultSteps))
	for i, step := range defaultSteps {
		stepStrings[i] = strings.Join(step, " ")
	}

	requiredSubstrings := []string{
		"server/tools/archcheck",
		"server/tools/migration-parity",
		"server/tools/speccheck",
		"server/tools/scopecheck",
		"go test ./...",
		"-count=10",
		"go vet",
	}

	for _, req := range requiredSubstrings {
		found := false
		for _, s := range stepStrings {
			if strings.Contains(s, req) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("verify steps missing required step containing %q; current steps: %v", req, stepStrings)
		}
	}
}

func TestVerifyConcurrencyStepExactArgv(t *testing.T) {
	t.Parallel()

	expectedConcurrencyStep := []string{
		"go", "test",
		"./server/internal/modules/commerce",
		"./server/internal/modules/staff",
		"./server/internal/modules/media",
		"-count=10",
	}

	found := false
	for _, step := range defaultSteps {
		if len(step) == len(expectedConcurrencyStep) {
			match := true
			for i := range step {
				if step[i] != expectedConcurrencyStep[i] {
					match = false
					break
				}
			}
			if match {
				found = true
				break
			}
		}
	}

	if !found {
		t.Fatalf("verify defaultSteps missing exact concurrency step %v; current steps: %v", expectedConcurrencyStep, defaultSteps)
	}
}
