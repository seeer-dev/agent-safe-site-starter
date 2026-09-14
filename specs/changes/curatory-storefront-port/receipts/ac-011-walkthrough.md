# AC-011 walkthrough — confirmation, tracking, notifications

## Observed live (fresh seed DB, dev server)

- `POST /api/orders` (variant SKU CUR-TEA-01-A ×1, home_delivery, cod)
  → order `TW-246be3c8a20e2437bf2da9deef0400b4`, items resolved to
  手作粗陶馬克杯/米白 @880, shipping 80, payment fee 15, total 975,
  status pending, timeline seeded.
- `GET /api/admin/notification-logs` immediately after → row with
  `code=order_placed`, `status=sent`, provider `mail`, subject/body
  rendered from the seeded template ({{order_id}}, {{customer_name}},
  {{total}} substituted). The log-mailer path records each attempt;
  missing/disabled templates record `skipped` rows (observed earlier:
  "skipped — template not found" before templates were seeded).
- `/order/?id=…` renders the confirmation with the stepper at pending,
  items/totals/shipping/payment/invoice cards, timeline; `/track/`
  looks the same order up via id + access token.
- Staff transitions (pending→processing→shipped→delivered→completed /
  cancel) are wired to the remaining template codes
  (order_paid/order_shipped/order_completed/order_cancelled) through
  the same notifyOrderEvent path; each emits a log row.
- Failure states: wrong/absent token → honest error card, not fixture
  data.
