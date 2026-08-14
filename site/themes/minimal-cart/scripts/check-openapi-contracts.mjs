/**
 * Source assertion: OpenAPI response schemas match Go/TS guarantees.
 *
 * This script checks that the OpenAPI required fields for response schemas
 * match the fields the Go server actually guarantees in successful
 * responses. It prevents contract drift where the OpenAPI spec omits
 * required fields that the server always emits (or vice versa).
 *
 * Run: node scripts/check-openapi-contracts.mjs
 */
import { readFileSync } from 'fs'
import { resolve, dirname } from 'path'
import { fileURLToPath } from 'url'

const __dirname = dirname(fileURLToPath(import.meta.url))
const root = resolve(__dirname, '..', '..', '..', '..')
const errors = []

const openapiPath = resolve(root, 'contracts/openapi.yaml')
const openapi = readFileSync(openapiPath, 'utf-8')
const bootstrap = readFileSync(resolve(root, 'server/internal/bootstrap/app.go'), 'utf-8')
const commerceHTTP = readFileSync(resolve(root, 'server/internal/modules/commerce/http.go'), 'utf-8')
const checkout = readFileSync(resolve(root, 'site/themes/minimal-cart/islands/CheckoutDialog/CheckoutDialog.vue'), 'utf-8')

// Helper: extract the required list for a schema by name.
function getRequired(schemaName) {
  // Match the schema block and its required line.
  const re = new RegExp(
    `    ${schemaName}:\\n      type: object\\n      required: \\[([^\\]]+)\\]`,
  )
  const m = openapi.match(re)
  if (!m) return null
  return m[1].split(',').map((s) => s.trim()).filter(Boolean)
}

// 1. QuoteResult must require items, subtotal, discount, shipping, total,
//    shipping_method, payment_method — the Go server always emits these
//    in a successful response (Quote validates shipping_method and
//    payment_method before returning).
const quoteResultRequired = getRequired('QuoteResult')
if (!quoteResultRequired) {
  errors.push('QuoteResult schema not found in openapi.yaml')
} else {
  const expected = ['items', 'subtotal', 'discount', 'shipping', 'total', 'shipping_method', 'payment_method']
  for (const field of expected) {
    if (!quoteResultRequired.includes(field)) {
      errors.push(`QuoteResult.required is missing "${field}" — Go server always emits this in successful responses`)
    }
  }
}

// 2. Order must require all fields the Go Order struct always emits
//    (no omitempty): id, status, subtotal, discount, shipping, total,
//    items, customer_name, email, phone, shipping_address,
//    shipping_method, payment_method, updated_unix.
//    idempotency_key has omitempty so it is NOT required.
const orderRequired = getRequired('Order')
if (!orderRequired) {
  errors.push('Order schema not found in openapi.yaml')
} else {
  const expectedOrderFields = [
    'id', 'status', 'subtotal', 'discount', 'shipping', 'total',
    'items', 'customer_name', 'email', 'phone', 'shipping_address',
    'shipping_method', 'payment_method', 'updated_unix',
  ]
  for (const field of expectedOrderFields) {
    if (!orderRequired.includes(field)) {
      errors.push(`Order.required is missing "${field}" — Go server always emits this (no omitempty)`)
    }
  }
  // idempotency_key has omitempty — it should NOT be in required.
  if (orderRequired.includes('idempotency_key')) {
    errors.push('Order.required includes "idempotency_key" but Go struct has omitempty — it is not always emitted')
  }
}

// 3. QuoteInput must require payment_method — the server validates it
//    and rejects empty with ErrInvalidPaymentMethod.
const quoteInputRequired = getRequired('QuoteInput')
if (!quoteInputRequired) {
  errors.push('QuoteInput schema not found in openapi.yaml')
} else {
  if (!quoteInputRequired.includes('payment_method')) {
    errors.push('QuoteInput.required is missing "payment_method" — server rejects empty with ErrInvalidPaymentMethod')
  }
  if (!quoteInputRequired.includes('shipping_method')) {
    errors.push('QuoteInput.required is missing "shipping_method" — server rejects empty with ErrInvalidShippingMethod')
  }
}

// 4. OrderInput must require payment_method — same validation as QuoteInput.
const orderInputRequired = getRequired('OrderInput')
if (!orderInputRequired) {
  errors.push('OrderInput schema not found in openapi.yaml')
} else {
  if (!orderInputRequired.includes('payment_method')) {
    errors.push('OrderInput.required is missing "payment_method" — server rejects empty with ErrInvalidPaymentMethod')
  }
  if (!orderInputRequired.includes('shipping_method')) {
    errors.push('OrderInput.required is missing "shipping_method" — server rejects empty with ErrInvalidShippingMethod')
  }
}

// 5. Promo codes are validated only when submitted to quote/order. Public
// enumeration would disclose every active code and let the browser become a
// competing eligibility authority.
if (openapi.includes('/api/promos/active:')) {
  errors.push('openapi.yaml exposes /api/promos/active — promo codes must not be publicly enumerated')
}
if (bootstrap.includes('GET /api/promos/active')) {
  errors.push('bootstrap registers GET /api/promos/active — promo codes must be validated only by quote/order')
}

// 6. Guest order creation has one canonical success envelope and uses the
// repository-wide 400 validation convention from INTEGRATION_PLAN.md.
const guestOrderPath = openapi.match(/  \/api\/orders:\n([\s\S]*?)(?=\n  \/api\/orders\/\{id\}:)/)?.[1] || ''
if (!guestOrderPath.includes('required: [order]') || !guestOrderPath.includes("order: {$ref: '#/components/schemas/Order'}")) {
  errors.push('POST /api/orders must document the { order: Order } success envelope')
}
if (!guestOrderPath.includes("'400':") || guestOrderPath.includes("'422':")) {
  errors.push('POST /api/orders validation errors must use 400, not 422')
}
if (!commerceHTTP.includes('http.StatusCreated, map[string]any{"order": order}')) {
  errors.push('CreateOrder handler must return the documented { order } success envelope')
}
if (checkout.includes('apiOrder.order ?? apiOrder')) {
  errors.push('CheckoutDialog accepts both raw and wrapped orders — consumer must require the canonical wrapper')
}

// 7. Product images are arrays at every API/consumer boundary. The database
// may persist JSON text internally, but transport must not expose that detail.
// Revision 8: ProductInput no longer has image/images fields — it uses
// product_images (array of ProductImageInput with verified object keys).
// The public Product schema still has images as string[] (derived URLs).
const productSchema = openapi.match(/    Product:\n([\s\S]*?)(?=\n    ProductInput:)/)?.[1] || ''
const productInputSchema = openapi.match(/    ProductInput:\n([\s\S]*?)(?=\n    OrderItem:)/)?.[1] || ''
if (!/images:\n\s+type: array\n\s+items: \{type: string\}/.test(productSchema)) {
  errors.push('Product.images must be documented as string[]')
}
// ProductInput must have product_images (not legacy image/images).
if (!/product_images:/.test(productInputSchema)) {
  errors.push('ProductInput.product_images must be documented')
}
if (/^\s+image:/m.test(productInputSchema)) {
  errors.push('ProductInput must NOT have legacy image field')
}
if (/^\s+images:/m.test(productInputSchema)) {
  errors.push('ProductInput must NOT have legacy images field')
}

if (errors.length > 0) {
  console.error('OpenAPI contract check FAILED:')
  for (const e of errors) {
    console.error(`  - ${e}`)
  }
  process.exit(1)
}

console.log('OpenAPI contract check PASSED: response schemas match Go/TS guarantees and promo enumeration is absent.')
