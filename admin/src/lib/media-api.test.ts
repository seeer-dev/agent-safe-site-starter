import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { presignUpload, uploadToR2, verifyUpload } from './media-api'
import * as apiConfig from './api-config'

describe('media-api request URL construction and operations', () => {
  const originalFetch = globalThis.fetch

  beforeEach(() => {
    vi.restoreAllMocks()
  })

  afterEach(() => {
    globalThis.fetch = originalFetch
  })

  describe('presignUpload', () => {
    it('uses /api default base prefix', async () => {
      const fetchMock = vi.fn().mockImplementation(() =>
        Promise.resolve(
          new Response(
            JSON.stringify({
              url: 'https://r2.example.com/upload/key1',
              method: 'PUT',
              key: 'temp-key-1',
              headers: { 'Content-Type': 'image/webp' },
            }),
            {
              status: 200,
              headers: { 'Content-Type': 'application/json' },
            },
          ),
        ),
      )
      globalThis.fetch = fetchMock

      const res = await presignUpload({
        filename: 'test.webp',
        content_type: 'image/webp',
        purpose: 'product_image',
      })

      expect(fetchMock).toHaveBeenCalledWith(
        '/api/media/presign',
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({
            filename: 'test.webp',
            content_type: 'image/webp',
            purpose: 'product_image',
          }),
        }),
      )
      expect(res.key).toBe('temp-key-1')
    })

    it('uses custom prefix when getApiBase resolves to remote HTTPS prefix', async () => {
      vi.spyOn(apiConfig, 'getApiBase').mockReturnValue('https://api.example.com/api')

      const fetchMock = vi.fn().mockImplementation(() =>
        Promise.resolve(
          new Response(
            JSON.stringify({
              url: 'https://r2.example.com/upload/key1',
              method: 'PUT',
              key: 'temp-key-1',
              headers: {},
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          ),
        ),
      )
      globalThis.fetch = fetchMock

      await presignUpload({
        filename: 'test.webp',
        content_type: 'image/webp',
        purpose: 'product_image',
      })

      expect(fetchMock).toHaveBeenCalledWith(
        'https://api.example.com/api/media/presign',
        expect.anything(),
      )
    })
  })

  describe('verifyUpload', () => {
    it('uses /api default base prefix', async () => {
      const fetchMock = vi.fn().mockImplementation(() =>
        Promise.resolve(
          new Response(
            JSON.stringify({
              key: 'verified/key1',
              content_type: 'image/webp',
              bytes: 1024,
              width: 200,
              height: 200,
            }),
            {
              status: 200,
              headers: { 'Content-Type': 'application/json' },
            },
          ),
        ),
      )
      globalThis.fetch = fetchMock

      const res = await verifyUpload({
        key: 'temp-key-1',
      })

      expect(fetchMock).toHaveBeenCalledWith(
        '/api/media/verify',
        expect.objectContaining({
          method: 'POST',
          body: JSON.stringify({
            key: 'temp-key-1',
          }),
        }),
      )
      expect(res.key).toBe('verified/key1')
    })

    it('uses custom prefix when getApiBase resolves to remote HTTPS prefix', async () => {
      vi.spyOn(apiConfig, 'getApiBase').mockReturnValue('https://api.example.com/api')

      const fetchMock = vi.fn().mockImplementation(() =>
        Promise.resolve(
          new Response(
            JSON.stringify({
              key: 'verified/key1',
              content_type: 'image/webp',
              bytes: 1024,
              width: 200,
              height: 200,
            }),
            { status: 200, headers: { 'Content-Type': 'application/json' } },
          ),
        ),
      )
      globalThis.fetch = fetchMock

      await verifyUpload({
        key: 'temp-key-1',
      })

      expect(fetchMock).toHaveBeenCalledWith(
        'https://api.example.com/api/media/verify',
        expect.anything(),
      )
    })
  })

  describe('uploadToR2', () => {
    it('uploads binary directly to the provided presigned URL', async () => {
      const fetchMock = vi.fn().mockImplementation(() =>
        Promise.resolve(new Response('', { status: 200 })),
      )
      globalThis.fetch = fetchMock

      const blob = new Blob(['sample-bytes'], { type: 'image/webp' })
      await uploadToR2('https://r2-upload.example.com/direct/put', blob, {
        'Content-Type': 'image/webp',
      })

      expect(fetchMock).toHaveBeenCalledWith(
        'https://r2-upload.example.com/direct/put',
        expect.objectContaining({
          method: 'PUT',
          headers: expect.objectContaining({
            'Content-Type': 'image/webp',
          }),
        }),
      )
    })
  })
})
