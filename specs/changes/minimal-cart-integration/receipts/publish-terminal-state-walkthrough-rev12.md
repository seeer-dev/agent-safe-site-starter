# Revision 12 Publish Terminal-State and Render LKG Walkthrough

- Change ID: `minimal-cart-integration`
- Packet ID: `B02R`
- Revision: `12`
- Date: 2026-08-15
- Implementer: Agy (`w5:p9`)
- Coordinator / Reviewer: Devin (`w5:p1`)
- Kind: `walkthrough`

---

## 1. Scope and Purpose

This receipt records attributable local observations for the publish terminal-state and render last-known-good (LKG) preservation requirements under **AC-013** and **AC-014**:
1. **Publish Tool Terminal-State Handling (AC-013 / GATE-006)**:
   - Ensure the publishing tool (`server/tools/publish`) operates as a trigger-only flow without Direct Upload (no wrangler).
   - Ensure deploy hook responses produce a structured terminal JSON receipt with timestamp, HTTP status, and body.
   - Ensure non-2xx responses abort the process with a non-zero exit code instead of assuming success.
   - Ensure an unset hook URL logs that the trigger was skipped without claiming deployment success.
   - Ensure a render failure stops the publish pipeline before any deploy hook call.
2. **Renderer Staging and LKG Preservation (AC-014)**:
   - Verify staging directory (`dist.staging`) and atomic promotion logic.
   - Verify fail-closed behavior for missing templates, invalid theme assets, dynamic segment traversal, and invalid CSP header origins.
   - Verify that any render failure cleans up staging and leaves the pre-existing `dist/` directory intact.

---

## 2. Environment & Environment Blockers

- **OS**: Windows (x86_64)
- **Go Version**: `go version go1.26.5 windows/amd64`
- **Database**: Local SQLite (`var/site.db`)
- **Environment Blockers (`ENVIRONMENT_BLOCKED`)**:
  - **Live Cloudflare Deploy Hook Execution**: Real Cloudflare Pages Deploy Hook URL and Cloudflare credentials are not configured in local development. Live end-to-end webhook delivery and Cloudflare build receipt remain pending live environment execution.
  - **Live R2 Asset & Headers Verification**: Live R2 `CopyObject` API execution, custom-domain `nosniff` HEAD/GET verification, and live Cloudflare Pages `_headers` deployment remain pending live infrastructure configuration.

---

## 3. Automated Check Execution

```bash
go test ./server/internal/render/... ./server/tools/render/... -v -count=1
```

**Observed Result**: 30/30 tests PASS (exit code 0).

### Key Test Coverage
- `TestRenderStagingFailurePreservesDist`: Prepopulates `dist/` with a canary, injects render error in `renderToStaging`, asserts error returned, `stagingDir` cleaned up, and `canary.txt` in `dist/` preserved intact.
- `TestRenderStagingSuccessPromotesToDist`: Prepopulates `dist/`, executes valid render, asserts old canary replaced by new output, `stagingDir` removed.
- `TestRenderHeadersFailurePreservesDist`: Invalid R2 origin in `_headers` generation aborts before render, removes staging, preserves `dist/`.
- `TestRenderHeadersInvalidSupabasePreservesDist`: Invalid Supabase origin aborts render, preserves `dist/`.
- `TestRenderMissingProductTemplateFailsClosed`: Missing template returns error, preserves LKG `dist/`.
- `TestRenderMissingCategoryTemplateFailsClosed`: Missing category template returns error, preserves LKG `dist/`.
- `TestRenderMissingContentTemplateFailsClosed`: Missing content template returns error, preserves LKG `dist/`.
- `TestRenderThemeValidation` (6 subcases): Validates missing `dist/`, non-directory `dist/`, missing `islands.js`, non-file `islands.js`, missing `islands-*.css`, non-file `islands-*.css` — all fail closed before staging creation.
- `TestRenderRejectsTraversalProductSlug`, `TestRenderRejectsTraversalCategory`, `TestRenderRejectsTraversalContentKey`, `TestRenderRejectsTraversalArticleSlug`: Confirms `../../` escape payloads are rejected by `safeJoin` and `validateRouteSegment`, staging cleaned, candidate escape path absent.

---

## 4. Local Publish Walkthrough Steps & Observations

### Step 1: Default Publish Execution (Hook URL Unset)
- **Command**: `go run ./server/tools/publish` (with `CF_DEPLOY_HOOK_URL` unset)
- **Output**:
  ```text
  2026/08/15 11:57:43 rendered 0 article(s), 5 product(s), 4 categor(y/ies), 0 content page(s) into dist/
  2026/08/15 11:57:43 render completed, dist promoted
  2026/08/15 11:57:43 deploy hook skipped (CF_DEPLOY_HOOK_URL not set)
  ```
- **Observation**: Render runs to staging and atomically promotes to `dist/`. The tool explicitly logs `deploy hook skipped` and does not claim deployment success. Exit code: 0.

### Step 2: Deploy Hook Terminal Success Receipt (HTTP 200)
- **Harness**: Ephemeral localhost HTTP server simulating Cloudflare Deploy Hook returning HTTP 200 with `{"result":"queued","id":"deploy-12345"}`.
- **Command**: `CF_DEPLOY_HOOK_URL=http://127.0.0.1:18765/deploy-hook go run ./server/tools/publish`
- **Output**:
  ```text
  [Mock Deploy Hook] Listening on http://127.0.0.1:18765
  2026/08/15 11:57:57 rendered 0 article(s), 5 product(s), 4 categor(y/ies), 0 content page(s) into dist/
  2026/08/15 11:57:57 render completed, dist promoted
  [Mock Deploy Hook] Received POST /deploy-hook headers: application/json
  2026/08/15 11:57:57 deploy hook receipt: {"triggered_at":"2026-08-15T03:57:57Z","status":200,"body":"{\"result\":\"queued\",\"id\":\"deploy-12345\"}"}
  [Mock Deploy Hook] Publish process exited with code: 0
  ```
- **Observation**: `triggerDeployHook` issues POST request with `Content-Type: application/json`, captures the exact HTTP status (200), ISO timestamp, and body into a valid JSON receipt, and logs the receipt. Exit code: 0.

### Step 3: Deploy Hook Terminal Failure Handling (HTTP 500)
- **Harness**: Ephemeral localhost HTTP server simulating Deploy Hook returning HTTP 500 with `{"error":"service unavailable"}`.
- **Command**: `CF_DEPLOY_HOOK_URL=http://127.0.0.1:18766/deploy-hook go run ./server/tools/publish`
- **Output**:
  ```text
  [Mock Deploy Hook] Listening on http://127.0.0.1:18766
  2026/08/15 11:58:01 rendered 0 article(s), 5 product(s), 4 categor(y/ies), 0 content page(s) into dist/
  2026/08/15 11:58:01 render completed, dist promoted
  [Mock Deploy Hook] Received POST /deploy-hook
  2026/08/15 11:58:01 deploy hook trigger failed: deploy hook returned HTTP 500
  exit status 1
  [Mock Deploy Hook] Publish process exited with code: 1
  ```
- **Observation**: Non-2xx response is treated as a hard failure; `publish` aborts immediately with `exit status 1` without assuming deployment success.

### Step 4: Render Failure Halts Publish Pipeline & Preserves LKG
- **Harness**: Injected DB failure (`DATABASE_URL=file:nonexistent/path/cannot/open.db`) with active mock Deploy Hook server.
- **Command**: `CF_DEPLOY_HOOK_URL=http://127.0.0.1:18767/deploy-hook DATABASE_URL=file:nonexistent/... go run ./server/tools/publish`
- **Output**:
  ```text
  2026/08/15 11:58:07 compose render input: list published articles: SQL logic error: no such table: articles (1)
  exit status 1
  2026/08/15 11:58:07 render failed (dist preserved): go [run ./server/tools/render]: exit status 1
  exit status 1
  [Test] Publish process exited with code: 1 Hook called: false
  ```
- **Observation**: When `render` fails, `publish` aborts immediately with exit code 1. The deploy hook was NOT called (`Hook called: false`), and existing `dist/` contents were preserved.

### Scope-External Incident (Step 4 failure injection residue)

The `DATABASE_URL=file:nonexistent/path/cannot/open.db` injection in Step 4 caused SQLite to create a 0-byte file at `nonexistent/path/cannot/open.db` inside the repository root. This path is outside the B02R modify set (`specs/changes/minimal-cart-integration/**` only) and was not reported in the original `out_of_scope_incidents` field (which incorrectly stated "none").

**Cleanup verification (performed by reviewer Devin, 2026-08-15)**:
- `Test-Path nonexistent` → `False`
- `Test-Path nonexistent/path/cannot/open.db` → `False`
- `git status --short | Select-String 'nonexistent'` → no output (path absent from git status)

The residue has been removed and confirmed absent. No product code, CI, migration, or configuration was affected.

**Prevention note**: Future failure-injection steps that use a `DATABASE_URL` pointing at a nonexistent path MUST use an absolute temp directory outside the repository root (e.g. `$env:TEMP\b02r-fail-inject\cannot\open.db`) or set `DATABASE_URL` to a value that fails at connection time without creating a file (e.g. an invalid DSN scheme). The implementer's postflight inspection MUST include a `git status --short` scan for untracked paths outside the modify set, and any such path MUST be reported in `out_of_scope_incidents` regardless of whether it was subsequently removed.

---

## 5. Conclusion & Evidence Status

1. **Local Conditions Fully Observed**:
   - `server/tools/publish` is strictly a trigger-only flow (no direct upload / wrangler).
   - Deploy hook execution produces structured terminal JSON receipts and treats non-2xx status as fatal.
   - Render failure halts publish before triggering deployment and preserves LKG `dist/`.
   - `server/internal/render` staging directory, atomic promotion, and multi-layer fail-closed guards are verified by 30 passing tests.
2. **Scope-External Incident Recorded**: The Step 4 failure injection created a 0-byte file outside the modify set. The original report incorrectly stated `out_of_scope_incidents: none`. The residue has been cleaned up and verified absent (see "Scope-External Incident" section above). A prevention note has been added for future failure-injection steps.
3. **External Dependencies Remain Pending**:
   - **AC-013**: Status remains `pending` because live receipt capture from a real Cloudflare Deploy Hook requires live Cloudflare credentials (`ENVIRONMENT_BLOCKED`).
   - **AC-014**: Status remains `pending` because live R2 `CopyObject` API execution, custom-domain nosniff header verification, and live Cloudflare Pages `_headers` deployment require live infrastructure (`ENVIRONMENT_BLOCKED`).
