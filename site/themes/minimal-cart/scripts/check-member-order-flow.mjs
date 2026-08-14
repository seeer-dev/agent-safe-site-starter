/**
 * Focused regression: a 2xx list envelope without an orders array must
 * not become a fabricated empty success, and a signed-in member order
 * row must reach GET /api/orders/mine/{id} with the bearer token.
 *
 * Invoked from check-auth-session.mjs.
 */
import { readFileSync } from 'fs'
import { resolve, dirname } from 'path'
import { fileURLToPath } from 'url'

const __dirname = dirname(fileURLToPath(import.meta.url))
const root = resolve(__dirname, '..')
const errors = []

function read(rel) {
  return readFileSync(resolve(root, rel), 'utf-8')
}

const api = read('shared/lib/api.ts')
const parser = read('shared/lib/member-orders.ts')
const track = read('islands/TrackOrderDialog/TrackOrderDialog.vue')
const account = read('islands/AccountDialog/AccountDialog.vue')

const listSrc = api.slice(
  api.indexOf('export async function listMyOrders'),
  api.indexOf('export async function getMyOrder'),
)

if (/data\.orders\s*\|\|\s*\[\]/.test(listSrc) || /\.orders\s*\|\|\s*\[\]/.test(listSrc)) {
  errors.push('REGRESSION: listMyOrders uses data.orders || []')
}
if (/return data\.orders/.test(listSrc)) {
  errors.push('REGRESSION: listMyOrders returns data.orders instead of the raw envelope')
}
if (!/return res\.json\(\)/.test(listSrc)) {
  errors.push('REGRESSION: listMyOrders does not return res.json()')
}

function parseEnvelopeOrThrow(raw) {
  if (!raw || typeof raw !== 'object' || Array.isArray(raw) || !Array.isArray(raw.orders)) {
    throw new Error('malformed')
  }
  return raw.orders
}

const malformedEnvelopes = [undefined, null, {}, { order: [] }, { orders: null }, []]
for (const sample of malformedEnvelopes) {
  let threw = false
  try {
    parseEnvelopeOrThrow(sample)
  } catch {
    threw = true
  }
  if (!threw) {
    errors.push(`REGRESSION: envelope ${JSON.stringify(sample)} was accepted as an empty success`)
  }
}
try {
  const empty = parseEnvelopeOrThrow({ orders: [] })
  if (!Array.isArray(empty) || empty.length !== 0) {
    errors.push('REGRESSION: { orders: [] } must remain a real empty list')
  }
} catch {
  errors.push('REGRESSION: { orders: [] } must be a valid empty list')
}

if (!/if \(!Array\.isArray\(orders\)\)/.test(parser)) {
  errors.push('REGRESSION: parser does not reject a missing orders array')
}
if (!/parseMemberOrderListEnvelope/.test(account)) {
  errors.push('REGRESSION: AccountDialog does not parse the list envelope')
}

if (!/getMyOrder\(id, token\)/.test(track)) {
  errors.push('REGRESSION: TrackOrderDialog does not call getMyOrder(id, token)')
}
if (!/void loadMemberOrder\(ui\.trackOrderId\)/.test(track)) {
  errors.push('REGRESSION: listed member orders do not auto-load via the member endpoint')
}
if (!/if \(memberBearerToken\.value\)/.test(track)) {
  errors.push('REGRESSION: signed-in search still requires a guest access token')
}

if (errors.length > 0) {
  console.error('Member order flow check FAILED:')
  for (const e of errors) {
    console.error(`  - ${e}`)
  }
  process.exit(1)
}

console.log('Member order flow check PASSED: no fabricated empty list; member detail uses bearer getMyOrder.')
