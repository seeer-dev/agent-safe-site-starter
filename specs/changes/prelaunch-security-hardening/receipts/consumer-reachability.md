# Consumer-Reachability Receipt — AC-003

Date: 2026-09-22

The same `manual_test` availability predicate is consumed by every public surface:

- public payment-method listing (`payment_methods.go`) filters `manual_test` through `ManualTestCheckoutAllowed()`;
- `Quote` rejects `manual_test` when the window is closed;
- guest `CreateOrder` and member `CreateOrder` both run through the same payment-method validation;
- `manual_test_window_test.go` proves list/quote/order share the decision: disabled without a window, disabled after expiry, available inside the window, identical across all entry points.

No browser bundle contains the window value — the flag exists only in the Go process environment.
