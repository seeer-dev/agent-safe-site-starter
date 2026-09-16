// Media API client functions for presign and verify endpoints.
// Uses the same fetch + Bearer token pattern as api-client.ts but
// with custom request handling for the R2 PUT (binary upload).

import type {
  PresignRequest,
  PresignResponse,
  VerifyRequest,
  VerifyResponse,
} from './media-types'
import { getAccessToken } from '@/lib/auth/token'
import { getApiBase } from './api-config'

function getAuthToken(): string {
  return getAccessToken()
}

// readError extracts the public message from the shared response envelope
// ({ok:false, error:{code,message}}), tolerating the legacy {"error":"msg"}
// shape during mixed-version deploys.
async function readError(res: Response, fallback: string): Promise<string> {
  const body = await res.text()
  try {
    const obj = JSON.parse(body)
    if (obj?.error && typeof obj.error === 'object') {
      return obj.error.message || fallback
    }
    if (typeof obj?.error === 'string') {
      return obj.error
    }
  } catch { /* keep default */ }
  return fallback
}

// unwrap returns the payload inside the response envelope ({ok:true,data:T}),
// tolerating a legacy bare payload while Pages and the origin roll separately.
function unwrap<T>(body: unknown): T {
  if (body !== null && typeof body === 'object' && 'ok' in body) {
    return (body as { data?: T }).data as T
  }
  return body as T
}

/** Call POST /api/media/presign to get a presigned R2 upload URL. */
export async function presignUpload(req: PresignRequest, signal?: AbortSignal): Promise<PresignResponse> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  }
  const token = getAuthToken()
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }
  const base = getApiBase()
  const res = await fetch(`${base}/media/presign`, {
    method: 'POST',
    headers,
    body: JSON.stringify(req),
    signal,
  })
  if (!res.ok) {
    throw new Error(await readError(res, `presign failed: ${res.status}`))
  }
  return unwrap<PresignResponse>(await res.json())
}

/**
 * PUT the binary blob to the presigned R2 URL.
 * Uses the headers returned by presign (at minimum Content-Type).
 * The AbortSignal allows cancellation.
 */
export async function uploadToR2(
  url: string,
  blob: Blob,
  headers: Record<string, string>,
  signal?: AbortSignal,
): Promise<void> {
  const putHeaders: Record<string, string> = { ...headers }
  // Ensure Content-Type is set from presign response
  if (!putHeaders['Content-Type']) {
    putHeaders['Content-Type'] = blob.type || 'application/octet-stream'
  }
  const res = await fetch(url, {
    method: 'PUT',
    headers: putHeaders,
    body: blob,
    signal,
  })
  if (!res.ok) {
    throw new Error(`R2 upload failed: ${res.status} ${res.statusText}`)
  }
}

/** Call POST /api/media/verify to verify the uploaded object. */
export async function verifyUpload(
  req: VerifyRequest,
  signal?: AbortSignal,
): Promise<VerifyResponse> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  }
  const token = getAuthToken()
  if (token) {
    headers['Authorization'] = `Bearer ${token}`
  }
  const base = getApiBase()
  const res = await fetch(`${base}/media/verify`, {
    method: 'POST',
    headers,
    body: JSON.stringify(req),
    signal,
  })
  if (!res.ok) {
    throw new Error(await readError(res, `verify failed: ${res.status}`))
  }
  return unwrap<VerifyResponse>(await res.json())
}
