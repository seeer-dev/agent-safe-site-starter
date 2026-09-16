import { getAccessToken } from '@/lib/auth/token'
import { getApiBase } from './api-config'

// ApiResponse is the wire envelope every JSON API response shares:
// { ok: true, data: T } on success, { ok: false, error: { code, message } }
// on failure. request() unwraps it; callers always receive T directly.
export interface ApiResponse<T> {
  ok: boolean
  data?: T
  error?: ApiErrorBody
}

export interface ApiErrorBody {
  code: string
  message: string
}

export class ApiError extends Error {
  constructor(
    public readonly status: number,
    message: string,
    public readonly code?: string,
  ) {
    super(message)
    this.name = 'ApiError'
  }
}

function getAuthToken(): string {
  return getAccessToken()
}

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...((options?.headers as Record<string, string>) ?? {}),
  }
  const token = getAuthToken()
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }
  const base = getApiBase()
  const normalizedPath = path.startsWith('/') ? path : `/${path}`
  const res = await fetch(`${base}${normalizedPath}`, {
    ...options,
    headers,
  })
  if (!res.ok) {
    const body = await res.text()
    let errMsg = `API ${res.status}: ${body}`
    let errCode: string | undefined
    try {
      const errObj = JSON.parse(body)
      // Envelope: error is { code, message }; legacy shape was a string.
      if (errObj?.error && typeof errObj.error === 'object') {
        errMsg = errObj.error.message || errMsg
        errCode = errObj.error.code
      } else if (typeof errObj?.error === 'string') {
        errMsg = errObj.error
      }
    } catch {}
    throw new ApiError(res.status, errMsg, errCode)
  }
  const ct = res.headers.get('content-type') || ''
  if (ct.includes('application/json')) {
    const body = (await res.json()) as ApiResponse<T> | T
    if (body !== null && typeof body === 'object' && 'ok' in body) {
      const env = body as ApiResponse<T>
      if (!env.ok) {
        throw new ApiError(res.status, env.error?.message ?? `API ${res.status}`, env.error?.code)
      }
      return env.data as T
    }
    // Legacy bare payload (mixed-version deploy window).
    return body as T
  }
  return undefined as unknown as T
}

export const api = {
  get: <T>(path: string) => request<T>(path),
  post: <T>(path: string, body?: unknown) =>
    request<T>(path, { method: 'POST', body: body ? JSON.stringify(body) : undefined }),
  put: <T>(path: string, body?: unknown) =>
    request<T>(path, { method: 'PUT', body: body ? JSON.stringify(body) : undefined }),
  patch: <T>(path: string, body?: unknown) =>
    request<T>(path, { method: 'PATCH', body: body ? JSON.stringify(body) : undefined }),
  del: <T>(path: string) => request<T>(path, { method: 'DELETE' }),
}
