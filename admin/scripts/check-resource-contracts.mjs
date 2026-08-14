import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

const root = resolve(import.meta.dirname, '..')
const read = (path) => readFileSync(resolve(root, path), 'utf8')
const page = read('src/pages/ResourceListPage.vue')
const orders = read('src/config/resources/orders.ts')
const products = read('src/config/resources/products.ts')
const payments = read('src/config/resources/payment-methods.ts')
const shipping = read('src/config/resources/shipping-methods.ts')
const promos = read('src/config/resources/promos.ts')
const content = read('src/config/resources/content.ts')
const roles = read('src/config/roles.ts')
const utils = read('src/lib/utils.ts')
const types = read('src/lib/types.ts')
const confirmDialog = read('src/components/ui/ConfirmDialog.vue')
const errors = []
const promoForm = promos.slice(promos.indexOf('  form:'))

for (const marker of ["fd.w === 'switch'", "fd.w === 'number'", "fd.w === 'datetime'", 'buildFormPayload()', 'formDirty', 'requestCloseForm', 'e.status === 409']) {
  if (!page.includes(marker)) errors.push(`ResourceListPage missing payload coercion: ${marker}`)
}
// Revision 8: Product form now uses product_images (media-uploader widget)
// instead of legacy image/images fields. The legacy fields are removed from
// the form entirely — ProductInput no longer accepts them (DisallowUnknownFields).
if (!page.includes("fd.w === 'media-uploader'")) errors.push('product form must handle media-uploader widget')
if (!products.includes("{ k: 'slug'") || !products.includes("{ k: 'product_images'")) errors.push('product form must preserve slug and product_images')
// Legacy image/images fields must NOT be in the product form.
for (const field of ['image', 'images']) {
  const marker = `{ k: '${field}'`
  if (products.includes(marker)) errors.push(`product form must NOT have legacy ${field} field (ProductInput rejects it via DisallowUnknownFields)`)
}
if (products.includes("k: 'bulk-category-")) errors.push('product form exposes unsupported bulk category actions')
for (const field of ['provider_label', 'environment', 'readiness_status', 'enabled']) {
  if (!payments.includes(`{ k: '${field}'`)) errors.push(`payment form missing ${field}`)
}
if (!payments.includes("opts: ['sandbox', 'production']")) errors.push('payment environment options do not match Go validation')
if (!payments.includes("opts: ['pending_setup', 'ready']")) errors.push('payment readiness options do not match Go validation')
if (!payments.includes("updateCap: 'twcommerce.admin'")) errors.push('payment edit capability does not match Go authorization')
if (!promoForm.includes("{ k: 'starts_unix'") || !promoForm.includes("{ k: 'expires_unix'")) errors.push('promo form must send *_unix fields')
if (promoForm.includes("{ k: 'starts_at'") || promoForm.includes("{ k: 'expires_at'")) errors.push('promo form still sends display-only *_at fields')
// ----- Site content approval/version/expiry contract (B5/REQ-006/AC-011) -----
// The admin config must wire approve and publish as separate rowActions
// with separate endpoints and expect: 'draft_version' for optimistic
// concurrency. Approve must use expiryInput (operator-decided expiry),
// NOT a hardcoded expiryHours.
if (!content.includes('adminMinimalCartContentApprove') || !content.includes("'/admin/site-content/{id}/approve'")) {
  errors.push('site content approve rowAction + endpoint must be wired (approval gate is live)')
}
if (!content.includes('adminMinimalCartContentPublish') || !content.includes("'/admin/site-content/{id}/publish'")) {
  errors.push('site content publish rowAction + endpoint must be wired (approval gate is live)')
}
// Approve and publish must have separate endpoints (not the same path).
if (content.includes("'/admin/site-content/{id}/approve'") && content.includes("'/admin/site-content/{id}/publish'")) {
  // OK — both present and different
} else {
  errors.push('site content approve and publish must have separate API endpoints')
}
// Both approve and publish must send expected_draft_version. The rowAction
// definitions span multiple lines, so we extract each action block (from
// the op line to the closing brace) and check within the block.
const contentLines = content.split('\n')
function extractActionBlock(opName) {
  const opLineIdx = contentLines.findIndex((l) => l.includes(`op: '${opName}'`))
  if (opLineIdx < 0) return ''
  // Find the start of this rowAction object (search backwards for '{')
  let start = opLineIdx
  while (start > 0 && !contentLines[start].includes('{')) start--
  // Find the end (next standalone '}' at the same or lower indent)
  let end = opLineIdx
  let depth = 0
  for (let i = start; i < contentLines.length; i++) {
    for (const ch of contentLines[i]) {
      if (ch === '{') depth++
      if (ch === '}') depth--
    }
    if (depth <= 0 && i > opLineIdx) { end = i; break }
  }
  return contentLines.slice(start, end + 1).join('\n')
}
for (const op of ['adminMinimalCartContentApprove', 'adminMinimalCartContentPublish']) {
  const block = extractActionBlock(op)
  if (!block) {
    errors.push(`site content ${op} action not found in content config`)
    continue
  }
  if (!block.includes("expect: 'draft_version'")) {
    errors.push(`site content ${op} action must use expect: 'draft_version' for optimistic concurrency`)
  }
}
// Approve must use expiryInput (operator-decided), NOT expiryHours (hardcoded).
const approveBlock = extractActionBlock('adminMinimalCartContentApprove')
if (approveBlock) {
  if (!approveBlock.includes('expiryInput: true')) {
    errors.push('site content approve action must use expiryInput: true (operator decides expiry, not hardcoded)')
  }
  if (approveBlock.includes('expiryHours')) {
    errors.push('site content approve action must NOT use expiryHours (hardcoded expiry replaces operator decision)')
  }
}
// Content config must show governance columns for both current approval
// and frozen published approval. The raw unix fields must NOT appear as
// col keys — they must go through rowMap + formatUnix so the table shows
// human-readable datetime strings, not raw Unix integers.
const rawUnixFields = [
  'approved_unix',
  'approved_expiry_unix',
  'published_approved_unix',
  'published_approval_expiry_unix',
]
for (const f of rawUnixFields) {
  if (content.includes(`k: '${f}'`)) {
    errors.push(`site content cols must NOT use raw unix field '${f}' as a col key — use rowMap + formatUnix instead`)
  }
}
// rowMap must format all 4 raw unix fields via formatUnix.
for (const f of rawUnixFields) {
  if (!content.includes(`formatUnix(raw.${f})`)) {
    errors.push(`site content rowMap must format '${f}' via formatUnix (no raw unix in table)`)
  }
}
// Cols must include the formatted datetime columns and published governance fields.
const requiredCols = [
  'draft_version',
  'approved_version',
  'approver_user_id',
  'approved_at',
  'approved_expiry_at',
  'published_version',
  'published_approver_user_id',
  'published_approved_at',
  'published_approval_expiry_at',
]
for (const col of requiredCols) {
  if (!content.includes(`k: '${col}'`)) {
    errors.push(`site content cols must include ${col} for governance visibility`)
  }
}
// Content config must include policy in desc, filter, and form options
// (OpenAPI/REQ-006 define policy as a valid placement).
if (!content.includes('政策')) {
  errors.push('site content desc must mention policy (政策) placement')
}
if (!content.includes("['policy', '政策']")) {
  errors.push('site content filter must include policy option')
}
if (!content.includes("'policy'")) {
  errors.push('site content form placement options must include policy')
}
// Content form placement must be req:true (server rejects empty placement
// with ErrInvalidPlacement → 400; the UI must not allow empty either).
// Search only within the form section (after "form: {") to avoid matching
// filter/cols/rowAction lines.
const formIdx = content.indexOf('\n  form: {')
const formSection = formIdx >= 0 ? content.slice(formIdx) : ''
const formPlacementLine = formSection.split('\n').find((l) => l.includes("k: 'placement'") && l.includes("w: 'select'"))
if (!formPlacementLine) {
  errors.push('site content form must have a placement select field')
} else if (!formPlacementLine.includes('req: true')) {
  errors.push('site content form placement field must be req: true (server rejects empty placement with 400)')
}
// Content form placement options must match the server allowlist exactly
// (hero, announcement, popup, footer, policy). No more, no less.
const formOptsMatch = content.match(/opts:\s*\[(['"]hero['"],\s*['"]announcement['"],\s*['"]popup['"],\s*['"]footer['"],\s*['"]policy['"])\]/)
if (!formOptsMatch) {
  errors.push('site content form placement opts must be exactly [hero, announcement, popup, footer, policy] matching server allowlist')
}
// Action visibility: approve must use 'approve_always' (always visible to
// cap holders — missing/stale/expired approvals all need re-approve),
// publish must use 'publish_ready' (version current + approver + not expired).
if (!content.includes("showWhen: 'approve_always'")) {
  errors.push("site content approve action must use showWhen: 'approve_always' (operator can re-approve missing/stale/expired)")
}
if (!content.includes("showWhen: 'publish_ready'")) {
  errors.push("site content publish action must use showWhen: 'publish_ready' (version current + approver + not expired)")
}

// ----- evalShowWhen contract: approve_always + publish_ready + expired case -----
if (!utils.includes("'approve_always'")) {
  errors.push("evalShowWhen must handle 'approve_always' expression")
}
if (!utils.includes("'publish_ready'")) {
  errors.push("evalShowWhen must handle 'publish_ready' expression")
}
// publish_ready must check approved_expiry_unix > now (not just version + approver).
if (!utils.includes('approved_expiry_unix')) {
  errors.push("evalShowWhen publish_ready must check approved_expiry_unix for expiry")
}

// ----- Roles cap alignment: content.approve must be in owner + manager -----
if (!roles.includes("'content.approve'")) {
  errors.push("roles.ts must include content.approve (server resolver grants it to owner + manager)")
}
// Count occurrences — should appear at least twice (owner + manager).
const approveCapCount = (roles.match(/'content\.approve'/g) || []).length
if (approveCapCount < 2) {
  errors.push(`roles.ts must grant content.approve to both owner and manager (found ${approveCapCount} occurrence(s), need >= 2)`)
}
// Admin must not invent caps the server doesn't grant. The server resolver
// (resolver.go) grants: content.read, content.create, content.update,
// content.approve, content.publish. No other content.* caps should exist.
const allContentCaps = new Set()
for (const m of roles.matchAll(/'content\.[a-z]+'/g)) {
  allContentCaps.add(m[0].replace(/'/g, ''))
}
const validContentCaps = new Set(['content.read', 'content.create', 'content.update', 'content.approve', 'content.publish'])
for (const cap of allContentCaps) {
  if (!validContentCaps.has(cap)) {
    errors.push(`roles.ts invents cap '${cap}' not granted by server resolver`)
  }
}

// ----- Types contract: RowAction.expiryInput + OpsDef.approve + ResourceApiDef.approve -----
if (!types.includes('roOnEdit?: boolean')) {
  errors.push('FieldDef must include roOnEdit?: boolean (read-only on edit, editable on create)')
}
if (!types.includes('nullable?: boolean')) {
  errors.push('FieldDef must include nullable?: boolean (blank number -> JSON null)')
}
if (!types.includes('expectedVersionField?: string')) {
  errors.push('ResourceDef must include expectedVersionField?: string for expected_version injection')
}
if (!page.includes('fd.roOnEdit') || !page.includes('isFieldReadOnly')) {
  errors.push('ResourceListPage must honor roOnEdit via isFieldReadOnly')
}
if (!page.includes('fd.nullable')) {
  errors.push('ResourceListPage must send blank nullable numbers as JSON null')
}
if (!page.includes('payload.expected_version = Number(row[r.expectedVersionField])')) {
  errors.push('ResourceListPage must inject expected_version from expectedVersionField (not a hardcoded resource name)')
}
if (!shipping.includes("expectedVersionField: 'version'")) {
  errors.push("shipping resource must set expectedVersionField: 'version'")
}
if (!shipping.includes('roOnEdit: true')) {
  errors.push('shipping method field must be roOnEdit on create/edit')
}
if (!shipping.includes('nullable: true') || !shipping.includes("k: 'free_threshold'")) {
  errors.push('shipping free_threshold must be a nullable number field')
}
if (shipping.includes('delete:') || shipping.includes('ops.del') || shipping.includes("k: 'delete'") || shipping.includes("k: 'del'")) {
  errors.push('shipping resource must not expose a delete action')
}
if (!shipping.includes("createCap: 'twcommerce.create'") || !shipping.includes("updateCap: 'twcommerce.update'")) {
  errors.push('shipping resource capabilities must be twcommerce.create / twcommerce.update')
}

if (!types.includes('expiryInput?: boolean')) {
  errors.push("RowAction type must include expiryInput?: boolean")
}
if (!types.includes('allCaps?: Capability[]')) {
  errors.push("RowAction type must include allCaps?: Capability[] (all-of capability gate for multi-cap actions like restock)")
}
if (!types.includes('approve?: string') || !types.includes('approve?: string;')) {
  errors.push("OpsDef must include approve?: string")
}
if (!types.includes("approve?: string; // POST path with {id} for approve action")) {
  errors.push("ResourceApiDef must include approve?: string (POST path for approve)")
}

// ----- ConfirmDialog contract: expiry input UX -----
// Label must have for/id association for accessibility.
if (!confirmDialog.includes('for="confirm-expiry-input"') || !confirmDialog.includes('id="confirm-expiry-input"')) {
  errors.push('ConfirmDialog expiry label must have for/id association for accessibility')
}
// Must show timezone hint.
if (!confirmDialog.includes('tzHint')) {
  errors.push('ConfirmDialog must show local timezone hint for expiry input')
}
// Must validate future time (not just non-empty).
if (!confirmDialog.includes('到期時間必須在現在之後') && !confirmDialog.includes('ms <= now')) {
  errors.push('ConfirmDialog must validate expiry is in the future (not just non-empty)')
}
// Must preserve user value on invalid input (no clearing).
if (!confirmDialog.includes('Preserve the user')) {
  errors.push('ConfirmDialog must preserve user expiry value on invalid input')
}
// Must have role="alert" for inline validation.
if (!confirmDialog.includes('role="alert"')) {
  errors.push('ConfirmDialog inline validation must have role="alert" for accessibility')
}

// ----- ResourceListPage contract: buildPostBody, 409 handling, expected_draft_version injection -----
// buildPostBody must exist and use operator-decided expiry (not hardcoded).
if (!page.includes('function buildPostBody')) {
  errors.push('ResourceListPage must have buildPostBody for POST actions (approve/publish)')
}
if (!page.includes('datetimeLocalToUnix(confirmExpiry.value)')) {
  errors.push('buildPostBody must convert operator expiry input to unix (not hardcoded hours)')
}
if (page.includes('expiryHours')) {
  errors.push('ResourceListPage must NOT use expiryHours (hardcoded expiry replaces operator decision)')
}
// 409 must keep dialog open and show conflict (not fake success).
if (!page.includes('e?.status === 409')) {
  errors.push('ResourceListPage must detect 409 status and show conflict message (not fake success)')
}
if (!page.includes('衝突')) {
  errors.push('ResourceListPage 409 error must show conflict message (衝突)')
}
// Update form must inject expected_draft_version from row (not user-editable).
if (!page.includes('payload.expected_draft_version = Number(row.draft_version)')) {
  errors.push('ResourceListPage update form must inject expected_draft_version from row.draft_version (not user-editable)')
}
// resolveActionEndpoint must handle approve separately from publish.
if (!page.includes("a.op === r.ops.approve && r.api.approve")) {
  errors.push('resolveActionEndpoint must handle approve as a separate endpoint from publish')
}

// ----- Order restock action contract (B7) -----------------------------------
// The restock rowAction must use op: 'adminMinimalCartOrdersRestock',
// allCaps: ['orders.returns', 'inventory.adjust'] (the server requires BOTH
// capabilities — a single cap gate would let a principal with only one
// capability see the action and get a 403), expect: 'version',
// showWhen: 'return_request_status=received', reason: true, and
// restockItems: true. The Go handler decodes expected_version as int and
// requires a non-empty reason + idempotency_key + items array.
// restockItems: true opens the per-item restock modal (not the confirm dialog)
// so the operator can declare per-item returned/restocked quantities with
// defaults of 0 (not auto-full-quantity).
const restockLines = orders.split('\n').filter((line) => line.includes("op: 'adminMinimalCartOrdersRestock'"))
if (restockLines.length === 0) {
  errors.push('orders resource must have a restock rowAction targeting adminMinimalCartOrdersRestock')
}
for (const line of restockLines) {
  if (!line.includes("allCaps: ['orders.returns', 'inventory.adjust']")) {
    errors.push(`restock action must use allCaps: ['orders.returns', 'inventory.adjust'] (server requires BOTH capabilities, not a single cap): ${line.trim()}`)
  }
  if (!line.includes("expect: 'version'")) {
    errors.push(`restock action must use expect: 'version' (Go handler decodes expected_version as int): ${line.trim()}`)
  }
  if (!line.includes("showWhen: 'return_request_status=received'")) {
    errors.push(`restock action must use showWhen: 'return_request_status=received' (goods must be physically received before restock): ${line.trim()}`)
  }
  if (!line.includes('reason: true')) {
    errors.push(`restock action must use reason: true (Go handler requires non-empty reason): ${line.trim()}`)
  }
  if (!line.includes('restockItems: true')) {
    errors.push(`restock action must use restockItems: true (opens per-item restock modal, not confirm dialog): ${line.trim()}`)
  }
}
// ResourceListPage must resolve the restock endpoint.
if (!page.includes("a.op === r.ops.restock && r.api.restock")) {
  errors.push('ResourceListPage resolveActionEndpoint must handle restock op + api.restock endpoint')
}
// The restock modal must generate the idempotency key ONCE on open
// (openRestockModal), NOT on each submit. This ensures retries with
// unknown results reuse the same key so the server can detect replay.
if (!page.includes('restockIdempotencyKey.value = crypto.randomUUID()')) {
  errors.push('ResourceListPage openRestockModal must generate idempotency_key once on modal open (not on each submit)')
}
// The restock body must use the stable restockIdempotencyKey, not a
// per-submit crypto.randomUUID().
if (!page.includes('idempotency_key: restockIdempotencyKey.value')) {
  errors.push('ResourceListPage buildRestockBody must use the stable restockIdempotencyKey (not per-submit crypto.randomUUID())')
}
// The restock modal must default per-item returned/restocked to 0,
// NOT to the full ordered quantity. Auto-full-restock would violate
// per-item inspection and damage handling.
if (!page.includes('returned: 0') || !page.includes('restocked: 0')) {
  errors.push('ResourceListPage openRestockModal must default per-item returned/restocked to 0 (not full quantity)')
}
// The restock modal must open via askRowAction detecting restockItems.
if (!page.includes('a.restockItems')) {
  errors.push('ResourceListPage askRowAction must check a.restockItems to open the restock modal')
}
// The restock modal must fetch the AUTHORITATIVE order detail (GET) on
// open — the list row's items come from items_json and do NOT include
// the order_items ledger columns (returned_quantity, restocked_quantity).
if (!page.includes('api.get<Record<string, any>>(detailPath)')) {
  errors.push('ResourceListPage openRestockModal must GET authoritative order detail (list row items lack ledger columns)')
}
// The restock modal must display existing cumulative ledger values
// (existingReturned, existingRestocked) so the operator knows what has
// already been received/restocked.
if (!page.includes('existingReturned') || !page.includes('existingRestocked')) {
  errors.push('ResourceListPage restock modal must track existingReturned/existingRestocked from the authoritative order detail')
}
// The restocked input max must use the cumulative maxRestocked function,
// NOT a per-action max of it.returned. A restock-only follow-up
// (returned=0, restocked=1) is legal when existingReturned > existingRestocked.
if (!page.includes('maxRestocked(it)')) {
  errors.push('ResourceListPage restocked input must use maxRestocked(it) for cumulative max (not per-action it.returned)')
}
// The restockCanSubmit must NOT enforce a per-action restocked <= returned
// constraint — only cumulative constraints. A per-action check would block
// legal restock-only follow-ups.
if (page.includes('it.restocked > it.returned')) {
  errors.push('ResourceListPage restockCanSubmit must NOT enforce per-action restocked <= returned (blocks legal restock-only follow-ups)')
}
// The stable key must bind to the first-submit payload fingerprint and
// rotate on payload change.
if (!page.includes('restockSubmittedFingerprint')) {
  errors.push('ResourceListPage must track restockSubmittedFingerprint for stable key binding (rotate on payload change, reuse on retry)')
}

// ----- Order status/return-status expected_version contract -----------------
// The Go handlers for adminMinimalCartOrdersStatusUpdate and
// adminMinimalCartOrdersReturnStatusUpdate decode expected_version as an int
// (httpx.DecodeJSON -> int field). If the admin config sends a status string
// (e.g. "pending", "requested") as expected_version, the Go handler returns
// 400 and the admin consumer is broken. This check fail-closes on any order
// status/return-status rowAction that does not use expect: 'version'.
//
// buildStatusBody in ResourceListPage.vue reads row[a.expect] and sends it
// as expected_version. With expect: 'version' this sends the numeric
// aggregate version from the API response (Order.version, json:"version").
// With the old expect: 'status' / expect: 'return_request_status' it would
// send a status string, causing a 400.
const ORDER_CONCURRENCY_OPS = [
  'adminMinimalCartOrdersStatusUpdate',
  'adminMinimalCartOrdersReturnStatusUpdate',
]
const FORBIDDEN_EXPECT_VALUES = ['status', 'return_request_status', 'fulfillment_status', 'payment_status']

// Extract all rowAction lines that reference an order concurrency op.
const orderLines = orders.split('\n').filter((line) =>
  ORDER_CONCURRENCY_OPS.some((op) => line.includes(`op: '${op}'`)),
)

if (orderLines.length === 0) {
  errors.push('orders resource has no rowActions targeting order status/return-status ops')
}

for (const line of orderLines) {
  // Every order concurrency action MUST have expect: 'version'.
  if (!line.includes("expect: 'version'")) {
    // Check if it has a forbidden expect value (the old broken config).
    const hasForbidden = FORBIDDEN_EXPECT_VALUES.some((v) => line.includes(`expect: '${v}'`))
    if (hasForbidden) {
      errors.push(`order status/return-status action sends a status string as expected_version (broken): ${line.trim()}`)
    } else if (line.includes('expect:')) {
      errors.push(`order status/return-status action uses wrong expect field (must be 'version'): ${line.trim()}`)
    } else {
      errors.push(`order status/return-status action missing expect: 'version' (no optimistic-concurrency guard): ${line.trim()}`)
    }
  }
  // Fail-closed: no order concurrency action may use a forbidden expect value.
  for (const forbidden of FORBIDDEN_EXPECT_VALUES) {
    if (line.includes(`expect: '${forbidden}'`)) {
      errors.push(`order status/return-status action must not use expect: '${forbidden}' — Go handler decodes expected_version as int, a status string causes 400: ${line.trim()}`)
    }
  }
}

// Verify buildStatusBody reads row[a.expect] for expected_version.
// This is the runtime contract: the config field 'expect' names the row
// property that becomes expected_version. If buildStatusBody is changed to
// hardcode a field, the config contract breaks.
if (!page.includes('expected_version: row[a.expect]')) {
  errors.push("buildStatusBody must read expected_version from row[a.expect] (the config 'expect' field names the row property)")
}

// Contract regression: prove the check fails on the old broken config.
// If someone reverts expect to 'status' or 'return_request_status', this
// block must catch it. We simulate by checking that no line in orders.ts
// has the old pattern.
for (const line of orders.split('\n')) {
  for (const op of ORDER_CONCURRENCY_OPS) {
    if (line.includes(`op: '${op}'`)) {
      for (const forbidden of FORBIDDEN_EXPECT_VALUES) {
        if (line.includes(`expect: '${forbidden}'`)) {
          errors.push(`CONTRACT REGRESSION: order action uses expect: '${forbidden}' instead of 'version' — Go handler would 400 on string expected_version: ${line.trim()}`)
        }
      }
    }
  }
}

if (errors.length) {
  console.error('Resource contract check FAILED:')
  for (const error of errors) console.error(`  - ${error}`)
  process.exit(1)
}
console.log('Resource contract check PASSED: admin form payloads match active Go contracts.')
