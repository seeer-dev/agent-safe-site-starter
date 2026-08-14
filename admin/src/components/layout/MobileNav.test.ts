import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, enableAutoUnmount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import { ref } from 'vue'
import MobileNav from './MobileNav.vue'

enableAutoUnmount(afterEach)

const mockCaps = ref<string[]>([])

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    can: (cap: string) => {
      if (!cap) return true
      return mockCaps.value.includes(cap)
    },
  }),
}))

async function mountNav() {
  const pinia = createPinia()
  setActivePinia(pinia)
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', name: 'dashboard', component: { template: '<div/>' } },
      { path: '/res/:resourceKey', name: 'resource', component: { template: '<div/>' } },
    ],
  })
  await router.push('/')
  await router.isReady()
  return mount(MobileNav, {
    global: { plugins: [pinia, router] },
  })
}

describe('MobileNav', () => {
  beforeEach(() => {
    mockCaps.value = ['twcommerce.read']
  })

  it('renders href-aware links for the mobile resource set', async () => {
    const wrapper = await mountNav()
    const hrefs = wrapper.findAll('a').map((a) => a.attributes('href'))
    expect(hrefs).toContain('/')
    expect(hrefs).toContain('/res/minimal-cart-products')
    expect(hrefs).toContain('/res/minimal-cart-orders')
    expect(hrefs).toContain('/res/minimal-cart-members')
    expect(wrapper.get('a[href="/"]').text()).toContain('總覽')
  })

  it('omits items the current principal cannot access', async () => {
    mockCaps.value = []
    const wrapper = await mountNav()
    expect(wrapper.findAll('a')).toHaveLength(0)
  })
})
