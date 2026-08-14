import {
  getSupabasePublishableKey,
  getSupabaseURL,
  getAuthMode,
  isSupabaseConfigured,
  isOAuthProvider,
  isOAuthProviderEnabled,
  type OAuthProvider,
} from './config'

export type { OAuthProvider }

export function requireOAuthProvider(value: string): OAuthProvider {
  if (isOAuthProvider(value)) return value
  throw new Error('unsupported_provider')
}

export function oauthRedirectTo(origin = typeof window !== 'undefined' ? window.location.origin : ''): string {
  const trimmed = origin.trim().replace(/\/+$/, '')
  if (!/^https?:\/\/[^/?#]+$/i.test(trimmed)) {
    throw new Error('invalid_redirect')
  }
  return trimmed
}

export interface AuthSession {
  accessToken: string
  user: {
    id: string
    email: string | null
  }
}

export interface SessionAdapter {
  init(): Promise<AuthSession | null>
  subscribe(listener: (session: AuthSession | null) => void): () => void
  signInWithPassword(email: string, password: string): Promise<AuthSession>
  signInWithOAuth(provider: OAuthProvider): Promise<void>
  signInWithDevToken?(token: string): Promise<AuthSession>
  signOut(): Promise<void>
}

let testAdapter: SessionAdapter | null = null
let cachedAdapter: SessionAdapter | null = null

export function setSessionAdapterForTests(adapter: SessionAdapter | null): void {
  testAdapter = adapter
  cachedAdapter = null
}

export function getTestSessionAdapter(): SessionAdapter | null {
  return testAdapter
}

export function resetSessionAdapterForTests(): void {
  testAdapter = null
  cachedAdapter = null
}

export function resolveSessionAdapter(): SessionAdapter {
  if (testAdapter) return testAdapter
  if (!cachedAdapter) {
    cachedAdapter = createAdminSessionAdapter()
  }
  return cachedAdapter
}

export function createAdminSessionAdapter(): SessionAdapter {
  const mode = getAuthMode()
  if (mode === 'dev') return createMemoryDevAdapter()
  if (mode === 'supabase' && isSupabaseConfigured()) return createSupabaseSessionAdapter()
  return createUnavailableAdapter()
}

function mapSupabaseSession(session: { access_token?: string; user?: { id: string; email?: string | null } } | null): AuthSession | null {
  const accessToken = session?.access_token
  const user = session?.user
  if (!accessToken || !user?.id) return null
  return {
    accessToken,
    user: { id: user.id, email: user.email ?? null },
  }
}

function createUnavailableAdapter(): SessionAdapter {
  return {
    async init() {
      return null
    },
    subscribe() {
      return () => {}
    },
    async signInWithPassword() {
      throw new Error('unavailable')
    },
    async signInWithOAuth() {
      throw new Error('unavailable')
    },
    async signOut() {},
  }
}

function createMemoryDevAdapter(): SessionAdapter {
  let current: AuthSession | null = null
  const listeners = new Set<(session: AuthSession | null) => void>()

  function emit(session: AuthSession | null) {
    current = session
    listeners.forEach((listener) => listener(session))
  }

  return {
    async init() {
      return current
    },
    subscribe(listener) {
      listeners.add(listener)
      return () => listeners.delete(listener)
    },
    async signInWithPassword() {
      throw new Error('dev_token_only')
    },
    async signInWithOAuth() {
      throw new Error('unavailable')
    },
    async signInWithDevToken(token: string) {
      const accessToken = token.trim()
      if (!accessToken) throw new Error('empty_token')
      const session: AuthSession = {
        accessToken,
        user: { id: 'dev', email: null },
      }
      emit(session)
      return session
    },
    async signOut() {
      emit(null)
    },
  }
}

function createSupabaseSessionAdapter(): SessionAdapter {
  let clientPromise: Promise<import('@supabase/supabase-js').SupabaseClient> | null = null

  function getClient() {
    if (!clientPromise) {
      clientPromise = import('@supabase/supabase-js').then(({ createClient }) =>
        createClient(getSupabaseURL(), getSupabasePublishableKey(), {
          auth: {
            persistSession: true,
            autoRefreshToken: true,
            detectSessionInUrl: true,
          },
        }),
      )
    }
    return clientPromise
  }

  return {
    async init() {
      const client = await getClient()
      const { data } = await client.auth.getSession()
      return mapSupabaseSession(data.session)
    },
    subscribe(listener) {
      let unsubscribe = () => {}
      void getClient().then((client) => {
        const { data } = client.auth.onAuthStateChange((_event, session) => {
          listener(mapSupabaseSession(session))
        })
        unsubscribe = () => data.subscription.unsubscribe()
      })
      return () => unsubscribe()
    },
    async signInWithPassword(email: string, password: string) {
      const client = await getClient()
      const { data, error } = await client.auth.signInWithPassword({ email, password })
      if (error || !data.session) {
        throw new Error('signin_failed')
      }
      const session = mapSupabaseSession(data.session)
      if (!session) throw new Error('signin_failed')
      return session
    },
    async signInWithOAuth(provider: OAuthProvider) {
      const allowed = requireOAuthProvider(provider)
      if (!isOAuthProviderEnabled(allowed)) throw new Error('unavailable')
      const redirectTo = oauthRedirectTo()
      const client = await getClient()
      const { error } = await client.auth.signInWithOAuth({
        provider: allowed as 'google',
        options: {
          redirectTo,
          ...(allowed === 'custom:line' ? { scopes: 'openid profile email' } : {}),
        },
      })
      if (error) throw new Error('oauth_failed')
    },
    async signOut() {
      const client = await getClient()
      await client.auth.signOut()
    },
  }
}
