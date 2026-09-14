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

  let data: unknown = null
  try {
    data = await res.json()
  } catch {
    /* no body */
  }

  if (!res.ok) {
    const message =
      (data && typeof data === 'object' && 'error' in data && typeof (data as { error: unknown }).error === 'string'
        ? (data as { error: string }).error
        : null) ??
      (data && typeof data === 'object' && 'message' in data && typeof (data as { message: unknown }).message === 'string'
        ? (data as { message: string }).message
        : null) ??
      `請求失敗（${res.status}）`
    throw new ApiError(message, res.status)
  }

  return data as T
}

export const apiGet = <T>(path: string, headers?: Record<string, string>) => api<T>(path, { headers })
export const apiPost = <T>(path: string, body?: unknown, headers?: Record<string, string>) =>
  api<T>(path, { method: 'POST', body, headers })
