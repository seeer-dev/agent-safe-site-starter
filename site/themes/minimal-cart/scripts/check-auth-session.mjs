/**
 * Source assertion: public auth has a real session producer, bootstrap
 * initializes it, custom bearer-token persistence is gone, checkout uses
 * the member endpoint only with an authenticated token, and the browser
 * never queries Supabase Database.
 *
 * Run: node scripts/check-auth-session.mjs
 */
import { readFileSync } from 'fs'
import { resolve, dirname } from 'path'
import { fileURLToPath } from 'url'
import { spawnSync } from 'child_process'

const __dirname = dirname(fileURLToPath(import.meta.url))
const root = resolve(__dirname, '..')
const errors = []

function read(rel) {
  return readFileSync(resolve(root, rel), 'utf-8')
}

const session = read('shared/lib/auth/session.ts')
const bootstrap = read('islands/bootstrap.ts')
const userStore = read('shared/stores/user.ts')
const checkout = read('islands/CheckoutDialog/CheckoutDialog.vue')
const account = read('islands/AccountDialog/AccountDialog.vue')
const track = read('islands/TrackOrderDialog/TrackOrderDialog.vue')
const api = read('shared/lib/api.ts')
const memberOrders = read('shared/lib/member-orders.ts')
const vite = read('vite.config.ts')

const themeSources = [
  session,
  bootstrap,
  userStore,
  checkout,
  account,
  api,
]

if (!/export async function signInWithPassword/.test(session)) {
  errors.push('session producer does not export signInWithPassword')
}
if (!/export async function signUp/.test(session)) {
  errors.push('session producer does not export signUp')
}
if (!/export async function signOut/.test(session)) {
  errors.push('session producer does not export signOut')
}
if (!/export async function signInWithOAuth/.test(session)) {
  errors.push('session producer does not export signInWithOAuth')
}
if (!/custom:line/.test(session) || !/'google'/.test(session) || !/requireOAuthProvider/.test(session)) {
  errors.push('OAuth initiation must be typed to google and custom:line')
}
if (!/oauthRedirectTo/.test(session) || !/location\.origin/.test(session) || !/invalid_redirect/.test(session)) {
  errors.push('OAuth redirect must be origin-only and fail-closed')
}
if (/provider_token|provider_refresh_token/.test(session)) {
  errors.push('session producer must not persist provider access/refresh tokens')
}

if (!/initPublicAuth/.test(bootstrap)) {
  errors.push('bootstrap.ts does not initialize public auth')
}
if (!/syncFromSession/.test(bootstrap)) {
  errors.push('bootstrap.ts does not mirror the session into the user store')
}

if (/bearerToken:\s*bearerToken\.value/.test(userStore) || /JSON\.stringify\(\{[\s\S]*bearerToken/.test(userStore)) {
  errors.push('user store still persists bearerToken')
}
if (/user:\s*user\.value/.test(userStore) && /localStorage\.setItem\('minimal-user'/.test(userStore)) {
  errors.push('user store still persists user to minimal-user')
}
if (!/removeItem\(['"]minimal-user['"]\)/.test(userStore)) {
  errors.push('user store does not clear the legacy minimal-user key')
}
if (/function setUser\b/.test(userStore) || /function setBearerToken\b/.test(userStore) || /setUser,/.test(userStore) || /setBearerToken,/.test(userStore)) {
  errors.push('user store still exports setUser/setBearerToken — session sync must be the sole producer')
}

if (!/createOrderForMember/.test(checkout)) {
  errors.push('CheckoutDialog does not call createOrderForMember')
}
if (!/const token = userStore\.bearerToken\.trim\(\)/.test(checkout)) {
  errors.push('CheckoutDialog does not read the authenticated bearer token before choosing an order endpoint')
}
if (!/token\s*\n\s*\? await createOrderForMember\(orderPayload, token\)\s*\n\s*: await createOrder\(orderPayload\)/.test(checkout)
  && !/token\s*\?\s*await createOrderForMember\(orderPayload,\s*token\)\s*:\s*await createOrder\(orderPayload\)/.test(checkout)) {
  errors.push('CheckoutDialog must call createOrderForMember only when an authenticated token exists, otherwise createOrder')
}
if (/member_id/.test(checkout)) {
  errors.push('CheckoutDialog must never send member_id from the browser')
}

if (!/signInWithPassword/.test(account) || !/signUp/.test(account) || !/signOut/.test(account)) {
  errors.push('AccountDialog must call the session producer for signIn/signUp/signOut')
}
if (!/signInWithOAuth/.test(account) || !/isGoogleOAuthEnabled/.test(account) || !/isLineOAuthEnabled/.test(account)) {
  errors.push('AccountDialog must call signInWithOAuth and gate buttons on Google/LINE flags')
}
if (!/handleOAuth\('google'\)/.test(account) || !/handleOAuth\('custom:line'\)/.test(account)) {
  errors.push('AccountDialog OAuth buttons must use google and custom:line')
}
if (!/v-if="googleOAuthEnabled"/.test(account) || !/v-if="lineOAuthEnabled"/.test(account)) {
  errors.push('AccountDialog must render Google/LINE buttons only when their flags are enabled')
}

if (!/signOut\(\)/.test(account) || !/status === 401/.test(account)) {
  errors.push('AccountDialog 401 handling must sign out the real session')
}

if (!/parseMemberOrderListEnvelope/.test(account) || !/MalformedMemberOrdersError/.test(memberOrders)) {
  errors.push('AccountDialog must fail closed through parseMemberOrderListEnvelope')
}
const listMyOrdersSrc = api.slice(
  api.indexOf('export async function listMyOrders'),
  api.indexOf('export async function getMyOrder'),
)
if (!listMyOrdersSrc || /data\.orders\s*\|\|\s*\[\]/.test(listMyOrdersSrc) || /orders\s*\|\|\s*\[\]/.test(listMyOrdersSrc)) {
  errors.push('listMyOrders fabricates [] when the 2xx envelope lacks an orders array')
}
if (!/return res\.json\(\)/.test(listMyOrdersSrc)) {
  errors.push('listMyOrders must return the raw JSON envelope for fail-closed parsing')
}
if (!/!Array\.isArray\(orders\)/.test(memberOrders) || !/parseMemberOrderListEnvelope/.test(memberOrders)) {
  errors.push('parseMemberOrderListEnvelope must require a real orders array')
}
if (/subtotal:\s*o\.subtotal\s*\?\?/.test(account) || /subtotal:\s*o\.subtotal\s*\|\|/.test(memberOrders)) {
  errors.push('member order mapping fabricates subtotal')
}
if (/\?\?\s*0/.test(account) || /\|\|\s*['"]pending['"]/.test(account) || /Date\.now\(\)/.test(memberOrders)) {
  errors.push('AccountDialog/member-orders fabricates 0, pending, or Date.now for authoritative fields')
}
if (!/無法載入訂單，請稍後再試。/.test(account)) {
  errors.push('AccountDialog is missing the generic order load error')
}
if (!/requirePositiveUnix\(o\.updated_unix\)/.test(memberOrders) || !/value <= 0/.test(memberOrders)) {
  errors.push('member-orders must require a positive updated_unix')
}
if (!/value < 0/.test(memberOrders) || !/requirePositiveInteger/.test(memberOrders)) {
  errors.push('member-orders must reject negative amounts and non-positive quantities')
}

if (!/getMyOrder/.test(track) || !/memberBearerToken/.test(track)) {
  errors.push('TrackOrderDialog must call getMyOrder with the authenticated bearer token')
}
if (!/await getMyOrder\(id, token\)/.test(track) && !/getMyOrder\(id, token\)/.test(track)) {
  errors.push('TrackOrderDialog member path must pass the bearer token to getMyOrder')
}
if (!/if \(memberBearerToken\.value\)/.test(track) || !/loadMemberOrder/.test(track)) {
  errors.push('signed-in member search must use the member endpoint without a guest access token')
}
if (!/status === 401/.test(track) || !/signOut\(\)/.test(track)) {
  errors.push('TrackOrderDialog member 401 must sign out the real Supabase session')
}
if (!/parseMemberOrder\(result\)/.test(track)) {
  errors.push('TrackOrderDialog must fail closed through parseMemberOrder on member detail')
}
const loadMemberSrc = track.slice(
  track.indexOf('async function loadMemberOrder'),
  track.indexOf('async function handleSearch'),
)
if (!loadMemberSrc || /tokenInput/.test(loadMemberSrc)) {
  errors.push('member order load must not require a guest access token')
}
if (/\|\|\s*['"]pending['"]/.test(loadMemberSrc) || /Date\.now\(\)/.test(loadMemberSrc) || /\?\?\s*0/.test(loadMemberSrc)) {
  errors.push('member order load fabricates pending, Date.now, or 0')
}
if (!/ui\.trackOrderId && memberBearerToken\.value/.test(track) || !/void loadMemberOrder\(ui\.trackOrderId\)/.test(track)) {
  errors.push('opening a listed member order must call getMyOrder via loadMemberOrder without a guest token')
}

if (/Date\.now\(\)/.test(session)) {
  errors.push('public session mapping must not invent joinedAt with Date.now')
}
if (!/joinedAt: Number\.isFinite\(parsedJoinedAt\) \? parsedJoinedAt : null/.test(session)) {
  errors.push('public session mapping must leave unknown joinedAt as null')
}

const closeButtons = account.match(/<button[\s\S]*?@click="ui\.closeAccount\(\)"/g) || []
if (closeButtons.length < 3) {
  errors.push('AccountDialog is missing icon-only close buttons')
}
for (const btn of closeButtons) {
  if (!/type="button"/.test(btn) || !/aria-label="/.test(btn)) {
    errors.push('AccountDialog close buttons must be type=button with an aria-label')
    break
  }
}

for (const [name, src] of [
  ['session.ts', session],
  ['bootstrap.ts', bootstrap],
  ['user.ts', userStore],
  ['CheckoutDialog.vue', checkout],
  ['AccountDialog.vue', account],
  ['TrackOrderDialog.vue', track],
  ['api.ts', api],
  ['member-orders.ts', memberOrders],
]) {
  if (/\.from\(/.test(src) || /supabase\.schema\(/.test(src) || /SERVICE_ROLE/.test(src) || /service_role/.test(src)) {
    errors.push(`${name} appears to call Supabase Database or use a service-role key`)
  }
}

if (/auth\.admin/.test(session)) {
  errors.push('session producer must not use supabase.auth.admin')
}

if (!/import\('@supabase\/supabase-js'\)/.test(session)) {
  errors.push('public SDK should be lazy-imported from @supabase/supabase-js')
}

if (/DEV_AUTH_TOKEN/.test(vite) || /SERVICE_ROLE/.test(vite) || /CLIENT_SECRET/.test(vite) || /CHANNEL_SECRET/.test(vite)) {
  errors.push('theme Vite config must not expose secrets or DEV_AUTH_TOKEN')
}
if (!/import\.meta\.env\.AUTH_MODE/.test(vite) || !/import\.meta\.env\.SUPABASE_URL/.test(vite) || !/import\.meta\.env\.SUPABASE_PUBLISHABLE_KEY/.test(vite)) {
  errors.push('theme Vite config must expose AUTH_MODE, SUPABASE_URL, and SUPABASE_PUBLISHABLE_KEY')
}
if (!/import\.meta\.env\.AUTH_GOOGLE_ENABLED/.test(vite) || !/import\.meta\.env\.AUTH_LINE_ENABLED/.test(vite)) {
  errors.push('theme Vite config must expose fail-closed AUTH_GOOGLE_ENABLED and AUTH_LINE_ENABLED flags')
}

const publicConfig = read('shared/lib/auth/config.ts')
if (!/AUTH_GOOGLE_ENABLED/.test(publicConfig) || !/AUTH_LINE_ENABLED/.test(publicConfig)) {
  errors.push('public auth config must read AUTH_GOOGLE_ENABLED and AUTH_LINE_ENABLED')
}
if (!/value === '1' \|\| value === 'true'/.test(publicConfig)) {
  errors.push('OAuth flags must be fail-closed (only 1/true/yes/on enable the provider)')
}

if (errors.length > 0) {
  console.error('Auth session check FAILED:')
  for (const e of errors) {
    console.error(`  - ${e}`)
  }
  process.exit(1)
}

const flow = spawnSync(process.execPath, [resolve(root, 'scripts/check-member-order-flow.mjs')], {
  encoding: 'utf-8',
})
if (flow.stdout) process.stdout.write(flow.stdout)
if (flow.stderr) process.stderr.write(flow.stderr)
if (flow.status !== 0) {
  process.exit(flow.status ?? 1)
}

console.log('Auth session check PASSED: producer, bootstrap, OAuth google/custom:line, no custom token persistence, member checkout gated by token, no database calls.')
