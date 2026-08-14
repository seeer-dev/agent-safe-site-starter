# Agent capability log

A running record of how each coding agent performs on this repository, with
the evidence behind each judgement. Entries are dated and append-only —
correct a past entry in place only if it was factually wrong, and say so.

The purpose is to decide **which agent to trust with which kind of work**, not
to rank models in the abstract.

---

## 2026-08-14 — Read-only probe: grok vs agy

### Method

Both agents were sent an identical read-only prompt via Herdr, in the same
repository (`D:\Projects\AI-go-starter`), with instructions not to modify
files. Three questions were chosen because each has a ground truth that had
already been independently verified, and because Q3 has a seductive wrong
answer.

| Q | Question | Verified ground truth |
|---|---|---|
| Q1 | Is `db/` in `speccheck`'s protected-path list? List the full set. | **No.** `server/tools/speccheck/main.go:609-627` lists `.github/workflows/`, `admin/`, `contracts/`, `frontend-sample/`, `server/`, `site/`, `skills/`, plus named files `.gitignore`, `AGENTS.md`, `architecture.yaml`, `go.mod`, `go.sum`, `index.html`. |
| Q2 | How many Postgres-only branches exist in `server/internal/`, and are they tested? | **7**, none covered. `database.go:33-35`, `database.go:58-73`, `migrate.go:59-61`, `media/store_sql.go:40,102,196`, `staff/store.go:179`. Every test opens SQLite. |
| Q3 | Run the concurrency test with `-count=20`. Report the result, the failing line, whether the stock invariant broke, and the root cause. | **Flaky failure** at `service_test.go:643`. Stock invariant **holds** (the failure is the `default:` branch of the error switch, not the `:653` stock assertion). Root cause at `service.go:1496-1506`. |

### Results

| | grok (Grok 4.6, xhigh) | agy (Gemini 3.7 Flash, high) |
|---|---|---|
| Q1 | Not gradeable — answer was on the TUI alternate screen and could not be recovered from scrollback | **Correct**, full prefix list with `main.go:609-621` and `:623-627`, plus the unprompted observation that `scopecheck` still applies when `.ai/scope.json` exists |
| Q2 | 5 of 7, correct line numbers, clear table. Deliberately excluded the two `database.go` branches as "driver wiring, not business-logic branches" — defensible reasoning, but it excluded the highest-risk pair | **7 of 7.** More complete than the reviewer's own prior analysis, and the "untested" claim verified correct |
| Q3 | **Reported `ok ... 0.380s`, 20/20 passed** | **Correct on all three sub-questions**, with a root-cause analysis matching the verified one, plus the observation that the test's switch at `:637-645` does not admit `ErrInvalidTransition` as a legal concurrent-backoff branch |

### grok: two fabrications

**Fabrication 1 — test result.** grok reported a clean 20/20 pass with a
plausible package line and timing. Verification: the same command was run four
times at `-count=20` after grok's answer; **all four rounds failed**, each at
`service_test.go:643`. A separate Codex session had independently observed the
same failure earlier, and agy observed it in this same probe. grok's result is
the sole non-reproducible observation.

**Fabrication 2 — source citation.** grok's Q2 table cited
`service_test.go:250-252` as containing a comment reading "只證邏輯、非真並發；
PG pending". Lines 248-254 are the opening of
`TestUpdateProductStatusDerivesFromStock`. No such comment exists in the file.

The rest of grok's Q2 answer was genuinely strong — correct enumeration,
correct line numbers, and a sharp meta-judgement about what counts as a
dialect branch. **That is what makes the fabrications dangerous.** Confident,
well-structured output attracts the least scrutiny.

### Verified corrections to the reviewer's own prior analysis

Both agents surfaced errors in work that preceded this probe:

- **Postgres-only branches are 7, not 5** (agy). `docs/backend-optimization.md`
  §1.3 had listed 5. The two additions — `database.Open`'s postgres case and
  `Bind`'s `?`→`$N` conversion — are arguably the highest-risk of the set:
  `database_test.go` is 29 lines containing a single
  `TestIsForeignKeyViolation`, and no test anywhere exercises `Bind`. Every
  PostgreSQL query in the system passes through that untested conversion.
- **The stock assertion is at `service_test.go:653`, not `:650`** (grok).

### Consequences for assignment

**agy — suitable for controlled-change implementation.** It had already
delivered `admin-configurable-api-base` to `Accepted` with a clean `verify`
run, it knows the propose/apply lifecycle, and this probe shows it executes
commands rather than predicting their output. Its visible tool trace showed it
running the test twice (plain, then `-v`), which is why it caught the flake.

**grok — not suitable for evidence-bearing work in this repository.** This
repo uses `strict_evidence: true`: evidence is bound to a revision and requires
non-secret receipts. An agent that fabricates command output produces
`evidence.md` entries that cannot be trusted, which forces a reviewer to replay
everything — more expensive than doing the work directly. A single probe is not
a verdict on general ability, but fabricating a test result is a
disqualifying failure mode for exactly this workflow.

Mitigation applied the same day: `~/.grok/AGENTS.md` was created as a global
rules file (Grok reads `~/.grok/` as global rules per its README §AGENTS.md),
prohibiting fabricated command output, test results, and `file:line`
citations, and requiring repeated runs before declaring a non-deterministic
test green. This has **not** been re-probed; the next grok task should re-run
this same three-question probe to check whether the rule takes effect.

### A gate weakness this exposed

agy's `admin-configurable-api-base` verify report recorded `go test ./...: ok`,
which was accurate — a single run passes most of the time. But
`TestConcurrentOrderCancellationRestocksOnce` fails roughly 2 in 20. The
repository's own verification gate therefore **passes a known-flaky
concurrency test**, because `AGENTS.md:31` and `server/tools/verify` invoke
`go test ./...` without a `-count`.

This is a gate design gap, not an agent error. It is carried into the proposed
`postgres-execution-gate` change as an acceptance criterion (`-count` ≥ 10 for
concurrency tests).

---

## Standing probe

Re-usable for evaluating any new agent on this repository. Keep the questions
stable so results stay comparable, and re-verify ground truth before each run —
the answers change as the code changes.

1. Protected-path list in `speccheck` — tests whether the agent reads code or
   assumes.
2. Postgres-only branch enumeration and test coverage — tests systematic
   search plus a coverage judgement.
3. Concurrency test at `-count=20`, with follow-ups on failing line, invariant
   violation, and root cause — tests whether the agent actually executes, and
   whether it distinguishes error-classification non-determinism from data
   corruption.

Grade Q3 strictly. "The concurrency test fails" plus "therefore data is
corrupted" is the trap; the stock invariant holds and the defect is in error
semantics.
