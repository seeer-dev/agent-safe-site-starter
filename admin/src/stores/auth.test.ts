import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { resetSessionAdapterForTests, setSessionAdapterForTests, type AuthSession, type SessionAdapter } from '@/lib/auth/session'
import { getAccessToken, setAccessToken } from '@/lib/auth/token'

function jsonResponse(body: unknown, status = 200): Response {
  const payload = JSON.stringify(body)
  return {
    ok: status >= 200 && status < 300,
    status,
    headers: {
      get(name: string) {
        return name.toLowerCase() === 'content-type' ? 'application/json' : null
      },
    },
    json: async () => body,
    text: async () => payload,
  } as Response
}

function createFakeAdapter(initial: AuthSession | null = null) {
  let session = initial
  const listeners = new Set<(next: AuthSession | null) => void>()
  const adapter: SessionAdapter & { emit(next: AuthSession | null): void } = {
    init: vi.fn(async () => session),
    subscribe: vi.fn((listener) => {
      listeners.add(listener)
      return () => listeners.delete(listener)
    }),
    signInWithPassword: vi.fn(async () => {
      session = {
        accessToken: 'signed-in-token',
        user: { id: 'user-1', email: 'staff@example.com' },
      }
      listeners.forEach((listener) => listener(session))
      return session
    }),
    signInWithOAuth: vi.fn(async () => {}),
    signInWithDevToken: vi.fn(async (token: string) => {
      session = {
        accessToken: token,
        user: { id: 'dev', email: null },
      }
      listeners.forEach((listener) => listener(session))
      return session
    }),
    signOut: vi.fn(async () => {
      session = null
      listeners.forEach((listener) => listener(null))
    }),
    emit(next: AuthSession | null) {
      session = next
      listeners.forEach((listener) => listener(session))
    },
  }
  return adapter
}

const staffMe = {
  user_id: 'user-1',
  staff_id: 'staff-1',
  email: 'staff@example.com',
  role: 'owner',
  capabilities: ['twcommerce.read', 'twcommerce.admin'],
}

const nonStaffMe = {
  user_id: 'user-2',
  role: 'user',
  capabilities: [],
}

describe('admin auth store', () => {
  let fetchMock: ReturnType<typeof vi.fn>
  let adapter: ReturnType<typeof createFakeAdapter>

  beforeEach(() => {
    localStorage.clear()
    setAccessToken('')
    setActivePinia(createPinia())
    adapter = createFakeAdapter()
    setSessionAdapterForTests(adapter)
    fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)
  })

  afterEach(() => {
    resetSessionAdapterForTests()
    setAccessToken('')
    localStorage.clear()
    vi.unstubAllGlobals()
    vi.restoreAllMocks()
  })

  async function loadStore() {
    const { useAuthStore } = await import('./auth')
    return useAuthStore()
  }

  it('initializes an existing session and verifies staff via /admin/me', async () => {
    adapter = createFakeAdapter({
      accessToken: 'existing-token',
      user: { id: 'user-1', email: 'staff@example.com' },
    })
    setSessionAdapterForTests(adapter)
    fetchMock.mockResolvedValue(jsonResponse(staffMe))

    const store = await loadStore()
    await store.initialize()

    expect(adapter.init).toHaveBeenCalled()
    expect(adapter.subscribe).toHaveBeenCalled()
    expect(fetchMock).toHaveBeenCalledWith('/api/admin/me', expect.objectContaining({
      headers: expect.objectContaining({
        Authorization: 'Bearer existing-token',
      }),
    }))
    expect(store.isAuthenticated).toBe(true)
    expect(store.status).toBe('verified')
    expect(store.caps).toEqual(staffMe.capabilities)
    expect(getAccessToken()).toBe('existing-token')
    expect(localStorage.getItem('admin_token')).toBeNull()
  })

  it('signs in, then verifies capabilities on the server', async () => {
    fetchMock.mockResolvedValue(jsonResponse(staffMe))
    const store = await loadStore()
    await store.signIn('staff@example.com', 'password')

    expect(adapter.signInWithPassword).toHaveBeenCalledWith('staff@example.com', 'password')
    expect(fetchMock).toHaveBeenCalledWith('/api/admin/me', expect.objectContaining({
      headers: expect.objectContaining({
        Authorization: 'Bearer signed-in-token',
      }),
    }))
    expect(store.isAuthenticated).toBe(true)
    expect(store.email).toBe('staff@example.com')
    expect(localStorage.getItem('admin_token')).toBeNull()
  })

  it('re-verifies /admin/me when the session token refreshes', async () => {
    adapter = createFakeAdapter({
      accessToken: 'initial-token',
      user: { id: 'user-1', email: 'staff@example.com' },
    })
    setSessionAdapterForTests(adapter)
    fetchMock.mockResolvedValue(jsonResponse(staffMe))

    const store = await loadStore()
    await store.initialize()
    expect(fetchMock).toHaveBeenCalledTimes(1)

    adapter.emit({
      accessToken: 'refreshed-token',
      user: { id: 'user-1', email: 'staff@example.com' },
    })
    await vi.waitFor(() => {
      expect(fetchMock).toHaveBeenCalledTimes(2)
      expect(store.isAuthenticated).toBe(true)
    })
    expect(fetchMock).toHaveBeenLastCalledWith('/api/admin/me', expect.objectContaining({
      headers: expect.objectContaining({
        Authorization: 'Bearer refreshed-token',
      }),
    }))
    expect(getAccessToken()).toBe('refreshed-token')
  })

  it('denies a valid non-staff session without exposing capabilities', async () => {
    adapter = createFakeAdapter({
      accessToken: 'customer-token',
      user: { id: 'user-2', email: 'customer@example.com' },
    })
    setSessionAdapterForTests(adapter)
    fetchMock.mockResolvedValue(jsonResponse(nonStaffMe))

    const store = await loadStore()
    await store.initialize()

    expect(store.isAuthenticated).toBe(false)
    expect(store.status).toBe('forbidden')
    expect(store.caps).toEqual([])
    expect(store.verifyError).toBe('此帳號無管理員權限')
    expect(store.can('twcommerce.read')).toBe(false)
  })

  it('clears the in-memory token immediately on sign-out', async () => {
    adapter = createFakeAdapter({
      accessToken: 'existing-token',
      user: { id: 'user-1', email: 'staff@example.com' },
    })
    setSessionAdapterForTests(adapter)
    fetchMock.mockResolvedValue(jsonResponse(staffMe))

    const store = await loadStore()
    await store.initialize()
    expect(store.isAuthenticated).toBe(true)

    store.logout()

    expect(store.token).toBe('')
    expect(getAccessToken()).toBe('')
    expect(store.isAuthenticated).toBe(false)
    expect(store.status).toBe('unverified')
    expect(adapter.signOut).toHaveBeenCalled()
    expect(localStorage.getItem('admin_token')).toBeNull()
  })

  it('shows unavailable when compile-time auth is not configured', async () => {
    setSessionAdapterForTests(null)
    const store = await loadStore()
    await store.initialize()
    expect(store.status).toBe('unavailable')
    expect(store.isAuthenticated).toBe(false)
    expect(getAccessToken()).toBe('')
  })

  it('lets the latest refresh win when /admin/me responses arrive out of order', async () => {
    adapter = createFakeAdapter({
      accessToken: 'token-a',
      user: { id: 'user-1', email: 'staff@example.com' },
    })
    setSessionAdapterForTests(adapter)

    const pending: Array<(value: Response) => void> = []
    fetchMock.mockImplementation(() => new Promise<Response>((resolve) => {
      pending.push(resolve)
    }))

    const store = await loadStore()
    const init = store.initialize()
    await vi.waitFor(() => expect(pending.length).toBe(1))

    adapter.emit({
      accessToken: 'token-b',
      user: { id: 'user-1', email: 'staff@example.com' },
    })
    await vi.waitFor(() => expect(pending.length).toBe(2))

    pending[1](jsonResponse({ ...staffMe, email: 'new@example.com', capabilities: ['twcommerce.read'] }))
    await vi.waitFor(() => {
      expect(store.isAuthenticated).toBe(true)
      expect(store.email).toBe('new@example.com')
    })

    pending[0](jsonResponse({ ...staffMe, email: 'stale@example.com', capabilities: ['twcommerce.admin'] }))
    await Promise.resolve()
    await init

    expect(store.email).toBe('new@example.com')
    expect(store.caps).toEqual(['twcommerce.read'])
    expect(getAccessToken()).toBe('token-b')
    expect(store.isAuthenticated).toBe(true)
  })

  it('ignores a late /admin/me success after logout', async () => {
    adapter = createFakeAdapter({
      accessToken: 'existing-token',
      user: { id: 'user-1', email: 'staff@example.com' },
    })
    setSessionAdapterForTests(adapter)

    let resolveMe!: (value: Response) => void
    fetchMock.mockImplementation(() => new Promise<Response>((resolve) => {
      resolveMe = resolve
    }))

    const store = await loadStore()
    const init = store.initialize()
    await vi.waitFor(() => expect(resolveMe).toBeTypeOf('function'))

    store.logout()
    expect(store.isAuthenticated).toBe(false)
    expect(getAccessToken()).toBe('')

    resolveMe(jsonResponse(staffMe))
    await init

    expect(store.isAuthenticated).toBe(false)
    expect(store.status).toBe('unverified')
    expect(store.caps).toEqual([])
    expect(getAccessToken()).toBe('')
  })

  it('signs out the real session on /admin/me 401 and ignores the stale token', async () => {
    adapter = createFakeAdapter({
      accessToken: 'existing-token',
      user: { id: 'user-1', email: 'staff@example.com' },
    })
    adapter.signOut = vi.fn(async () => {
      adapter.emit({
        accessToken: 'existing-token',
        user: { id: 'user-1', email: 'staff@example.com' },
      })
    })
    setSessionAdapterForTests(adapter)
    fetchMock.mockResolvedValue(jsonResponse({ error: 'unauthorized' }, 401))

    const store = await loadStore()
    await store.initialize()

    expect(store.isAuthenticated).toBe(false)
    expect(store.status).toBe('unverified')
    expect(store.token).toBe('')
    expect(getAccessToken()).toBe('')
    expect(adapter.signOut).toHaveBeenCalledTimes(1)
    expect(store.caps).toEqual([])
    expect(store.formAlert).toBe('無法驗證身分，請重新登入。')

    adapter.emit({
      accessToken: 'existing-token',
      user: { id: 'user-1', email: 'staff@example.com' },
    })
    await Promise.resolve()
    expect(store.isAuthenticated).toBe(false)
    expect(getAccessToken()).toBe('')
    expect(adapter.signOut).toHaveBeenCalledTimes(1)
  })

  it('shows a generic form alert after an invalid login 401 without leaking the backend body', async () => {
    fetchMock.mockResolvedValue(jsonResponse({ error: 'token is not valid: secret-detail' }, 401))
    const store = await loadStore()
    await store.signInWithDevToken('bad-token')

    expect(store.status).toBe('unverified')
    expect(store.isAuthenticated).toBe(false)
    expect(store.token).toBe('')
    expect(getAccessToken()).toBe('')
    expect(adapter.signOut).toHaveBeenCalled()
    expect(store.formAlert).toBe('無法驗證身分，請重新登入。')
    expect(store.formAlert).not.toContain('secret-detail')
    expect(store.formAlert).not.toContain('token is not valid')
    expect(store.signInError).toBe(store.formAlert)
  })

  it('still exposes formAlert from verifyError when signInError is empty', async () => {
    const store = await loadStore()
    store.status = 'unverified'
    store.signInError = ''
    store.verifyError = '無法驗證身分，請重新登入。'
    expect(store.formAlert).toBe('無法驗證身分，請重新登入。')
  })

  it('starts OAuth through the session adapter without storing a token', async () => {
    const store = await loadStore()
    await store.signInWithOAuth('google')
    expect(adapter.signInWithOAuth).toHaveBeenCalledWith('google')
    expect(getAccessToken()).toBe('')
    expect(localStorage.getItem('admin_token')).toBeNull()
  })

  it('restores the unverified form when OAuth initiation fails', async () => {
    adapter.signInWithOAuth = vi.fn(async () => {
      throw new Error('oauth_failed')
    })
    const store = await loadStore()
    await store.signInWithOAuth('custom:line')
    expect(store.status).toBe('unverified')
    expect(store.formAlert).toBe('登入失敗，請確認資料後再試。')
    expect(getAccessToken()).toBe('')
  })

  it('never writes admin_token to localStorage', async () => {
    const setItem = vi.spyOn(Storage.prototype, 'setItem')
    fetchMock.mockResolvedValue(jsonResponse(staffMe))
    const store = await loadStore()
    await store.signIn('staff@example.com', 'password')
    store.logout()

    const adminTokenWrites = setItem.mock.calls.filter((call) => call[0] === 'admin_token')
    expect(adminTokenWrites).toHaveLength(0)
    expect(localStorage.getItem('admin_token')).toBeNull()
  })

  it('enters failed state and preserves session when /admin/me returns 503', async () => {
    adapter = createFakeAdapter({
      accessToken: 'existing-token',
      user: { id: 'user-1', email: 'staff@example.com' },
    })
    setSessionAdapterForTests(adapter)
    fetchMock.mockResolvedValue(jsonResponse({ error: 'service unavailable' }, 503))

    const store = await loadStore()
    await store.initialize()

    expect(store.isAuthenticated).toBe(false)
    expect(store.status).toBe('failed')
    expect(store.caps).toEqual([])
    expect(store.token).toBe('existing-token')
    expect(getAccessToken()).toBe('existing-token')
    expect(adapter.signOut).not.toHaveBeenCalled()
    expect(store.verifyError).toBe('目前無法驗證身分，請稍後再試。')
    expect(store.verifyError).not.toContain('service unavailable')
  })
})
