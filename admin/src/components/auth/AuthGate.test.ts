import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, enableAutoUnmount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'

vi.mock('@/lib/auth/config', () => ({
  getAuthMode: () => 'dev',
  isAdminAuthReady: () => true,
  isSupabaseConfigured: () => false,
  getSupabaseURL: () => '',
  getSupabasePublishableKey: () => '',
}))

import AuthGate from './AuthGate.vue'
import { useAuthStore } from '@/stores/auth'
import { resetSessionAdapterForTests } from '@/lib/auth/session'

enableAutoUnmount(afterEach)

describe('AuthGate', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  afterEach(() => {
    resetSessionAdapterForTests()
  })

  it('shows a generic role=alert on the unverified form when only verifyError is set', async () => {
    const store = useAuthStore()
    store.status = 'unverified'
    store.signInError = ''
    store.verifyError = '無法驗證身分，請重新登入。'

    const wrapper = mount(AuthGate)
    const alert = wrapper.get('[role="alert"]')
    expect(alert.text()).toBe('無法驗證身分，請重新登入。')
    expect(wrapper.text()).not.toContain('unauthorized')
    expect(wrapper.text()).not.toContain('secret')
  })
})
