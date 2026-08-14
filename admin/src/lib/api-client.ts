import { getAccessToken } from '@/lib/auth/token'

const API_BASE = '/api'

export interface ApiResponse<T> {
  data: T
  error?: string
}

export class ApiError extends Error {
  constructor(public readonly status: number, message: string) {
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
  const res = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers,
  })
  if (!res.ok) {
    const body = await res.text()
    let errMsg = `API ${res.status}: ${body}`
    try {
      const errObj = JSON.parse(body)
      errMsg = errObj.error || errMsg
    } catch {}
    throw new ApiError(res.status, errMsg)
  }
  const ct = res.headers.get('content-type') || ''
  if (ct.includes('application/json')) {
    return res.json() as Promise<T>
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
