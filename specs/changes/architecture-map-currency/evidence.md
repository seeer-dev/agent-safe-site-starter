# Evidence

## Delivery status

Revision 1 implemented and verified on 2026-08-18. Documentation-only change: no runtime behavior, route, schema, or dependency was altered.

## Observed evidence

| ID | Status | Proof |
|---|---|---|
| REQ-001 | passed | All six directories under server/internal/modules/ now appear in the modules section, and each owns list was derived from the tables that module actually writes rather than from the previous map. |
| REQ-002 | passed | tools.deterministic lists all 13 command directories under server/tools/ (excluding the shared internal helper package). A reverse check confirmed no entry names a directory that does not exist. |
| REQ-003 | passed | The map now states that module isolation is enforced at the import level only, and records the one known cross-module data write under known_cross_module_data_writes with its file:line and detected_by: none. |
| AC-001 | passed | Verified mechanically: every basename under server/internal/modules/ matches a modules key (commerce, contact, content, media, sitecontent, staff). staff_members now appears exactly once, owned by staff. commerce gained order_items, shipping_methods, and restock_idempotency; media gained media_assets and media_gc_jobs. |
| AC-002 | passed | Verified mechanically in both directions: every command directory under server/tools/ except internal is present, and every server/tools/ path named in the map resolves to an existing directory. local-postgres-gate, postgres-live-gate, migration-parity, media-gc, preview, and seed are all listed. |
| AC-003 | passed | known_cross_module_data_writes records commerce writing media_assets.unassociated_since_unix at server/internal/modules/commerce/store.go:434,440, owned by media, with detected_by: none. archcheck was inspected and walks Go imports only, so no current gate covers this. |

## Baseline observations

- `architecture.yaml` lists 4 modules; `server/internal/modules/` contains 6.
- `architecture.yaml` lists 8 deterministic tools; `server/tools/` contains 14
  command directories plus a shared `internal` helper package.
- `commerce.owns` includes `staff_members`, which only the `staff` module writes.
- `server/internal/modules/commerce/store.go:434,440` writes `media_assets`, a
  table owned by the `media` module. `archcheck` inspects Go imports only and
  does not detect this.
