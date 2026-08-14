package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateControlAcceptsApplyingChange(t *testing.T) {
	path := writeChange(t, "Applying", "pending", "")
	_, problems := validateControl(path)
	if len(problems) > 0 {
		t.Fatalf("validateControl() problems:\n%s", strings.Join(problems, "\n"))
	}
}

func TestValidateControlAcceptsAcceptedWithPassedEvidence(t *testing.T) {
	path := writeChange(t, "Accepted", "passed", "go test ./server/tools/speccheck: passed")
	_, problems := validateControl(path)
	if len(problems) > 0 {
		t.Fatalf("validateControl() problems:\n%s", strings.Join(problems, "\n"))
	}
}

func TestValidateControlRejectsAcceptedWithoutPassedEvidence(t *testing.T) {
	path := writeChange(t, "Accepted", "pending", "")
	_, problems := validateControl(path)
	assertProblemContains(t, problems, "status Accepted requires passed evidence for REQ-001")
}

func TestValidateControlRejectsContradictoryPassedProof(t *testing.T) {
	path := writeChange(t, "Applying", "passed", "automated checks pass; user walkthrough not yet captured")
	_, problems := validateControl(path)
	assertProblemContains(t, problems, "proof contains unresolved-gap phrase \"not yet captured\"")
}

func TestValidateControlAcceptsNegativeProofLanguageWithoutGap(t *testing.T) {
	path := writeChange(t, "Applying", "passed", "browser does not run database queries; wrong tokens are rejected")
	_, problems := validateControl(path)
	if len(problems) > 0 {
		t.Fatalf("validateControl() problems:\n%s", strings.Join(problems, "\n"))
	}
}

func TestValidateControlRejectsAllGatesResolvedWithPendingEvidence(t *testing.T) {
	path := writeChange(t, "Applying", "pending", "")
	rewriteControl(t, path, func(control *Control) {
		control.Evidence["REQ-001"] = EvidenceRecord{Status: "passed", Proof: "all gates resolved"}
	})
	rewriteEvidence(t, path, map[string]EvidenceRecord{
		"REQ-001": {Status: "passed", Proof: "all gates resolved"},
		"AC-001":  {Status: "pending", Proof: ""},
	})
	_, problems := validateControl(path)
	assertProblemContains(t, problems, "claims all gates resolved while other evidence is not passed")
}

func TestValidateControlStrictEvidenceRequiresCurrentRevision(t *testing.T) {
	path := writeChange(t, "Applying", "passed", "go test ./server/tools/speccheck: passed")
	rewriteControl(t, path, func(control *Control) {
		control.StrictEvidence = true
		for id, record := range control.Evidence {
			record.ObservedRevision = control.Revision - 1
			control.Evidence[id] = record
		}
	})
	_, problems := validateControl(path)
	assertProblemContains(t, problems, "observed_revision 0 must equal control revision 1")
}

func TestValidateControlStrictEvidenceRequiresDeclaredReceipt(t *testing.T) {
	path := writeChange(t, "Applying", "passed", "walkthrough receipt required")
	rewriteControl(t, path, func(control *Control) {
		control.StrictEvidence = true
		control.EvidenceRequirements = map[string][]string{"AC-001": []string{"walkthrough"}}
		for id, record := range control.Evidence {
			record.ObservedRevision = control.Revision
			control.Evidence[id] = record
		}
	})
	_, problems := validateControl(path)
	assertProblemContains(t, problems, "passed evidence AC-001 requires walkthrough receipt")
}

func TestValidateControlStrictEvidenceAcceptsCurrentReceipt(t *testing.T) {
	proof := "walkthrough passed; receipt receipts/example-walkthrough.md"
	path := writeChange(t, "Applying", "passed", proof)
	receiptPath := filepath.Join(filepath.Dir(path), "receipts", "example-walkthrough.md")
	if err := os.MkdirAll(filepath.Dir(receiptPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(receiptPath, []byte("Source revision: example\nObserved behavior: passed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rewriteControl(t, path, func(control *Control) {
		control.StrictEvidence = true
		control.EvidenceRequirements = map[string][]string{"AC-001": []string{"walkthrough"}}
		for id, record := range control.Evidence {
			record.ObservedRevision = control.Revision
			if id == "AC-001" {
				record.Receipts = []EvidenceReceipt{{Kind: "walkthrough", Path: "receipts/example-walkthrough.md"}}
			}
			control.Evidence[id] = record
		}
	})
	_, problems := validateControl(path)
	if len(problems) > 0 {
		t.Fatalf("validateControl() problems:\n%s", strings.Join(problems, "\n"))
	}
}

func TestValidateControlRejectsUnsafeReceiptPath(t *testing.T) {
	path := writeChange(t, "Applying", "passed", "receipt ../outside.md")
	rewriteControl(t, path, func(control *Control) {
		control.StrictEvidence = true
		for id, record := range control.Evidence {
			record.ObservedRevision = control.Revision
			if id == "AC-001" {
				record.Receipts = []EvidenceReceipt{{Kind: "walkthrough", Path: "../outside.md"}}
			}
			control.Evidence[id] = record
		}
	})
	_, problems := validateControl(path)
	assertProblemContains(t, problems, "receipt path \"../outside.md\" must be a safe relative file")
}

func TestValidateControlRejectsMissingReceiptFile(t *testing.T) {
	path := writeChange(t, "Applying", "passed", "receipt receipts/missing.md")
	rewriteControl(t, path, func(control *Control) {
		control.StrictEvidence = true
		for id, record := range control.Evidence {
			record.ObservedRevision = control.Revision
			if id == "AC-001" {
				record.Receipts = []EvidenceReceipt{{Kind: "walkthrough", Path: "receipts/missing.md"}}
			}
			control.Evidence[id] = record
		}
	})
	_, problems := validateControl(path)
	assertProblemContains(t, problems, "receipt receipts/missing.md: read:")
}

func TestValidateControlRejectsArtifactDrift(t *testing.T) {
	path := writeChange(t, "Applying", "pending", "")
	planPath := filepath.Join(filepath.Dir(path), "plan.md")
	if err := os.WriteFile(planPath, []byte("# Plan\n\nREQ-001 only\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, problems := validateControl(path)
	assertProblemContains(t, problems, "plan.md does not trace AC-001")
}

func TestCoveringChangesRequiresOneAuthorizingSpec(t *testing.T) {
	controls := []controlledChange{
		{Dir: "specs/changes/draft-change", Control: Control{ChangeID: "draft-change", Status: "Draft", AppliesTo: []string{"server/internal/**"}}},
		{Dir: "specs/changes/active-change", Control: Control{ChangeID: "active-change", Status: "Applying", AppliesTo: []string{"server/internal/**"}}},
	}
	changedDirs := map[string]struct{}{
		"specs/changes/draft-change":  {},
		"specs/changes/active-change": {},
	}
	owners := coveringChanges("server/internal/config/config.go", controls, changedDirs, false)
	if len(owners) != 1 || owners[0] != "active-change" {
		t.Fatalf("coveringChanges() = %v, want [active-change]", owners)
	}
	if owners := coveringChanges("server/internal/config/config.go", controls, map[string]struct{}{}, false); len(owners) != 0 {
		t.Fatalf("unchanged controlled specs must not authorize a new diff, got %v", owners)
	}

	controls = append(controls, controlledChange{Dir: "specs/changes/second-change", Control: Control{ChangeID: "second-change", Status: "Ready", AppliesTo: []string{"server/internal/config/config.go"}}})
	changedDirs["specs/changes/second-change"] = struct{}{}
	owners = coveringChanges("server/internal/config/config.go", controls, changedDirs, false)
	if len(owners) != 2 {
		t.Fatalf("coveringChanges() = %v, want two owners", owners)
	}

	if owners := coveringChanges("server/internal/config/config.go", controls, changedDirs, true); len(owners) != 0 {
		t.Fatalf("CI Accepted-only coveringChanges() = %v, want none", owners)
	}
}

func TestAcceptedAndSupersededControlsAreImmutable(t *testing.T) {
	for _, status := range []string{"Accepted", "Superseded"} {
		if !immutableStatus(status) {
			t.Errorf("immutableStatus(%q) = false, want true", status)
		}
	}
	for _, status := range []string{"Draft", "Ready", "Applying", "Verifying"} {
		if immutableStatus(status) {
			t.Errorf("immutableStatus(%q) = true, want false", status)
		}
	}
}

func TestRequiresControlledSpecProtectsImplementationAndGovernance(t *testing.T) {
	tests := map[string]bool{
		"server/internal/config/config.go": true,
		"server/tools/speccheck/main.go":   true,
		"server/tools/verify/main.go":      true,
		".github/workflows/ci.yml":         true,
		"skills/site/SKILL.md":             true,
		"AGENTS.md":                        true,
		"architecture.yaml":                true,
		".gitignore":                       true,
		"specs/changes/example/spec.md":    false,
		".ai/scope.json":                   false,
		"docs/architecture.md":             false,
		"README.md":                        false,
	}
	for path, want := range tests {
		if got := requiresControlledSpec(path); got != want {
			t.Errorf("requiresControlledSpec(%q) = %v, want %v", path, got, want)
		}
	}
}

func TestGovernanceWiringKeepsSpeccheckInVerifyAndCI(t *testing.T) {
	root := repositoryRoot(t)
	assertFileContains(t, filepath.Join(root, "server", "tools", "verify", "main.go"), `{"go", "run", "./server/tools/speccheck"}`)
	assertFileContains(t, filepath.Join(root, ".github", "workflows", "ci.yml"), "fetch-depth: 0")
	assertFileContains(t, filepath.Join(root, ".github", "workflows", "ci.yml"), "branches: [main, master]")
	assertFileContains(t, filepath.Join(root, ".github", "workflows", "ci.yml"), "SPEC_BASE_REF:")
	assertFileContains(t, filepath.Join(root, ".github", "workflows", "ci.yml"), "SPEC_REQUIRE_ACCEPTED:")
	assertFileContains(t, filepath.Join(root, ".github", "workflows", "ci.yml"), "go run ./server/tools/speccheck")
}

func TestSiteSkillRequiresReachabilityRecoveryReceiptsAndIndependentReplay(t *testing.T) {
	root := repositoryRoot(t)
	assertFileContains(t, filepath.Join(root, "skills", "site", "SKILL.md"), "real entry point and identity/session producer")
	assertFileContains(t, filepath.Join(root, "skills", "site", "references", "integration-planning.md"), "A symbol with no caller is not a consumer")
	assertFileContains(t, filepath.Join(root, "skills", "site", "references", "integration-planning.md"), "authoritative empty result")
	assertFileContains(t, filepath.Join(root, "skills", "site", "references", "auth.md"), "Contact-field equality alone is never sufficient")
	assertFileContains(t, filepath.Join(root, "skills", "site", "references", "user-walkthrough.md"), "walkthrough recommended")
	assertFileContains(t, filepath.Join(root, "skills", "site", "references", "spec-driven-delivery.md"), "strict_evidence: true")
	assertFileContains(t, filepath.Join(root, "skills", "site", "references", "delivery-lifecycle.md"), "implementation report from another agent is a handoff")
}

func writeChange(t *testing.T, status, evidenceStatus, proof string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "example-change")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	control := Control{
		ChangeID:           "example-change",
		Revision:           1,
		Status:             status,
		DecisionAuthority:  "Repository owner/user",
		ApprovalBasis:      "User approved revision 1 in the tracked task request.",
		RepositoryBaseline: strings.Repeat("a", 40),
		Supersedes:         []string{},
		AppliesTo:          []string{"server/internal/**"},
		Requirements:       []string{"REQ-001"},
		Acceptance:         []string{"AC-001"},
		Evidence: map[string]EvidenceRecord{
			"REQ-001": {Status: evidenceStatus, Proof: proof},
			"AC-001":  {Status: evidenceStatus, Proof: proof},
		},
	}
	body, err := json.MarshalIndent(control, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "control.json")
	if err := os.WriteFile(path, append(body, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	spec := `# Example

Change ID: example-change
Revision: 1
Status: ` + status + `
Decision authority: Repository owner/user
Approval basis: User approved revision 1 in the tracked task request.
Repository baseline: ` + strings.Repeat("a", 40) + `
Supersedes: none

### REQ-001: Example behavior
The system MUST reject invalid state.

#### AC-001: Invalid state
- GIVEN invalid state exists
- WHEN validation runs
- THEN validation fails
`
	plan := "# Plan\n\nScope: server/internal/**\n\n| REQ/AC | Slice |\n|---|---|\n| REQ-001 | Slice 1 |\n| AC-001 | Slice 1 |\n"
	evidence := "# Evidence\n\n| REQ/AC | Status | Proof |\n|---|---|---|\n| REQ-001 | " + evidenceStatus + " | " + proof + " |\n| AC-001 | " + evidenceStatus + " | " + proof + " |\n"
	for name, content := range map[string]string{"spec.md": spec, "plan.md": plan, "evidence.md": evidence} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return path
}

func rewriteControl(t *testing.T, path string, mutate func(*Control)) {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var control Control
	if err := json.Unmarshal(body, &control); err != nil {
		t.Fatal(err)
	}
	mutate(&control)
	body, err = json.MarshalIndent(control, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(body, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}

func rewriteEvidence(t *testing.T, controlPath string, records map[string]EvidenceRecord) {
	t.Helper()
	body := "# Evidence\n\n| REQ/AC | Status | Proof |\n|---|---|---|\n"
	for _, id := range []string{"REQ-001", "AC-001"} {
		record := records[id]
		body += "| " + id + " | " + record.Status + " | " + record.Proof + " |\n"
	}
	if err := os.WriteFile(filepath.Join(filepath.Dir(controlPath), "evidence.md"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertProblemContains(t *testing.T, problems []string, substring string) {
	t.Helper()
	for _, problem := range problems {
		if strings.Contains(problem, substring) {
			return
		}
	}
	t.Fatalf("problems do not contain %q:\n%s", substring, strings.Join(problems, "\n"))
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("repository root not found")
		}
		dir = parent
	}
}

func assertFileContains(t *testing.T, path, substring string) {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), substring) {
		t.Fatalf("%s does not contain %q", path, substring)
	}
}
