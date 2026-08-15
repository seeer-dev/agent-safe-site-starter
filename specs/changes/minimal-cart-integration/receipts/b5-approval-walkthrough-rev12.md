# Revision 12 B5 Site Content Approval & Publish Gate Walkthrough

Change: `minimal-cart-integration`
Revision: `12`
Date: 2026-08-15 (Asia/Taipei)
Reviewer: Agy
Kind: `walkthrough`
Scope: B5 Site Content Approval/Publish Gate (`REQ-006`, `AC-011`, `AC-012`)

---

## 1. Environment & Pre-requisites

- **OS**: Windows (AMD64)
- **Go Version**: `go version go1.26.4 windows/amd64`
- **Database**: SQLite (`var/site.db` with WAL mode and foreign keys enabled)
- **Node**: v22.22.2

---

## 2. Automated Test Results

### 2.1 Backend Site Content Module Tests (60 / 60 Passed)
- Command: `go test ./server/internal/modules/sitecontent/ -v -count=1`
- Output:
  ```text
  === RUN   TestValidateKeyNormalization
  --- PASS: TestValidateKeyNormalization (0.00s)
  === RUN   TestMigration009Backfill
  --- PASS: TestMigration009Backfill (0.01s)
  === RUN   TestPublishedListOmitsGovernanceFields
  --- PASS: TestPublishedListOmitsGovernanceFields (0.20s)
  === RUN   TestListAllAcceptsContentRead
  --- PASS: TestListAllAcceptsContentRead (0.20s)
  === RUN   TestPublishedSortOrderIsolation
  --- PASS: TestPublishedSortOrderIsolation (0.20s)
  === RUN   TestPublishNotFound
  --- PASS: TestPublishNotFound (0.21s)
  === RUN   TestExpiredApprovalRejectsPublish
  --- PASS: TestExpiredApprovalRejectsPublish (0.21s)
  === RUN   TestMigration012SQLiteFullApply
  --- PASS: TestMigration012SQLiteFullApply (0.21s)
  === RUN   TestPublishRequiresPublishCapability
  --- PASS: TestPublishRequiresPublishCapability (0.21s)
  === RUN   TestApproveNotFound
  --- PASS: TestApproveNotFound (0.21s)
  === RUN   TestConcurrentEditInvalidatesApproval
  --- PASS: TestConcurrentEditInvalidatesApproval (0.22s)
  === RUN   TestPublishNormalizesLegacyWhitespaceKey
  --- PASS: TestPublishNormalizesLegacyWhitespaceKey (0.22s)
  === RUN   TestApproveRecordsApproverIdentityAndVersion
  --- PASS: TestApproveRecordsApproverIdentityAndVersion (0.22s)
  === RUN   TestPublishedKeyCollisionFails
  --- PASS: TestPublishedKeyCollisionFails (0.22s)
  === RUN   TestCreateAlwaysSavesDraft
  --- PASS: TestCreateAlwaysSavesDraft (0.22s)
  === RUN   TestPublishedSnapshotExpiryFilter
  --- PASS: TestPublishedSnapshotExpiryFilter (0.22s)
  === RUN   TestApproveAndPublishCapabilitySeparation
  --- PASS: TestApproveAndPublishCapabilitySeparation (0.22s)
  === RUN   TestConcurrentEditInvalidatesPublish
  --- PASS: TestConcurrentEditInvalidatesPublish (0.22s)
  === RUN   TestPublishWithStaleVersionFails
  --- PASS: TestPublishWithStaleVersionFails (0.22s)
  === RUN   TestApproveRequiresApproveCapability
  --- PASS: TestApproveRequiresApproveCapability (0.22s)
  === RUN   TestPublishedRouteIsolationAfterDraftEdit
  --- PASS: TestPublishedRouteIsolationAfterDraftEdit (0.22s)
  === RUN   TestPublishedContentWithDraftEditStillRenderable
  --- PASS: TestPublishedContentWithDraftEditStillRenderable (0.17s)
  === RUN   TestPublishWithoutApprovalFailsClosed
  --- PASS: TestPublishWithoutApprovalFailsClosed (0.17s)
  === RUN   TestUnpublishedDraftNotInPublishedList
  --- PASS: TestUnpublishedDraftNotInPublishedList (0.16s)
  === RUN   TestPublishSucceedsWithCapabilityAndApproval
  --- PASS: TestPublishSucceedsWithCapabilityAndApproval (0.18s)
  === RUN   TestDeletePublishedContentRequiresPublishCap
  --- PASS: TestDeletePublishedContentRequiresPublishCap (0.18s)
  === RUN   TestDeletePublishedRequiresPublishCap
  --- PASS: TestDeletePublishedRequiresPublishCap (0.18s)
  === RUN   TestDeleteDraftContentRequiresUpdateCap
  --- PASS: TestDeleteDraftContentRequiresUpdateCap (0.18s)
  === RUN   TestPublishHTTPWithoutApproval409
  --- PASS: TestPublishHTTPWithoutApproval409 (0.19s)
  === RUN   TestPublishSwitchesToNewVersion
  --- PASS: TestPublishSwitchesToNewVersion (0.17s)
  === RUN   TestPublishHTTPStaleVersion409
  --- PASS: TestPublishHTTPStaleVersion409 (0.19s)
  === RUN   TestUpdatePublishedContentKeepsPublishedCopyLive
  --- PASS: TestUpdatePublishedContentKeepsPublishedCopyLive (0.17s)
  === RUN   TestApproveHTTPStaleVersion409
  --- PASS: TestApproveHTTPStaleVersion409 (0.19s)
  === RUN   TestDeleteDraftSucceeds
  --- PASS: TestDeleteDraftSucceeds (0.18s)
  === RUN   TestApproveRejectsPastExpiry
  --- PASS: TestApproveRejectsPastExpiry (0.17s)
  === RUN   TestApproveHTTPSuccess
  --- PASS: TestApproveHTTPSuccess (0.19s)
  === RUN   TestApproveRejectsZeroExpiry
  --- PASS: TestApproveRejectsZeroExpiry (0.18s)
  === RUN   TestApproveHTTPRejectsEmptyUserID
  --- PASS: TestApproveHTTPRejectsEmptyUserID (0.18s)
  === RUN   TestUpdateDoesNotPromoteToPublished
  --- PASS: TestUpdateDoesNotPromoteToPublished (0.18s)
  === RUN   TestDeleteDraftAfterConcurrentPublishFails
  --- PASS: TestDeleteDraftAfterConcurrentPublishFails (0.18s)
  === RUN   TestPublishedSnapshotExpiryEditDoesNotChangeFrozenMetadata
  --- PASS: TestPublishedSnapshotExpiryEditDoesNotChangeFrozenMetadata (0.17s)
  === RUN   TestEditInvalidatesApproval
  --- PASS: TestEditInvalidatesApproval (0.18s)
  === RUN   TestUpdateHTTPRejectsEmptyKey
  --- PASS: TestUpdateHTTPRejectsEmptyKey (0.19s)
  === RUN   TestUpdateHTTPStaleVersion409
  --- PASS: TestUpdateHTTPStaleVersion409 (0.19s)
  === RUN   TestListPublishedHTTPNoGovernanceFields
  --- PASS: TestListPublishedHTTPNoGovernanceFields (0.17s)
  === RUN   TestPublishHTTPSuccess
  --- PASS: TestPublishHTTPSuccess (0.18s)
  === RUN   TestCreateRejectsInvalidPlacement
  --- PASS: TestCreateRejectsInvalidPlacement (0.17s)
  === RUN   TestUnapprovedPublishedRowFailClosed
  --- PASS: TestUnapprovedPublishedRowFailClosed (0.19s)
  === RUN   TestUpdateHTTPRejectsInvalidPlacement
  --- PASS: TestUpdateHTTPRejectsInvalidPlacement (0.13s)
  === RUN   TestUpdateRejectsEmptyKeyFailClosed
  --- PASS: TestUpdateRejectsEmptyKeyFailClosed (0.08s)
  === RUN   TestApproveWithStaleVersionFails
  --- PASS: TestApproveWithStaleVersionFails (0.11s)
  === RUN   TestUpdateWithStaleVersionFails
  --- PASS: TestUpdateWithStaleVersionFails (0.10s)
  === RUN   TestCreateHTTPRejectsInvalidPlacement
  --- PASS: TestCreateHTTPRejectsInvalidPlacement (0.10s)
  === RUN   TestUpdateRejectsUnsafeKey
  --- PASS: TestUpdateRejectsUnsafeKey (0.09s)
  === RUN   TestUpdateRejectsInvalidPlacement
  --- PASS: TestUpdateRejectsInvalidPlacement (0.09s)
  === RUN   TestCreateNormalizesOuterWhitespaceKey
  --- PASS: TestCreateNormalizesOuterWhitespaceKey (0.09s)
  PASS
  ok  	github.com/example/ai-site-starter/server/internal/modules/sitecontent	0.690s
  ```

### 2.2 Admin Typecheck and Production Build
- Command: `cd admin && npm run typecheck && npm run build`
- Output: `✓ built in 5.75s`, Exit code: 0.

### 2.3 Theme OpenAPI Contracts Check
- Command: `cd site/themes/minimal-cart && npm run check:openapi-contracts`
- Output: `OpenAPI contract check PASSED: response schemas match Go/TS guarantees and promo enumeration is absent.`, Exit code: 0.

---

## 3. Independent API End-to-End Walkthrough (not full UI acceptance)

The Go development server was launched via `go run ./server/tools/dev` with `AUTH_MODE=dev`.

### Step a: Create a Draft Site Content Block
- **Request**:
  ```http
  POST /api/admin/site-content HTTP/1.1
  Host: localhost:8080
  Authorization: Bearer dev-admin
  Content-Type: application/json

  {
    "key": "policy.terms",
    "placement": "policy",
    "title": "服務條款",
    "body": "本網站服務條款內容（草稿）。",
    "sort_order": 10
  }
  ```
- **Response**: `HTTP/1.1 201 Created`
  ```json
  {
    "id": "fb322d84aa9c7797faceb54b15db67a3",
    "key": "policy.terms",
    "placement": "policy",
    "title": "服務條款",
    "body": "本網站服務條款內容（草稿）。",
    "status": "draft",
    "sort_order": 10,
    "updated_unix": 1786758347,
    "draft_version": 1
  }
  ```
- **Observation**: Draft created with `draft_version: 1`, `status: "draft"`. It does not appear in public `GET /api/site-content/published` (`items: null`).

### Step b: Approve with Future Expiry
- **Request**:
  ```http
  POST /api/admin/site-content/fb322d84aa9c7797faceb54b15db67a3/approve HTTP/1.1
  Host: localhost:8080
  Authorization: Bearer dev-admin
  Content-Type: application/json

  {
    "expected_draft_version": 1,
    "expiry_unix": 1787363155
  }
  ```
- **Response**: `HTTP/1.1 200 OK`
  ```json
  {
    "id": "fb322d84aa9c7797faceb54b15db67a3",
    "key": "policy.terms",
    "placement": "policy",
    "status": "draft",
    "draft_version": 1,
    "approved_version": 1,
    "approver_user_id": "local-admin",
    "approved_unix": 1786758355,
    "approved_expiry_unix": 1787363155
  }
  ```
- **Observation**: Approval recorded atomically against `draft_version: 1`. Approver identity is securely derived from `auth.Principal` (`"local-admin"`). Status remains `"draft"`.

### Step c: Publish the Approved Content
- **Request**:
  ```http
  POST /api/admin/site-content/fb322d84aa9c7797faceb54b15db67a3/publish HTTP/1.1
  Host: localhost:8080
  Authorization: Bearer dev-admin
  Content-Type: application/json

  {
    "expected_draft_version": 1
  }
  ```
- **Response**: `HTTP/1.1 200 OK`
  ```json
  {
    "id": "fb322d84aa9c7797faceb54b15db67a3",
    "key": "policy.terms",
    "placement": "policy",
    "status": "published",
    "published_title": "服務條款",
    "published_body": "本網站服務條款內容（草稿）。",
    "published_key": "policy.terms",
    "published_placement": "policy",
    "published_sort_order": 10,
    "published_version": 1,
    "published_approver_user_id": "local-admin",
    "published_approved_unix": 1786758355,
    "published_approval_expiry_unix": 1787363155
  }
  ```
- **Observation**: Published atomically. Frozen published snapshot metadata (`published_version: 1`, approver, timestamps) populated.

### Step d: Verify Public API
- **Request**: `GET /api/site-content/published HTTP/1.1`
- **Response**: `HTTP/1.1 200 OK`
  ```json
  {
    "items": [
      {
        "id": "fb322d84aa9c7797faceb54b15db67a3",
        "key": "policy.terms",
        "placement": "policy",
        "title": "服務條款",
        "body": "本網站服務條款內容（草稿）。",
        "status": "published",
        "sort_order": 10,
        "updated_unix": 1786758358
      }
    ]
  }
  ```
- **Observation**: Public endpoint returns the active published item and strictly strips internal governance fields (`published_version`, `published_approver_user_id`, etc.).

### Step e: Render and Inspect `dist/`
- **Command**: `go run ./server/tools/render`
- **Output**: `rendered 0 article(s), 5 product(s), 4 categor(y/ies), 1 content page(s) into dist/`
- **Inspection**: `dist/content/policy.terms/index.html` exists and contains server-rendered title and body.

### Step f: Edit Draft and Verify Published Copy Unchanged
- **Request**:
  ```http
  PUT /api/admin/site-content/fb322d84aa9c7797faceb54b15db67a3 HTTP/1.1
  Host: localhost:8080
  Authorization: Bearer dev-admin
  Content-Type: application/json

  {
    "key": "policy.terms",
    "placement": "policy",
    "title": "服務條款（修訂草稿）",
    "body": "本網站服務條款已更新（尚未批准）。",
    "sort_order": 10,
    "expected_draft_version": 1
  }
  ```
- **Response**: `HTTP/1.1 200 OK` (`draft_version: 2`, `approved_version: 1`, `published_version: 1`)
- **Verification**: `GET /api/site-content/published` returned original unedited content (`title: "服務條款"`, `body: "本網站服務條款內容（草稿）。"`). Draft edits do NOT alter the live published copy.

### Step g: Attempt Publish Without Re-Approval (409 Conflict)
- **Request (with `expected_draft_version: 2`)**:
  ```http
  POST /api/admin/site-content/fb322d84aa9c7797faceb54b15db67a3/publish HTTP/1.1
  Host: localhost:8080
  Authorization: Bearer dev-admin
  Content-Type: application/json

  {
    "expected_draft_version": 2
  }
  ```
- **Response**: `HTTP/1.1 409 Conflict`
  ```json
  {"error":"no current approval"}
  ```
- **Request (with stale `expected_draft_version: 1`)**:
  ```http
  POST /api/admin/site-content/fb322d84aa9c7797faceb54b15db67a3/publish HTTP/1.1
  Host: localhost:8080
  Authorization: Bearer dev-admin
  Content-Type: application/json

  {
    "expected_draft_version": 1
  }
  ```
- **Response**: `HTTP/1.1 409 Conflict`
  ```json
  {"error":"stale draft version"}
  ```
- **Observation**: Publish is strictly blocked (409 Conflict) when draft version is unapproved or stale.

### Step h: Re-Approve and Re-Publish
- **Approve Request**: `POST /api/admin/site-content/fb322d84aa9c7797faceb54b15db67a3/approve` with `expected_draft_version: 2` -> `HTTP/1.1 200 OK` (`approved_version: 2`).
- **Publish Request**: `POST /api/admin/site-content/fb322d84aa9c7797faceb54b15db67a3/publish` with `expected_draft_version: 2` -> `HTTP/1.1 200 OK` (`published_version: 2`).
- **Verification**: `GET /api/site-content/published` and `go run ./server/tools/render` immediately reflect version 2.

### Step i: Test Expired Approval (400 on Past Approve / 409 on Expired Publish)
1. `POST /api/admin/site-content` created draft `policy.privacy`.
2. `POST /api/admin/site-content/{id}/approve` with `expiry_unix: 1000` (past epoch):
   - **Response**: `HTTP/1.1 400 Bad Request` -> `{"error":"approval expiry must be in the future"}`.
3. Approved `policy.privacy` with valid future expiry, then simulated expiry by setting `approved_expiry_unix = 1000` in the database.
4. `POST /api/admin/site-content/{id}/publish` with `expected_draft_version: 1`:
   - **Response**: `HTTP/1.1 409 Conflict` -> `{"error":"no current approval"}`.
5. `GET /api/site-content/published` verified that the expired unapproved block is NOT published.

---

## 4. Post-review residue check

- **Reviewer**: Codex
- **Time**: 2026-08-15 20:39 (Asia/Taipei)
- **Method**: read-only SQLite query against `var/site.db`.
- **Observed state**: no rows with `key` equal to `policy.terms` or `policy.privacy` remain. The database contains only the pre-existing `footer.about`, `home.hero`, and `home.popup` rows, so the walkthrough did not leave either temporary policy block behind.
- **Limitation**: this verifies the current database state; it does not retroactively establish the exact cleanup command or time. The missing UI and no-JavaScript observations remain blockers.

---

## 5. Findings & Conclusion

- **Findings**:
  - The Go backend, SQLite storage, and Go API routes for B5 approval, versioning, expiry, capability separation, and public snapshot isolation contracts behave exactly as specified. 60 top-level site content unit/integration tests pass.
  - Interactive local Vue admin UI walkthrough (button clicks, interactive dialog flow) and JavaScript-disabled public policy / footer navigation inspection were not observed in this receipt.
- **Conclusion**:
  - Backend and API contract behaviors are validated.
  - `AC-011` and `AC-012` remain `pending` with the interactive local admin UI walkthrough and JavaScript-disabled policy navigation walkthrough recorded as explicit blockers.
