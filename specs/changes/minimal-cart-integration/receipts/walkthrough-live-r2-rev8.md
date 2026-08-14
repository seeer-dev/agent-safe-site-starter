# Independent Walkthrough Receipt (Revision 8) - B4 Live R2 Upload

Date: 2026-08-13

## Source revision and configuration

- Controlled change: `minimal-cart-integration`, revision 8 (`Applying`).
- Repository baseline: `7c45b616fbe3a632ffe2a39d872c98485466c991` plus the recorded dirty working tree.
- Admin route: `http://127.0.0.1:4174/res/minimal-cart-products`.
- Go API: `http://127.0.0.1:8080`.
- Persona: local development admin with server-confirmed `media.upload`.
- R2 credentials, bucket, and public base URL were present without exposing their values.
- The R2 CORS policy allowed exact origin `http://127.0.0.1:4174`, method `PUT`, and request header `Content-Type`.
- Relevant evidence: REQ-003, REQ-008, REQ-012, AC-006, AC-023, AC-024.

## Action and observed behavior

1. Opened existing draft product `SKU-APP-05` and selected the non-personal system image `C:\Windows\Web\Wallpaper\Theme1\img13.jpg`.
2. Source image: JPEG, 1,277,444 bytes, 3840x1200 pixels.
3. The long edge exceeded the 2560px direct threshold, so the uploader used the frontend Worker-compression path.
4. The browser completed presign, direct R2 PUT, and Go post-upload verification. The editor received a stable `verified/product-images/...jpg` key and exposed the image remove control.
5. No `Failed to fetch`, CORS, upload, verification, warning, or console error remained.
6. A public GET of the verified object returned HTTP 200, `image/jpeg`, 196,145 bytes, and 2560x800 pixels.
7. The verified payload was 84.6% smaller than the source while preserving the aspect ratio and the configured 2560px long-edge limit.
8. Saved the verified key to draft product `SKU-APP-05`. The editor closed and the product update timestamp changed, proving the permitted mutation returned successfully.
9. Reloaded the admin route from the Go API and reopened the product. The verified image and remove control were present while Save remained disabled, proving authoritative save/refetch persistence rather than browser-local state.
10. Removed the image, saved again, reloaded the route, and reopened the product. The image list was empty, no remove control remained, and Save was disabled, proving authoritative removal persistence.

## Mutation boundary and cleanup

- `SKU-APP-05` was temporarily associated with the verified JPEG and then restored to an empty image list. The final `product_images` state matches the initial state; the product's `updated_unix` reflects the final cleanup save at 2026-08-13 11:22:44 Asia/Taipei.
- The Go verify path promoted the temporary upload to a stable verified key and deleted the temporary upload key.
- This walkthrough created one verified JPEG object plus its media registry row, currently unassociated with a product. An earlier direct diagnostic also created one unassociated verified PNG. The application has no media-deletion workflow, so these were not deleted directly to avoid R2/database inconsistency.
- The agent-created browser tabs were closed after observation.

## Mechanical replay

- `go test ./server/internal/modules/media/... -count=1`: passed.
- `go test ./server/internal/modules/commerce/... -run 'Test.*(Image|Media)' -count=1`: passed.
- `npm test -- --run src/components/MediaUploader.test.ts`: 1 file, 28 tests passed.
- `go run ./server/tools/speccheck`: passed before this evidence-only synchronization and replayed afterward.

## Result

B4 live R2 upload is **COMPLETE for presign, browser PUT, frontend large-image compression, post-upload byte verification, verified-key promotion, product association save/refetch, association removal/refetch, and public object delivery**.

B4 remains **PENDING** for verified-media deletion/retention handling, live PostgreSQL migration 013, R2 response-header/CDN behavior, and full release acceptance.
