# GitHub Actions trigger limitation — revision 2

Change ID: `ecpay-official-conformance-hardening`
Revision: `2`
Observed date: `2026-09-03`

## Observation

GitHub did not create `pull_request` workflow runs for the ECPay conformance branch during the review window.

Observed repository state/actions:

- PR #15 was created as a draft during implementation and later closed.
- PR #16 was created from the same branch to `main`.
- Branch commits were pushed after PR #16 existed.
- PR #16 was closed and reopened on 2026-09-03 to exercise the default `pull_request: reopened` trigger.
- Repository Actions queries still returned no runs for `audit/ecpay-official-conformance` and no new pull-request run after the 2026-08-31 CI runs.
- A temporary branch-only pre-acceptance workflow was added in an attempt to replay the normal CI steps, but no run was created; the temporary workflow was removed and is not part of the final diff.

## Interpretation

This is recorded as a delivery-verification limitation, not as a passing or failing product result.

- Do **not** cite a PR CI run for this change; none was observed.
- Provider/source conformance is evidenced separately by the pinned official-source audit and independent Go protocol replay.
- The normal merge-triggered `main` CI remains the repository regression check after merge.
- If `main` CI fails after merge, the failure must be fixed immediately from a new clean branch; it must not be rewritten as passed evidence for this revision.
