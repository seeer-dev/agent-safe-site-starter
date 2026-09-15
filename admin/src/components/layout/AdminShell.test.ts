import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, enableAutoUnmount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import { ref, h } from 'vue'
import AdminShell from './AdminShell.vue'
import { useLayoutStore } from '@/stores/layout'

enableAutoUnmount(afterEach)

const mockCaps = ref<string[]>([])

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    can: (cap: string) => {
      if (!cap) return true
      return mockCaps.value.includes(cap)
    },
    email: 'admin@example.com',
    role: 'owner',
    logout: vi.fn(),
  }),
}))

async function mountShell() {
  const pinia = createPinia()
  setActivePinia(pinia)
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', name: 'dashboard', component: { template: '<div/>' } },
      { path: '/res/:resourceKey', name: 'resource', component: { template: '<div/>' } },
      { path: '/settings', name: 'store-settings', component: { template: '<div/>' } },
      { path: '/roles', name: 'roles', component: { template: '<div/>' } },
      { path: '/states', name: 'states', component: { template: '<div/>' } },
    ],
  })
  await router.push('/')
  await router.isReady()
  const wrapper = mount(AdminShell, {
    global: { plugins: [pinia, router] },
    slots: { default: () => h('div', { class: 'slot-content' }) },
  })
  const layout = useLayoutStore()
  return { wrapper, router, layout }
}

describe('AdminShell', () => {
  beforeEach(() => {
    mockCaps.value = ['twcommerce.read', 'content.read', 'staff.read']
  })

  it('renders the mobile drawer OUTSIDE .app so collapsed-state CSS cannot reach it', async () => {
    const { wrapper } = await mountShell()
    const app = wrapper.get('.app')
    const drawer = wrapper.get('.mobile-drawer')
    // Structural guard: .app[data-sidebar="collapsed"] selectors must never
    // match the drawer's Sidebar instance (labels would be hidden).
    expect(app.element.contains(drawer.element)).toBe(false)
    expect(drawer.element.parentElement).toBe(wrapper.element)
    expect(wrapper.get('.mobilenav').element.parentElement).toBe(wrapper.element)
  })

  it('opens the mobile drawer and closes it after child navigation', async () => {
    const { wrapper, router, layout } = await mountShell()
    expect(wrapper.get('.mobile-drawer').classes()).not.toContain('open')

    layout.openMobileDrawer()
    await wrapper.vm.$nextTick()
    const drawer = wrapper.get('.mobile-drawer')
    expect(drawer.classes()).toContain('open')

    // The drawer hosts a second Sidebar with the full grouped nav.
    const drawerNav = drawer.get('.sidebar .nav')
    expect(drawerNav.findAll('.nav-item-wrapper').length).toBeGreaterThanOrEqual(5)

    const link = drawerNav
      .findAll('.flyout .nav-child, .nav-children .nav-child, a[href^="/res/"]')
      .map((a) => a)
      .find((a) => a.attributes('href') === '/res/minimal-cart-orders')
    expect(link).toBeTruthy()
    await link!.trigger('click', { button: 0 })
    await flushPromises()
    await wrapper.vm.$nextTick()

    expect(router.currentRoute.value.path).toBe('/res/minimal-cart-orders')
    expect(layout.mobileDrawerOpen).toBe(false)
    expect(wrapper.get('.mobile-drawer').classes()).not.toContain('open')
  })

  it('keeps capability-gated destinations out of the drawer nav', async () => {
    mockCaps.value = ['twcommerce.read']
    const { wrapper, layout } = await mountShell()
    layout.openMobileDrawer()
    await wrapper.vm.$nextTick()
    const hrefs = wrapper
      .get('.mobile-drawer .sidebar')
      .findAll('a')
      .map((a) => a.attributes('href'))
    expect(hrefs).toContain('/res/minimal-cart-products')
    expect(hrefs).not.toContain('/res/staff')
    expect(hrefs).not.toContain('/roles')
  })
})
