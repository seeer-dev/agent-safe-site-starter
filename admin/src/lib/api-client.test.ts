import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { api, ApiError } from './api-client'
import * as apiConfig from './api-config'
import * as tokenModule from '@/lib/auth/token'

describe('api-client request URL construction and HTTP methods', () => {
  const originalFetch = globalThis.fetch

  beforeEach(() => {
    vi.restoreAllMocks()
  })

  afterEach(() => {
    globalThis.fetch = originalFetch
  })

  it('uses /api default prefix when resolveApiBase returns /api', async () => {
    const fetchMock = vi.fn().mockImplementation(() =>
      Promise.resolve(
        new Response(JSON.stringify({ user_id: 'u1', role: 'owner' }), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        }),
      ),
    )
    globalThis.fetch = fetchMock

    const res = await api.get<{ user_id: string }>('/admin/me')

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/admin/me',
      expect.objectContaining({
        headers: expect.objectContaining({
          'Content-Type': 'application/json',
        }),
      }),
    )
    expect(res).toEqual({ user_id: 'u1', role: 'owner' })
  })

  it('uses custom prefix when getApiBase resolves to remote HTTPS prefix', async () => {
    vi.spyOn(apiConfig, 'getApiBase').mockReturnValue('https://api.example.com/api')

    const fetchMock = vi.fn().mockImplementation(() =>
      Promise.resolve(
        new Response(JSON.stringify({ products: [] }), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        }),
      ),
    )
    globalThis.fetch = fetchMock

    await api.get('/admin/products')

    expect(fetchMock).toHaveBeenCalledWith(
      'https://api.example.com/api/admin/products',
      expect.anything(),
    )
  })

  it('normalizes paths without leading slashes', async () => {
    vi.spyOn(apiConfig, 'getApiBase').mockReturnValue('https://api.example.com/api')

    const fetchMock = vi.fn().mockImplementation(() =>
      Promise.resolve(
        new Response(JSON.stringify({ ok: true }), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        }),
      ),
    )
    globalThis.fetch = fetchMock

    await api.get('admin/orders')

    expect(fetchMock).toHaveBeenCalledWith(
      'https://api.example.com/api/admin/orders',
      expect.anything(),
    )
  })

  it('attaches Authorization header when access token is present', async () => {
    vi.spyOn(tokenModule, 'getAccessToken').mockReturnValue('test-bearer-token')

    const fetchMock = vi.fn().mockImplementation(() =>
      Promise.resolve(
        new Response(JSON.stringify({ ok: true }), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        }),
      ),
    )
    globalThis.fetch = fetchMock

    await api.get('/admin/me')

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/admin/me',
      expect.objectContaining({
        headers: expect.objectContaining({
          Authorization: 'Bearer test-bearer-token',
        }),
      }),
    )
  })

  it('dispatches POST, PUT, PATCH, and DELETE with correct methods and bodies', async () => {
    const fetchMock = vi.fn().mockImplementation(() =>
      Promise.resolve(
        new Response(JSON.stringify({ success: true }), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        }),
      ),
    )
    globalThis.fetch = fetchMock

    await api.post('/admin/products', { name: 'Item 1' })
    expect(fetchMock).toHaveBeenLastCalledWith(
      '/api/admin/products',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ name: 'Item 1' }),
      }),
    )

    await api.put('/admin/products/p1', { name: 'Updated' })
    expect(fetchMock).toHaveBeenLastCalledWith(
      '/api/admin/products/p1',
      expect.objectContaining({
        method: 'PUT',
        body: JSON.stringify({ name: 'Updated' }),
      }),
    )

    await api.patch('/admin/products/p1/status', { status: 'active' })
    expect(fetchMock).toHaveBeenLastCalledWith(
      '/api/admin/products/p1/status',
      expect.objectContaining({
        method: 'PATCH',
        body: JSON.stringify({ status: 'active' }),
      }),
    )

    await api.del('/admin/products/p1')
    expect(fetchMock).toHaveBeenLastCalledWith(
      '/api/admin/products/p1',
      expect.objectContaining({
        method: 'DELETE',
      }),
    )
  })

  it('throws ApiError on non-ok HTTP responses', async () => {
    const fetchMock = vi.fn().mockImplementation(() =>
      Promise.resolve(
        new Response(JSON.stringify({ error: 'forbidden action' }), {
          status: 403,
          headers: { 'Content-Type': 'application/json' },
        }),
      ),
    )
    globalThis.fetch = fetchMock

    await expect(api.get('/admin/secret')).rejects.toThrow(ApiError)
  })
})
