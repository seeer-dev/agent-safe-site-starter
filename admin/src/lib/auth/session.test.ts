import { describe, it, expect, afterEach } from 'vitest'
import {
  oauthRedirectTo,
  requireOAuthProvider,
  resolveSessionAdapter,
  resetSessionAdapterForTests,
  setSessionAdapterForTests,
  type SessionAdapter,
} from './session'

function stubAdapter(label: string): SessionAdapter {
  return {
    async init() { return null },
    subscribe() { return () => {} },
    async signInWithPassword() { throw new Error(label) },
    async signInWithOAuth() { throw new Error(label) },
    async signOut() {},
  }
}

describe('resolveSessionAdapter', () => {
  afterEach(() => {
    resetSessionAdapterForTests()
  })

  it('returns the same production adapter for initialize, sign-in, and logout', () => {
    resetSessionAdapterForTests()
    const forInit = resolveSessionAdapter()
    const forSignIn = resolveSessionAdapter()
    const forLogout = resolveSessionAdapter()
    expect(forInit).toBe(forSignIn)
    expect(forSignIn).toBe(forLogout)
  })

  it('is resettable so a later resolve does not reuse a discarded adapter', () => {
    resetSessionAdapterForTests()
    const first = resolveSessionAdapter()
    resetSessionAdapterForTests()
    const second = resolveSessionAdapter()
    expect(second).not.toBe(first)
  })

  it('uses the injected test adapter instead of a cached production instance', () => {
    const injected = stubAdapter('test')
    setSessionAdapterForTests(injected)
    expect(resolveSessionAdapter()).toBe(injected)
    expect(resolveSessionAdapter()).toBe(injected)
  })
})

describe('oauth initiation contract', () => {
  it('accepts only google and custom:line', () => {
    expect(requireOAuthProvider('google')).toBe('google')
    expect(requireOAuthProvider('custom:line')).toBe('custom:line')
    expect(() => requireOAuthProvider('facebook')).toThrow('unsupported_provider')
    expect(() => requireOAuthProvider('line')).toThrow('unsupported_provider')
  })

  it('allows origin-only redirects and rejects paths or query strings', () => {
    expect(oauthRedirectTo('http://localhost:4173')).toBe('http://localhost:4173')
    expect(oauthRedirectTo('https://shop.example.com/')).toBe('https://shop.example.com')
    expect(() => oauthRedirectTo('https://shop.example.com/callback')).toThrow('invalid_redirect')
    expect(() => oauthRedirectTo('https://shop.example.com?next=/admin')).toThrow('invalid_redirect')
    expect(() => oauthRedirectTo('')).toThrow('invalid_redirect')
  })
})
