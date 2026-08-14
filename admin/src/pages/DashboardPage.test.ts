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
    expect(kpis).toHaveLength(4)
    expect(wrapper.text()).toContain('待處理訂單')
    expect(wrapper.text()).toContain('待出貨')
    expect(wrapper.text()).toContain('退貨待審')
    expect(wrapper.text()).toContain('低庫存商品')

    const values = kpis.map((el) => el.find('b').text())
    expect(values).toEqual(['1', '1', '0', '1'])

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
})
