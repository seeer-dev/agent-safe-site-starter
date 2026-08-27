# Plan

1. Add explicit all-or-none ECPay runtime configuration.
2. Add SQLite/PostgreSQL parity migration for durable ECPay payment attempts.
3. Add concrete commerce-owned ECPay signer, launch, callback verification, replay handling, and atomic paid transition.
4. Add HTTP launch/ReturnURL/browser-return routes.
5. Wire minimal-cart to launch ECPay only after CreateOrder succeeds.
6. Run format, migration parity, commerce tests, config/bootstrap tests, and storefront build/contracts.
