# Independent contract-parity mutation review — revision 2

Change: `restore-http-contract-truth`
Revision: 2
Baseline: `9755c31048e3594d457748bf3d5dfb9f864a482f`

## Restored contract baseline

After the runtime-first OpenAPI restoration, PR CI run `33401094224` reached the frontend contract gate successfully. After both deliberate mutations were removed, clean CI run `33401872093` (job `99519881674`, head `eb3b97310c7eb32bba46febc2184d051a20743a0`) again produced:

```text
Runtime/OpenAPI parity check PASSED: 56 registered operations match the OpenAPI surface.
Resource contract check PASSED: admin form payloads match active Go contracts.
OpenAPI contract check PASSED: response schemas match Go/TS guarantees and promo enumeration is absent.
```

The clean run then stopped at the expected PR lifecycle gate because the controlled change was still `Applying`; the contract checks themselves were green before acceptance promotion.

## Mutation 1 — registered route omission

A one-line temporary mutation was injected into the parity checker's loaded OpenAPI text so the valid path `/api/payments/ecpay/browser-return` appeared as `/api/payments/ecpay/browser-return-mutated`. No Go runtime file or persisted OpenAPI operation was redesigned.

CI run `33401535363` (job `99518753250`) failed at `make verify-contracts` with the expected diagnostics:

```text
runtime operation missing from OpenAPI: POST /api/payments/ecpay/browser-return
OpenAPI operation is not registered by runtime: POST /api/payments/ecpay/browser-return-mutated
cannot inspect missing OpenAPI operation POST /api/payments/ecpay/browser-return
```

The mutation was removed before the next commit.

## Mutation 2 — observable success status drift

With the route mutation already removed, a second one-line temporary input mutation changed only the loaded admin-product create success status from `201` to `200`.

CI run `33401725071` (job `99519392203`) failed at the frontend contract gate with the exact diagnostic:

```text
POST /api/admin/products contract is missing "'201':"
```

The mutation was removed in commit `eb3b97310c7eb32bba46febc2184d051a20743a0`.

## Residue and scope review

A baseline-to-branch comparison after restoration contains only the intended final files:

- `Makefile`
- `contracts/check-runtime-openapi.mjs`
- `contracts/openapi.yaml`
- `specs/changes/restore-http-contract-truth/**`

Neither mutated route text nor the synthetic product-create `200` replacement remains in the final checker. The final checker reads `contracts/openapi.yaml` directly and the clean run proves the restored input passes 56/56 route parity plus both pre-existing contract checks.

## Review result

AC-004 is satisfied: both required drift classes were independently replayed through PR CI, each turned the contract gate red with a specific diagnostic, and restoration returned the contract gate to green with no mutation residue in the final diff.
