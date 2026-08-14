package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const changesRoot = "specs/changes"

type Control struct {
	ChangeID             string                    `json:"change_id"`
	Revision             int                       `json:"revision"`
	Status               string                    `json:"status"`
	DecisionAuthority    string                    `json:"decision_authority"`
	ApprovalBasis        string                    `json:"approval_basis"`
	RepositoryBaseline   string                    `json:"repository_baseline"`
	Supersedes           []string                  `json:"supersedes"`
	AppliesTo            []string                  `json:"applies_to"`
	Requirements         []string                  `json:"requirements"`
	Acceptance           []string                  `json:"acceptance"`
	StrictEvidence       bool                      `json:"strict_evidence,omitempty"`
	EvidenceRequirements map[string][]string       `json:"evidence_requirements,omitempty"`
	Evidence             map[string]EvidenceRecord `json:"evidence"`
}

type EvidenceRecord struct {
	Status           string            `json:"status"`
	Proof            string            `json:"proof"`
	ObservedRevision int               `json:"observed_revision,omitempty"`
	Receipts         []EvidenceReceipt `json:"receipts,omitempty"`
}

type EvidenceReceipt struct {
	Kind string `json:"kind"`
	Path string `json:"path"`
}

type controlledChange struct {
	Path    string
	Dir     string
	Control Control
}

var (
	changeIDPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	commitPattern   = regexp.MustCompile(`^[0-9a-f]{40}$`)
	requirementID   = regexp.MustCompile(`^REQ-[0-9]{3,}$`)
	acceptanceID    = regexp.MustCompile(`^AC-[0-9]{3,}$`)
	requirementHead = regexp.MustCompile(`(?m)^### (REQ-[0-9]{3,}):[^\r\n]+$`)
	acceptanceHead  = regexp.MustCompile(`(?m)^#### (AC-[0-9]{3,}):[^\r\n]+$`)
	normativeWord   = regexp.MustCompile(`\b(?:MUST|SHALL)\b`)
	evidenceRow     = regexp.MustCompile(`(?m)^\|\s*((?:REQ|AC)-[0-9]{3,})\s*\|\s*(pending|passed|blocked)\s*\|\s*([^|\r\n]*)\s*\|\s*$`)
	unresolvedGap   = regexp.MustCompile(`(?i)\b(?:not yet (?:captured|validated|verified|approved|run|available|implemented|completed)|(?:has|have|was|were) not (?:captured|validated|verified|approved|run|implemented|completed)|(?:still|remains?) pending|recommended before (?:release|production)|walkthrough (?:is )?recommended)\b`)
)

var allowedReceiptKinds = map[string]struct{}{
	"consumer-reachability":    {},
	"independent-review":       {},
	"production-content-audit": {},
	"security-review":          {},
	"walkthrough":              {},
}

func main() {
	changed, err := changedFiles()
	if err != nil {
		fatal(err)
	}

	controls, violations := loadControls(changesRoot)
	violations = append(violations, validateChangedSpecDirs(changed, controls)...)
	violations = append(violations, validateImmutableControlledSpecs(changed, comparisonBase())...)
	changedSpecDirs := changedSpecDirectories(changed)
	requireAccepted := os.Getenv("SPEC_REQUIRE_ACCEPTED") == "1"

	protectedCount := 0
	for _, path := range changed {
		if !requiresControlledSpec(path) {
			continue
		}
		protectedCount++
		owners := coveringChanges(path, controls, changedSpecDirs, requireAccepted)
		switch len(owners) {
		case 0:
			requiredState := "Ready, Applying, Verifying, or Accepted"
			if requireAccepted {
				requiredState = "Accepted"
			}
			violations = append(violations, fmt.Sprintf("%s: protected change is not covered by exactly one %s controlled spec present in this comparison diff", path, requiredState))
		case 1:
			// Exactly one owner is the intended state.
		default:
			violations = append(violations, fmt.Sprintf("%s: protected change is covered by multiple controlled specs: %s", path, strings.Join(owners, ", ")))
		}
	}

	if len(violations) > 0 {
		sort.Strings(violations)
		fmt.Fprintln(os.Stderr, "speccheck: violations:")
		for _, violation := range violations {
			fmt.Fprintln(os.Stderr, " -", violation)
		}
		os.Exit(1)
	}

	fmt.Printf("speccheck: ok (%d controlled spec(s), %d protected changed file(s))\n", len(controls), protectedCount)
}

func loadControls(root string) ([]controlledChange, []string) {
	paths, err := filepath.Glob(filepath.Join(root, "*", "control.json"))
	if err != nil {
		return nil, []string{fmt.Sprintf("scan controlled specs: %v", err)}
	}
	sort.Strings(paths)

	controls := make([]controlledChange, 0, len(paths))
	var violations []string
	seenIDs := map[string]string{}
	for _, path := range paths {
		change, problems := validateControl(path)
		controls = append(controls, change)
		violations = append(violations, problems...)
		if change.Control.ChangeID == "" {
			continue
		}
		if previous, exists := seenIDs[change.Control.ChangeID]; exists {
			violations = append(violations, fmt.Sprintf("%s: duplicate change_id %q also used by %s", slash(path), change.Control.ChangeID, previous))
		} else {
			seenIDs[change.Control.ChangeID] = slash(path)
		}
	}
	return controls, violations
}

func validateControl(path string) (controlledChange, []string) {
	change := controlledChange{Path: slash(path), Dir: slash(filepath.Dir(path))}
	prefix := change.Path
	body, err := os.ReadFile(path)
	if err != nil {
		return change, []string{fmt.Sprintf("%s: read: %v", prefix, err)}
	}

	decoder := json.NewDecoder(strings.NewReader(string(body)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&change.Control); err != nil {
		return change, []string{fmt.Sprintf("%s: parse: %v", prefix, err)}
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return change, []string{fmt.Sprintf("%s: must contain exactly one JSON object", prefix)}
	}

	control := change.Control
	var problems []string
	problem := func(format string, args ...any) {
		problems = append(problems, fmt.Sprintf("%s: %s", prefix, fmt.Sprintf(format, args...)))
	}

	directoryID := filepath.Base(filepath.Dir(path))
	if !changeIDPattern.MatchString(control.ChangeID) {
		problem("change_id %q must be lowercase hyphen-case", control.ChangeID)
	} else if control.ChangeID != directoryID {
		problem("change_id %q does not match directory %q", control.ChangeID, directoryID)
	}
	if control.Revision < 1 {
		problem("revision must be at least 1")
	}
	if !validStatus(control.Status) {
		problem("status %q is invalid", control.Status)
	}
	if placeholder(control.DecisionAuthority) || agentOnlyAuthority(control.DecisionAuthority) {
		problem("decision_authority must identify a non-agent authority")
	}
	if !commitPattern.MatchString(control.RepositoryBaseline) {
		problem("repository_baseline must be a full 40-character lowercase commit SHA")
	}
	if control.Status != "Draft" && (placeholder(control.ApprovalBasis) || len(strings.TrimSpace(control.ApprovalBasis)) < 20) {
		problem("status %s requires a specific non-placeholder approval_basis", control.Status)
	}
	if control.Status == "Draft" && strings.TrimSpace(control.ApprovalBasis) == "" {
		problem("Draft approval_basis must state pending or identify delegated approval")
	}

	problems = append(problems, validateSupersedes(prefix, control.Supersedes)...)
	problems = append(problems, validatePatterns(prefix, control.AppliesTo)...)
	problems = append(problems, validateIDs(prefix, "requirements", control.Requirements, requirementID)...)
	problems = append(problems, validateIDs(prefix, "acceptance", control.Acceptance, acceptanceID)...)
	problems = append(problems, validateEvidenceControl(prefix, change.Dir, control)...)

	specBody, specProblems := readArtifact(filepath.Join(filepath.Dir(path), "spec.md"))
	planBody, planProblems := readArtifact(filepath.Join(filepath.Dir(path), "plan.md"))
	evidenceBody, evidenceProblems := readArtifact(filepath.Join(filepath.Dir(path), "evidence.md"))
	problems = append(problems, specProblems...)
	problems = append(problems, planProblems...)
	problems = append(problems, evidenceProblems...)

	if specBody != "" {
		problems = append(problems, validateSpec(prefix, specBody, control)...)
	}
	if planBody != "" {
		problems = append(problems, validatePlan(prefix, planBody, control)...)
	}
	if evidenceBody != "" {
		problems = append(problems, validateEvidenceArtifact(prefix, evidenceBody, control)...)
	}

	return change, problems
}

func validateSupersedes(prefix string, values []string) []string {
	seen := map[string]struct{}{}
	var problems []string
	for _, value := range values {
		if !changeIDPattern.MatchString(value) {
			problems = append(problems, fmt.Sprintf("%s: supersedes value %q must be lowercase hyphen-case", prefix, value))
		}
		if _, exists := seen[value]; exists {
			problems = append(problems, fmt.Sprintf("%s: duplicate supersedes value %q", prefix, value))
		}
		seen[value] = struct{}{}
	}
	return problems
}

func validatePatterns(prefix string, patterns []string) []string {
	if len(patterns) == 0 {
		return []string{fmt.Sprintf("%s: applies_to must not be empty", prefix)}
	}
	seen := map[string]struct{}{}
	var problems []string
	for _, raw := range patterns {
		pattern := slash(strings.TrimSpace(raw))
		base := strings.TrimSuffix(pattern, "/**")
		if pattern == "" || pattern == "*" || pattern == "**" || filepath.IsAbs(pattern) || strings.Contains(pattern, "..") || strings.ContainsAny(base, "*?[") {
			problems = append(problems, fmt.Sprintf("%s: applies_to pattern %q must be a narrow relative path or directory/**", prefix, raw))
		}
		if _, exists := seen[pattern]; exists {
			problems = append(problems, fmt.Sprintf("%s: duplicate applies_to pattern %q", prefix, pattern))
		}
		seen[pattern] = struct{}{}
	}
	return problems
}

func validateIDs(prefix, field string, ids []string, pattern *regexp.Regexp) []string {
	if len(ids) == 0 {
		return []string{fmt.Sprintf("%s: %s must not be empty", prefix, field)}
	}
	seen := map[string]struct{}{}
	var problems []string
	for _, id := range ids {
		if !pattern.MatchString(id) {
			problems = append(problems, fmt.Sprintf("%s: invalid %s ID %q", prefix, field, id))
		}
		if _, exists := seen[id]; exists {
			problems = append(problems, fmt.Sprintf("%s: duplicate %s ID %q", prefix, field, id))
		}
		seen[id] = struct{}{}
	}
	return problems
}

func validateEvidenceControl(prefix, changeDir string, control Control) []string {
	expected := append(append([]string{}, control.Requirements...), control.Acceptance...)
	expectedSet := stringSet(expected)
	var problems []string
	hasIncompleteEvidence := false
	for _, id := range expected {
		record, exists := control.Evidence[id]
		if !exists {
			problems = append(problems, fmt.Sprintf("%s: evidence is missing %s", prefix, id))
			continue
		}
		if record.Status != "pending" && record.Status != "passed" && record.Status != "blocked" {
			problems = append(problems, fmt.Sprintf("%s: evidence %s has invalid status %q", prefix, id, record.Status))
		}
		if (record.Status == "passed" || record.Status == "blocked") && placeholder(record.Proof) {
			problems = append(problems, fmt.Sprintf("%s: evidence %s status %s requires proof", prefix, id, record.Status))
		}
		if record.Status != "passed" {
			hasIncompleteEvidence = true
		}
		if record.Status == "passed" {
			if phrase := contradictoryPassedProof(record.Proof); phrase != "" {
				problems = append(problems, fmt.Sprintf("%s: evidence %s is passed but proof contains unresolved-gap phrase %q", prefix, id, phrase))
			}
			if control.StrictEvidence && record.ObservedRevision != control.Revision {
				problems = append(problems, fmt.Sprintf("%s: strict evidence %s observed_revision %d must equal control revision %d", prefix, id, record.ObservedRevision, control.Revision))
			}
		}
		problems = append(problems, validateEvidenceReceipts(prefix, changeDir, id, record, control.EvidenceRequirements[id])...)
		if (control.Status == "Accepted" || control.Status == "Superseded") && record.Status != "passed" {
			problems = append(problems, fmt.Sprintf("%s: status %s requires passed evidence for %s", prefix, control.Status, id))
		}
	}
	for id := range control.Evidence {
		if _, exists := expectedSet[id]; !exists {
			problems = append(problems, fmt.Sprintf("%s: evidence contains undeclared ID %s", prefix, id))
		}
	}
	for id, kinds := range control.EvidenceRequirements {
		if _, exists := expectedSet[id]; !exists {
			problems = append(problems, fmt.Sprintf("%s: evidence_requirements contains undeclared ID %s", prefix, id))
		}
		seen := map[string]struct{}{}
		for _, kind := range kinds {
			if _, ok := allowedReceiptKinds[kind]; !ok {
				problems = append(problems, fmt.Sprintf("%s: evidence_requirements %s has unsupported receipt kind %q", prefix, id, kind))
			}
			if _, duplicate := seen[kind]; duplicate {
				problems = append(problems, fmt.Sprintf("%s: evidence_requirements %s has duplicate receipt kind %q", prefix, id, kind))
			}
			seen[kind] = struct{}{}
		}
	}
	if hasIncompleteEvidence {
		for id, record := range control.Evidence {
			if record.Status == "passed" && strings.Contains(strings.ToLower(record.Proof), "all gates resolved") {
				problems = append(problems, fmt.Sprintf("%s: evidence %s claims all gates resolved while other evidence is not passed", prefix, id))
			}
		}
	}
	return problems
}

func contradictoryPassedProof(proof string) string {
	if match := unresolvedGap.FindString(proof); match != "" {
		return strings.ToLower(match)
	}
	return ""
}

func validateEvidenceReceipts(prefix, changeDir, id string, record EvidenceRecord, requiredKinds []string) []string {
	var problems []string
	seenKinds := map[string]struct{}{}
	for _, receipt := range record.Receipts {
		kind := strings.TrimSpace(receipt.Kind)
		path := slash(strings.TrimSpace(receipt.Path))
		if _, ok := allowedReceiptKinds[kind]; !ok {
			problems = append(problems, fmt.Sprintf("%s: evidence %s has unsupported receipt kind %q", prefix, id, receipt.Kind))
		}
		if _, duplicate := seenKinds[kind]; duplicate {
			problems = append(problems, fmt.Sprintf("%s: evidence %s has duplicate receipt kind %q", prefix, id, kind))
		}
		seenKinds[kind] = struct{}{}

		nativePath := filepath.FromSlash(path)
		cleanPath := filepath.Clean(nativePath)
		if path == "" || filepath.IsAbs(nativePath) || filepath.VolumeName(nativePath) != "" || cleanPath == ".." || strings.HasPrefix(cleanPath, ".."+string(filepath.Separator)) || strings.ContainsAny(path, "*?[") {
			problems = append(problems, fmt.Sprintf("%s: evidence %s receipt path %q must be a safe relative file", prefix, id, receipt.Path))
			continue
		}
		if !strings.Contains(slash(record.Proof), path) {
			problems = append(problems, fmt.Sprintf("%s: evidence %s proof must mention receipt path %s", prefix, id, path))
		}
		body, err := os.ReadFile(filepath.Join(filepath.FromSlash(changeDir), cleanPath))
		if err != nil {
			problems = append(problems, fmt.Sprintf("%s: evidence %s receipt %s: read: %v", prefix, id, path, err))
			continue
		}
		if strings.TrimSpace(string(body)) == "" {
			problems = append(problems, fmt.Sprintf("%s: evidence %s receipt %s must not be empty", prefix, id, path))
		}
	}
	if record.Status == "passed" {
		for _, kind := range requiredKinds {
			if _, exists := seenKinds[kind]; !exists {
				problems = append(problems, fmt.Sprintf("%s: passed evidence %s requires %s receipt", prefix, id, kind))
			}
		}
	}
	return problems
}

func validateSpec(prefix, body string, control Control) []string {
	expectedHeaders := map[string]string{
		"Change ID":           control.ChangeID,
		"Revision":            strconv.Itoa(control.Revision),
		"Status":              control.Status,
		"Decision authority":  control.DecisionAuthority,
		"Approval basis":      control.ApprovalBasis,
		"Repository baseline": control.RepositoryBaseline,
		"Supersedes":          supersedesHeader(control.Supersedes),
	}
	keys := make([]string, 0, len(expectedHeaders))
	for key := range expectedHeaders {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var problems []string
	for _, key := range keys {
		values := headerValues(body, key)
		if len(values) != 1 {
			problems = append(problems, fmt.Sprintf("%s: spec.md must contain exactly one %q header", prefix, key))
			continue
		}
		if values[0] != expectedHeaders[key] {
			problems = append(problems, fmt.Sprintf("%s: spec.md %s %q does not match control.json %q", prefix, key, values[0], expectedHeaders[key]))
		}
	}

	requirementMatches := requirementHead.FindAllStringSubmatchIndex(body, -1)
	globalAcceptance := captureIDs(acceptanceHead.FindAllStringSubmatch(body, -1))
	var requirements, acceptance []string
	for index, match := range requirementMatches {
		id := body[match[2]:match[3]]
		requirements = append(requirements, id)
		end := len(body)
		if index+1 < len(requirementMatches) {
			end = requirementMatches[index+1][0]
		}
		section := body[match[1]:end]
		acceptanceMatches := acceptanceHead.FindAllStringSubmatchIndex(section, -1)
		normative := section
		if len(acceptanceMatches) > 0 {
			normative = section[:acceptanceMatches[0][0]]
		}
		if !normativeWord.MatchString(normative) {
			problems = append(problems, fmt.Sprintf("%s: %s must contain a MUST or SHALL normative statement", prefix, id))
		}
		if len(acceptanceMatches) == 0 {
			problems = append(problems, fmt.Sprintf("%s: %s must contain at least one acceptance scenario", prefix, id))
		}
		for acceptanceIndex, acceptanceMatch := range acceptanceMatches {
			acceptanceIDValue := section[acceptanceMatch[2]:acceptanceMatch[3]]
			acceptance = append(acceptance, acceptanceIDValue)
			acceptanceEnd := len(section)
			if acceptanceIndex+1 < len(acceptanceMatches) {
				acceptanceEnd = acceptanceMatches[acceptanceIndex+1][0]
			}
			scenario := section[acceptanceMatch[1]:acceptanceEnd]
			for _, keyword := range []string{"GIVEN", "WHEN", "THEN"} {
				if !regexp.MustCompile(`(?m)^- ` + keyword + ` .+`).MatchString(scenario) {
					problems = append(problems, fmt.Sprintf("%s: %s must contain a '- %s ...' line", prefix, acceptanceIDValue, keyword))
				}
			}
		}
	}
	if len(globalAcceptance) != len(acceptance) {
		problems = append(problems, fmt.Sprintf("%s: every AC heading must be nested under a REQ heading", prefix))
	}
	problems = append(problems, compareIDLists(prefix, "spec.md requirements", requirements, control.Requirements)...)
	problems = append(problems, compareIDLists(prefix, "spec.md acceptance", acceptance, control.Acceptance)...)
	return problems
}

func validatePlan(prefix, body string, control Control) []string {
	var problems []string
	for _, id := range append(append([]string{}, control.Requirements...), control.Acceptance...) {
		if !strings.Contains(body, id) {
			problems = append(problems, fmt.Sprintf("%s: plan.md does not trace %s", prefix, id))
		}
	}
	for _, pattern := range control.AppliesTo {
		if !strings.Contains(body, pattern) {
			problems = append(problems, fmt.Sprintf("%s: plan.md scope does not include applies_to pattern %s", prefix, pattern))
		}
	}
	return problems
}

func validateEvidenceArtifact(prefix, body string, control Control) []string {
	rows := map[string]EvidenceRecord{}
	var problems []string
	for _, match := range evidenceRow.FindAllStringSubmatch(body, -1) {
		id := strings.TrimSpace(match[1])
		if _, exists := rows[id]; exists {
			problems = append(problems, fmt.Sprintf("%s: evidence.md contains duplicate row %s", prefix, id))
			continue
		}
		rows[id] = EvidenceRecord{Status: strings.TrimSpace(match[2]), Proof: strings.TrimSpace(match[3])}
	}
	expected := append(append([]string{}, control.Requirements...), control.Acceptance...)
	expectedSet := stringSet(expected)
	for _, id := range expected {
		row, exists := rows[id]
		if !exists {
			problems = append(problems, fmt.Sprintf("%s: evidence.md is missing row %s", prefix, id))
			continue
		}
		controlRecord := control.Evidence[id]
		if row.Status != controlRecord.Status || row.Proof != strings.TrimSpace(controlRecord.Proof) {
			problems = append(problems, fmt.Sprintf("%s: evidence.md row %s does not match control.json", prefix, id))
		}
	}
	for id := range rows {
		if _, exists := expectedSet[id]; !exists {
			problems = append(problems, fmt.Sprintf("%s: evidence.md contains undeclared row %s", prefix, id))
		}
	}
	return problems
}

func validateChangedSpecDirs(changed []string, controls []controlledChange) []string {
	dirs := map[string]struct{}{}
	for _, control := range controls {
		dirs[control.Dir] = struct{}{}
	}
	seen := map[string]struct{}{}
	var problems []string
	for _, path := range changed {
		path = slash(path)
		if !strings.HasPrefix(path, changesRoot+"/") {
			continue
		}
		parts := strings.Split(path, "/")
		if len(parts) < 4 {
			problems = append(problems, fmt.Sprintf("%s: controlled change files must live under %s/<change-id>/", path, changesRoot))
			continue
		}
		dir := strings.Join(parts[:3], "/")
		if _, exists := seen[dir]; exists {
			continue
		}
		seen[dir] = struct{}{}
		if _, exists := dirs[dir]; !exists {
			problems = append(problems, fmt.Sprintf("%s: changed controlled spec directory has no control.json", dir))
		}
	}
	return problems
}

func changedSpecDirectories(changed []string) map[string]struct{} {
	dirs := map[string]struct{}{}
	for _, path := range changed {
		path = slash(path)
		if !strings.HasPrefix(path, changesRoot+"/") {
			continue
		}
		parts := strings.Split(path, "/")
		if len(parts) >= 4 {
			dirs[strings.Join(parts[:3], "/")] = struct{}{}
		}
	}
	return dirs
}

func validateImmutableControlledSpecs(changed []string, base string) []string {
	if base == "" {
		return nil
	}
	var problems []string
	for dir := range changedSpecDirectories(changed) {
		controlPath := dir + "/control.json"
		listing, err := gitOutput("ls-tree", "--name-only", base, "--", controlPath)
		if err != nil {
			problems = append(problems, fmt.Sprintf("%s: inspect base control: %v", controlPath, err))
			continue
		}
		if strings.TrimSpace(listing) == "" {
			continue
		}
		body, err := gitOutput("show", base+":"+controlPath)
		if err != nil {
			problems = append(problems, fmt.Sprintf("%s: read base control: %v", controlPath, err))
			continue
		}
		var previous struct {
			Status string `json:"status"`
		}
		if err := json.Unmarshal([]byte(body), &previous); err != nil {
			problems = append(problems, fmt.Sprintf("%s: parse base control: %v", controlPath, err))
			continue
		}
		if immutableStatus(previous.Status) {
			problems = append(problems, fmt.Sprintf("%s: base status %s is immutable; create a new controlled change and use supersedes", dir, previous.Status))
		}
	}
	return problems
}

func coveringChanges(path string, controls []controlledChange, changedDirs map[string]struct{}, requireAccepted bool) []string {
	var owners []string
	for _, change := range controls {
		if _, changed := changedDirs[change.Dir]; !changed {
			continue
		}
		if requireAccepted && change.Control.Status != "Accepted" {
			continue
		}
		if !requireAccepted && !authorizesImplementation(change.Control.Status) {
			continue
		}
		for _, pattern := range change.Control.AppliesTo {
			if covered(path, pattern) {
				owners = append(owners, change.Control.ChangeID)
				break
			}
		}
	}
	sort.Strings(owners)
	return owners
}

func requiresControlledSpec(path string) bool {
	path = slash(path)
	if strings.HasPrefix(path, changesRoot+"/") || strings.HasPrefix(path, ".ai/") {
		return false
	}
	for _, prefix := range []string{
		".github/workflows/",
		"admin/",
		"contracts/",
		"frontend-sample/",
		"server/",
		"site/",
		"skills/",
	} {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	switch path {
	case ".gitignore", "AGENTS.md", "architecture.yaml", "go.mod", "go.sum", "index.html":
		return true
	default:
		return false
	}
}

func changedFiles() ([]string, error) {
	set := map[string]struct{}{}
	base := strings.TrimSpace(os.Getenv("SPEC_BASE_REF"))
	if base != "" {
		var args []string
		if strings.Trim(base, "0") == "" {
			args = []string{"diff-tree", "--root", "--no-commit-id", "--name-only", "-r", "HEAD"}
		} else {
			args = []string{"diff", "--name-only", base + "...HEAD"}
		}
		out, err := gitOutput(args...)
		if err != nil {
			return nil, fmt.Errorf("inspect changed files from SPEC_BASE_REF %q: %w", base, err)
		}
		addLines(set, out)
	} else {
		out, err := gitOutput("diff", "--name-only", "HEAD")
		if err != nil {
			return nil, fmt.Errorf("inspect tracked changes: %w", err)
		}
		addLines(set, out)
		out, err = gitOutput("ls-files", "--others", "--exclude-standard")
		if err != nil {
			return nil, fmt.Errorf("inspect untracked changes: %w", err)
		}
		addLines(set, out)
	}

	paths := make([]string, 0, len(set))
	for path := range set {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths, nil
}

func comparisonBase() string {
	base := strings.TrimSpace(os.Getenv("SPEC_BASE_REF"))
	if base == "" {
		return "HEAD"
	}
	if strings.Trim(base, "0") == "" {
		return ""
	}
	return base
}

func gitOutput(args ...string) (string, error) {
	output, err := exec.Command("git", args...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return string(output), nil
}

func readArtifact(path string) (string, []string) {
	body, err := os.ReadFile(path)
	if err != nil {
		return "", []string{fmt.Sprintf("%s: read: %v", slash(path), err)}
	}
	if strings.TrimSpace(string(body)) == "" {
		return "", []string{fmt.Sprintf("%s: must not be empty", slash(path))}
	}
	return string(body), nil
}

func compareIDLists(prefix, label string, actual, expected []string) []string {
	actualSet := stringSet(actual)
	expectedSet := stringSet(expected)
	var problems []string
	if len(actualSet) != len(actual) {
		problems = append(problems, fmt.Sprintf("%s: %s contains duplicate IDs", prefix, label))
	}
	for _, id := range expected {
		if _, exists := actualSet[id]; !exists {
			problems = append(problems, fmt.Sprintf("%s: %s is missing %s", prefix, label, id))
		}
	}
	for _, id := range actual {
		if _, exists := expectedSet[id]; !exists {
			problems = append(problems, fmt.Sprintf("%s: %s contains undeclared %s", prefix, label, id))
		}
	}
	return problems
}

func captureIDs(matches [][]string) []string {
	ids := make([]string, 0, len(matches))
	for _, match := range matches {
		ids = append(ids, match[1])
	}
	return ids
}

func headerValues(body, key string) []string {
	pattern := regexp.MustCompile(`(?m)^` + regexp.QuoteMeta(key) + `:\s*(.*)\s*$`)
	matches := pattern.FindAllStringSubmatch(body, -1)
	values := make([]string, 0, len(matches))
	for _, match := range matches {
		values = append(values, strings.TrimSpace(match[1]))
	}
	return values
}

func supersedesHeader(values []string) string {
	if len(values) == 0 {
		return "none"
	}
	return strings.Join(values, ", ")
}

func validStatus(status string) bool {
	switch status {
	case "Draft", "Ready", "Applying", "Verifying", "Accepted", "Superseded":
		return true
	default:
		return false
	}
}

func authorizesImplementation(status string) bool {
	switch status {
	case "Ready", "Applying", "Verifying", "Accepted":
		return true
	default:
		return false
	}
}

func immutableStatus(status string) bool {
	return status == "Accepted" || status == "Superseded"
}

func placeholder(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "", "-", "pending", "tbd", "todo", "none", "n/a", "na", "unknown", "agent-approved", "self-approved":
		return true
	default:
		return false
	}
}

func agentOnlyAuthority(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "agent", "ai", "assistant", "codex", "claude", "devin":
		return true
	default:
		return false
	}
}

func covered(path, pattern string) bool {
	path = slash(path)
	pattern = slash(strings.TrimSpace(pattern))
	if strings.HasSuffix(pattern, "/**") {
		return strings.HasPrefix(path, strings.TrimSuffix(pattern, "**"))
	}
	return path == pattern
}

func stringSet(values []string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		set[value] = struct{}{}
	}
	return set
}

func addLines(set map[string]struct{}, body string) {
	for _, line := range strings.Split(body, "\n") {
		line = slash(strings.TrimSpace(line))
		if line != "" {
			set[line] = struct{}{}
		}
	}
}

func slash(path string) string {
	return filepath.ToSlash(path)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "speccheck:", err)
	os.Exit(1)
}
