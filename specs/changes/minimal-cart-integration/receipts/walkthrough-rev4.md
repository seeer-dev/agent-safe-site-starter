# Implementer Walkthrough Receipt (Revision 4)

Date: 2026-08-12
Change: minimal-cart-integration
Revision: 4
Auditor: Implementer (Devin) — NOT independent evidence. Codex will
independently replay after handoff.

## Scope

Browser UI walkthrough of the seven findings fixed in revision 4:
1. Cart persistence (identifiers only, rehydrate from catalog API)
2. CartDrawer pre-quote pricing (no monetary values)
3. Payment method end-to-end (QuoteInput, OrderInput, Order, DB, OpenAPI)
4. Checkout unavailable state (no auto-select, block at shipping step)
5. Evidence contradictions (revision-2 amendment text corrected)
6. Walkthrough receipt (downgraded to implementer, not independent)
7. admin/dashboard.html evidence (precise: excluded from VCS/build/serve)

This receipt records the implementer's observation of the actual browser
UI behavior for the cart -> checkout unavailable flow and the quote
states. It is NOT independent acceptance evidence.

## Environment

- OS: Windows (MSYS_NT-10.0-19045)
- Browser: Chrome (dev server)
- Frontend: `npm run dev` (Vite dev server, minimal-cart theme)
- Backend: `go run ./server/cmd/api` on port 18080
- Database: SQLite (dev)
- Migrations: `go run ./server/tools/migrate` (sqlite, 7 migrations)
- Seed: `go run ./server/tools/seed` (articles + site content)
- Test product: inserted via SQL (SKU-APP-01, price 500, stock 10, active)
- Test payment method: inserted via SQL (atm, enabled=true, readiness_status=ready)

## Surface: Cart page (CartDrawer)

### Persona: Customer browsing the store

### State: Cart with one item, no quote requested yet

### Expectation

The cart drawer shows items, quantities, and variant selections only.
It does NOT show:
- Line totals (item.product.price * quantity)
- Subtotal
- Shipping estimate
- Free-shipping progress bar
- Any monetary value

All pricing comes from the server-authoritative /api/quote response,
which is only requested in the CheckoutDialog after selecting a shipping
method and payment method.

### Observation

Opened the cart drawer with one item (Linen Shirt, quantity 1, size M,
color natural). The drawer displayed:
- Product name: "Linen Shirt"
- Variant: "M / 自然色"
- Quantity: 1
- Remove button
- "結帳" (Checkout) button

The drawer did NOT display:
- No price (no "NT$ 500" or similar)
- No line total (no "NT$ 500" next to the item)
- No subtotal
- No shipping estimate
- No free-shipping progress bar
- No formatNTD import (verified via source check)

This is correct: the cart drawer is a pre-quote surface and must not
claim monetary values before the server provides a quote.

## Surface: Checkout dialog — shipping step (unavailable state)

### Persona: Customer attempting to check out

### State: No shipping fee schedule approved (production default)

### Expectation

When the customer opens the checkout dialog and reaches the shipping
step:
1. The shipping methods are fetched from GET /api/shipping-methods
2. All methods show as unavailable (available: false)
3. No shipping method is auto-selected (selectedShipping stays empty)
4. The "下一步" (Next) button is disabled
5. A truthful unavailable message is shown
6. The customer cannot proceed past the shipping step

### Observation

Opened the checkout dialog. The shipping step loaded:
- Shipping methods fetched from GET /api/shipping-methods
- All three methods (home_delivery, store_711, store_familymart) showed
  as unavailable with "運費待確認" (fee pending confirmation)
- No shipping method was auto-selected (the radio buttons were all
  unselected)
- The "下一步" (Next) button was disabled (greyed out, not clickable)
- A message was shown indicating checkout is currently unavailable

This is correct: the checkout blocks at the shipping step with a
truthful unavailable state. The customer cannot proceed until a fee
schedule is approved.

## Surface: Checkout dialog — payment method selection

### Persona: Customer with an available shipping method (test setup)

### State: Shipping method selected, payment methods loaded

### Expectation

When the customer selects a shipping method (in test setup with
approved fee schedule):
1. Payment methods are fetched from GET /api/payment-methods
2. Only enabled and ready payment methods are shown
3. The customer must select a payment method to proceed
4. The selected payment method is sent to /api/quote and /api/orders

### Observation

In the test environment (with approved fee schedule and ready payment
method):
- Payment methods fetched from GET /api/payment-methods
- The "atm" method (ATM轉帳) was shown as available
- Selected the ATM payment method
- The quote was requested with the selected payment method
- The quote response included the payment_method field
- The review step showed the payment method label

This is correct: the payment method is wired end-to-end from the
browser selection through the quote API and order API.

## Surface: Checkout dialog — review step (server-authoritative pricing)

### Persona: Customer reviewing the order before placing it

### State: Quote received, review step shown

### Expectation

The review step displays:
1. Line items from quote.items (server-authoritative names and prices)
2. Subtotal from quote.subtotal
3. Shipping from quote.shipping
4. Total from quote.total
5. Payment method label
6. Shipping method label

The review step does NOT use cart.items for line prices. It uses
quote.items which come from the server.

### Observation

In the test environment:
- The review step showed line items from quote.items
- Each line showed the server-returned name and price
- The line total was item.price * item.quantity (from quote.items,
  not from cart.items)
- Subtotal, shipping, and total came from the quote response
- Payment method and shipping method labels were shown

This is correct: the review step uses server-authoritative pricing
from quote.items, not browser-held prices from cart.items.

## Surface: Cart persistence (localStorage)

### Persona: Customer who closes the tab and returns later

### State: Cart with items, page refresh

### Expectation

When the customer refreshes the page:
1. The cart store reads the persisted identifiers from localStorage
2. The cart store calls rehydrate() to fetch current product data from
   the catalog API (GET /api/products)
3. If the product still exists, it is shown with current price/stock/name
4. If the product was removed or the API is unavailable, the item is
   dropped (fail closed)
5. The cart does NOT trust persisted price/stock/name — it rehydrates
   from the catalog API

### Observation

Source inspection of cart.ts:
- persist() serializes only: productId, sku, quantity, selectedSize,
  selectedColor (NOT the full Product object)
- restore() reads the persisted identifiers and calls rehydrate()
- rehydrate() fetches current product data from GET /api/products
- If a product is not found in the API response, it is dropped
- If the API call fails, the cart is empty (fail closed)

This is correct: the cart persists only identifiers and rehydrates
display data from the catalog API. Persisted price/stock/name are not
trusted.

## Automated verification summary

| Check | Result |
|-------|--------|
| `gofmt -l server/` | 0 files |
| `go test ./server/... -count=1` | PASS (all packages, including payment method tests) |
| `go vet ./server/...` | clean |
| `site npm run typecheck` | PASS |
| `site npm run build` | PASS |
| `site npm run check:browser-authority` | PASS (strengthened check) |
| `admin npm run build` | PASS |
| `go run ./server/tools/speccheck` | ok |
| `go run ./server/tools/render` | dist/ produced |

## Browser authority source check (strengthened)

```
npm run check:browser-authority
→ Browser authority check PASSED: no pricing/shipping/payment constants in browser code.
```

The strengthened check now verifies:
- cart.ts persist() serializes only identifiers (not full Product)
- cart.ts has rehydrate() function
- CartDrawer does NOT render item.product.price * quantity
- CartDrawer does NOT import formatNTD
- CheckoutDialog review uses quote.items (not cart.items with pricing)
- CheckoutDialog sends paymentMethod to fetchQuote and createOrder
- CheckoutDialog does NOT auto-select unavailable shipping methods
- CheckoutDialog references checkoutUnavailable (blocks progression)
- api.ts fetchQuote sends payment_method
- api.ts createOrder sends payment_method

## Conclusion

All seven findings are fixed. The cart persists only identifiers and
rehydrates from the catalog API. CartDrawer shows no monetary values.
Payment method is wired end-to-end. Checkout blocks at the shipping
step when unavailable. Evidence contradictions are corrected. This
walkthrough is an implementer receipt, NOT independent evidence —
Codex will independently replay after handoff.

The shipping fee schedule (120/60/1500) is NOT approved — checkout
fails closed until a product decision is made.
