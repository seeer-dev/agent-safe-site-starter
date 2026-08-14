import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, enableAutoUnmount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import { ref } from 'vue'
import Sidebar from './Sidebar.vue'

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

async function mountSidebar() {
  const pinia = createPinia()
  setActivePinia(pinia)
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', name: 'dashboard', component: { template: '<div/>' } },
      { path: '/res/:resourceKey', name: 'resource', component: { template: '<div/>' } },
      { path: '/states', name: 'states', component: { template: '<div/>' } },
    ],
  })
  await router.push('/')
  await router.isReady()
  const wrapper = mount(Sidebar, {
    global: { plugins: [pinia, router] },
  })
  return { wrapper, router }
}

describe('Sidebar', () => {
  beforeEach(() => {
    mockCaps.value = ['twcommerce.read', 'content.read', 'staff.read']
  })

  it('renders real hrefs for dashboard, resources, and states', async () => {
    const { wrapper } = await mountSidebar()
    const hrefs = wrapper.findAll('a').map((a) => a.attributes('href'))
    expect(hrefs).toContain('/')
    expect(hrefs).toContain('/res/minimal-cart-products')
    expect(hrefs).toContain('/res/minimal-cart-orders')
    expect(hrefs).toContain('/states')
    expect(wrapper.get('a[href="/"]').text()).toContain('總覽')
    expect(wrapper.get('a[href="/states"]').text()).toContain('五狀態參考')
  })

  it('hides capability-gated items and still exposes remaining links', async () => {
    mockCaps.value = ['twcommerce.read']
    const { wrapper } = await mountSidebar()
    const hrefs = wrapper.findAll('a').map((a) => a.attributes('href'))
    expect(hrefs).toContain('/')
    expect(hrefs).toContain('/res/minimal-cart-products')
    expect(hrefs).not.toContain('/res/minimal-cart-content')
    expect(hrefs).not.toContain('/res/staff')
    expect(hrefs).toContain('/states')
  })

  it('navigates via the href when a resource link is clicked', async () => {
    const { wrapper, router } = await mountSidebar()
    await router.push('/res/minimal-cart-orders')
    await wrapper.vm.$nextTick()
    expect(router.currentRoute.value.path).toBe('/res/minimal-cart-orders')
    expect(wrapper.get('a[href="/res/minimal-cart-orders"]').classes()).toContain('active')
  })
})
