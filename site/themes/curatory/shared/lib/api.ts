// API client for the curatory storefront. Same-origin in dev (the dev
// server proxies /api/*); in production the rendered <html data-api-base>
// attribute carries PUBLIC_API_BASE.

export class ApiError extends Error {
  status: number
  constructor(message: string, status: number) {
    super(message)
    this.status = status
  }
}

export function apiBase(): string {
  return document.documentElement.getAttribute('data-api-base')?.replace(/\/$/, '') ?? ''
}

interface ApiOptions {
  method?: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'
  body?: unknown
  headers?: Record<string, string>
}

export async function api<T>(path: string, options?: ApiOptions): Promise<T> {
  const headers: Record<string, string> = { 'Content-Type': 'application/json', ...options?.headers }
  const res = await fetch(`${apiBase()}${path}`, {
    method: options?.method ?? 'GET',
    headers,
    body: options?.body !== undefined ? JSON.stringify(options.body) : undefined,
  })

  let body: unknown = null
  try {
    body = await res.json()
  } catch {
    /* no body */
  }

  if (!res.ok) {
    const message = errorMessage(body) ?? `請求失敗（${res.status}）`
    throw new ApiError(message, res.status)
  }

  // Shared response envelope: { status: 'success', data: T }. Tolerate a
  // legacy bare payload while Pages and the origin deploy independently.
  // Detection is value-based so a bare payload carrying its own status field
  // is never mistaken for the envelope.
  if (body !== null && typeof body === 'object') {
    const env = body as { status?: unknown; data?: T; error?: { message?: string } }
    if (env.status === 'error') {
      throw new ApiError(env.error?.message ?? `請求失敗（${res.status}）`, res.status)
    }
    if (env.status === 'success') {
      return env.data as T
    }
  }
  return body as T
}

// errorMessage reads the public message from the shared envelope
// ({status:"error", error:{code,message}}) or the legacy {"error":"msg"} /
// {"message":"msg"} shapes.
function errorMessage(body: unknown): string | null {
  if (body === null || typeof body !== 'object') return null
  const error = (body as { error?: unknown }).error
  if (error && typeof error === 'object' && typeof (error as { message?: unknown }).message === 'string') {
    return (error as { message: string }).message
  }
  if (typeof error === 'string') return error
  const message = (body as { message?: unknown }).message
  return typeof message === 'string' ? message : null
}

export const apiGet = <T>(path: string, headers?: Record<string, string>) => api<T>(path, { headers })
export const apiPost = <T>(path: string, body?: unknown, headers?: Record<string, string>) =>
  api<T>(path, { method: 'POST', body, headers })
