# Harden implementation handoffs plan

## Repository reality

- `expand-implementation` correctly makes repository work low-inference but intentionally excludes assignment and scheduling.
- The S01G replay showed two delivery failures around a PowerShell planner: Bash-only shell syntax and multiline Herdr prompt quoting.
- The same replay showed an unrequested temporary helper path and a planner-written expansion in an active controlled plan; neither should be hidden by later cleanup.

## Scope lock

- `skills/expand-implementation/**`
- `specs/changes/harden-implementation-handoffs/**`

## Vertical slices

### Slice 1: Define a neutral handoff envelope

Outcome: a packet has stable identity, sender/reviewer responsibility, path boundaries, expected response fields, and proportionate detail.

Covers: REQ-001, AC-001.

### Slice 2: Define safe delivery and planner boundaries

Outcome: long Windows/Herdr payloads use a non-secret temporary artifact plus explicit submission, and planners write controlled artifacts only with authority.

Covers: REQ-002, AC-002, AC-003.

### Slice 3: Require postflight incident accounting

Outcome: implementer/reviewer handoff records scope residue, test/mutation status, and unresolved blockers before evidence changes.

Covers: REQ-003, AC-004.

## Verification plan

- Replay one short PowerShell/Herdr handoff using a temporary non-secret packet path.
- Verify a planner without controlled-artifact write authority leaves the controlled plan unchanged.
- Verify postflight reports a deliberately created scope-external temporary path, then confirms its removal.
- Run the relevant skill validation plus `go run ./server/tools/speccheck`, scopecheck, and the repository verifier.
