/**
 * Source assertion: CheckoutDialog uses a stable idempotency key across
 * transport retries and fails closed on malformed 2xx responses.
 *
 * This script statically checks the CheckoutDialog source for:
 * 1. No Date.now()/Math.random() in idempotency key generation.
 * 2. crypto.randomUUID() is used for key generation.
 * 3. The key is stored in a ref (stable across retries).
 * 4. The key is NOT regenerated on every placeOrder call (only if null).
 * 5. No fabricated order ID fallback (e.g. `TW-${Date.now()}`).
 * 6. Fail-closed validation on missing server fields (order id, totals).
 * 7. The key is reset after confirmed success (in placeOrder, not open watcher).
 * 8. Fail closed if crypto.randomUUID is unavailable.
 * 9. The key is NOT reset inside the ui.checkoutOpen watcher — reopening
 *    an unchanged checkout must preserve the submission intent so a
 *    committed-but-lost response can be retried with the same key.
 * 10. checkoutFingerprint computed and its watch are declared AFTER
 *     shippingForm, selectedShipping, and selectedPayment to avoid a
 *     Temporal Dead Zone violation when Vue evaluates the computed
 *     getter to collect dependencies.
 *
 * Run: node scripts/check-checkout-idempotency.mjs
 */
import { readFileSync } from 'fs'
import { resolve, dirname } from 'path'
import { fileURLToPath } from 'url'

const __dirname = dirname(fileURLToPath(import.meta.url))
const errors = []

const dialogPath = resolve(__dirname, '..', 'islands', 'CheckoutDialog', 'CheckoutDialog.vue')
const src = readFileSync(dialogPath, 'utf-8')

// 1. No Date.now() or Math.random() in idempotency key generation.
// The old pattern was: `${Date.now()}-${Math.random().toString(36)...}`
const oldKeyPattern = /Date\.now\(\).*Math\.random/
if (oldKeyPattern.test(src)) {
  errors.push('CheckoutDialog still uses Date.now()/Math.random for idempotency key — use crypto.randomUUID()')
}

// 2. crypto.randomUUID() is used for key generation.
if (!/crypto\.randomUUID\(\)/.test(src)) {
  errors.push('CheckoutDialog does not use crypto.randomUUID() for idempotency key generation')
}

// 3. The key is stored in a ref (stable across retries).
if (!/idempotencyKey\s*=\s*ref</.test(src)) {
  errors.push('CheckoutDialog does not store idempotency key in a ref — it must be stable across retries')
}

// 4. The key is NOT regenerated on every placeOrder call (only if null).
// Look for the pattern: if (!idempotencyKey.value) { ... generate ... }
if (!/if\s*\(!idempotencyKey\.value\)/.test(src)) {
  errors.push('CheckoutDialog does not guard key generation with "if (!idempotencyKey.value)" — key must only be generated once per submission intent')
}

// 5. No fabricated order ID fallback.
const fabricatedIdPattern = /o\.id\s*\|\|\s*`TW-|o\.id\s*\|\|\s*['"]TW-/
if (fabricatedIdPattern.test(src)) {
  errors.push('CheckoutDialog fabricates order ID on missing server field — fail closed instead')
}

// 6. Fail-closed validation on ALL required server-authoritative fields.
//    The Go server always emits id, status, subtotal, discount,
//    shipping, total, shipping_method, payment_method. The frontend
//    must validate each and throw on missing — no ?? or || fallbacks.
const requiredFieldChecks = [
  { field: 'id', pattern: /throw new Error.*缺少訂單編號/, label: 'order ID' },
  { field: 'status', pattern: /throw new Error.*缺少訂單狀態/, label: 'order status' },
  { field: 'subtotal', pattern: /throw new Error.*缺少小計金額/, label: 'subtotal' },
  { field: 'discount', pattern: /throw new Error.*缺少折扣金額/, label: 'discount' },
  { field: 'shipping', pattern: /throw new Error.*缺少運費金額/, label: 'shipping cost' },
  { field: 'total', pattern: /throw new Error.*缺少總金額/, label: 'total' },
  { field: 'shipping_method', pattern: /throw new Error.*缺少配送方式/, label: 'shipping method' },
  { field: 'payment_method', pattern: /throw new Error.*缺少付款方式/, label: 'payment method' },
]
for (const check of requiredFieldChecks) {
  if (!check.pattern.test(src)) {
    errors.push(`CheckoutDialog does not fail closed on missing ${check.label} from server response`)
  }
}

// 6b. No nullish coalescing (??) or logical-or (||) fallbacks on
//     required server-authoritative order fields in the Order
//     construction block. These fields must come directly from the
//     validated server response.
//     Extract the Order construction block and check for fallbacks
//     on subtotal, discount, shipping, total, status.
const orderBlockMatch = src.match(/const order:\s*Order\s*=\s*\{[\s\S]*?\}/)
if (orderBlockMatch) {
  const orderBlock = orderBlockMatch[0]
  // Check for ?? or || on required fields (excluding accessToken which
  // is optional, and shippingMethod/paymentMethod which use label
  // fallback from the server-validated method id).
  const fallbackPattern = /(?:subtotal|discount|shipping|total|status)\s*:\s*o\.\w+\s*(?:\?\?|\|\|)/
  if (fallbackPattern.test(orderBlock)) {
    errors.push('CheckoutDialog uses ?? or || fallback on required server-authoritative order field — use direct value from validated response')
  }
}

// 7. The key is reset after confirmed success.
// Must exist somewhere in the file (the success path in placeOrder).
if (!/idempotencyKey\.value\s*=\s*null/.test(src)) {
  errors.push('CheckoutDialog does not reset idempotency key after confirmed success')
}

// 8. Fail closed if crypto.randomUUID is unavailable.
if (!/crypto\.randomUUID.*function/.test(src) && !/typeof crypto\.randomUUID.*function/.test(src)) {
  errors.push('CheckoutDialog does not check for crypto.randomUUID availability — must fail closed if unavailable')
}

// 9. The key is NOT reset inside the ui.checkoutOpen watcher.
// Extract the open-watcher block and check it does not contain
// idempotencyKey.value = null.
const openWatcherMatch = src.match(/watch\(\(\)\s*=>\s*ui\.checkoutOpen[^{]*\{[^}]*\}\s*\)/s)
if (openWatcherMatch) {
  const watcherBody = openWatcherMatch[0]
  if (/idempotencyKey\.value\s*=\s*null/.test(watcherBody)) {
    errors.push('CheckoutDialog resets idempotencyKey inside the ui.checkoutOpen watcher — reopening must preserve the key for replay safety')
  }
} else {
  // Try a broader match for the watcher block
  const broadMatch = src.match(/watch\(\(\)\s*=>\s*ui\.checkoutOpen[\s\S]*?\n\}\)/)
  if (broadMatch) {
    const watcherBody = broadMatch[0]
    if (/idempotencyKey\.value\s*=\s*null/.test(watcherBody)) {
      errors.push('CheckoutDialog resets idempotencyKey inside the ui.checkoutOpen watcher — reopening must preserve the key for replay safety')
    }
  }
}

// 10. Declaration order: checkoutFingerprint and its watch must appear
//     AFTER shippingForm, selectedShipping, and selectedPayment in the
//     source. Vue's watch evaluates the computed getter immediately to
//     collect dependencies; accessing a const binding before its
//     initializer runs is a TDZ violation.
const fingerprintIdx = src.indexOf('checkoutFingerprint')
const watchFingerprintIdx = src.indexOf('watch(checkoutFingerprint')
const shippingFormIdx = src.indexOf('const shippingForm')
const selectedShippingIdx = src.indexOf('const selectedShipping')
const selectedPaymentIdx = src.indexOf('const selectedPayment')

if (fingerprintIdx < 0) {
  errors.push('CheckoutDialog does not declare checkoutFingerprint computed')
} else if (watchFingerprintIdx < 0) {
  errors.push('CheckoutDialog does not watch checkoutFingerprint')
} else {
  if (shippingFormIdx < 0 || fingerprintIdx < shippingFormIdx) {
    errors.push('CheckoutDialog: checkoutFingerprint is declared before shippingForm — TDZ violation risk')
  }
  if (selectedShippingIdx < 0 || fingerprintIdx < selectedShippingIdx) {
    errors.push('CheckoutDialog: checkoutFingerprint is declared before selectedShipping — TDZ violation risk')
  }
  if (selectedPaymentIdx < 0 || fingerprintIdx < selectedPaymentIdx) {
    errors.push('CheckoutDialog: checkoutFingerprint is declared before selectedPayment — TDZ violation risk')
  }
  if (watchFingerprintIdx < fingerprintIdx) {
    errors.push('CheckoutDialog: watch(checkoutFingerprint) is declared before checkoutFingerprint — TDZ violation')
  }
}

// 11. checkoutFingerprint must use JSON.stringify (not delimiter
//     concatenation) to avoid collisions when user-controlled values
//     contain delimiter characters. A delimiter-based fingerprint
//     (e.g. sku:qty.join(',') + '|' + fields) can collide when a field
//     value contains the delimiter, leaving the old idempotency key
//     active after a material input change.
//     Extract the checkoutFingerprint computed body and check.
const fingerprintBlockMatch = src.match(/checkoutFingerprint\s*=\s*computed\(\(\)\s*=>\s*\{([\s\S]*?)\n\}\)/)
if (fingerprintBlockMatch) {
  const fingerprintBody = fingerprintBlockMatch[1]
  if (!/JSON\.stringify/.test(fingerprintBody)) {
    errors.push('checkoutFingerprint does not use JSON.stringify — delimiter concatenation is collision-prone with user-controlled values')
  }
  // Check for old delimiter patterns: .join(',') with ':' or '|' separators
  if (/\.join\(['"],?['"]\)/.test(fingerprintBody) && /\|/.test(fingerprintBody)) {
    errors.push('checkoutFingerprint uses delimiter concatenation (join + |) — use JSON.stringify over a typed object instead')
  }
} else {
  errors.push('Could not extract checkoutFingerprint computed body for JSON.stringify check')
}

if (errors.length > 0) {
  console.error('Checkout idempotency check FAILED:')
  for (const e of errors) {
    console.error(`  - ${e}`)
  }
  process.exit(1)
}

console.log('Checkout idempotency check PASSED: stable key, crypto.randomUUID, fail-closed on all required fields, no open-watcher reset, correct declaration order, JSON fingerprint.')
