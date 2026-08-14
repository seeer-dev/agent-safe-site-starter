// In-memory access token used by API clients. Supabase (or the dev
// adapter) owns session persistence; this module must not write a
// second copy to localStorage/sessionStorage.

let accessToken = ''

export function setAccessToken(token: string): void {
  accessToken = token
}

export function getAccessToken(): string {
  return accessToken
}
