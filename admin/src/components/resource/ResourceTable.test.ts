import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises, enableAutoUnmount } from '@vue/test-utils'
import { nextTick, ref } from 'vue'
import { createPinia, setActivePinia } from 'pinia'
import ResourceTable from '@/components/resource/ResourceTable.vue'
import type { ResourceDef } from '@/lib/types'

// enableAutoUnmount is file-scoped — call once at top level, not per-describe.
enableAutoUnmount(afterEach)

// The auth store is mocked so we can control which capabilities the
// current principal holds. This lets us test the all-of capability gate
// (allCaps) on RowAction — specifically the restock action which requires
// BOTH orders.returns AND inventory.adjust.
const mockCaps = ref<string[]>([])

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    caps: mockCaps,
    can: (cap: string) => {
      if (!cap) return true
      return mockCaps.value.includes(cap)
    },
    canAll: (...required: string[]) => required.every((c) => mockCaps.value.includes(c)),
  }),
}))

// Minimal resource definition that includes a restock action with allCaps
// and a simple status action with a single cap (for backward compat).
function makeResource(): ResourceDef {
  return {
    label: '訂單',
    desc: 'test',
    pageSize: 10,
    ops: { list: 'list', restock: 'restock' },
    updateCap: 'twcommerce.update',
    cols: [
      { k: 'id', l: '訂單' },
      { k: 'return_request_status', l: '退貨', r: 'badge' },
    ],
    rowActions: [
      {
        k: 'restock',
        l: '驗收回補',
        op: 'restock',
        allCaps: ['orders.returns', 'inventory.adjust'],
        expect: 'version',
        showWhen: 'return_request_status=received',
        variant: 'sec',
        restockItems: true,
      },
      {
        k: 'processing',
        l: '處理中',
        op: 'statusUpdate',
        cap: 'twcommerce.update',
        expect: 'version',
        showWhen: 'status=pending',
      },
    ],
    filters: [],
    form: { title: 'test', sections: [] },
    rows: [],
  }
}

function makeRows(): Record<string, any>[] {
  return [
    { id: 'ORD-1', status: 'delivered', return_request_status: 'received', version: 3 },
  ]
}

async function mountTable() {
  const pinia = createPinia()
  setActivePinia(pinia)
  const wrapper = mount(ResourceTable, {
    props: {
      resource: makeResource(),
      rows: makeRows(),
      selected: new Set<number>(),
    },
    global: {
      plugins: [pinia],
    },
  })
  await flushPromises()
  await nextTick()
  return wrapper
}

describe('RowAction allCaps capability gate', () => {
  beforeEach(() => {
    mockCaps.value = []
  })

  afterEach(() => {
    vi.clearAllMocks()
  })

  it('restock button is disabled when principal has NEITHER capability', async () => {
    const wrapper = await mountTable()
    const buttons = wrapper.findAll('button')
    const restockBtn = buttons.find((b) => b.text().includes('驗收回補'))
    expect(restockBtn).toBeTruthy()
    expect(restockBtn!.attributes('disabled')).toBeDefined()
  })

  it('restock button is disabled when principal has ONLY orders.returns (missing inventory.adjust)', async () => {
    mockCaps.value = ['orders.returns']
    const wrapper = await mountTable()
    const buttons = wrapper.findAll('button')
    const restockBtn = buttons.find((b) => b.text().includes('驗收回補'))
    expect(restockBtn).toBeTruthy()
    expect(restockBtn!.attributes('disabled')).toBeDefined()
  })

  it('restock button is disabled when principal has ONLY inventory.adjust (missing orders.returns)', async () => {
    mockCaps.value = ['inventory.adjust']
    const wrapper = await mountTable()
    const buttons = wrapper.findAll('button')
    const restockBtn = buttons.find((b) => b.text().includes('驗收回補'))
    expect(restockBtn).toBeTruthy()
    expect(restockBtn!.attributes('disabled')).toBeDefined()
  })

  it('restock button is ENABLED when principal has BOTH orders.returns AND inventory.adjust', async () => {
    mockCaps.value = ['orders.returns', 'inventory.adjust']
    const wrapper = await mountTable()
    const buttons = wrapper.findAll('button')
    const restockBtn = buttons.find((b) => b.text().includes('驗收回補'))
    expect(restockBtn).toBeTruthy()
    expect(restockBtn!.attributes('disabled')).toBeUndefined()
  })

  it('single-cap action (cap, not allCaps) still works: disabled without cap, enabled with cap', async () => {
    // The "處理中" action uses cap: 'twcommerce.update' (not allCaps).
    // It should be disabled without the cap and enabled with it.
    // This proves backward compatibility — allCaps does not break cap.
    mockCaps.value = []
    const wrapper = await mountTable()
    const buttons = wrapper.findAll('button')
    // "處理中" has showWhen: 'status=pending' but the row has status='delivered',
    // so it won't be visible. We need a row with status='pending'.
    await wrapper.setProps({
      rows: [{ id: 'ORD-2', status: 'pending', return_request_status: '', version: 1 }],
    })
    await nextTick()
    const buttonsAfter = wrapper.findAll('button')
    const processingBtn = buttonsAfter.find((b) => b.text().includes('處理中'))
    expect(processingBtn).toBeTruthy()
    expect(processingBtn!.attributes('disabled')).toBeDefined()

    // Now grant the cap — button should be enabled.
    mockCaps.value = ['twcommerce.update']
    await nextTick()
    const buttonsEnabled = wrapper.findAll('button')
    const processingBtnEnabled = buttonsEnabled.find((b) => b.text().includes('處理中'))
    expect(processingBtnEnabled).toBeTruthy()
    expect(processingBtnEnabled!.attributes('disabled')).toBeUndefined()
  })

  it('disabled restock button title lists the missing capabilities', async () => {
    mockCaps.value = ['orders.returns'] // missing inventory.adjust
    const wrapper = await mountTable()
    const buttons = wrapper.findAll('button')
    const restockBtn = buttons.find((b) => b.text().includes('驗收回補'))
    expect(restockBtn).toBeTruthy()
    const title = restockBtn!.attributes('title')
    expect(title).toContain('inventory.adjust')
    expect(title).not.toContain('orders.returns')
  })
})

describe('orders resource config restock allCaps', () => {
  it('restock action uses allCaps with both orders.returns and inventory.adjust', async () => {
    const { ordersResource } = await import('@/config/resources/orders')
    const restockAction = ordersResource.rowActions.find((a) => a.k === 'restock')
    expect(restockAction).toBeTruthy()
    expect(restockAction!.allCaps).toEqual(['orders.returns', 'inventory.adjust'])
    // Must NOT use the old single-cap gate (which only checked inventory.adjust).
    expect(restockAction!.cap).toBeUndefined()
  })

  it('other order actions still use single cap (backward compatibility)', async () => {
    const { ordersResource } = await import('@/config/resources/orders')
    const processingAction = ordersResource.rowActions.find((a) => a.k === 'processing')
    expect(processingAction).toBeTruthy()
    expect(processingAction!.cap).toBe('twcommerce.update')
    expect(processingAction!.allCaps).toBeUndefined()
  })
})

// Resource where an action sets BOTH cap and allCaps — the merged
// all-of list must require every unique capability from both fields.
// This catches the bug where allCaps non-empty used to shadow cap.
function makeCombinedResource(): ResourceDef {
  return {
    label: '訂單',
    desc: 'test combined',
    pageSize: 10,
    ops: { list: 'list' },
    updateCap: 'twcommerce.update',
    cols: [
      { k: 'id', l: '訂單' },
      { k: 'return_request_status', l: '退貨', r: 'badge' },
    ],
    rowActions: [
      {
        // cap + allCaps with NO overlap — merged list = [cap, ...allCaps]
        k: 'combined-distinct',
        l: '合併不相交',
        op: 'combinedA',
        cap: 'twcommerce.admin',
        allCaps: ['orders.returns', 'inventory.adjust'],
        showWhen: 'return_request_status=received',
      },
      {
        // cap + allCaps WITH overlap (cap duplicated in allCaps) —
        // deduped merged list must not double-require
        k: 'combined-overlap',
        l: '合併重複',
        op: 'combinedB',
        cap: 'orders.returns',
        allCaps: ['orders.returns', 'inventory.adjust'],
        showWhen: 'return_request_status=received',
      },
    ],
    filters: [],
    form: { title: 'test', sections: [] },
    rows: [],
  }
}

async function mountCombinedTable() {
  const pinia = createPinia()
  setActivePinia(pinia)
  const wrapper = mount(ResourceTable, {
    props: {
      resource: makeCombinedResource(),
      rows: makeRows(),
      selected: new Set<number>(),
    },
    global: {
      plugins: [pinia],
    },
  })
  await flushPromises()
  await nextTick()
  return wrapper
}

describe('RowAction combined cap + allCaps merge', () => {
  beforeEach(() => {
    mockCaps.value = []
  })

  afterEach(() => {
    vi.clearAllMocks()
  })

  it('combined-distinct: disabled when missing cap (has both allCaps)', async () => {
    mockCaps.value = ['orders.returns', 'inventory.adjust'] // missing twcommerce.admin
    const wrapper = await mountCombinedTable()
    const btn = wrapper.findAll('button').find((b) => b.text().includes('合併不相交'))
    expect(btn).toBeTruthy()
    expect(btn!.attributes('disabled')).toBeDefined()
    const title = btn!.attributes('title')
    expect(title).toContain('twcommerce.admin')
    // Held caps should NOT appear in the missing list
    expect(title).not.toContain('orders.returns')
    expect(title).not.toContain('inventory.adjust')
  })

  it('combined-distinct: disabled when missing one allCaps (has cap + other allCaps)', async () => {
    mockCaps.value = ['twcommerce.admin', 'orders.returns'] // missing inventory.adjust
    const wrapper = await mountCombinedTable()
    const btn = wrapper.findAll('button').find((b) => b.text().includes('合併不相交'))
    expect(btn).toBeTruthy()
    expect(btn!.attributes('disabled')).toBeDefined()
    const title = btn!.attributes('title')
    expect(title).toContain('inventory.adjust')
    expect(title).not.toContain('twcommerce.admin')
    expect(title).not.toContain('orders.returns')
  })

  it('combined-distinct: disabled when missing ALL (empty caps)', async () => {
    mockCaps.value = []
    const wrapper = await mountCombinedTable()
    const btn = wrapper.findAll('button').find((b) => b.text().includes('合併不相交'))
    expect(btn).toBeTruthy()
    expect(btn!.attributes('disabled')).toBeDefined()
    const title = btn!.attributes('title')
    // All three should be listed as missing
    expect(title).toContain('twcommerce.admin')
    expect(title).toContain('orders.returns')
    expect(title).toContain('inventory.adjust')
  })

  it('combined-distinct: enabled only when ALL three (cap + both allCaps) are held', async () => {
    mockCaps.value = ['twcommerce.admin', 'orders.returns', 'inventory.adjust']
    const wrapper = await mountCombinedTable()
    const btn = wrapper.findAll('button').find((b) => b.text().includes('合併不相交'))
    expect(btn).toBeTruthy()
    expect(btn!.attributes('disabled')).toBeUndefined()
  })

  it('combined-overlap: dedup — cap duplicated in allCaps does not double-require', async () => {
    // cap='orders.returns', allCaps=['orders.returns', 'inventory.adjust']
    // Merged deduped list = ['orders.returns', 'inventory.adjust']
    // Holding both should enable; holding only one should disable.
    mockCaps.value = ['orders.returns', 'inventory.adjust']
    const wrapper = await mountCombinedTable()
    const btn = wrapper.findAll('button').find((b) => b.text().includes('合併重複'))
    expect(btn).toBeTruthy()
    expect(btn!.attributes('disabled')).toBeUndefined()

    // Missing inventory.adjust → disabled, tooltip lists only inventory.adjust
    mockCaps.value = ['orders.returns']
    await nextTick()
    const btnDisabled = wrapper.findAll('button').find((b) => b.text().includes('合併重複'))
    expect(btnDisabled).toBeTruthy()
    expect(btnDisabled!.attributes('disabled')).toBeDefined()
    const title = btnDisabled!.attributes('title')
    expect(title).toContain('inventory.adjust')
    expect(title).not.toContain('orders.returns')
  })
})

// Resource with a number column to test null/undefined/zero rendering.
function makeNumberResource(): ResourceDef {
  return {
    label: '內容',
    desc: 'test number cols',
    pageSize: 10,
    ops: { list: 'list' },
    updateCap: 'twcommerce.update',
    cols: [
      { k: 'id', l: 'ID' },
      { k: 'approved_version', l: '已核可版', r: 'number' },
      { k: 'published_version', l: '發布版', r: 'number' },
    ],
    rowActions: [],
    filters: [],
    form: { title: 'test', sections: [] },
    rows: [],
  }
}

describe('cellContent null/undefined/zero rendering for number columns', () => {
  beforeEach(() => {
    mockCaps.value = []
  })

  afterEach(() => {
    vi.clearAllMocks()
  })

  async function mountNumberTable(rows: Record<string, any>[]) {
    const pinia = createPinia()
    setActivePinia(pinia)
    const wrapper = mount(ResourceTable, {
      props: {
        resource: makeNumberResource(),
        rows,
        selected: new Set<number>(),
      },
      global: {
        plugins: [pinia],
      },
    })
    await flushPromises()
    await nextTick()
    return wrapper
  }

  it('renders em dash for undefined number cell, not literal "undefined"', async () => {
    const wrapper = await mountNumberTable([
      { id: 'C-1', approved_version: undefined, published_version: undefined },
    ])
    const cells = wrapper.findAll('td')
    // cols: id, approved_version, published_version — no selectable column
    const approvedCell = cells[1]
    const publishedCell = cells[2]
    expect(approvedCell.text()).toBe('—')
    expect(publishedCell.text()).toBe('—')
    expect(approvedCell.text()).not.toContain('undefined')
    expect(publishedCell.text()).not.toContain('undefined')
  })

  it('renders em dash for null number cell, not literal "null"', async () => {
    const wrapper = await mountNumberTable([
      { id: 'C-2', approved_version: null, published_version: null },
    ])
    const cells = wrapper.findAll('td')
    const approvedCell = cells[1]
    const publishedCell = cells[2]
    expect(approvedCell.text()).toBe('—')
    expect(publishedCell.text()).toBe('—')
    expect(approvedCell.text()).not.toContain('null')
    expect(publishedCell.text()).not.toContain('null')
  })

  it('preserves numeric zero as "0", does not conflate with absent', async () => {
    const wrapper = await mountNumberTable([
      { id: 'C-3', approved_version: 0, published_version: 0 },
    ])
    const cells = wrapper.findAll('td')
    const approvedCell = cells[1]
    const publishedCell = cells[2]
    expect(approvedCell.text()).toBe('0')
    expect(publishedCell.text()).toBe('0')
  })

  it('renders real version numbers correctly', async () => {
    const wrapper = await mountNumberTable([
      { id: 'C-4', approved_version: 3, published_version: 2 },
    ])
    const cells = wrapper.findAll('td')
    expect(cells[1].text()).toBe('3')
    expect(cells[2].text()).toBe('2')
  })

  it('mixed row: present number, absent number, and zero coexist correctly', async () => {
    const wrapper = await mountNumberTable([
      { id: 'C-5', approved_version: 1, published_version: undefined },
      { id: 'C-6', approved_version: 0, published_version: 5 },
    ])
    const rows = wrapper.findAll('tbody tr')
    // First row: approved=1, published=undefined -> "—"
    const row1Cells = rows[0].findAll('td')
    expect(row1Cells[1].text()).toBe('1')
    expect(row1Cells[2].text()).toBe('—')
    // Second row: approved=0, published=5
    const row2Cells = rows[1].findAll('td')
    expect(row2Cells[1].text()).toBe('0')
    expect(row2Cells[2].text()).toBe('5')
  })
})

describe('ResourceTable sortable columns', () => {
  const sortableResource: ResourceDef = {
    label: '商品',
    desc: 'test',
    pageSize: 10,
    ops: { list: 'list' },
    cols: [
      { k: 'name', l: '名稱', sortable: true },
      { k: 'price', l: '售價', r: 'number', sortable: true },
      { k: 'sku', l: 'SKU' },
    ],
    rowActions: [],
    rows: [],
    filters: [],
    form: { title: '商品', sections: [] },
  }
  const rows = [
    { id: '1', name: '芭樂', price: 30, sku: 'B' },
    { id: '2', name: '蘋果', price: 10, sku: 'A' },
    { id: '3', name: '芒果', price: 20, sku: 'C' },
  ]

  function mountSortable(res: ResourceDef = sortableResource, r: any[] = rows) {
    const pinia = createPinia()
    setActivePinia(pinia)
    return mount(ResourceTable, {
      props: { resource: res, rows: r, selected: new Set<number>() },
      global: { plugins: [pinia] },
    })
  }

  // sortableResource has no bulkActions → no checkbox column:
  // ths = [name(sort), price(sort), sku(plain), 動作(plain)]; tds = [name, price, sku, actions]
  function nameCells(wrapper: ReturnType<typeof mount>) {
    return wrapper.findAll('tbody tr').map((tr) => tr.findAll('td')[0].text())
  }

  it('renders sort buttons only on sortable columns', () => {
    const wrapper = mountSortable()
    const ths = wrapper.findAll('thead th')
    expect(ths[0].find('.th-sort').exists()).toBe(true)
    expect(ths[1].find('.th-sort').exists()).toBe(true)
    expect(ths[2].find('.th-sort').exists()).toBe(false)
    expect(ths[2].attributes('aria-sort')).toBeUndefined()
  })

  it('first click sorts ascending, second click descending, indicator reflects direction', async () => {
    const wrapper = mountSortable()
    const btn = wrapper.findAll('thead th')[0].find('.th-sort')
    // zh-Hant collation order is ICU-dependent; compute expectations with the
    // same comparator and assert asc/desc are exact reverses of each other.
    const asc = [...rows.map((r) => r.name)].sort((a, b) => a.localeCompare(b, 'zh-Hant'))

    await btn.trigger('click')
    expect(nameCells(wrapper)).toEqual(asc)
    expect(wrapper.findAll('thead th')[0].attributes('aria-sort')).toBe('ascending')

    await btn.trigger('click')
    expect(nameCells(wrapper)).toEqual([...asc].reverse())
    expect(wrapper.findAll('thead th')[0].attributes('aria-sort')).toBe('descending')
  })

  it('numeric columns sort numerically', async () => {
    const wrapper = mountSortable()
    await wrapper.findAll('thead th')[1].find('.th-sort').trigger('click')
    const prices = wrapper.findAll('tbody tr').map((tr) => tr.findAll('td')[1].text())
    expect(prices).toEqual(['NT$10', 'NT$20', 'NT$30'])
  })

  it('unsorted and other columns keep server order; switching columns resets direction', async () => {
    const wrapper = mountSortable()
    expect(nameCells(wrapper)).toEqual(['芭樂', '蘋果', '芒果']) // server order

    await wrapper.findAll('thead th')[0].find('.th-sort').trigger('click') // name asc
    await wrapper.findAll('thead th')[0].find('.th-sort').trigger('click') // name desc
    await wrapper.findAll('thead th')[1].find('.th-sort').trigger('click') // price asc
    expect(wrapper.findAll('thead th')[0].attributes('aria-sort')).toBe('none')
    expect(wrapper.findAll('thead th')[1].attributes('aria-sort')).toBe('ascending')
  })

  it('sorting preserves original row indexes for row actions', async () => {
    const res: ResourceDef = {
      ...sortableResource,
      rowActions: [{ k: 'view', l: '檢視' }],
    }
    const wrapper = mountSortable(res)
    await wrapper.findAll('thead th')[0].find('.th-sort').trigger('click')
    // First displayed row's name determines its original index.
    const firstName = nameCells(wrapper)[0]
    const expectedIdx = rows.findIndex((r) => r.name === firstName)
    await wrapper.findAll('tbody tr')[0].find('button').trigger('click')
    expect(wrapper.emitted('rowAction')).toEqual([[expectedIdx, 'view']])
  })

  it('empty values sort after non-empty values regardless of direction', async () => {
    const r = [
      { id: '1', name: '乙', price: 1, sku: 'x' },
      { id: '2', name: '', price: 2, sku: 'y' },
      { id: '3', name: '甲', price: 3, sku: 'z' },
    ]
    const wrapper = mountSortable(sortableResource, r)
    const btn = wrapper.findAll('thead th')[0].find('.th-sort')
    const asc = ['乙', '甲'].sort((a, b) => a.localeCompare(b, 'zh-Hant'))
    await btn.trigger('click')
    expect(nameCells(wrapper)).toEqual([...asc, ''])
    await btn.trigger('click')
    expect(nameCells(wrapper)).toEqual([...[...asc].reverse(), ''])
  })
})

describe('ResourceTable dropdown row actions', () => {
  const menuResource: ResourceDef = {
    label: '商品',
    desc: 'test',
    pageSize: 10,
    ops: { list: 'list' },
    cols: [{ k: 'name', l: '名稱' }],
    rows: [],
    filters: [],
    rowActions: [
      { k: 'edit', l: '編輯', cap: 'twcommerce.update' },
      { k: 'publish', l: '上架', cap: 'twcommerce.update' },
      { k: 'archive', l: '封存', cap: 'twcommerce.delete', variant: 'danger' },
    ],
    form: { title: '商品', sections: [] },
  }
  const rows = [{ id: '1', name: 'T恤' }]

  function mountMenu(caps: string[] = ['twcommerce.update', 'twcommerce.delete']) {
    const pinia = createPinia()
    setActivePinia(pinia)
    mockCaps.value = caps
    return mount(ResourceTable, {
      props: { resource: menuResource, rows, selected: new Set<number>() },
      global: { plugins: [pinia] },
      attachTo: document.body,
    })
  }

  it('shows the first action inline and collapses the rest into a 更多 menu', async () => {
    const wrapper = mountMenu()
    const cell = wrapper.find('tbody tr td.num:last-child')
    expect(cell.findAll('button.btn').length).toBe(1)
    const trigger = cell.find('.ddown-trigger')
    expect(trigger.exists()).toBe(true)

    await trigger.trigger('click')
    const items = document.querySelectorAll('.ddown-item')
    expect(items.length).toBe(2)
    expect(items[0].textContent).toContain('上架')
    expect(items[1].textContent).toContain('封存')
    expect(items[1].classList.contains('danger')).toBe(true)
    wrapper.unmount()
  })

  it('selecting a menu item emits rowAction with the original index and key', async () => {
    const wrapper = mountMenu()
    await wrapper.find('.ddown-trigger').trigger('click')
    const items = document.querySelectorAll<HTMLElement>('.ddown-item')
    items[1].click() // 封存
    await nextTick()
    expect(wrapper.emitted('rowAction')).toEqual([[0, 'archive']])
    wrapper.unmount()
  })

  it('shows missing-capability actions disabled with a hint and does not emit', async () => {
    const wrapper = mountMenu(['twcommerce.update']) // missing twcommerce.delete
    await wrapper.find('.ddown-trigger').trigger('click')
    const items = document.querySelectorAll<HTMLElement>('.ddown-item')
    expect(items.length).toBe(2)
    const archive = items[1] as HTMLButtonElement
    expect(archive.disabled).toBe(true)
    expect(archive.textContent).toContain('需要 twcommerce.delete')
    archive.click()
    await nextTick()
    expect(wrapper.emitted('rowAction')).toBeUndefined()
    wrapper.unmount()
  })

  it('Escape dismisses the menu without invoking an action', async () => {
    const wrapper = mountMenu()
    await wrapper.find('.ddown-trigger').trigger('click')
    expect(document.querySelector('.ddown-menu')).toBeTruthy()
    document.dispatchEvent(new KeyboardEvent('keydown', { key: 'Escape' }))
    await nextTick()
    expect(document.querySelector('.ddown-menu')).toBeFalsy()
    expect(wrapper.emitted('rowAction')).toBeUndefined()
    wrapper.unmount()
  })
})

describe('ResourceTable horizontal scroll + pinned columns', () => {
  beforeEach(() => {
    mockCaps.value = ['twcommerce.read']
  })

  const pinnedResource: ResourceDef = {
    label: '訂單',
    desc: 'test',
    pageSize: 10,
    ops: { list: 'list' },
    pinActions: true,
    cols: [
      { k: 'id', l: '訂單', r: 'mono', pin: 'left' },
      { k: 'customer', l: '顧客' },
      { k: 'total', l: '金額', r: 'number' },
    ],
    rowActions: [{ k: 'view', l: '檢視' }],
    bulkActions: [{ k: 'export', l: '匯出', op: 'adminOrdersExport', cap: 'twcommerce.read' }],
    rows: [],
    filters: [],
    form: { title: '訂單', sections: [] },
  }
  const pinRows = [
    { id: 'TW-1', customer: 'A', total: 100 },
    { id: 'TW-2', customer: 'B', total: 200 },
  ]

  function mountPinned(res: ResourceDef = pinnedResource) {
    const pinia = createPinia()
    setActivePinia(pinia)
    return mount(ResourceTable, {
      props: { resource: res, rows: pinRows, selected: new Set<number>() },
      global: { plugins: [pinia] },
    })
  }

  it('wraps the table in a horizontal-scroll container', () => {
    const wrapper = mountPinned()
    const wrap = wrapper.get('.rtable-wrap')
    expect(wrap.find('table').exists()).toBe(true)
  })

  it('pins the identifier column left and the actions column right', () => {
    const wrapper = mountPinned()
    const ths = wrapper.findAll('thead th')
    // ths = [checkbox(pin), id(pin-l), customer, total, 動作(pin-r)]
    expect(ths[0].classes()).toContain('pin-l')          // checkbox auto-pins
    expect(ths[1].classes()).toContain('pin-l')          // id col
    expect(ths[1].attributes('style')).toContain('left')
    expect(ths[4].classes()).toContain('pin-r')          // actions
    expect(ths[4].attributes('style')).toContain('right: 0px')
    // Middle columns are not pinned.
    expect(ths[2].classes()).not.toContain('pin-l')
    expect(ths[2].classes()).not.toContain('pin-r')

    const firstRowTds = wrapper.findAll('tbody tr')[0].findAll('td')
    expect(firstRowTds[0].classes()).toContain('pin-l')
    expect(firstRowTds[1].classes()).toContain('pin-l')
    expect(firstRowTds[4].classes()).toContain('pin-r')
  })

  it('does not pin the checkbox column when no left-pinned column exists', () => {
    const res: ResourceDef = {
      ...pinnedResource,
      cols: [
        { k: 'id', l: '訂單', r: 'mono' },
        { k: 'customer', l: '顧客' },
      ],
    }
    const wrapper = mountPinned(res)
    const ths = wrapper.findAll('thead th')
    expect(ths[0].classes()).not.toContain('pin-l')       // checkbox stays static
    expect(ths[ths.length - 1].classes()).toContain('pin-r') // actions still pinned
  })

  it('leaves all cells unpinned when the resource declares no pins', () => {
    const res: ResourceDef = {
      ...pinnedResource,
      pinActions: false,
      cols: [
        { k: 'id', l: '訂單', r: 'mono' },
        { k: 'customer', l: '顧客' },
      ],
    }
    const wrapper = mountPinned(res)
    expect(wrapper.findAll('.pin-l')).toHaveLength(0)
    expect(wrapper.findAll('.pin-r')).toHaveLength(0)
  })
})
