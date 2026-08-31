# Restore HTTP Contract Truth Evidence

Change ID: restore-http-contract-truth
Revision: 2
Status: Accepted

| ID | Status | Proof |
| --- | --- | --- |
| REQ-001 | passed | contracts/openapi.yaml now represents all 56 operations registered in server/internal/bootstrap/app.go; clean PR CI run 33401872093 reports Runtime/OpenAPI parity check PASSED: 56 registered operations match the OpenAPI surface. |
| REQ-002 | passed | OpenAPI 0.3.0 now follows current runtime success behavior, including admin product 201 create, 200 delete with ID body, admin DTOs, media presign purpose, staff envelopes/statuses, and ECPay HTTP boundaries; both existing contract checks pass in clean run 33401872093. |
| REQ-003 | passed | contracts/check-runtime-openapi.mjs uses only Node standard-library modules, performs symmetric registered-operation parity plus guarded observable-contract checks, and is wired into make verify-contracts before the existing admin and public-theme contract checks; clean run 33401872093 passes all three. |
| AC-001 | passed | The restored contract has 56 of 56 registered operations and clean run 33401872093 passes route/method parity; route-omission mutation run 33401535363 failed with exact missing-runtime and extra-OpenAPI diagnostics before restoration. |
| AC-002 | passed | Admin product OpenAPI now uses AdminProductResponse, declares 201 for create and 200 with an ID response body for delete, and includes status/bulk operations; status mutation run 33401725071 failed specifically because the guarded 201 disappeared. |
| AC-003 | passed | OpenAPI now represents ECPay prepare, server ReturnURL, and browser-return POST operations with the order access header, signed launch response, form callback boundary, plain-text successful callback acknowledgement, and 303 browser redirect semantics. |
| AC-004 | passed | Independent CI mutation replay is recorded in receipts/independent-review-rev2.md: route mutation run 33401535363 and product-create status mutation run 33401725071 both failed with specific parity diagnostics; clean restoration run 33401872093 returned all contract checks to green and the baseline-to-branch diff contains no mutation residue. |
