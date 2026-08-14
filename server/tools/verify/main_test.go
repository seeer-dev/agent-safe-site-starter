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
