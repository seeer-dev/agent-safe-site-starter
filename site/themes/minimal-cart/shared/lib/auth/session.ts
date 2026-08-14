import {
  getSupabasePublishableKey,
  getSupabaseURL,
  isPublicAuthEnabled,
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

export interface PublicAuthUser {
  id: string
  email: string
  name: string
  joinedAt: number | null
}

export interface PublicAuthSession {
  accessToken: string
  user: PublicAuthUser
}

export type SignUpResult =
  | { kind: 'signed_in'; session: PublicAuthSession }
  | { kind: 'check_email' }

export class PublicAuthError extends Error {
  constructor(public readonly kind: 'signin' | 'signup' | 'signout' | 'unavailable' | 'network' | 'oauth') {
    super('auth_failed')
    this.name = 'PublicAuthError'
  }
}

type SessionListener = (session: PublicAuthSession | null) => void

let listener: SessionListener | null = null
let clientPromise: Promise<import('@supabase/supabase-js').SupabaseClient> | null = null
let unsubscribeAuth: (() => void) | null = null

function notify(session: PublicAuthSession | null) {
  listener?.(session)
}

function displayName(email: string, metadata: Record<string, unknown> | undefined): string {
  const raw = metadata?.name
  if (typeof raw === 'string' && raw.trim()) return raw.trim()
  if (email.includes('@')) return email.split('@')[0] || '會員'
  return '會員'
}

function mapSession(session: {
  access_token?: string
  user?: {
    id: string
    email?: string | null
    created_at?: string
    user_metadata?: Record<string, unknown>
  }
} | null): PublicAuthSession | null {
  const accessToken = session?.access_token
  const user = session?.user
  if (!accessToken || !user?.id) return null
  const email = user.email ?? ''
  const parsedJoinedAt = user.created_at ? Date.parse(user.created_at) : Number.NaN
  return {
    accessToken,
    user: {
      id: user.id,
      email,
      name: displayName(email, user.user_metadata),
      joinedAt: Number.isFinite(parsedJoinedAt) ? parsedJoinedAt : null,
    },
  }
}

function getClient() {
  if (!isPublicAuthEnabled()) {
    return Promise.reject(new PublicAuthError('unavailable'))
  }
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

export async function initPublicAuth(onSession: SessionListener): Promise<void> {
  listener = onSession
  unsubscribeAuth?.()
  unsubscribeAuth = null
  if (!isPublicAuthEnabled()) {
    notify(null)
    return
  }
  try {
    const client = await getClient()
    const { data } = await client.auth.getSession()
    notify(mapSession(data.session))
    const { data: sub } = client.auth.onAuthStateChange((_event, session) => {
      notify(mapSession(session))
    })
    unsubscribeAuth = () => sub.subscription.unsubscribe()
  } catch {
    notify(null)
  }
}

export async function signInWithPassword(email: string, password: string): Promise<PublicAuthSession> {
  if (!isPublicAuthEnabled()) throw new PublicAuthError('unavailable')
  try {
    const client = await getClient()
    const { data, error } = await client.auth.signInWithPassword({ email, password })
    const session = mapSession(data.session)
    if (error || !session) throw new PublicAuthError('signin')
    notify(session)
    return session
  } catch (error) {
    if (error instanceof PublicAuthError) throw error
    throw new PublicAuthError('network')
  }
}

export async function signInWithOAuth(provider: OAuthProvider): Promise<void> {
  try {
    const allowed = requireOAuthProvider(provider)
    if (!isPublicAuthEnabled() || !isOAuthProviderEnabled(allowed)) {
      throw new PublicAuthError('unavailable')
    }
    const redirectTo = oauthRedirectTo()
    const client = await getClient()
    const { error } = await client.auth.signInWithOAuth({
      provider: allowed as 'google',
      options: {
        redirectTo,
        ...(allowed === 'custom:line' ? { scopes: 'openid profile email' } : {}),
      },
    })
    if (error) throw new PublicAuthError('oauth')
  } catch (error) {
    if (error instanceof PublicAuthError) throw error
    throw new PublicAuthError('oauth')
  }
}

export async function signUp(email: string, password: string, name?: string): Promise<SignUpResult> {
  if (!isPublicAuthEnabled()) throw new PublicAuthError('unavailable')
  try {
    const client = await getClient()
    const { data, error } = await client.auth.signUp({
      email,
      password,
      options: name?.trim() ? { data: { name: name.trim() } } : undefined,
    })
    if (error) throw new PublicAuthError('signup')
    const session = mapSession(data.session)
    if (!session) return { kind: 'check_email' }
    notify(session)
    return { kind: 'signed_in', session }
  } catch (error) {
    if (error instanceof PublicAuthError) throw error
    throw new PublicAuthError('network')
  }
}

export async function signOut(): Promise<void> {
  try {
    if (isPublicAuthEnabled() && clientPromise) {
      const client = await getClient()
      await client.auth.signOut()
    }
  } catch {
    // UI must clear even if the provider call fails.
  } finally {
    notify(null)
  }
}

export function publicAuthErrorMessage(error: unknown): string {
  if (error instanceof PublicAuthError && error.kind === 'unavailable') {
    return '會員登入尚未開放。'
  }
  return '無法完成操作，請稍後再試。'
}
