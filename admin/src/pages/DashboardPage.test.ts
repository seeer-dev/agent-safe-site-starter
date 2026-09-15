import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises, enableAutoUnmount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import DashboardPage from './DashboardPage.vue'
import { useAuthStore } from '@/stores/auth'

vi.mock('@/lib/api-client', () => ({
  api: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    patch: vi.fn(),
    del: vi.fn(),
  },
}))

import { api } from '@/lib/api-client'
const mockedApi = vi.mocked(api)

enableAutoUnmount(afterEach)

describe('DashboardPage loaded KPI and module content', () => {
  beforeEach(() => {
    mockedApi.get.mockReset()
    mockedApi.get.mockImplementation(async (path: string) => {
      if (path.includes('/admin/orders')) {
        return {
          orders: [
            { id: 'TW-1', status: 'pending', return_request_status: '', customer_name: 'Alice', total: 560 },
            { id: 'TW-2', status: 'processing', return_request_status: '', customer_name: 'Bob', total: 300 },
          ],
        }
      }
      if (path.includes('/admin/products')) {
        return {
          products: [
            { sku: 'SKU-LOW', name: '低庫存商品', stock: 2, status: 'active' },
          ],
        }
      }
      if (path.includes('/admin/stats')) {
        return { revenue: 4280, pending_comments: 1, orders: 2, products: 1 }
      }
      if (path.includes('/admin/comments')) {
        return {
          comments: [
            { id: 'c1', status: 'pending', nickname: 'Cat', content: '待審評論內容' },
          ],
        }
      }
      return {}
    })
  })

  afterEach(() => {
    vi.clearAllMocks()
  })

  async function mountDashboard() {
    const pinia = createPinia()
    setActivePinia(pinia)
    const store = useAuthStore()
    store.status = 'verified'
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [
        { path: '/', component: DashboardPage },
        { path: '/res/:resource', component: { template: '<div/>' } },
      ],
    })
    await router.push('/')
    await router.isReady()
    const wrapper = mount(DashboardPage, {
      global: { plugins: [pinia, router] },
    })
    await flushPromises()
    await nextTick()
    return wrapper
  }

  it('renders live KPI values and module rows after orders/products resolve', async () => {
    const wrapper = await mountDashboard()

    expect(mockedApi.get).toHaveBeenCalledWith('/admin/orders')
    expect(mockedApi.get).toHaveBeenCalledWith('/admin/products')

    const kpis = wrapper.findAll('.kpi')
    expect(kpis).toHaveLength(6)
    expect(wrapper.text()).toContain('待處理訂單')
    expect(wrapper.text()).toContain('待出貨')
    expect(wrapper.text()).toContain('退貨待審')
    expect(wrapper.text()).toContain('低庫存商品')
    expect(wrapper.text()).toContain('總營收')
    expect(wrapper.text()).toContain('待審評論')

    const values = kpis.map((el) => el.find('b').text())
    expect(values).toEqual(['1', '1', '0', '1', '4280', '1'])

    expect(wrapper.text()).toContain('TW-1')
    expect(wrapper.text()).toContain('開始處理')
    expect(wrapper.text()).toContain('twcommerce')
    expect(wrapper.text()).toContain('staff')
    expect(wrapper.text()).toContain('商品 · 訂單 · 會員 · 優惠 · 付款方式')
    // The original bug wrapped KPIs/panels in a nested native <template>
    // (no class), which keeps its children inert. A classed selector
    // would miss that wrapper.
    expect(wrapper.find('.pagehd + template').exists()).toBe(false)
    expect(wrapper.find('template').exists()).toBe(false)
  })

  it('KPI cards click through to their resource', async () => {
    const wrapper = await mountDashboard()
    const router = (wrapper.vm as any).$router
    const spy = vi.spyOn(router, 'push')
    await wrapper.findAll('.kpi')[0].trigger('click') // 待處理訂單
    expect(spy).toHaveBeenCalledWith('/res/minimal-cart-orders')
    await wrapper.findAll('.kpi')[3].trigger('click') // 低庫存商品
    expect(spy).toHaveBeenCalledWith('/res/minimal-cart-products')
  })

  it('shows skeleton placeholders while data is in flight (no fake values)', async () => {
    mockedApi.get.mockImplementation(() => new Promise(() => {})) // never resolves
    const pinia = createPinia()
    setActivePinia(pinia)
    const store = useAuthStore()
    store.status = 'verified'
    const router = createRouter({
      history: createMemoryHistory(),
      routes: [{ path: '/', component: DashboardPage }],
    })
    await router.push('/')
    await router.isReady()
    const wrapper = mount(DashboardPage, { global: { plugins: [pinia, router] } })
    await nextTick()
    expect(wrapper.find('.skel').exists()).toBe(true)
    expect(wrapper.text()).not.toContain('載入中')
    // No fabricated KPI values while loading
    expect(wrapper.findAll('.kpi b').map((el) => el.text())).toEqual([])
  })

  it('shows an explicit error state and no fabricated KPIs when all APIs fail', async () => {
    mockedApi.get.mockRejectedValue(new Error('network down'))
    const wrapper = await mountDashboard()
    expect(wrapper.text()).toContain('載入失敗')
    // KPI area is empty — nothing fabricated
    expect(wrapper.findAll('.kpi')).toHaveLength(0)
  })
})
