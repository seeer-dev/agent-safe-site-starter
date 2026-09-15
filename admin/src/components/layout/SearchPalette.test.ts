import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises, enableAutoUnmount } from '@vue/test-utils'
import { nextTick, ref } from 'vue'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import SearchPalette from './SearchPalette.vue'
import { useLayoutStore } from '@/stores/layout'

const mockCaps = ref<string[]>([])

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    caps: mockCaps,
    can: (cap: string) => !cap || mockCaps.value.includes(cap),
    canAll: (...required: string[]) => required.every((c) => mockCaps.value.includes(c)),
  }),
}))

function makeRouter() {
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/', name: 'dashboard', component: { template: '<div/>' } },
      { path: '/settings', name: 'store-settings', component: { template: '<div/>' } },
      { path: '/roles', name: 'roles', component: { template: '<div/>' } },
      { path: '/states', name: 'states', component: { template: '<div/>' } },
      { path: '/res/:resourceKey', name: 'resource', component: { template: '<div/>' } },
    ],
  })
}

async function mountPalette(caps: string[]) {
  mockCaps.value = caps
  const pinia = createPinia()
  setActivePinia(pinia)
  const layout = useLayoutStore()
  layout.paletteOpen = true
  const router = makeRouter()
  await router.push('/')
  await router.isReady()
  const wrapper = mount(SearchPalette, {
    global: { plugins: [pinia, router] },
    attachTo: document.body,
  })
  await flushPromises()
  await nextTick()
  return { wrapper, router, layout }
}

describe('SearchPalette', () => {
  enableAutoUnmount(afterEach)

  beforeEach(() => {
    mockCaps.value = []
  })

  afterEach(() => {
    document.body.innerHTML = ''
  })

  it('indexes only capability-visible destinations', async () => {
    const { wrapper } = await mountPalette(['twcommerce.read', 'staff.read'])
    const labels = Array.from(document.querySelectorAll('.palette-item-label')).map((el) => el.textContent)
    expect(labels).toContain('商品')
    expect(labels).toContain('人員')
    expect(labels).toContain('角色權限')
    expect(labels).toContain('五狀態參考')
    expect(labels).not.toContain('公告文章') // needs content.publish
    expect(wrapper.find('.palette-empty').exists()).toBe(false)
  })

  it('never lists destinations the principal lacks caps for', async () => {
    await mountPalette(['twcommerce.read']) // no staff.read, no content.*
    const labels = Array.from(document.querySelectorAll('.palette-item-label')).map((el) => el.textContent)
    expect(labels).not.toContain('人員')
    expect(labels).not.toContain('角色權限')
    expect(labels).not.toContain('公告文章')
    expect(labels).not.toContain('商店設定')
  })

  it('filters by query and navigates on Enter', async () => {
    const { router, layout } = await mountPalette(['twcommerce.read'])
    const input = document.querySelector<HTMLInputElement>('.palette-input input')!
    input.value = '訂單'
    input.dispatchEvent(new Event('input'))
    await nextTick()
    const items = document.querySelectorAll('.palette-item')
    expect(items.length).toBe(1)
    expect(items[0].textContent).toContain('訂單')
    input.dispatchEvent(new KeyboardEvent('keydown', { key: 'Enter', bubbles: true }))
    await flushPromises()
    await nextTick()
    expect(router.currentRoute.value.path).toBe('/res/minimal-cart-orders')
    expect(layout.paletteOpen).toBe(false)
  })

  it('Escape closes the palette without navigating', async () => {
    const { router, layout } = await mountPalette(['twcommerce.read'])
    const input = document.querySelector<HTMLInputElement>('.palette-input input')!
    input.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape', bubbles: true }))
    await nextTick()
    expect(layout.paletteOpen).toBe(false)
    expect(router.currentRoute.value.path).toBe('/')
  })

  it('clicking the overlay backdrop closes the palette', async () => {
    const { wrapper, layout } = await mountPalette(['twcommerce.read'])
    await wrapper.find('.palette-overlay').trigger('click')
    await nextTick()
    expect(layout.paletteOpen).toBe(false)
  })
})
