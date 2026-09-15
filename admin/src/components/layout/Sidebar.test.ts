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
    email: 'admin@example.com',
    role: 'owner',
    logout: vi.fn(),
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
      { path: '/settings', name: 'store-settings', component: { template: '<div/>' } },
      { path: '/roles', name: 'roles', component: { template: '<div/>' } },
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

function allHrefs(wrapper: ReturnType<typeof mount>): string[] {
  return wrapper.findAll('a').map((a) => a.attributes('href') ?? '')
}

/** v-show truthiness via the inline display style — jsdom's
 *  checkVisibility() does not reflect it reliably. */
function shown(el: { element: Element }): boolean {
  return (el.element as HTMLElement).style.display !== 'none'
}

describe('Sidebar', () => {
  beforeEach(() => {
    mockCaps.value = ['twcommerce.read', 'content.read', 'staff.read']
  })

  it('renders real hrefs for dashboard, resources, and states', async () => {
    const { wrapper } = await mountSidebar()
    const hrefs = allHrefs(wrapper)
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
    const hrefs = allHrefs(wrapper)
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
    // 訂單 is now a flat leaf — leaves carry `active`, children `is-current`.
    expect(wrapper.get('a[href="/res/minimal-cart-orders"]').classes()).toContain('active')
  })
})

describe('Sidebar grouped navigation', () => {
  beforeEach(() => {
    mockCaps.value = ['twcommerce.read', 'content.read', 'content.publish', 'staff.read']
  })

  it('renders named groups whose children expand and collapse on click', async () => {
    const { wrapper } = await mountSidebar()
    const groups = wrapper.findAll('.nav-item-wrapper')
    expect(groups.length).toBeGreaterThanOrEqual(5)

    const customerGroup = groups.find((g) => g.get('a.nav-item').text().includes('顧客'))
    expect(customerGroup).toBeTruthy()
    const children = customerGroup!.find('.nav-children')
    // Dashboard route has no active child, so the group starts collapsed.
    expect(shown(children!)).toBe(false)

    await customerGroup!.get('a.nav-item').trigger('click')
    expect(shown(children!)).toBe(true)
    expect(children!.text()).toContain('會員')
    expect(children!.text()).toContain('評論')

    await customerGroup!.get('a.nav-item').trigger('click')
    expect(shown(children!)).toBe(false)
  })

  it('auto-expands the group containing the active child and marks it current', async () => {
    const { wrapper, router } = await mountSidebar()
    await router.push('/res/minimal-cart-members')
    await wrapper.vm.$nextTick()

    const customerGroup = wrapper.findAll('.nav-item-wrapper').find((g) => g.get('a.nav-item').text().includes('顧客'))
    expect(customerGroup).toBeTruthy()
    expect(shown(customerGroup!.find('.nav-children')!)).toBe(true)
    expect(customerGroup!.get('a.nav-item').classes()).toContain('active')
    const current = customerGroup!.findAll('.nav-child').find((c) => c.classes().includes('is-current'))
    expect(current?.attributes('href')).toBe('/res/minimal-cart-members')
  })

  it('renders the labeled divider once between module and universal groups', async () => {
    const { wrapper } = await mountSidebar()
    const dividers = wrapper.findAll('.nav .divider')
    expect(dividers).toHaveLength(1)
    expect(dividers[0].text()).toBe('通用後台')
  })

  it('still emits the divider when the first universal group is fully gated', async () => {
    // content.* missing -> 內容 group dropped; 通知 (twcommerce.read) still
    // visible, so the 通用後台 divider must render before it, not vanish.
    mockCaps.value = ['twcommerce.read', 'staff.read']
    const { wrapper } = await mountSidebar()
    const dividers = wrapper.findAll('.nav .divider')
    expect(dividers).toHaveLength(1)
    const navText = wrapper.get('.nav').text()
    expect(navText.indexOf('通用後台')).toBeLessThan(navText.indexOf('通知'))
    expect(navText).not.toContain('內容')
  })

  it('drops a group entirely when every child is capability-gated', async () => {
    mockCaps.value = ['twcommerce.read']
    const { wrapper } = await mountSidebar()
    const navText = wrapper.get('.nav').text()
    // 系統 children all require staff.read -> group absent; 內容 children
    // require content.* -> absent.
    expect(navText).not.toContain('系統')
    expect(navText).not.toContain('內容')
    // Business entries still render: flat leaves and the 顧客 group, and the
    // 商店 group survives on its twcommerce.read children (優惠/付款/配送)
    // even though 商店設定 (content.read) drops out of it.
    expect(navText).toContain('顧客')
    expect(navText).toContain('商店')
    expect(navText).not.toContain('商店設定')
    expect(navText).not.toContain('人員')
  })

  it('hides a gated child inside an otherwise visible group', async () => {
    // content.read present, content.publish missing -> 前台內容 shows,
    // 公告文章 must not appear anywhere (inline children OR flyout).
    mockCaps.value = ['twcommerce.read', 'content.read', 'staff.read']
    const { wrapper } = await mountSidebar()
    const hrefs = allHrefs(wrapper)
    expect(hrefs).toContain('/res/minimal-cart-content')
    expect(hrefs).not.toContain('/res/articles')
    expect(wrapper.get('.nav').text()).not.toContain('公告文章')
  })

  it('keeps collapsed-mode flyout markup for every visible group', async () => {
    const { wrapper } = await mountSidebar()
    const groups = wrapper.findAll('.nav-item-wrapper')
    for (const g of groups) {
      const flyout = g.find('.flyout')
      const bridge = g.find('.flyout-bridge')
      expect(flyout.exists()).toBe(true)
      expect(bridge.exists()).toBe(true)
      // The flyout/bridge must be DIRECT children of the positioned
      // .nav-item-wrapper — the collapsed CSS anchors them with
      // position:absolute relative to that wrapper.
      expect(flyout.element.parentElement).toBe(g.element)
      expect(bridge.element.parentElement).toBe(g.element)
      expect(g.findAll('.flyout .nav-child').length).toBeGreaterThan(0)
      // Parent links opt OUT of the leaf tooltip via data-has-children;
      // without it the leaf ::after tooltip would overlap the flyout.
      expect(g.get('a.nav-item').attributes('data-has-children')).toBe('true')
    }
    // Flat leaves must NOT carry data-has-children (they get the tooltip).
    for (const a of wrapper.findAll('.nav > a')) {
      expect(a.attributes('data-has-children')).toBeUndefined()
    }
    // The flyout mirrors the gated inline children — no gated href leaks.
    mockCaps.value = ['twcommerce.read']
    const restricted = await mountSidebar()
    const flyoutHrefs = restricted.wrapper
      .findAll('.flyout .nav-child')
      .map((a) => a.attributes('href'))
    expect(flyoutHrefs).not.toContain('/res/staff')
    expect(flyoutHrefs).not.toContain('/roles')
  })

  it('renders the pinned user block with the current identity and logout', async () => {
    const { wrapper } = await mountSidebar()
    const block = wrapper.get('.sidefoot .userblock')
    expect(block.text()).toContain('admin@example.com')
    expect(block.text()).toContain('owner')
    expect(block.get('.avatar').text()).toBe('A')
    expect(wrapper.get('button[aria-label="登出"]')).toBeTruthy()
  })
})
