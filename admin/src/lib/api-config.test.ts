import { describe, expect, it } from 'vitest'
import { getApiBase, resolveApiBase } from './api-config'

describe('resolveApiBase pure resolver', () => {
  describe('default fallback for empty or whitespace inputs', () => {
    it('returns /api when raw value is undefined', () => {
      expect(resolveApiBase(undefined)).toBe('/api')
    })

    it('returns /api when raw value is null', () => {
      expect(resolveApiBase(null)).toBe('/api')
    })

    it('returns /api when raw value is empty string', () => {
      expect(resolveApiBase('')).toBe('/api')
    })

    it('returns /api when raw value is whitespace only', () => {
      expect(resolveApiBase('   \t\n  ')).toBe('/api')
    })
  })

  describe('valid relative prefix resolution and trailing slash normalization', () => {
    it('returns /api for standard /api', () => {
      expect(resolveApiBase('/api')).toBe('/api')
    })

    it('normalizes single trailing slash on /api/', () => {
      expect(resolveApiBase('/api/')).toBe('/api')
    })

    it('normalizes multiple trailing slashes on /api///', () => {
      expect(resolveApiBase('/api///')).toBe('/api')
    })

    it('rejects root-relative paths other than /api and falls back to /api', () => {
      expect(resolveApiBase('/custom/api')).toBe('/api')
      expect(resolveApiBase('/custom/api/')).toBe('/api')
      expect(resolveApiBase('/api/v1')).toBe('/api')
      expect(resolveApiBase('/admin')).toBe('/api')
    })
  })

  describe('valid absolute prefix resolution', () => {
    it('resolves remote HTTPS prefix ending in /api', () => {
      expect(resolveApiBase('https://api.example.com/api')).toBe('https://api.example.com/api')
    })

    it('normalizes trailing slashes on remote HTTPS prefix', () => {
      expect(resolveApiBase('https://api.example.com/api/')).toBe('https://api.example.com/api')
      expect(resolveApiBase('https://api.example.com/api///')).toBe('https://api.example.com/api')
    })

    it('resolves remote HTTPS prefix with port and subpath ending in /api', () => {
      expect(resolveApiBase('https://api.example.com:8443/v1/api/')).toBe('https://api.example.com:8443/v1/api')
      expect(resolveApiBase('https://api.example.com/custom/api')).toBe('https://api.example.com/custom/api')
    })

    it('allows valid in-segment dots and encoded dots without dot-segments', () => {
      expect(resolveApiBase('https://api.example.com/v1%2ebeta/api')).toBe('https://api.example.com/v1%2ebeta/api')
      expect(resolveApiBase('https://api.example.com/v1..beta/api')).toBe('https://api.example.com/v1..beta/api')
      expect(resolveApiBase('https://api.example.com/v1.0.0/api')).toBe('https://api.example.com/v1.0.0/api')
    })

    it('allows @ in path segments when ending in /api while rejecting userinfo and @api endings', () => {
      expect(resolveApiBase('https://api.example.com/v1/@scope/api')).toBe('https://api.example.com/v1/@scope/api')
      expect(resolveApiBase('https://api.example.com/@scope/api')).toBe('https://api.example.com/@scope/api')
      expect(resolveApiBase('https://api.example.com/v1/@api')).toBe('/api')
      expect(resolveApiBase('https://api.example.com/@api')).toBe('/api')
      expect(resolveApiBase('https://admin:secret@api.example.com/api')).toBe('/api')
      expect(resolveApiBase('https://user@api.example.com/api')).toBe('/api')
    })

    it('allows HTTP on canonical localhost loopback', () => {
      expect(resolveApiBase('http://localhost:8080/api')).toBe('http://localhost:8080/api')
      expect(resolveApiBase('http://localhost:8080/api/')).toBe('http://localhost:8080/api')
    })

    it('allows HTTP on canonical 127.0.0.1 loopback', () => {
      expect(resolveApiBase('http://127.0.0.1:8080/api')).toBe('http://127.0.0.1:8080/api')
      expect(resolveApiBase('http://127.0.0.1:8080/api/')).toBe('http://127.0.0.1:8080/api')
    })

    it('allows HTTP on canonical [::1] IPv6 loopback', () => {
      expect(resolveApiBase('http://[::1]:8080/api')).toBe('http://[::1]:8080/api')
      expect(resolveApiBase('http://[::1]:8080/api/')).toBe('http://[::1]:8080/api')
    })

    it('allows HTTPS on loopback', () => {
      expect(resolveApiBase('https://localhost:8080/api')).toBe('https://localhost:8080/api')
    })
  })

  describe('security rejection and safe fallback', () => {
    it('rejects bare origins without /api path', () => {
      expect(resolveApiBase('https://api.example.com')).toBe('/api')
      expect(resolveApiBase('https://api.example.com/')).toBe('/api')
      expect(resolveApiBase('http://localhost:8080')).toBe('/api')
    })

    it('rejects remote non-loopback HTTP URLs', () => {
      expect(resolveApiBase('http://api.example.com/api')).toBe('/api')
      expect(resolveApiBase('http://192.168.1.50:8080/api')).toBe('/api')
      expect(resolveApiBase('http://10.0.0.1/api')).toBe('/api')
    })

    it('rejects non-canonical or alternative loopback spellings for HTTP', () => {
      expect(resolveApiBase('http://127.0.0.2:8080/api')).toBe('/api')
      expect(resolveApiBase('http://0177.0.0.1:8080/api')).toBe('/api')
      expect(resolveApiBase('http://localtest.me:8080/api')).toBe('/api')
    })

    it('rejects protocol-relative URLs', () => {
      expect(resolveApiBase('//api.example.com/api')).toBe('/api')
      expect(resolveApiBase('//localhost:8080/api')).toBe('/api')
    })

    it('rejects query strings', () => {
      expect(resolveApiBase('https://api.example.com/api?debug=true')).toBe('/api')
      expect(resolveApiBase('/api?foo=bar')).toBe('/api')
    })

    it('rejects fragments', () => {
      expect(resolveApiBase('https://api.example.com/api#section')).toBe('/api')
      expect(resolveApiBase('/api#top')).toBe('/api')
    })

    it('rejects literal dot-segments in relative and absolute paths', () => {
      expect(resolveApiBase('https://api.example.com/v1/../api')).toBe('/api')
      expect(resolveApiBase('https://api.example.com/v1/./api')).toBe('/api')
      expect(resolveApiBase('/v1/../api')).toBe('/api')
      expect(resolveApiBase('/./api')).toBe('/api')
      expect(resolveApiBase('/api/..')).toBe('/api')
      expect(resolveApiBase('/api/.')).toBe('/api')
    })

    it('rejects standalone percent-encoded dot-segments case-insensitively', () => {
      expect(resolveApiBase('https://api.example.com/v1/%2e%2e/api')).toBe('/api')
      expect(resolveApiBase('https://api.example.com/v1/%2E%2E/api')).toBe('/api')
      expect(resolveApiBase('https://api.example.com/v1/%2e./api')).toBe('/api')
      expect(resolveApiBase('https://api.example.com/v1/.%2e/api')).toBe('/api')
      expect(resolveApiBase('https://api.example.com/v1/%2e/api')).toBe('/api')
      expect(resolveApiBase('https://api.example.com/v1/%2E/api')).toBe('/api')
      expect(resolveApiBase('/%2e/api')).toBe('/api')
      expect(resolveApiBase('/%2e%2e/api')).toBe('/api')
      expect(resolveApiBase('/.%2e/api')).toBe('/api')
      expect(resolveApiBase('/%2e./api')).toBe('/api')
      expect(resolveApiBase('/%2E/api')).toBe('/api')
    })

    it('rejects non-root-relative paths', () => {
      expect(resolveApiBase('api')).toBe('/api')
      expect(resolveApiBase('api/endpoint')).toBe('/api')
      expect(resolveApiBase('custom/api')).toBe('/api')
    })

    it('rejects absolute paths not ending in /api', () => {
      expect(resolveApiBase('https://api.example.com/v1/admin')).toBe('/api')
      expect(resolveApiBase('https://api.example.com/endpoint')).toBe('/api')
    })

    it('rejects non-http(s) schemes', () => {
      expect(resolveApiBase('javascript:alert(1)')).toBe('/api')
      expect(resolveApiBase('ftp://api.example.com/api')).toBe('/api')
      expect(resolveApiBase('ws://localhost:8080/api')).toBe('/api')
    })

    it('rejects control characters and embedded whitespace', () => {
      expect(resolveApiBase('https://api.example.com/api\x00')).toBe('/api')
      expect(resolveApiBase('https://api.example.com/ api')).toBe('/api')
      expect(resolveApiBase('/api\n/test')).toBe('/api')
    })
  })

  describe('getApiBase helper', () => {
    it('returns /api by default', () => {
      expect(getApiBase()).toBe('/api')
    })
  })
})
