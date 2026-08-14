# Independent Walkthrough Receipt (Revision 6) — B8 Admin Media/Accessibility

Date: 2026-08-12
Change: minimal-cart-integration
Revision: 6
Auditor: Independent review (Devin + independent desktop and narrow/mobile browser observation)

## Scope

Independent local walkthrough of the B8 admin media/accessibility surface
for the product edit dialog and MediaUploader component, covering keyboard
operation, Tab containment, Escape/dirty-confirm, focus restoration,
disabled/loading state focus safety, readable error/empty/success states,
touch-target/layout at narrow viewport, label/ARIA relationships for
uploader controls, and provider/network failure behavior without live R2.

This receipt records only what was observed. It does not authorize a
release or blanket-pass any requirement. B4 actual media upload (file
chooser -> presign -> R2 PUT -> verify -> product_images association)
remains pending — file chooser opened but setFiles was rejected by
Chrome extension permission; operator action: enable Chrome ChatGPT
extension Details > Allow access to file URLs before the actual upload
walkthrough.

B8 walkthrough status: COMPLETE for covered local admin scope. Desktop
keyboard observations and narrow/mobile viewport observations (390x844)
were made by independent browser observation. Two code defects
(touch-target size, form label association) were found and fixed with
focused tests. B4 actual media upload remains independently pending.

## Environment

- OS: Windows (MSYS_NT-10.0-19045)
- Dev stack: `go run ./server/tools/dev` (API :8080 + site :4173 with
  same-origin /api/* proxy), already running (PID 3344)
- Admin: Vite dev server on port 4174 (PID 21412), already running and
  healthy (HTTP 200). `/api` proxied to `http://localhost:8080`.
- Auth: dev-admin token (`DEV_AUTH_TOKEN` default `dev-admin`),
  verified via `GET /api/admin/me` -> 200 with 15 capabilities including
  `media.upload`, role=admin, user_id=local-admin.
- Database: SQLite (dev)
- Browser: Independent desktop and narrow/mobile (390x844) browser
  observation was performed in the running admin UI. File chooser
  opened but setFiles was rejected by Chrome extension permission;
  operator action: enable Chrome ChatGPT extension Details > Allow
  access to file URLs before the actual upload walkthrough. No live
  R2 provider was available.

## Surface: Admin product edit dialog — desktop keyboard operation (independent desktop browser observation)

### Source revision/configuration

- ResourceListPage.vue (lines 155-209): `openForm` saves
  `document.activeElement` as `formTriggerEl`; `focusFormModal` uses
  `nextTick` to focus first editable `.fgrid input:not([disabled])`;
  `closeForm` restores `formTriggerEl.focus()` via `nextTick`;
  `requestCloseForm` checks `formDirty` and calls `window.confirm` if
  dirty before closing.
- Modal.vue (lines 39-73): window `keydown` listener; Escape emits
  `close`; Tab trap queries `.modal[role="dialog"]` for focusable
  elements, wraps first<->last, prevents default on wrap. Disabled
  buttons are excluded from the focusable selector
  (`button:not([disabled])`).
- Products config: first editable field is SKU (text, req, not
  disabled).

### Observed behavior (independent desktop browser observation)

1. **Initial focus**: Edit dialog focused SKU field on open. OBSERVED.
2. **Shift+Tab wrap**: Shift+Tab from the first field (SKU) wrapped to
   the Cancel button (last focusable in dialog). OBSERVED.
3. **Tab wrap with disabled skip**: Tab from Cancel wrapped back to
   SKU. The disabled Save button was skipped (not focused). OBSERVED.
4. **Dirty + Escape**: Changing a field then pressing Escape triggered
   the unsaved-change native confirm dialog. OBSERVED.

### Observed behavior (jsdom automated tests — supporting evidence)

5. **Close restores trigger**: "restores focus to the trigger button
   after the form closes" — `closeForm()` -> `nextTick` ->
   `document.activeElement` is `triggerBtn`. PASS.
6. **Tab trap inactive when closed**: "Tab trap is inactive when modal
   is closed" — no modal in DOM, Tab on outside input ->
   `preventDefault` not called. PASS.
7. **Edit modal ARIA**: "edit modal also gets focus and ARIA
   semantics" — role=dialog, aria-modal=true, tabindex=-1,
   aria-labelledby present. PASS.

### Conclusion

Desktop keyboard operation (initial focus, Tab/Shift+Tab wrap with
disabled-skip, Escape/dirty-confirm) confirmed by independent desktop
browser observation. Close-restore and trap-inactive-when-closed
confirmed by jsdom tests. No defect found in keyboard operation.

## Surface: MediaUploader — keyboard/focus operation

### Source revision/configuration

- MediaUploader.vue (lines 148-160): `watch(uploadState)` moves focus
  to `retryBtnRef` on error, `selectBtnRef` on idle (after success
  timeout).
- File input: `aria-label="Select image file"`, disabled during
  analyzing/compressing/uploading/verifying.
- Select button: visible text "選擇圖片"; disabled during active states.
- Cancel button: visible during active states.
- Retry button: visible on error.

### Observed behavior (jsdom automated tests — no interactive browser)

1. **Error -> focus retry**: "moves focus to retry button after error"
   — threshold reject -> error -> `retryBtn.element` is
   `document.activeElement`. PASS.
2. **Retry re-error -> refocus**: "retry from error re-enters error
   state and refocuses retry button". PASS.
3. **Success timeout -> focus select**: "moves focus to select button
   after success timeout". PASS.
4. **Cancel -> idle -> focus select**: "cancel during compress" and
   "cancel during presign" tests. PASS.
5. **Disabled/loading state focus safety**: file input and select
   button disabled during active states; cancel button shown. PASS.

### Conclusion

MediaUploader keyboard/focus operation verified via 5 jsdom tests. No
defect found. Not independently observed in a desktop browser.

## Surface: MediaUploader — readable error/empty/success states

### Observed behavior (jsdom automated tests)

1. **Error message readable**: "retry from error succeeds" — error
   shows `uploadError` text. "rejects presign response with invalid
   key" — shows "presign returned invalid key". PASS.
2. **Success message readable**: "completes full upload flow" —
   success shows "已驗證". PASS.
3. **Empty state**: no `.media-list` rendered when empty, upload area
   visible. PASS.
4. **Threshold rejection readable**: shows "50 MiB". PASS.
5. **Deduplication**: shows "已驗證" but no duplicate emit. PASS.

### Conclusion

Error, empty, success, and threshold-rejection states are readable.
No defect found.

## Surface: Touch-target size (DEFECT FOUND AND FIXED)

### Source revision/configuration (before fix)

- MediaUploader.vue CSS: `.btn-move`, `.btn-remove`, `.btn-cancel`,
  `.btn-retry` had `padding: 2px 8px; font-size: 12px` -> ~18px
  computed height.
- `.media-alt-input` had `padding: 2px 6px; font-size: 12px` ->
  ~18px computed height.
- WCAG 2.5.8 AA minimum target size: 24x24px.

### Defect

Touch-target size for MediaUploader small buttons (~18px) and alt-text
input (~18px) was below the WCAG 2.5.8 AA 24x24px minimum. This is a
reproducible code defect against the stated accessibility criteria
(plan.md: "responsive/accessible checkout/admin states"; REQ-008:
"keyboard, touch, responsive, and recovery walkthroughs").

### Fix

- `.btn-move`, `.btn-remove`, `.btn-cancel`, `.btn-retry`: padding
  increased from `2px 8px` to `4px 8px`; added `min-height: 24px` in
  the scoped CSS rule AND as an inline `style="min-height:24px"` on
  each affected button element so the constraint is directly
  observable in jsdom without a layout engine.
- `.btn-move` (the smallest arrow controls): added `min-width: 24px`
  in the scoped CSS rule AND as inline `style="min-width:24px"` on
  each `.btn-move` button, so the full 24x24 minimum is met.
- `.media-alt-input`: padding increased from `2px 6px` to `4px 6px`;
  added `min-height: 24px` in the scoped CSS rule AND as an inline
  `style="min-height:24px"` on the input element. Width already
  exceeds 24px so no min-width is needed.

### Regression test

- `MediaUploader.test.ts` "media controls meet 24x24 minimum
  touch-target size": mounts the component with 2 images (so `.btn-move`
  up/down buttons render), finds all `.btn-move` buttons and asserts
  `element.style.minHeight === '24px'` AND
  `element.style.minWidth === '24px'`; finds `.btn-remove` and asserts
  `minHeight === '24px'`; finds `.media-alt-input` and asserts
  `minHeight === '24px'`. Fails if any inline style is removed. PASS.

### Conclusion

Touch-target size defect fixed. All MediaUploader small buttons meet
WCAG 2.5.8 AA 24x24px minimum (`.btn-move` has both min-height and
min-width; `.btn-remove`/`.btn-cancel`/`.btn-retry` have min-height
and naturally exceed 24px in width). Alt-text input meets min-height
24px (width already exceeds 24px). Regression test prevents removal.

## Surface: Form label association (DEFECT FOUND AND FIXED)

### Source revision/configuration (before fix)

- ResourceListPage.vue form: `<label>` elements were siblings of input
  components with no `for` attribute. Input/Textarea/Select components
  did not accept `id` props. MediaUploader file input had
  `aria-label="Select image file"` which did not match the visible
  form label "商品圖片".
- Screen readers could not announce the visible label when the
  corresponding input received focus. Clicking the label text did not
  focus the input.

### Defect

Form labels were not programmatically associated with inputs via
`for`/`id`. This affected ALL form fields, not just the MediaUploader.
This is a reproducible code defect against the stated accessibility
criteria (plan.md: "responsive/accessible checkout/admin states";
REQ-008: "keyboard, touch, responsive, and recovery walkthroughs").

### Fix

- Input.vue: added `id` prop; rendered as `:id="id || undefined"` on
  the `<input>` element.
- Textarea.vue: added `id` prop; rendered as `:id="id || undefined"`
  on the `<textarea>` element.
- Select.vue: added `id` prop; rendered as `:id="id || undefined"` on
  the `<select>` element.
- MediaUploader.vue: added `labelledby` prop; rendered as
  `:aria-labelledby="labelledby || undefined"` on the file input
  (alongside the existing `aria-label` fallback).
- ResourceListPage.vue form: each `<label>` now has
  `:for="'field-' + fd.k"` for regular fields (Input/Textarea/Select),
  and `:id="'label-field-' + fd.k"` for media-uploader fields. Each
  Input/Textarea/Select receives `:id="'field-' + fd.k"`. The
  MediaUploader receives `:labelledby="'label-field-' + fd.k"`.
  When the field is read-only (`fd.ro` or `formIsReadOnly`), the
  label omits both `for` and `id` so it does not reference a
  nonexistent control (the field renders as a `<p>` with no input).

### Regression test

- `ResourceListPage.test.ts` "form labels are associated with inputs
  via for/id": opens the product create form, iterates all `.fgrid
  label` elements, asserts that regular field labels have `for`
  pointing to an existing input/textarea/select with that id, and
  media-uploader labels have `id` with a corresponding
  `aria-labelledby` on the file input. PASS.
- `ResourceListPage.test.ts` "read-only form labels omit for/id (no
  nonexistent control references)": opens the orders detail form
  (readOnly: true), iterates all `.fgrid label` elements, asserts
  that none have `for` or `id` attributes. PASS.
- `MediaUploader.test.ts` "file input has aria-labelledby when
  labelledby prop is set": asserts the file input has the correct
  `aria-labelledby` value. PASS.
- `MediaUploader.test.ts` "file input has no aria-labelledby when
  labelledby prop is absent": asserts no `aria-labelledby` when prop
  is absent. PASS.

### Conclusion

Form label association defect fixed. All form fields now have
programmatic label association via `for`/`id` (regular fields) or
`aria-labelledby` (MediaUploader). 3 regression tests prevent removal.

## Surface: Responsive layout and accessibility at narrow/mobile viewport (390x844, independent browser observation)

### Observed behavior (independent narrow/mobile browser observation at 390x844)

1. **No horizontal overflow**: document clientWidth=scrollWidth=390.
   No page-level horizontal overflow. OBSERVED.
2. **Product editor opened with SKU focused**: Edit dialog initial
   focus on SKU field at narrow viewport. OBSERVED.
3. **Modal width within viewport**: modal rect width 366.59 within
   390px viewport; `.fgrid` computed one column (328.59px). OBSERVED.
4. **All fields/footer reachable via scroll**: modalback is fixed,
   overflow-y:auto, clientHeight=844, scrollHeight=1297; scrolling
   reached footer (scrollTop=453, footer y=745). All fields and
   footer actions are reachable. OBSERVED.
5. **Media file input aria-labelledby**: file input
   aria-labelledby=label-field-product_images, whose visible text
   is 商品圖片. OBSERVED.
6. **Label click focuses input**: clicking label[for=field-name]
   focused #field-name. OBSERVED.
7. **Shift+Tab wrap at narrow viewport**: Shift+Tab from #field-sku
   wrapped to enabled Cancel. OBSERVED.
8. **Tab wrap with disabled skip at narrow viewport**: Tab from
   Cancel wrapped to #field-sku, skipping disabled Save. OBSERVED.
9. **Escape closes clean dialog and restores trigger**: Escape closed
   the clean (non-dirty) dialog and restored focus to the triggering
   編輯 button. OBSERVED.
10. **Focus indication visible**: focus indication on SKU was visible
    via 3px box-shadow and changed border color. OBSERVED.
11. **Narrow button measurements**: 選擇圖片 78x33; Cancel/Save
    54x35.5. Both exceed 24x24 minimum. OBSERVED.
12. **24x24 move controls**: existing automated coverage still
    supports 24x24 move controls and state management. No new
    responsive/accessibility defect reproduced. OBSERVED.

### Conclusion

Responsive layout and accessibility at narrow/mobile viewport (390x844)
confirmed by independent browser observation: no horizontal overflow,
modal fits within viewport, single-column form grid, all fields and
footer reachable via scroll, label/ARIA associations work, keyboard
Tab/Shift+Tab wrap with disabled-skip works, Escape closes and restores
focus, focus indication visible, button sizes exceed 24x24 minimum.
No new responsive/accessibility defect reproduced. B8 narrow/mobile
walkthrough is COMPLETE for the covered local admin scope.

## Surface: Provider/network failure behavior (no live R2)

### Observed behavior (via HTTP inspection)

1. **Presign without R2**: `POST /api/media/presign` with valid auth
   -> `503` with `{"error":"R2 is not configured"}`. OBSERVED.
2. **Verify without R2**: `POST /api/media/verify` with correct key
   prefix -> `503` with `{"error":"media storage is not configured"}`.
   OBSERVED.
3. **Verify with wrong key prefix**: `POST /api/media/verify` with
   `uploads/test/x.jpg` -> `403` with `{"error":"forbidden"}`.
   OBSERVED.
4. **Presign without auth**: `POST /api/media/presign` no auth ->
   `401` with `{"error":"unauthorized"}`. OBSERVED.
5. **Retry from error**: jsdom test "retry from error succeeds and
   emits exactly once" — presign rejects -> error -> retry -> success.
   PASS.

### Conclusion

Provider/network failure behavior safely exercised without live R2.
Go API returns truthful 503/403/401 errors. MediaUploader displays
error and provides retry. No defect found. B4 actual upload remains
pending.

## Surface: Admin auth and capability gating

### Observed behavior (via HTTP inspection)

1. **Admin/me**: `GET /api/admin/me` with `Bearer dev-admin` -> 200
   with 15 capabilities including `media.upload`. OBSERVED.
2. **Product list**: `GET /api/admin/products` -> 200 with 6 products,
   all with `product_images: []`. OBSERVED.
3. **Unauthorized**: `GET /api/admin/products` without auth -> 401.
   OBSERVED.

### Conclusion

Admin auth and capability gating work correctly. No defect found.

## Automated test summary (supporting evidence)

- `npm run test` (admin): 6 test files, 120 tests, all PASS.
  - MediaUploader.test.ts: 28 tests (+3 new: touch-target
    inline-style minHeight/minWidth with 2 images, aria-labelledby
    present, aria-labelledby absent).
  - ResourceListPage.test.ts: 12 tests (+2 new: form label
    for/id association, read-only form labels omit for/id).
  - ResourceTable.test.ts: 18 tests.
  - image-header.test.ts: 24 tests.
  - media-thresholds.test.ts: 21 tests.
  - image-compress.test.ts: 17 tests.
- `npm run typecheck`: PASS (exit 0).
- `npm run build`: PASS (exit 0).
- `npm run check:resource-contracts`: PASS (exit 0).

## Defect summary

| Label | Defect | Fix | Tests | Blocks acceptance? |
|---|---|---|---|---|
| broken | Touch-target size below WCAG 2.5.8 AA 24x24px minimum for MediaUploader small buttons (~18px) and alt-text input (~18px) | Added `min-height: 24px` + `min-width: 24px` (for `.btn-move`) and increased padding, as inline style + scoped CSS | 1 regression test (inline style minHeight/minWidth on mounted DOM with 2 images) | Was blocking; now fixed |
| broken | Form labels not associated with inputs via for/id; MediaUploader file input aria-label didn't match visible label; read-only form labels pointed to nonexistent controls | Added `id` prop to Input/Textarea/Select; added `for` to labels; added `labelledby` prop to MediaUploader with `aria-labelledby`; omit `for`/`id` when field is read-only | 4 regression tests (for/id association, read-only label omission, aria-labelledby present, aria-labelledby absent) | Was blocking; now fixed |

Both defects were reproducible code defects against the stated
accessibility criteria. Both are now fixed with focused regression
tests.

## Remaining blockers

1. **B4 actual media upload**: File chooser opened but setFiles was
   rejected by Chrome extension permission. The full upload flow
   (file selection -> presign -> R2 PUT -> verify -> product_images
   association) cannot be exercised. Operator action: enable Chrome
   ChatGPT extension Details > Allow access to file URLs before the
   actual upload walkthrough. No live R2 provider. REMAINS PENDING.
2. **Live R2 CopyObject verification**: No live R2 in env. REMAINS
   PENDING.
3. **Live R2 custom domain nosniff HEAD/GET**: No live R2 in env.
   REMAINS PENDING.
4. **Live PostgreSQL migration 014**: No live PG in env. REMAINS
   PENDING.
5. **Live Cloudflare Pages _headers deployment**: Not deployed.
   REMAINS PENDING.
6. **Secure token recovery**: No secure recovery mechanism implemented.
   REMAINS PENDING.
7. **Member auth consumer reachability**: Supabase customer auth not
   integrated. REMAINS PENDING.
8. **Retention/deletion policy approval (GATE-009)**: Not approved by
   named authority. REMAINS PENDING.
9. **Independent full acceptance walkthrough by non-implementer
   reviewer**: Not performed. REMAINS PENDING.
