# Harden implementation handoffs

Change ID: harden-implementation-handoffs
Revision: 1
Status: Verifying
Decision authority: Repository owner/user
Approval basis: User authorized plain apply implementation of harden-implementation-handoffs revision 1 on 2026-08-15.
Repository baseline: 7e5aa90b92d23c8c316d44ca935be7af8d150a1c
Supersedes: none

## Outcome

Implementation blueprints remain executor-neutral, while a relay between a planner, an implementer, and a reviewer is attributable, transport-safe, and prevented from silently modifying controlled artifacts or leaving scope residue.

## Scope

In scope: a provider-neutral handoff reference used alongside `expand-implementation`, including packet identity, write authority, PowerShell-safe Herdr delivery guidance, acknowledgement, postflight inspection, and proportionate packet sizing.

Out of scope: selecting agents, changing product code, creating a new user lifecycle action, modifying Herdr itself, or changing accepted controlled specifications.

## Requirements

### REQ-001: Every delegated packet is attributable
The handoff procedure MUST assign one stable packet ID, source change/revision, explicit modify set, forbidden paths, expected response fields, and one named reviewer before implementation starts.

#### AC-001: Recipient acknowledgement is inspectable
- GIVEN a planner delegates an implementation packet
- WHEN the recipient starts work
- THEN its acknowledgement names the packet ID, allowed paths, forbidden paths, and reporting destination.

### REQ-002: Handoff transport is robust across shells
The procedure MUST define a shell-safe fallback for long or multiline packets and preserve the recipient's current work without assuming Bash syntax.

#### AC-002: PowerShell relay does not truncate the packet
- GIVEN a Windows PowerShell planner sends a multiline packet through Herdr
- WHEN direct prompt argument transport is unsafe or rejected
- THEN the planner writes a non-secret OS-temp packet, delivers its path via `herdr agent prompt` which guarantees explicit submission, and cleans up the temporary file in finally.

### REQ-003: Planning and implementation boundaries remain enforced
The procedure MUST state where a blueprint may be written, prohibit unauthorized controlled-artifact edits, require postflight scope/residue inspection, and use compact packets for small changes.

#### AC-003: Planner cannot silently amend a controlled plan
- GIVEN a planner lacks explicit controlled-artifact write authority for an already active change
- WHEN it prepares a relay packet
- THEN it returns the packet through the approved OS-temp transport rather than modifying `plan.md`.

#### AC-004: Scope residue is reported even if removed
- GIVEN an implementer creates any path outside its modify set
- WHEN postflight inspection runs
- THEN the report records the incident and confirms that no residue remains relative to baseline before acceptance evidence can advance.
