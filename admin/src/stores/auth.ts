import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import type { Capability } from '@/lib/types'
import { api, ApiError } from '@/lib/api-client'
import { getAuthMode, isAdminAuthReady, isGoogleOAuthEnabled, isLineOAuthEnabled, type OAuthProvider } from '@/lib/auth/config'
import { getTestSessionAdapter, resolveSessionAdapter } from '@/lib/auth/session'
import type { AuthSession } from '@/lib/auth/session'
import { getAccessToken, setAccessToken } from '@/lib/auth/token'

export type AuthStatus =
  | 'connecting'
  | 'unverified'
  | 'verified'
  | 'forbidden'
  | 'failed'
  | 'unavailable'

interface MeResponse {
  user_id: string
  staff_id?: string
  email?: string
  role: string
  capabilities: Capability[]
}

export const useAuthStore = defineStore('auth', () => {
  const token = ref('')
  const status = ref<AuthStatus>('connecting')
  const serverCaps = ref<Capability[]>([])
  const serverRole = ref('')
  const serverUserID = ref('')
  const serverStaffID = ref('')
  const serverEmail = ref('')
  const verifyError = ref('')
  const signInError = ref('')
  const mode = computed(() => getAuthMode())
  const googleOAuthEnabled = computed(() => isGoogleOAuthEnabled())
  const lineOAuthEnabled = computed(() => isLineOAuthEnabled())

  const isAuthenticated = computed(() => status.value === 'verified')
  const caps = computed(() => serverCaps.value)
  const formAlert = computed(() => {
    if (status.value !== 'unverified') return ''
    return signInError.value || verifyError.value
  })

  let applyGeneration = 0
  let unsubscribe: (() => void) | null = null
  let signOutInFlight = false
  const invalidatedTokens = new Set<string>()

  function clearServerIdentity() {
    serverCaps.value = []
    serverRole.value = ''
    serverUserID.value = ''
    serverStaffID.value = ''
    serverEmail.value = ''
  }

  function clearSessionLocally() {
    token.value = ''
    setAccessToken('')
    clearServerIdentity()
    verifyError.value = ''
  }

  function isCurrent(generation: number, expectedToken?: string): boolean {
    if (generation !== applyGeneration) return false
    if (expectedToken && getAccessToken() !== expectedToken) return false
    return true
  }

  async function guardedSignOut(): Promise<void> {
    if (signOutInFlight) return
    signOutInFlight = true
    try {
      await resolveSessionAdapter().signOut()
    } finally {
      signOutInFlight = false
    }
  }

  async function invalidateUnauthorizedSession(badToken: string): Promise<void> {
    invalidatedTokens.add(badToken)
    applyGeneration += 1
    clearSessionLocally()
    status.value = 'unverified'
    const generic = '無法驗證身分，請重新登入。'
    signInError.value = generic
    verifyError.value = generic
    await guardedSignOut()
  }

  async function applySession(session: AuthSession | null) {
    if (session?.accessToken && invalidatedTokens.has(session.accessToken)) {
      return
    }
    if (signOutInFlight && !session) {
      return
    }
    const generation = ++applyGeneration
    if (!session?.accessToken) {
      if (!isCurrent(generation)) return
      clearSessionLocally()
      if (status.value !== 'unavailable') {
        status.value = 'unverified'
      }
      return
    }
    token.value = session.accessToken
    setAccessToken(session.accessToken)
    await verify(session.accessToken, generation)
  }

  async function initialize(): Promise<void> {
    if (!getTestSessionAdapter() && !isAdminAuthReady()) {
      clearSessionLocally()
      status.value = 'unavailable'
      return
    }
    status.value = 'connecting'
    signInError.value = ''
    try {
      const adapter = resolveSessionAdapter()
      unsubscribe?.()
      unsubscribe = adapter.subscribe((session) => {
        void applySession(session)
      })
      const session = await adapter.init()
      await applySession(session)
    } catch {
      clearSessionLocally()
      status.value = 'failed'
      verifyError.value = '目前無法驗證身分，請稍後再試。'
    }
  }

  async function verify(expectedToken?: string, generation?: number): Promise<boolean> {
    const tokenToVerify = expectedToken || getAccessToken() || token.value
    const gen = generation ?? applyGeneration
    if (!tokenToVerify) {
      if (!isCurrent(gen)) return false
      status.value = 'unverified'
      return false
    }
    if (!isCurrent(gen, tokenToVerify)) return false
    status.value = 'connecting'
    verifyError.value = ''
    try {
      const me = await api.get<MeResponse>('/admin/me')
      if (!isCurrent(gen, tokenToVerify)) return false
      serverUserID.value = me.user_id ?? ''
      serverStaffID.value = me.staff_id ?? ''
      serverEmail.value = me.email ?? ''
      serverRole.value = me.role ?? ''
      serverCaps.value = me.capabilities ?? []
      const isAdmin =
        serverCaps.value.length > 0 &&
        serverRole.value !== 'user' &&
        serverRole.value !== 'disabled'
      if (isAdmin) {
        status.value = 'verified'
        return true
      }
      status.value = 'forbidden'
      verifyError.value = serverRole.value === 'disabled'
        ? '此帳號已停用'
        : '此帳號無管理員權限'
      return false
    } catch (error) {
      if (!isCurrent(gen, tokenToVerify)) return false
      if (error instanceof ApiError && error.status === 401) {
        await invalidateUnauthorizedSession(tokenToVerify)
        return false
      }
      status.value = 'failed'
      serverCaps.value = []
      verifyError.value = '目前無法驗證身分，請稍後再試。'
      return false
    }
  }

  async function signIn(email: string, password: string): Promise<void> {
    signInError.value = ''
    status.value = 'connecting'
    try {
      const session = await resolveSessionAdapter().signInWithPassword(email, password)
      await applySession(session)
    } catch {
      status.value = 'unverified'
      signInError.value = '登入失敗，請確認資料後再試。'
    }
  }

  async function signInWithOAuth(provider: OAuthProvider): Promise<void> {
    signInError.value = ''
    status.value = 'connecting'
    try {
      await resolveSessionAdapter().signInWithOAuth(provider)
    } catch {
      status.value = 'unverified'
      signInError.value = '登入失敗，請確認資料後再試。'
    }
  }

  async function signInWithDevToken(devToken: string): Promise<void> {
    signInError.value = ''
    const adapter = resolveSessionAdapter()
    if (!adapter.signInWithDevToken) {
      signInError.value = '此環境不支援開發權杖登入。'
      return
    }
    status.value = 'connecting'
    try {
      const session = await adapter.signInWithDevToken(devToken)
      await applySession(session)
    } catch {
      status.value = 'unverified'
      signInError.value = '登入失敗，請確認資料後再試。'
    }
  }

  function logout() {
    const current = token.value || getAccessToken()
    if (current) invalidatedTokens.add(current)
    applyGeneration += 1
    clearSessionLocally()
    signInError.value = ''
    status.value = 'unverified'
    void guardedSignOut()
  }

  function can(cap: Capability): boolean {
    if (!cap) return true
    return caps.value.includes(cap)
  }

  function canAll(...required: Capability[]): boolean {
    return required.every((c) => caps.value.includes(c))
  }

  return {
    token,
    status,
    caps,
    role: serverRole,
    userID: serverUserID,
    staffID: serverStaffID,
    email: serverEmail,
    verifyError,
    signInError,
    formAlert,
    mode,
    googleOAuthEnabled,
    lineOAuthEnabled,
    isAuthenticated,
    initialize,
    verify,
    signIn,
    signInWithOAuth,
    signInWithDevToken,
    logout,
    can,
    canAll,
  }
})
