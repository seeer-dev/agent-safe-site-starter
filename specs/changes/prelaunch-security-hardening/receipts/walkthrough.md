# Walkthrough Receipt — AC-006 (guest order token retention)

Date: 2026-09-22
Environment: locally rendered `dist/` served statically at 127.0.0.1:8091 (no backend needed; the TrackPage island mounts client-side).

## Steps observed in a real browser session

1. Seeded legacy `localStorage.curatory_recent_orders` with two entries containing `token` fields (`LEGACY-SECRET-AAAA/BBBB`).
2. Reloaded `/track/`. The TrackPage island mounted and rendered both recent-order chips (`TW-LEGACY1`, `TW-LEGACY2`) showing order ID + date only.
3. Read `localStorage` again: the stored list was re-saved de-identified — no `token` keys, no `SECRET` substrings.
4. Clicked the `TW-LEGACY1` chip: `#track-order` was prefilled with the order ID, `#track-token` remained empty, no `/api/orders` request was issued (fetch hook observed zero calls), and focus moved to the query-code field.

## Result

- A later/new session shows recent-order metadata but requires the query code before any protected read — matches AC-006.
- Same-session continuity for a just-created order is preserved through `sessionStorage.curatory_last_order` (unchanged code path in `OrderPage`).
- Unit-level coverage: `shared/lib/store.test.ts` 5/5 (strip, metadata-only write, dedupe/cap, malformed input).
