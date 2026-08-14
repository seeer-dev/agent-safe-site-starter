export type AuthMode = 'dev' | 'supabase' | ''
export type OAuthProvider = 'google' | 'custom:line'

export function getAuthMode(): AuthMode {
  const mode = String(import.meta.env.AUTH_MODE ?? '').trim().toLowerCase()
  if (mode === 'dev' || mode === 'supabase') return mode
  return ''
}

export function getSupabaseURL(): string {
  return String(import.meta.env.SUPABASE_URL ?? '').trim().replace(/\/+$/, '')
}

export function getSupabasePublishableKey(): string {
  return String(import.meta.env.SUPABASE_PUBLISHABLE_KEY ?? '').trim()
}

export function isSupabaseConfigured(): boolean {
  return getSupabaseURL() !== '' && getSupabasePublishableKey() !== ''
}

export function isAdminAuthReady(): boolean {
  const mode = getAuthMode()
  if (mode === 'dev') return true
  return mode === 'supabase' && isSupabaseConfigured()
}

function envFlag(name: string): boolean {
  const env = import.meta.env as Record<string, string | boolean | undefined>
  const value = String(env[name] ?? '').trim().toLowerCase()
  return value === '1' || value === 'true' || value === 'yes' || value === 'on'
}

export function isOAuthProvider(value: string): value is OAuthProvider {
  return value === 'google' || value === 'custom:line'
}

export function isGoogleOAuthEnabled(): boolean {
  return getAuthMode() === 'supabase' && isSupabaseConfigured() && envFlag('AUTH_GOOGLE_ENABLED')
}

export function isLineOAuthEnabled(): boolean {
  return getAuthMode() === 'supabase' && isSupabaseConfigured() && envFlag('AUTH_LINE_ENABLED')
}

export function isOAuthProviderEnabled(provider: OAuthProvider): boolean {
  if (provider === 'google') return isGoogleOAuthEnabled()
  if (provider === 'custom:line') return isLineOAuthEnabled()
  return false
}
