import { test } from 'node:test'
import assert from 'node:assert/strict'
import { strictOrigin, buildAdminHeaders } from './security-headers.mjs'

const has = (haystack, needle) => assert.ok(haystack.includes(needle), `expected to contain ${needle}`)
const throwsWith = (fn, needle) =>
  assert.throws(fn, (e) => e.message.includes(needle))

test('empty inputs produce self-only connect-src', () => {
  const out = buildAdminHeaders({})
  has(out, "connect-src 'self';")
  assert.equal((out.match(/connect-src/g) || []).length, 1)
})

test('companion headers are always emitted', () => {
  const out = buildAdminHeaders({})
  has(out, '/*')
  has(out, 'X-Content-Type-Options: nosniff')
  has(out, 'Referrer-Policy: strict-origin-when-cross-origin')
  has(out, 'X-Frame-Options: DENY')
  has(out, "frame-ancestors 'none'")
  has(out, "object-src 'none'")
  has(out, "base-uri 'self'")
  has(out, "form-action 'self'")
  has(out, "script-src 'self'")
})

test('https supabase origin is added to connect-src', () => {
  const out = buildAdminHeaders({ supabaseUrl: 'https://xyz.supabase.co' })
  has(out, "connect-src 'self' https://xyz.supabase.co")
})

test('same-origin api base is not duplicated', () => {
  const out = buildAdminHeaders({ adminApiBase: '/api' })
  has(out, "connect-src 'self'")
  assert.ok(!out.includes('/api '), 'relative prefix must not leak into CSP')
})

test('cross-origin api base ending in /api contributes its origin', () => {
  const out = buildAdminHeaders({ adminApiBase: 'https://api.example.com/api' })
  has(out, "connect-src 'self' https://api.example.com")
})

test('duplicate supabase and api origins collapse', () => {
  const out = buildAdminHeaders({
    supabaseUrl: 'https://shared.example.com',
    adminApiBase: 'https://shared.example.com/api',
  })
  assert.equal((out.match(/shared\.example\.com/g) || []).length, 1)
})

test('loopback http api base is allowed for local dev', () => {
  const out = buildAdminHeaders({ adminApiBase: 'http://localhost:8080/api' })
  has(out, 'connect-src \'self\' http://localhost:8080')
})

test('non-loopback http is rejected', () => {
  throwsWith(() => strictOrigin('http://api.example.com/api', { name: 'ADMIN_API_BASE', allowApiPath: true }), 'loopback')
})

test('userinfo is rejected', () => {
  throwsWith(() => strictOrigin('https://user:pass@evil.com', { name: 'SUPABASE_URL' }), 'userinfo')
})

test('query and fragment are rejected', () => {
  throwsWith(() => strictOrigin('https://a.com/?x=1', { name: 'SUPABASE_URL' }), 'query')
  throwsWith(() => strictOrigin('https://a.com/#f', { name: 'SUPABASE_URL' }), 'fragment')
})

test('non-/api paths are rejected even for the api base', () => {
  throwsWith(
    () => strictOrigin('https://a.com/evil/api', { name: 'ADMIN_API_BASE', allowApiPath: true }),
    'path',
  )
  throwsWith(() => strictOrigin('https://a.com/api', { name: 'SUPABASE_URL' }), 'path')
})

test('non-http schemes are rejected', () => {
  throwsWith(() => strictOrigin('javascript:alert(1)', { name: 'SUPABASE_URL' }), 'scheme')
  throwsWith(() => strictOrigin('ftp://a.com', { name: 'SUPABASE_URL' }), 'scheme')
})

test('control characters are rejected', () => {
  throwsWith(() => strictOrigin('https://a.com/\n.evil', { name: 'SUPABASE_URL' }), 'control')
  throwsWith(() => strictOrigin('https://a.c\tom', { name: 'SUPABASE_URL' }), 'control')
})

test('malformed values are rejected', () => {
  throwsWith(() => strictOrigin('not a url', { name: 'SUPABASE_URL' }), 'valid URL')
  throwsWith(() => strictOrigin('https://', { name: 'SUPABASE_URL' }), 'valid URL')
})

test('no wildcard sources anywhere in the policy', () => {
  const out = buildAdminHeaders({
    supabaseUrl: 'https://xyz.supabase.co',
    adminApiBase: 'https://api.example.com/api',
  })
  assert.ok(!/script-src[^;]*\*/.test(out), 'wildcard script-src')
  assert.ok(!/connect-src[^;]*\*/.test(out), 'wildcard connect-src')
  assert.ok(!/frame-src[^;]*\*/.test(out), 'wildcard frame-src')
})
