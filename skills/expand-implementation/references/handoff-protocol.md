# Implementation Handoff Protocol

Provider-neutral protocol for delegating implementation packets between a planner, an implementer, and a reviewer.

## Purpose

Maintain strict attribution, transport safety across shells (including Windows PowerShell and Herdr), explicit write-authority boundaries, and auditable scope accounting when delegating implementation work.

---

## 1. Neutral Handoff Envelope (REQ-001, AC-001)

Every delegated implementation packet must be packaged into a standalone, attributable envelope before implementation starts.

### Mandatory Envelope Fields

```markdown
### Implementation Handoff Envelope

- **Packet / Slice ID**: `<packet-id or slice-id>` (e.g. `Slice 1`, `Slice 2`, `S01`; preserves the source plan's existing packet or slice identifier; do not invent an unrecorded ID mapping unless the planner has explicit write authority to record it in `plan.md`)
- **Change ID & Revision**: `<change-id>` rev `<revision>`
- **Repository Baseline**: `<40-character hex commit>` (authoritative baseline commit from `control.json`)
- **Linked-Worktree Root**: `<OS-temp linked worktree path>` (isolated linked worktree path outside repository)
- **Assigned Reviewer**: `<named reviewer / authority>` (e.g. User, Repository Owner, or named Reviewer Agent)
- **Reporting Destination**: `<pane / session / file / channel destination>`
- **Active Ownership Preflight**: Verified zero overlap between `Modify Set` and active/dirty files in any other worktree (must stop immediately on conflict)
- **Modify Set (Allowed Paths)**:
  - `exact/path/1`
  - `exact/path/2/**`
- **Forbidden Paths**:
  - In-repository scratch files (`tmp/...`, `.scratch/...`, or unapproved scratch scripts)
  - Secrets, credentials, or private authentication tokens
  - Unowned specs (`specs/changes/<other-id>/**`)
  - `.ai/scope.json` (unless explicitly delegated)
  - Product code / `.env*` / CI configurations outside approved scope
  - Pre-existing uncommitted worktree changes
- **Packet Instructions**:
  <Concise, proportionate ordered edits and constraints>
- **Verification Commands**:
  <Exact working directory, argv, and expected assertions>
- **Required Response Fields**:
  1. Recipient Acknowledgement
  2. Modified Paths & Full Selected Diff (baseline-relative comparison)
  3. Validation Results (exact commands & exit codes)
  4. Evidence Status (REQ/AC mapping)
  5. Worktree Cleanup & Scope Incident Log (even if cleaned up)
```

### Proportionate Packet Sizing

- Size packets proportionally to the task. Small changes (e.g. docs, single-function fixes) must receive concise, compact packets without unnecessary boilerplate.
- Cross-cutting changes must be split into dependency-ordered packets with isolated outcomes and rollback boundaries.

### Inspectable Recipient Acknowledgement (AC-001)

Before executing edits, the recipient must emit an acknowledgement recording:
1. Assigned **Packet / Slice ID**, **Change ID / Revision**, and **Repository Baseline**.
2. **Linked-Worktree Root** and confirmation of zero active ownership overlap.
3. **Allowed Paths** (Modify Set).
4. **Forbidden Paths** (including zero in-repository scratch files and zero secrets).
5. **Reporting Destination** / Assigned Reviewer.

---

## 2. Shell-Safe Transport & PowerShell / Herdr Relay (REQ-002, AC-002)

Direct transmission of multiline markdown or JSON payloads across shell sessions (such as Windows PowerShell or Herdr panes) can cause quote stripping, newline truncation, or syntax failures. Do not assume Bash syntax or unescaped multiline string handling.

### Safe Relay & Isolated Linked-Worktree Procedure

1. **OS Temp Only (No In-Repository Scratch Files)**:
   - In-repository scratch paths (e.g. `tmp/...`) create untracked file residue and pollute `git status`.
   - The relay MUST write temporary packet envelopes strictly into the operating system temporary directory (e.g. `$env:TEMP` / `[System.IO.Path]::GetTempPath()`) using unguessable filenames.
   - **Zero Secrets**: Never include credentials, tokens, or private secrets in the temporary handoff artifact or report.
2. **One OS-Temporary Linked Worktree Per Packet**:
   - For parallel delegated packets or dirty-primary repositories, create an isolated linked worktree in the OS temporary directory branching from the controlled `repository_baseline`.
   - Before edits, compare the packet's `applies_to` with active/dirty files in all existing worktrees; if any path overlaps, STOP immediately.
   - In the linked worktree root, run validation with `$env:SCOPE_CHANGE_ID='<change-id>'; go run ./server/tools/scopecheck` and `go run ./server/tools/verify`. Selected mode is local-only and forbidden in CI.
3. **Real Herdr Primitive (`herdr agent prompt`)**:
   - Send only a concise invocation referencing the unguessable OS-temp path to the target agent or pane using `herdr agent prompt <target> "<invocation>"`.
   - `herdr agent prompt` natively handles bracketed paste and automatically delivers an encoded Enter after submission; explicit submission is guaranteed by this command without prescribing separate raw newline steps.
4. **Guaranteed Cleanup in Finally**:
   - In a `finally` block, remove the linked worktree (`git worktree remove --force`), prune Git metadata (`git worktree prune`), and delete temporary envelope files.
5. **Exact PowerShell Implementation Pattern**:

```powershell
$tempEnvelope = [System.IO.Path]::Combine([System.IO.Path]::GetTempPath(), [System.IO.Path]::GetRandomFileName() + ".md")
$tempWorktree = [System.IO.Path]::Combine([System.IO.Path]::GetTempPath(), "wt-" + [System.IO.Path]::GetRandomFileName())
try {
    Set-Content -LiteralPath $tempEnvelope -Value $packetEnvelopeContent -Encoding utf8
    git -C $repositoryRoot worktree add --detach $tempWorktree $repositoryBaseline
    Push-Location $tempWorktree
    try {
        $env:SCOPE_CHANGE_ID = $changeId
        herdr agent prompt <target-pane-or-agent> "Implement packet from $tempEnvelope in $tempWorktree" --wait --timeout 120000
    } finally {
        Pop-Location
    }
} finally {
    if (Test-Path -LiteralPath $tempWorktree) {
        git -C $repositoryRoot worktree remove --force $tempWorktree
        git -C $repositoryRoot worktree prune
    }
    if (Test-Path -LiteralPath $tempEnvelope) {
        Remove-Item -LiteralPath $tempEnvelope -Force
    }
}
```

6. **Preserve Recipient State**:
   - The relay must respect and preserve any pre-existing dirty files in the recipient's workspace.

---

## 3. Planner Write-Authority Boundary (REQ-003, AC-003)

Planners and relay agents must not exceed their designated authority:

- **Controlled Plan Protection**: If a planner does not possess explicit write authority for controlled artifacts on an active change (e.g. status `Applying` or locked), it **MUST NOT** modify `plan.md`, `spec.md`, or `control.json`.
- **Approved Temporary Transport**: When preparing or relaying packets without controlled write authority, return the packet through the approved OS-temp transport mechanism or review response rather than editing repository specifications.

---

## 4. Postflight Scope Inspection & Incident Accounting (REQ-003, AC-004)

Upon completing implementation, the implementer and reviewer must audit the working tree for out-of-scope modifications.

### Postflight Audit Rules

1. **Baseline-Relative Scope Comparison & Full Selected Diff**:
   - In a shared dirty worktree, never claim `git status` is clean unless the entire repository working tree is actually clean.
   - Record the pre-existing dirty baseline before work begins.
   - Inspect the complete selected diff (`<baseline>...HEAD` plus local/untracked) against `control.json` `applies_to`:
     - Verify every file modified by the implementer is in the declared modify set.
     - Verify pre-existing dirty files and concurrent changes were preserved intact.
     - Verify zero unapproved implementer-created files exist.
2. **Scope-Residue Incident Log (AC-004)**:
   - If any path outside the modify set was created during work (e.g. temporary debug scripts, scratch files, scratch tests, or fixtures), even if deleted before reporting, the implementer MUST record this as an incident in the final report.
   - Record:
     - Path of temporary file
     - Purpose and creation time
     - Deletion / restoration confirmation
     - Confirmation of zero residual files relative to baseline
3. **Evidence Gate**:
   - Controlled acceptance evidence cannot advance to `passed` while unrecorded scope residue exists or while temporary mutations remain unrestored.

---

## 5. Standard Implementation Report Schema

The implementer must return this structured format:

```markdown
### Implementation Report: <Packet / Slice ID> (<Change ID> rev <Revision>)

- **Reviewer**: `<Named Reviewer>`
- **Status**: Complete | Blocked

#### 1. Modified Paths & Selected Diff
- `path/to/modified/file` (Allowed)
- Full selected diff inspected and verified against `applies_to`

#### 2. Scope & Residue Audit (Baseline-Relative)
- Pre-existing dirty paths preserved: Yes
- Unauthorized paths touched: None
- Scope Incidents: None (or list temporary files created and cleaned up)
- Residue Check: Clean relative to pre-existing baseline (no implementer residue)
- Worktree Cleanup: Linked worktree removed and Git metadata pruned

#### 3. Verification Results
- `SCOPE_CHANGE_ID=<id> go run ./server/tools/scopecheck`: PASS (<details>)
- `go run ./server/tools/verify`: PASS (<details>)

#### 4. Evidence Traceability
- `REQ-NNN`: passed / pending / blocked (<proof>)
- `AC-NNN`: passed / pending / blocked (<proof>)

#### 5. Blockers / Open Items
- <None or explicit blockers>
```
