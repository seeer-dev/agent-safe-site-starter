# Independent Walkthrough Receipt (Revision 7) — B4 Pre-R2 Media Upload

Date: 2026-08-13

## Scope and boundary

This receipt records an independent local browser walkthrough of the B4
admin product-media path before object storage. Source state was the existing
controlled change `minimal-cart-integration`, revision 8, at repository
baseline `7c45b616fbe3a632ffe2a39d872c98485466c991` plus its recorded dirty
working tree.

The Chrome file-URL permission blocker recorded in
`receipts/walkthrough-rev6.md` is superseded for this run: file selection now
works. Cloudflare R2 was not configured (`R2_ACCOUNT_ID`, access key, secret,
bucket, and public base URL were all absent), so this receipt does not claim
an R2 PUT, post-upload verification, verified object creation, or
`product_images` association.

## Surface

- Route: `http://127.0.0.1:4174/res/minimal-cart-products`
- Persona: local development admin with server-confirmed `media.upload`
- API: local Go service on `http://127.0.0.1:8080`
- Product editor: existing `SKU-APP-01` product, opened without saving
- Relevant evidence: REQ-003, REQ-008, REQ-012, AC-006, AC-015, AC-023,
  AC-024

## Observations

### File chooser and small-file direct path

1. The product editor opened and exposed a single-file image chooser.
2. `setFiles` accepted the non-personal bundled test PNG
   `google-chrome.png` (113,744 bytes). This confirms the Chrome extension
   file-URL permission is active.
3. The small PNG did not show the compression state and reached the presign
   step.
4. The Go API returned the configured fail-closed error `R2 is not
   configured`. The uploader displayed that message and moved focus to the
   `重試` button.
5. No image entry was added and the product `儲存` action remained disabled.

### Large-image Worker-compression path

1. `C:\Windows\Web\Wallpaper\Theme1\img13.jpg` was used as a non-personal
   system test asset. It is 1,277,444 bytes and 3840x1200 pixels.
2. The long edge exceeds the 2560px direct-upload threshold, so the expected
   decision is Worker compression even though the file is below 2 MiB.
3. Visible state polling after selection observed `壓縮中…` at approximately
   46, 97, 146, and 206 ms, then `上傳中…` at approximately 263 ms, followed
   by `R2 is not configured` at approximately 312 ms.
4. Reaching `上傳中…` and the presign failure proves the Worker returned an
   output that passed the client's compressed-output validation. The polling
   cadence stayed responsive during this specific 3840x1200 run; it does not
   constitute a general device-performance benchmark.
5. The failure remained recoverable: the message was visible and focus moved
   to `重試`. No product mutation occurred.

## Security and data handling

- Only bundled/system test images were selected; no personal user file,
  credential, token, or PII was transmitted.
- The browser requested a Go-generated presign before any direct object-store
  upload. With R2 disabled, the flow stopped at the authenticated Go endpoint.
- No R2 URL was produced, so no external PUT was attempted.
- No unverified key was written to the product form or database.

## Mechanical replay

- `npm test -- MediaUploader.test.ts image-compress.test.ts
  media-thresholds.test.ts image-header.test.ts`: 4 files, 90 tests passed.
- `npm run typecheck`: passed.
- `npm run build:only`: passed; the production build includes the dedicated
  `image-compress.worker` asset.
- `npm run check:resource-contracts`: passed.
- `go test ./server/internal/modules/media/... -count=1`: passed.
- `go run ./server/tools/speccheck`: passed after evidence synchronization.

## Result

B4 pre-R2 browser behavior is **COMPLETE for the covered local scope**:
file selection, small-file direct routing, large-dimension Worker compression,
truthful storage-unavailable feedback, retry focus, and no-association failure
behavior were observed.

B4 end-to-end media upload remains **PENDING LIVE**. A configured test R2
environment is required to observe presign success, browser-to-R2 PUT,
server-side byte verification and verified-key promotion, `product_images`
association, save/refetch persistence, and public asset delivery.
