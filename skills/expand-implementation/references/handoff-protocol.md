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
- **Assigned Reviewer**: `<named reviewer / authority>` (e.g. User, Repository Owner, or named Reviewer Agent)
- **Reporting Destination**: `<pane / session / file / channel destination>`
- **Modify Set (Allowed Paths)**:
  - `exact/path/1`
  - `exact/path/2/**`
- **Forbidden Paths**:
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
  2. Modified Paths & Residue Check (baseline-relative comparison)
  3. Validation Results (exact commands & exit codes)
  4. Evidence Status (REQ/AC mapping)
  5. Scope Incident Log (even if cleaned up)
```

### Proportionate Packet Sizing

- Size packets proportionally to the task. Small changes (e.g. docs, single-function fixes) must receive concise, compact packets without unnecessary boilerplate.
- Cross-cutting changes must be split into dependency-ordered packets with isolated outcomes and rollback boundaries.

### Inspectable Recipient Acknowledgement (AC-001)

Before executing edits, the recipient must emit an acknowledgement recording:
1. Assigned **Packet / Slice ID** and **Change ID / Revision**.
2. **Allowed Paths** (Modify Set).
3. **Forbidden Paths**.
4. **Reporting Destination** / Assigned Reviewer.

---

## 2. Shell-Safe Transport & PowerShell / Herdr Relay (REQ-002, AC-002)

Direct transmission of multiline markdown or JSON payloads across shell sessions (such as Windows PowerShell or Herdr panes) can cause quote stripping, newline truncation, or syntax failures. Do not assume Bash syntax or unescaped multiline string handling.

### Safe Relay Procedure

1. **OS Temp Only (No In-Repository Scratch Files)**:
   - In-repository scratch paths (e.g. `tmp/...`) create untracked file residue and pollute `git status`.
   - The relay MUST write temporary packet envelopes strictly into the operating system temporary directory (e.g. `$env:TEMP` / `[System.IO.Path]::GetTempPath()`) using unguessable filenames.
   - **Zero Secrets**: Never include credentials, tokens, or private secrets in the temporary handoff artifact.
2. **Real Herdr Primitive (`herdr agent prompt`)**:
   - Send only a concise invocation referencing the unguessable OS-temp path to the target agent or pane using `herdr agent prompt <target> "<invocation>"`.
   - `herdr agent prompt` natively handles bracketed paste and automatically delivers an encoded Enter after submission; explicit submission is guaranteed by this command without prescribing separate raw newline steps.
3. **Guaranteed Cleanup**:
   - Temporary OS-temp files must be cleaned up in a `finally` block or postflight step.
4. **Exact PowerShell Implementation Pattern**:

```powershell
$tempFile = [System.IO.Path]::Combine([System.IO.Path]::GetTempPath(), [System.IO.Path]::GetRandomFileName() + ".md")
try {
    Set-Content -LiteralPath $tempFile -Value $packetEnvelopeContent -Encoding utf8
    herdr agent prompt <target-pane-or-agent> "Implement packet from $tempFile" --wait --timeout 120000
} finally {
    if (Test-Path -LiteralPath $tempFile) {
        Remove-Item -LiteralPath $tempFile -Force
    }
}
```

5. **Preserve Recipient State**:
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

1. **Baseline-Relative Scope Comparison**:
   - In a shared dirty worktree, never claim `git status` is clean unless the entire repository working tree is actually clean.
   - Record the pre-existing dirty baseline before work begins.
   - Compare post-implementation state against the pre-existing baseline:
     - Verify every file modified by the implementer is in the declared modify set.
     - Verify pre-existing dirty files and concurrent changes were preserved intact.
     - Verify zero unapproved implementer-created files exist.
2. **Scope-Residue Incident Log (AC-004)**:
   - If any path outside the modify set was created during work (e.g. temporary debug scripts, scratch files, scratch tests, or fixtures)—**even if deleted before reporting**—the implementer MUST record this as an incident in the final report.
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

#### 1. Modified Paths
- `path/to/modified/file` (Allowed)

#### 2. Scope & Residue Audit (Baseline-Relative)
- Pre-existing dirty paths preserved: Yes
- Unauthorized paths touched: None
- Scope Incidents: None (or list temporary files created and cleaned up)
- Residue Check: Clean relative to pre-existing baseline (no implementer residue)

#### 3. Verification Results
- `<Command 1>`: PASS (<details>)
- `<Command 2>`: PASS (<details>)

#### 4. Evidence Traceability
- `REQ-NNN`: passed / pending / blocked (<proof>)
- `AC-NNN`: passed / pending / blocked (<proof>)

#### 5. Blockers / Open Items
- <None or explicit blockers>
```
