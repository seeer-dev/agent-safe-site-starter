// Generates admin/dist/_headers for Cloudflare Pages during the production
// build. The CSP is derived only from validated build inputs — provider
// IDs and installation hosts are never hardcoded here.
//
//   SUPABASE_URL    — identity-provider origin the SPA talks to directly
//   ADMIN_API_BASE  — optional cross-origin API prefix (must end in /api);
//                     unset means same-origin /api via the Pages Function
//
// Empty inputs keep the policy self-only. Non-empty but malformed inputs
// fail the build rather than broadening or silently dropping the policy.
import { writeFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'
import { resolve } from 'node:path'

const LOOPBACK_HOSTS = new Set(['localhost', '127.0.0.1', '[::1]', '::1'])

// strictOrigin parses an origin-bearing config value. Returns the bare
// scheme://host[:port] string or throws on any malformed input.
//   - http is allowed only for loopback hosts (local dev)
//   - userinfo, query, fragment, and control characters are rejected
//   - a path is allowed only for API prefixes and only when it is exactly
//     the documented /api suffix
export function strictOrigin(raw, { name, allowApiPath = false } = {}) {
  const fail = (why) => {
    throw new Error(`${name}: ${why}`)
  }
  if (raw === undefined || raw === null) return ''
  const value = String(raw).trim()
  if (value === '') return ''
  // Reject any C0/C1 control or DEL anywhere — header injection guard.
  if (/[\x00-\x1f\x7f-\x9f]/.test(value)) fail('contains a control character')
  // An explicit same-origin /api prefix adds nothing to connect-src.
  if (allowApiPath && value === '/api') return ''

  let u
  try {
    u = new URL(value)
  } catch {
    fail(`not a valid URL: ${JSON.stringify(value)}`)
  }
  if (u.protocol !== 'https:' && u.protocol !== 'http:') {
    fail(`scheme ${u.protocol} is not http/https`)
  }
  const host = u.hostname.toLowerCase()
  const loopback = LOOPBACK_HOSTS.has(host)
  if (u.protocol === 'http:' && !loopback) {
    fail('http is only allowed for loopback hosts')
  }
  if (u.username !== '' || u.password !== '') fail('userinfo is not allowed')
  if (u.search !== '' || u.hash !== '') fail('query and fragment are not allowed')
  const path = u.pathname.replace(/\/+$/, '')
  if (path !== '' && !(allowApiPath && path === '/api')) {
    fail(`path ${JSON.stringify(u.pathname)} is not allowed`)
  }
  return `${u.protocol}//${u.host}`
}

// buildAdminHeaders returns the _headers file content for the admin SPA.
export function buildAdminHeaders({ supabaseUrl = '', adminApiBase = '' } = {}) {
  const supabaseOrigin = strictOrigin(supabaseUrl, { name: 'SUPABASE_URL' })
  // ADMIN_API_BASE is a request prefix (…/api); only its origin belongs in
  // connect-src. Empty means the Pages Function serves /api same-origin.
  const apiOrigin = strictOrigin(adminApiBase, { name: 'ADMIN_API_BASE', allowApiPath: true })

  const connectSrc = ["'self'", apiOrigin, supabaseOrigin]
    .filter((o, i, a) => (i === 0 ? true : o !== '' && o !== "'self'"))
    .filter((o, i, a) => a.indexOf(o) === i)
    .join(' ')

  const csp = [
    "default-src 'self'",
    "script-src 'self'",
    "style-src 'self' 'unsafe-inline'",
    "img-src 'self' data:",
    `connect-src ${connectSrc}`,
    "frame-ancestors 'none'",
    "object-src 'none'",
    "base-uri 'self'",
    "form-action 'self'",
  ].join('; ')

  return [
    '/*',
    '  X-Content-Type-Options: nosniff',
    '  Referrer-Policy: strict-origin-when-cross-origin',
    '  X-Frame-Options: DENY',
    `  Content-Security-Policy: ${csp}`,
    '',
  ].join('\n')
}

function main() {
  const content = buildAdminHeaders({
    supabaseUrl: process.env.SUPABASE_URL ?? '',
    adminApiBase: process.env.ADMIN_API_BASE ?? '',
  })
  const out = resolve(fileURLToPath(new URL('..', import.meta.url)), 'dist', '_headers')
  writeFileSync(out, content)
  console.log(`wrote ${out}`)
}

if (process.argv[1] && fileURLToPath(import.meta.url) === resolve(process.argv[1])) {
  main()
}
