# AC-009 walkthrough — Taiwan address + CVS flows

## Home delivery path

- Checkout step 2 renders the city/district cascade from
  `shared/data/taiwan-regions.ts` (ported static dataset); district
  options are constrained by the chosen city and ZIP fills from the
  district selection.
- Same-as-recipient toggle copies name/phone and disables the
  recipient fields.
- Smoke order `acc-test-001` persisted `city`/`district`/address on the
  order record (verified in POST /api/orders response).

## CVS path

- Selecting a CVS shipping method (cvs_711 / cvs_family) swaps the
  address block for a store picker dialog fed by
  `shared/data/cvs-stores.ts`; submission is blocked until a store is
  chosen (per-step validation).
- Order persists `cvs_store_name`/`cvs_store_address`; admin order
  detail shows the 超商門市 field.

## Invoice + note

- Invoice type radio (personal/company) reveals 統編 field only for
  company; buyer note textarea persists to `buyer_note`.
