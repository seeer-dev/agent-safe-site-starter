# Restock Idempotency Validation Receipt (Revision 9)

Date: 2026-08-13

## Scope

Grok 4.6 implemented the bounded commerce fix through Herdr. Codex reproduced
the defect, constrained the implementation to `service.go` and
`service_test.go`, inspected the resulting code and tests, and independently
reran acceptance checks.

## Defect and resolution

Two concurrent requests with the same idempotency key could both miss the
early lookup. After the winner atomically committed its idempotency row and
ledger update, the loser could read the new ledger and return `ErrOverRestock`
before reaching the transaction's unique-key arbitration.

The service now re-resolves the key before returning errors derived from
mutable order or ledger state. A committed different fingerprint returns
`ErrIdempotencyConflict`; a matching fingerprint returns the exact stored
snapshot; `ErrNotFound` preserves the original mutable-state error. Authorization
still runs first, the fingerprint still includes the order ID, no public API or
database schema changed, and the stored admin snapshot remains behind both
`orders.returns` and `inventory.adjust` capabilities.

A deterministic store wrapper forces the first lookup to miss after a winner
has already committed. The regression test covers different-payload conflict,
same-payload replay, and genuine unused-key over-restock behavior. The original
scheduler-driven concurrent test remains.

## Independent validation

- Pre-fix: focused test with `-count=100` failed 5 times with
  `ErrOverRestock` instead of `ErrIdempotencyConflict`.
- `go test ./server/internal/modules/commerce -run '^TestRestockOrderItemsConcurrentSameKeyDifferentPayload$' -count=500`: passed.
- `go test ./server/internal/modules/commerce -run '^TestRestockOrderItemsPostEarlyCheckRaceResolvesIdempotency$' -count=100`: passed.
- `go test -race ./server/internal/modules/commerce -run 'Restock.*Idempoten|ConcurrentSameKeyDifferentPayload|PostEarlyCheckRace' -count=10`: passed.
- `go test ./server/internal/modules/commerce -count=1`: passed.
- `go vet ./server/internal/modules/commerce`: passed.
- `go test ./server/... -count=1`: passed.
- `go vet ./server/...`: passed.
- `gofmt -d server/internal/modules/commerce/service.go server/internal/modules/commerce/service_test.go`: no output.

The broader controlled change remains Applying. Live PostgreSQL concurrency and
migration validation, the governed refund/provider path, and other previously
recorded acceptance blockers remain pending.
