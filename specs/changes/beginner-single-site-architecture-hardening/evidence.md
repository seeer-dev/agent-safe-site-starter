# Beginner Single-Site Architecture Hardening Evidence

Change ID: beginner-single-site-architecture-hardening
Revision: 1
Status: Accepted

## Targeted CI evidence

PR #2 CI run `32968611776` passed:

- `gofmt`;
- `go test ./server/tools/archcheck -count=1`;
- the manifest-driven real repository architecture scan;
- frontend resource/OpenAPI contract checks;
- migration parity.

The run then stopped only at the expected Accepted-only controlled-spec gate while the change was still `Verifying`. The targeted architecture evidence was reconciled before changing the controlled status to `Accepted`.

## Architecture evidence

- `architecture.yaml` owns a versioned import policy with hard `deny` semantics.
- `archcheck` rejects unsafe/nonexistent roots and validates non-zero scan/module/platform coverage.
- `AGENTS.md`, `skills/site/SKILL.md`, and `architecture-boundaries.md` preserve the beginner single-site product shape using only starter-owned terminology.
- Cross-module synchronous collaboration uses consumer-owned typed ports wired in bootstrap.
- Large modules grow by cohesive internal seams before top-level proliferation.
- Checked-in UI/reference flows remain the user-facing acceptance contract unless a reviewed design change supersedes them.

Full repository CI is required after this acceptance update before merge.
