# Media GC security review receipt

Date: 2026-08-13  
Revision: 9  
Scope: migration 015, media verification reservations, commerce association updates, durable GC jobs, R2 lifecycle runbook, and `server/tools/media-gc`

## Threats reviewed

- TOCTOU between a product-image association and deletion of the same R2 object.
- TOCTOU between re-verification and a GC claim.
- Provider failure before deletion, after deletion, or before job acknowledgement.
- Stale verification after a server crash between CopyObject, temp deletion, and registry completion.
- Accidental bucket-wide lifecycle deletion of referenced `verified/` objects.
- Sensitive object keys leaking through cron output or error messages.
- Unbounded work, repeated provider failure, and starvation of new deletion jobs.

## Controls observed

- `media_assets` is canonical and `product_images(object_key, asset_state)` references only an `active` asset with `ON DELETE RESTRICT`.
- PostgreSQL claims lock candidates with `FOR UPDATE SKIP LOCKED`; SQLite uses the repository's single-connection serialization. The foreign key remains the final guard for either race ordering.
- GC creates `media_gc_jobs`, deletes source rows, and removes the canonical asset in one database transaction before issuing R2 DeleteObject.
- A failed R2 deletion increments attempts and retains the durable job. A delete that succeeded remotely but was not acknowledged is safe to retry because DeleteObject is idempotent.
- A pending job blocks `ReserveVerified`; successful re-verification of an active asset renews its unassociated timestamp before CopyObject.
- Product unlink/delete starts the seven-day grace only when no other product references the key. GC also rechecks `NOT EXISTS product_images` while claiming.
- Stale `verifying` reservations use a separate 24-hour threshold. Copy failures before verified bytes exist abort newly created reservations.
- The operational result contains counts only. GC claim errors no longer interpolate object keys. Batch size is limited to 1..1000.
- Unattempted jobs sort ahead of retries, preventing one persistent provider failure from starving newly claimed work.
- The runbook scopes the R2 lifecycle rule to `uploads/` only and explicitly forbids applying an age rule to `verified/`.

## Validation observed

- `go test -race -count=1 ./server/...`: pass.
- `go vet ./server/...`: pass.
- Migration tests cover SQLite upgrade/backfill, referenced-row protection, and static PostgreSQL parity beyond integer width.
- GC tests cover exact seven-day and 24-hour boundaries, referenced protection, both claim/association outcomes, renewal, dry-run, provider retry, durable acknowledgement, and retry fairness.

## Residual and external checks

- PostgreSQL migration 015 and its concurrent lock behavior are not live-verified in this environment.
- The user reported applying the Cloudflare `uploads/` one-day lifecycle rule. It could not be independently read back because the available Dashboard session was logged out and no management API token is configured. Existing R2 S3 credentials must not be used to replace an unknown lifecycle configuration without first reading and merging all rules.
- Schedule `media-gc --apply` in a restricted deployment job that owns the database and R2 credentials. Do not expose it as an HTTP endpoint.
- Direct out-of-band SQL writes can bypass grace-clock updates. They cannot bypass the foreign-key reference protection, but operators must use the application mutation path for the approved continuous-unassociation semantics.
