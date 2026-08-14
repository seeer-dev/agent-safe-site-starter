/**
 * Source assertion: browser policy constants are no longer authority.
 *
 * This script checks that the browser bundle does not contain hardcoded
 * pricing, shipping, or payment authority constants. Monetary values and
 * method availability must come from current Go API responses.
 *
 * It also checks that:
 * - cart.ts persists ONLY identifiers (not the full Product object)
 * - CartDrawer does not render item.product.price * quantity
 * - CheckoutDialog review uses quote.items (not cart.items) for line prices
 * - Payment method is wired end-to-end (sent to quote and order APIs)
 *
 * Run: node scripts/check-browser-authority.mjs
 */
import { readFileSync, existsSync } from 'fs'
import { resolve, dirname } from 'path'
import { fileURLToPath } from 'url'

const __dirname = dirname(fileURLToPath(import.meta.url))
const root = resolve(__dirname, '..')

const errors = []

// 1. mock-data.ts must NOT export pricing/shipping/payment constants
const mockDataPath = resolve(root, 'shared/lib/mock-data.ts')
const mockData = readFileSync(mockDataPath, 'utf-8')

const forbiddenExports = [
  'FREE_SHIPPING_THRESHOLD',
  'SHIPPING_FLAT_RATE',
  'SHIPPING_METHODS',
  'PAYMENT_METHODS',
]

for (const name of forbiddenExports) {
  const exportPattern = new RegExp(`export\\s+const\\s+${name}\\b`)
  if (exportPattern.test(mockData)) {
    errors.push(`mock-data.ts still exports ${name} — must be removed`)
  }
}

// 2. cart.ts must NOT import or compute monetary values
const cartPath = resolve(root, 'shared/stores/cart.ts')
const cart = readFileSync(cartPath, 'utf-8')

const forbiddenCartImports = [
  'FREE_SHIPPING_THRESHOLD',
  'SHIPPING_FLAT_RATE',
]

for (const name of forbiddenCartImports) {
  if (cart.includes(name)) {
    errors.push(`cart.ts still references ${name} — must not import pricing constants`)
  }
}

// cart.ts must NOT expose subtotal, discount, shipping, tax, total as computed
const forbiddenCartComputed = [
  'const subtotal',
  'const discount',
  'const shipping',
  'const tax',
  'const total',
]

for (const pattern of forbiddenCartComputed) {
  if (cart.includes(pattern)) {
    errors.push(`cart.ts still has local computed "${pattern}" — monetary values must come from server`)
  }
}

// cart.ts persist() must NOT serialize the full Product object.
// It must persist ONLY identifiers, quantity, and selections.
// Check that persist() maps to a minimal shape (productId, sku, quantity,
// selectedSize, selectedColor) — NOT the full product object.
if (cart.includes('items: items.value') || cart.includes('items: items.value,')) {
  errors.push('cart.ts persist() serializes full items (including Product) — must persist only identifiers')
}
// Verify persist maps to identifier-only shape
if (!cart.includes('productId: i.product.id') || !cart.includes('sku: i.product.sku')) {
  errors.push('cart.ts persist() does not map to identifier-only shape — must persist productId, sku, quantity, selections')
}
// Verify restore does NOT directly set items from localStorage
if (cart.includes('items.value = data.items')) {
  errors.push('cart.ts restore() sets items directly from localStorage — must rehydrate from catalog API')
}
// Verify rehydrate function exists
if (!cart.includes('function rehydrate') && !cart.includes('async function rehydrate')) {
  errors.push('cart.ts does not have a rehydrate() function — must rehydrate display data from catalog API')
}

// 3. CheckoutDialog must NOT import SHIPPING_METHODS or PAYMENT_METHODS from mock-data
const checkoutPath = resolve(root, 'islands/CheckoutDialog/CheckoutDialog.vue')
const checkout = readFileSync(checkoutPath, 'utf-8')

if (checkout.includes('SHIPPING_METHODS') && checkout.includes('mock-data')) {
  errors.push('CheckoutDialog still imports SHIPPING_METHODS from mock-data — must use fetchShippingMethods API')
}
if (checkout.includes('PAYMENT_METHODS') && checkout.includes('mock-data')) {
  errors.push('CheckoutDialog still imports PAYMENT_METHODS from mock-data — must use fetchPaymentMethods API')
}

// 4. CartDrawer must NOT import FREE_SHIPPING_THRESHOLD or show monetary totals
const cartDrawerPath = resolve(root, 'islands/CartDrawer/CartDrawer.vue')
const cartDrawer = readFileSync(cartDrawerPath, 'utf-8')

if (cartDrawer.includes('FREE_SHIPPING_THRESHOLD')) {
  errors.push('CartDrawer still references FREE_SHIPPING_THRESHOLD — must not use browser pricing constants')
}
if (cartDrawer.includes('已達免運門檻')) {
  errors.push('CartDrawer still shows "已達免運門檻" — no free-shipping threshold is approved')
}
if (cartDrawer.includes('運費免運') || cartDrawer.includes("'免運'")) {
  errors.push('CartDrawer still shows local shipping calculation — must use server quote')
}
// CartDrawer must NOT render item.product.price * quantity (pre-quote monetary claim)
if (cartDrawer.includes('item.product.price * item.quantity')) {
  errors.push('CartDrawer renders item.product.price * quantity — must not show pre-quote monetary totals')
}
// CartDrawer must NOT import formatNTD (no monetary display)
if (cartDrawer.includes('formatNTD')) {
  errors.push('CartDrawer imports formatNTD — must not show monetary values before server quote')
}

// 5. CheckoutDialog must fetch shipping/payment methods from the API
if (!checkout.includes('fetchShippingMethods')) {
  errors.push('CheckoutDialog does not call fetchShippingMethods — must fetch from server')
}
if (!checkout.includes('fetchPaymentMethods')) {
  errors.push('CheckoutDialog does not call fetchPaymentMethods — must fetch from server')
}

// 5a. CheckoutDialog review step must use quote.items, not cart.items, for line prices
if (checkout.includes('v-for="item in cart.items"') && checkout.includes('item.product.price')) {
  errors.push('CheckoutDialog review step loops over cart.items with item.product.price — must use quote.items for server-authoritative line names and prices')
}
// Verify quote.items is used in the review step
if (!checkout.includes('quote.items')) {
  errors.push('CheckoutDialog does not reference quote.items — review step must use server-authoritative line items')
}

// 5b. CheckoutDialog must send paymentMethod to fetchQuote and createOrder
if (!checkout.includes('selectedPayment.value') || !checkout.includes('paymentMethod')) {
  errors.push('CheckoutDialog does not send paymentMethod — payment selection must be wired to quote and order APIs')
}

// 5c. CheckoutDialog must NOT auto-select unavailable shipping methods
if (checkout.includes('firstAvailable ?? first') || checkout.includes('?? first')) {
  errors.push('CheckoutDialog auto-selects unavailable shipping method — must keep selectedShipping empty when no available method')
}
// Verify checkoutUnavailable blocks the shipping Next button
if (!checkout.includes('checkoutUnavailable')) {
  errors.push('CheckoutDialog does not reference checkoutUnavailable — must block progression when checkout is unavailable')
}

// 6. api.ts must export fetchShippingMethods and fetchPaymentMethods
const apiPath = resolve(root, 'shared/lib/api.ts')
const api = readFileSync(apiPath, 'utf-8')

if (!api.includes('export async function fetchShippingMethods')) {
  errors.push('api.ts does not export fetchShippingMethods — browser needs server-authoritative shipping methods')
}
if (!api.includes('export async function fetchPaymentMethods')) {
  errors.push('api.ts does not export fetchPaymentMethods — browser needs server-authoritative payment methods')
}

// 6c. The browser must submit a typed promo code to quote/order without
// downloading or locally matching the set of active codes.
if (api.includes('/api/promos/active') || api.includes('fetchActivePromos')) {
  errors.push('api.ts enumerates active promo codes — submit typed codes only to quote/order')
}
if (cart.includes('loadPromos') || cart.includes('promos.value.find')) {
  errors.push('cart.ts locally validates against an enumerated promo list — server quote/order must validate')
}
if (!cart.includes('優惠碼將於結帳報價時由伺服器驗證')) {
  errors.push('cart.ts does not describe promo input as pending server validation')
}

// 7. Hero claims must not contain an unapproved shipping threshold or
// availability promise. Shipping availability comes from the API.
const hero = readFileSync(resolve(root, 'islands/Hero/Hero.vue'), 'utf-8')
if (hero.includes('1,500') || hero.includes('1500') || hero.includes('免運')) {
  errors.push('Hero.vue contains an unapproved free-shipping claim')
}

// 6a. api.ts fetchQuote must include payment_method
if (!api.includes('payment_method: paymentMethod')) {
  errors.push('api.ts fetchQuote does not send payment_method — payment must be validated server-side')
}

// 6b. api.ts createOrder must include payment_method
if (!api.includes('payment_method: input.paymentMethod')) {
  errors.push('api.ts createOrder does not send payment_method — payment must be persisted on the order')
}

if (errors.length > 0) {
  console.error('Browser authority check FAILED:')
  for (const e of errors) {
    console.error(`  - ${e}`)
  }
  process.exit(1)
}

console.log('Browser authority check PASSED: no pricing/shipping/payment constants in browser code.')
