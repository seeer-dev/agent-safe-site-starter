/**
 * Admin API base prefix resolver and validator.
 *
 * Implements strict revision-2 configuration grammar:
 * - Relative prefix must strictly be "/api" (with optional trailing slashes).
 *   Any other relative path (e.g. "/custom/api", "/api/v1") is rejected.
 * - Absolute prefixes must end in "/api" (e.g. "https://api.example.com/api" or "https://api.example.com/v1/@scope/api").
 * - Remote absolute URLs must use HTTPS.
 * - HTTP is restricted to exact canonical loopback hosts (localhost, 127.0.0.1, [::1]).
 *   Alternative numeric, octal, hex, or non-canonical loopback spellings are rejected.
 * - Forbids userinfo (user:pass@host), queries, fragments, protocol-relative URLs,
 *   standalone literal or percent-encoded dot-segments (. , .. , %2e, %2e%2e, .%2e, %2e.),
 *   control characters, and embedded whitespace.
 * - Allows valid in-segment dots (v1..beta, v1%2ebeta) and in-segment @ characters (v1/@scope/api).
 * - Safely and silently falls back to "/api" on any invalid value without logging.
 */

const DEFAULT_API_BASE = '/api'
const CANONICAL_LOOPBACK_HOSTS = new Set(['localhost', '127.0.0.1', '[::1]'])
const DOT_SEGMENT_REGEX = /(?:^|\/)(?:\.|\.\.|%2e|%2e%2e|\.%2e|%2e\.)(?:\/|$)/i
const VALID_API_PATH_REGEX = /(?:^|\/)api$/

export function resolveApiBase(raw?: unknown): string {
  if (raw === undefined || raw === null) {
    return DEFAULT_API_BASE
  }
  const str = String(raw).trim()
  if (str === '') {
    return DEFAULT_API_BASE
  }

  // Reject control characters (\x00-\x1F, \x7F) and embedded whitespace
  if (/[\x00-\x1F\x7F\s]/.test(str)) {
    return DEFAULT_API_BASE
  }

  // Reject protocol-relative URLs (e.g. //example.com/api)
  if (str.startsWith('//')) {
    return DEFAULT_API_BASE
  }

  // Reject standalone literal or percent-encoded dot-segments
  if (DOT_SEGMENT_REGEX.test(str)) {
    return DEFAULT_API_BASE
  }

  // Reject query strings and fragments
  if (str.includes('?') || str.includes('#')) {
    return DEFAULT_API_BASE
  }

  // Relative path case: RelativePrefix MUST strictly be "/api" ["/"]*
  if (str.startsWith('/')) {
    if (str.replace(/\/+$/, '') !== '/api') {
      return DEFAULT_API_BASE
    }
    return DEFAULT_API_BASE
  }

  // Absolute URL case
  try {
    const parsed = new URL(str)
    const protocol = parsed.protocol.toLowerCase()

    // Reject userinfo credentials
    if (parsed.username || parsed.password) {
      return DEFAULT_API_BASE
    }

    if (protocol === 'http:') {
      // Require exact canonical loopback host in raw string without parser transformation
      const match = /^http:\/\/([^/:?#]+|\[[^\]]+\])(?::\d+)?\//i.exec(str)
      if (!match) {
        return DEFAULT_API_BASE
      }
      const rawHost = match[1].toLowerCase()
      if (!CANONICAL_LOOPBACK_HOSTS.has(rawHost)) {
        return DEFAULT_API_BASE
      }
    } else if (protocol === 'https:') {
      // Allowed for both remote and loopback
    } else {
      // Non-HTTP(S) schemes rejected
      return DEFAULT_API_BASE
    }

    // Path must end with /api
    const pathname = parsed.pathname
    const normalizedPath = pathname.replace(/\/+$/, '')
    if (!VALID_API_PATH_REGEX.test(normalizedPath)) {
      return DEFAULT_API_BASE
    }

    return `${parsed.origin}${normalizedPath}`
  } catch {
    return DEFAULT_API_BASE
  }
}

export function getApiBase(): string {
  const env = import.meta.env as Record<string, string | undefined> | undefined
  const raw = env?.ADMIN_API_BASE
  return resolveApiBase(raw)
}
