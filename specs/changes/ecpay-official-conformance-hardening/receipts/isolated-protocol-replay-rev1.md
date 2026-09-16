# Isolated ECPay protocol replay — revision 1

Change ID: `ecpay-official-conformance-hardening`
Revision: `1`
Observed date: `2026-09-01`

This receipt is **not** full repository CI. It records an independent local Go replay of the changed ECPay protocol/payment logic while GitHub Actions had not produced a run for the API-created PR branch.

## CheckMacValue vectors

The branch CheckMacValue algorithm was replayed against the expected SHA256 digests copied from official `ECPay/ECPay-API-Skill@test-vectors/checkmacvalue.json` at pinned commit `ae964f75b69ec90e1c205b136364ab6587fc328c`.

Observed results:

```text
baseline true 291CBA324D31FB5A4BBBFDF2CFE5D32598524753AFD4959C3BF590C5B2F57FB2
apostrophe true CF0A3D4901D99459D8641516EC57210700E8A5C9AB26B1D021301E9CB93EF78D
tilde true CEEAE01D2F9A8E74D4AC0DCE7735B046D73F35A5EC99558A31A2EE03159DA1C9
spaces true 7712A5E6EDC3B57086063C88568084C66CE882A21D40E74DE5ACA3B478C6F316
callback true 2AB536D86AFF8E1086744D59175040A32538C96B1C28C4135B551BD728E913B8
```

The apostrophe vector specifically exercises the Go-specific `' -> %27` correction added by this change.

## Minimal compile/flow harness

A minimal Go module used the changed `ecpay.go` / `ecpay_payment.go` function bodies with only repository Store/Order/PaymentMethod types replaced by narrow stubs. This isolates protocol and callback-authority behavior without inventing an alternative implementation.

Environment:

```text
go version go1.23.2 linux/amd64
GOTOOLCHAIN=local
```

Observed tests:

```text
=== RUN   TestSimulationDoesNotConsumeClaimAndRealCanCapture
--- PASS: TestSimulationDoesNotConsumeClaimAndRealCanCapture
=== RUN   TestOfficialApostropheVector
--- PASS: TestOfficialApostropheVector
=== RUN   TestNon443Rejected
--- PASS: TestNon443Rejected
=== RUN   TestExplicit443AndIPRejected
--- PASS: TestExplicit443AndIPRejected
PASS
```

The simulation→real test verifies the important one-effect boundary: an authenticated `SimulatePaid=1` notification returns `1|OK` with zero callback claims, then a later non-simulated `RtnCode=1` notification for the same `MerchantTradeNo` can acquire exactly one claim and capture.

## Verification boundary

These results support the protocol-specific review and falsification requirements, but they do not prove:

- the repository compiles as a whole;
- all repository unit/integration tests pass;
- `make verify-contracts` passes on the full branch;
- `speccheck` accepts the full PR diff;
- live PostgreSQL/stress/vet gates pass.

Those remain required before the controlled change can be marked `Accepted` or merged.
