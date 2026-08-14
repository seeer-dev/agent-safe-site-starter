import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount, flushPromises, enableAutoUnmount } from '@vue/test-utils'
import { nextTick } from 'vue'
import { createPinia, setActivePinia } from 'pinia'
import { createRouter, createMemoryHistory } from 'vue-router'
import ResourceListPage from './ResourceListPage.vue'

// Mock the api-client so loadRows doesn't make real network calls.
vi.mock('@/lib/api-client', () => ({
  api: {
    get: vi.fn().mockResolvedValue({ products: [] }),
    post: vi.fn().mockResolvedValue({}),
    put: vi.fn().mockResolvedValue({}),
    patch: vi.fn().mockResolvedValue({}),
    del: vi.fn().mockResolvedValue({}),
  },
  ApiError: class ApiError extends Error {
    constructor(public readonly status: number, message: string) {
      super(message)
      this.name = 'ApiError'
    }
  },
}))

import { api, ApiError } from '@/lib/api-client'
const mockedApi = vi.mocked(api)

// ResourceListPage uses <script setup>, so its internal refs/functions
// are not on the public instance type. Cast to a typed helper for
// test-only access to internals.
type ResourceListVM = {
  openForm(i: number | null): void
  closeForm(): void
  saveForm(): Promise<void>
  formOpen: boolean
  formIsNew: boolean
  formRowIndex: number | null
  formTriggerEl: HTMLElement | null
  formData: Record<string, any>
  rows: Record<string, any>[]
  error: string | null
}

function vm(wrapper: ReturnType<typeof mount>): ResourceListVM {
  return (wrapper.vm as unknown) as ResourceListVM
}

function makeRouter(path: string) {
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/res/:resource', component: ResourceListPage },
      { path: '/:pathMatch(.*)*', component: { template: '<div/>' } },
    ],
  })
}

async function mountProductsPage() {
  const pinia = createPinia()
  setActivePinia(pinia)
  const router = makeRouter('/res/minimal-cart-products')
  await router.push('/res/minimal-cart-products')
  await router.isReady()

  const wrapper = mount(ResourceListPage, {
    global: {
      plugins: [pinia, router],
    },
  })
  await flushPromises()
  await nextTick()
  return { wrapper, router, pinia }
}

async function mountOrdersPage() {
  const pinia = createPinia()
  setActivePinia(pinia)
  const router = makeRouter('/res/minimal-cart-orders')
  await router.push('/res/minimal-cart-orders')
  await router.isReady()

  mockedApi.get.mockResolvedValue({ orders: [] })

  const wrapper = mount(ResourceListPage, {
    global: {
      plugins: [pinia, router],
    },
  })
  await flushPromises()
  await nextTick()
  return { wrapper, router, pinia }
}

describe('ResourceListPage form modal focus management', () => {
  enableAutoUnmount(afterEach)

  beforeEach(() => {
    mockedApi.get.mockResolvedValue({ products: [] })
  })

  afterEach(() => {
    vi.clearAllMocks()
    // Clean up teleported modal content from document.body
    document.body.innerHTML = ''
  })

  it('moves focus to the first editable field (SKU) when the form opens', async () => {
    const { wrapper } = await mountProductsPage()

    // The "新增商品" button should be present (createCap is twcommerce.create,
    // and the dev auth store starts with no caps — but the button is rendered
    // when canCreate is true; in test, auth.can() returns false by default,
    // so we need to check the disabled fallback or directly trigger openForm).
    //
    // Instead of relying on capability checks, we directly call the openForm
    // method via the component instance to test the focus behavior.
    //
    // First, set up a trigger element as if the user clicked a button.
    const triggerBtn = document.createElement('button')
    triggerBtn.textContent = '新增商品'
    document.body.appendChild(triggerBtn)
    triggerBtn.focus()
    expect(document.activeElement).toBe(triggerBtn)

    // Call openForm(null) to open the create form
    vm(wrapper).openForm(null)
    await flushPromises()
    await nextTick()
    await flushPromises()
    await nextTick()

    // The modal should be open with role=dialog
    const modal = document.querySelector('.modal[role="dialog"]')
    expect(modal).toBeTruthy()

    // Focus should have moved to the first editable input (SKU field)
    const activeEl = document.activeElement
    expect(activeEl).toBeInstanceOf(HTMLInputElement)
    // The SKU input should have placeholder "SKU" or be the first input in .fgrid
    const firstInput = document.querySelector('.fgrid input')
    expect(activeEl).toBe(firstInput)

    // Cleanup
    triggerBtn.remove()
  })

  it('restores focus to the trigger button after the form closes', async () => {
    const { wrapper } = await mountProductsPage()

    const triggerBtn = document.createElement('button')
    triggerBtn.textContent = '新增商品'
    document.body.appendChild(triggerBtn)
    triggerBtn.focus()
    expect(document.activeElement).toBe(triggerBtn)

    // Open the form
    vm(wrapper).openForm(null)
    await flushPromises()
    await nextTick()
    await flushPromises()
    await nextTick()

    // Focus should be in the modal
    expect(document.activeElement).not.toBe(triggerBtn)

    // Close the form
    vm(wrapper).closeForm()
    await flushPromises()
    await nextTick()
    await flushPromises()
    await nextTick()

    // Focus should be restored to the trigger button
    expect(document.activeElement).toBe(triggerBtn)

    // Cleanup
    triggerBtn.remove()
  })

  it('modal has ARIA semantics: role=dialog, aria-modal=true, aria-labelledby, tabindex=-1', async () => {
    const { wrapper } = await mountProductsPage()

    vm(wrapper).openForm(null)
    await flushPromises()
    await nextTick()
    await flushPromises()
    await nextTick()

    const modal = document.querySelector('.modal[role="dialog"]') as HTMLElement
    expect(modal).toBeTruthy()
    expect(modal.getAttribute('aria-modal')).toBe('true')
    expect(modal.getAttribute('tabindex')).toBe('-1')

    // aria-labelledby should point to the title element's id
    const labelledBy = modal.getAttribute('aria-labelledby')
    expect(labelledBy).toBeTruthy()

    const titleEl = modal.querySelector(`#${labelledBy}`)
    expect(titleEl).toBeTruthy()
    expect(titleEl?.textContent).toContain('商品')

    // The title id should be stable (non-empty string)
    expect(labelledBy).toMatch(/vue-id-|v-|aria-/i)
  })

  it('Escape key closes the form and restores trigger focus', async () => {
    const { wrapper } = await mountProductsPage()

    const triggerBtn = document.createElement('button')
    triggerBtn.textContent = '新增商品'
    document.body.appendChild(triggerBtn)
    triggerBtn.focus()
    expect(document.activeElement).toBe(triggerBtn)

    vm(wrapper).openForm(null)
    await flushPromises()
    await nextTick()
    await flushPromises()
    await nextTick()

    // Modal should be open
    expect(vm(wrapper).formOpen).toBe(true)
    // formTriggerEl should have captured the trigger button
    expect(vm(wrapper).formTriggerEl).toBe(triggerBtn)

    // Press Escape — Modal.vue listens on window keydown
    const escapeEvent = new KeyboardEvent('keydown', { key: 'Escape', bubbles: true })
    window.dispatchEvent(escapeEvent)
    await flushPromises()
    await nextTick()
    await flushPromises()
    await nextTick()

    // Form should be closed (requestCloseForm -> closeForm since form is not dirty)
    expect(vm(wrapper).formOpen).toBe(false)

    // Focus should be restored to the trigger button
    expect(document.activeElement).toBe(triggerBtn)

    triggerBtn.remove()
  })

  it('Escape respects dirty-confirm and does not close when user cancels', async () => {
    const { wrapper } = await mountProductsPage()

    const triggerBtn = document.createElement('button')
    triggerBtn.textContent = '新增商品'
    document.body.appendChild(triggerBtn)
    triggerBtn.focus()

    vm(wrapper).openForm(null)
    await flushPromises()
    await nextTick()
    await flushPromises()
    await nextTick()

    // Make the form dirty by setting a field value
    vm(wrapper).formData.sku = 'TEST-001'
    await nextTick()

    // Mock window.confirm to return false (user cancels)
    const confirmSpy = vi.spyOn(window, 'confirm').mockReturnValue(false)

    // Press Escape
    const escapeEvent = new KeyboardEvent('keydown', { key: 'Escape', bubbles: true })
    window.dispatchEvent(escapeEvent)
    await flushPromises()
    await nextTick()

    // Modal should still be open (dirty-confirm blocked close)
    expect(document.querySelector('.modal[role="dialog"]')).toBeTruthy()
    expect(confirmSpy).toHaveBeenCalled()

    confirmSpy.mockRestore()
    triggerBtn.remove()
  })

  it('edit modal also gets focus and ARIA semantics', async () => {
    // Seed a product row via the API mock so loadRows populates rows
    mockedApi.get.mockResolvedValue({
      products: [
        { id: 'p1', sku: 'SKU-001', name: 'Test', category: 'apparel', price: 100, stock: 5, status: 'draft', updated_unix: 0 },
      ],
    })
    const { wrapper } = await mountProductsPage()

    // Verify the row was loaded
    expect(vm(wrapper).rows.length).toBe(1)

    const triggerBtn = document.createElement('button')
    triggerBtn.textContent = '編輯'
    document.body.appendChild(triggerBtn)
    triggerBtn.focus()

    // Open edit form for row 0
    vm(wrapper).openForm(0)
    await flushPromises()
    await nextTick()
    await flushPromises()
    await nextTick()

    // formIsNew should be false (formRowIndex=0, not null)
    expect(vm(wrapper).formRowIndex).toBe(0)
    expect(vm(wrapper).formIsNew).toBe(false)

    // Modal should be open with ARIA semantics
    const modals = document.querySelectorAll('.modal[role="dialog"]')
    expect(modals.length).toBe(1)
    const modal = modals[0] as HTMLElement
    expect(modal).toBeTruthy()
    expect(modal.getAttribute('aria-modal')).toBe('true')
    expect(modal.getAttribute('tabindex')).toBe('-1')

    // Focus should be on the first editable input (SKU)
    const firstInput = document.querySelector('.fgrid input')
    expect(document.activeElement).toBe(firstInput)

    // Title should say "編輯" (edit mode, not "新增" create mode)
    const labelledBy = modal.getAttribute('aria-labelledby')
    const titleEl = modal.querySelector(`#${labelledBy}`)
    // Log the actual title for debugging
    const actualTitle = titleEl?.textContent ?? ''
    expect(actualTitle).toContain('編輯')

    triggerBtn.remove()
  })

  it('number fields are seeded as strings to match Input modelValue contract', async () => {
    // Regression: API returns price/stock as numbers, but Input.vue
    // declares modelValue?: string. seedFieldValue must convert number
    // fields to string so Vue does not emit "Invalid prop type" warnings.
    mockedApi.get.mockResolvedValue({
      products: [
        { id: 'p1', sku: 'SKU-001', name: 'Test', category: 'apparel', price: 100, stock: 5, status: 'draft', updated_unix: 0 },
      ],
    })
    const { wrapper } = await mountProductsPage()

    // Capture Vue warnings via console.error spy
    const warnSpy = vi.spyOn(console, 'error').mockImplementation(() => {})

    vm(wrapper).openForm(0)
    await flushPromises()
    await nextTick()
    await flushPromises()
    await nextTick()

    // formData should have string values for number fields
    expect(vm(wrapper).formData.price).toBe('100')
    expect(vm(wrapper).formData.stock).toBe('5')
    expect(typeof vm(wrapper).formData.price).toBe('string')
    expect(typeof vm(wrapper).formData.stock).toBe('string')

    // No Vue "Invalid prop type check" warnings should have been emitted
    const warningCalls = warnSpy.mock.calls.filter(
      (args) => typeof args[0] === 'string' && args[0].includes('Invalid prop'),
    )
    expect(warningCalls).toHaveLength(0)

    warnSpy.mockRestore()
    vm(wrapper).closeForm()
    await flushPromises()
    await nextTick()
  })

  it('Tab from last focusable wraps to first focusable inside dialog', async () => {
    const { wrapper } = await mountProductsPage()

    vm(wrapper).openForm(null)
    await flushPromises()
    await nextTick()
    await flushPromises()
    await nextTick()

    const dialog = document.querySelector('.modal[role="dialog"]') as HTMLElement
    expect(dialog).toBeTruthy()

    // Collect focusable elements inside the dialog (same logic as Modal.vue)
    const FOCUSABLE = [
      'a[href]',
      'button:not([disabled])',
      'input:not([disabled])',
      'textarea:not([disabled])',
      'select:not([disabled])',
      '[tabindex]:not([tabindex="-1"])',
    ].join(',')
    function isVisible(el: HTMLElement): boolean {
      if (el.offsetParent !== null) return true
      if (el === document.activeElement) return true
      const style = window.getComputedStyle(el)
      return style.display !== 'none' && style.visibility !== 'hidden'
    }
    const focusable = Array.from(
      dialog.querySelectorAll<HTMLElement>(FOCUSABLE),
    ).filter(isVisible)
    expect(focusable.length).toBeGreaterThan(1)

    const first = focusable[0]
    const last = focusable[focusable.length - 1]

    // Focus the last focusable element
    last.focus()
    expect(document.activeElement).toBe(last)

    // Press Tab — should wrap to first
    const tabEvent = new KeyboardEvent('keydown', { key: 'Tab', bubbles: true })
    window.dispatchEvent(tabEvent)
    await nextTick()

    expect(document.activeElement).toBe(first)

    // Cleanup
    vm(wrapper).closeForm()
    await flushPromises()
    await nextTick()
  })

  it('Shift+Tab from first focusable wraps to last focusable inside dialog', async () => {
    const { wrapper } = await mountProductsPage()

    vm(wrapper).openForm(null)
    await flushPromises()
    await nextTick()
    await flushPromises()
    await nextTick()

    const dialog = document.querySelector('.modal[role="dialog"]') as HTMLElement
    expect(dialog).toBeTruthy()

    const FOCUSABLE = [
      'a[href]',
      'button:not([disabled])',
      'input:not([disabled])',
      'textarea:not([disabled])',
      'select:not([disabled])',
      '[tabindex]:not([tabindex="-1"])',
    ].join(',')
    function isVisible(el: HTMLElement): boolean {
      if (el.offsetParent !== null) return true
      if (el === document.activeElement) return true
      const style = window.getComputedStyle(el)
      return style.display !== 'none' && style.visibility !== 'hidden'
    }
    const focusable = Array.from(
      dialog.querySelectorAll<HTMLElement>(FOCUSABLE),
    ).filter(isVisible)
    expect(focusable.length).toBeGreaterThan(1)

    const first = focusable[0]
    const last = focusable[focusable.length - 1]

    // Focus the first focusable element
    first.focus()
    expect(document.activeElement).toBe(first)

    // Press Shift+Tab — should wrap to last
    const shiftTabEvent = new KeyboardEvent('keydown', {
      key: 'Tab',
      shiftKey: true,
      bubbles: true,
    })
    window.dispatchEvent(shiftTabEvent)
    await nextTick()

    expect(document.activeElement).toBe(last)

    // Cleanup
    vm(wrapper).closeForm()
    await flushPromises()
    await nextTick()
  })

  it('Tab trap is inactive when modal is closed (no global listener leak)', async () => {
    const { wrapper } = await mountProductsPage()

    // Modal is closed — no dialog in DOM
    expect(document.querySelector('.modal[role="dialog"]')).toBeFalsy()

    // Create a focusable element outside the modal
    const outsideInput = document.createElement('input')
    outsideInput.type = 'text'
    document.body.appendChild(outsideInput)
    outsideInput.focus()
    expect(document.activeElement).toBe(outsideInput)

    // Press Tab — should NOT be intercepted (no modal open)
    const tabEvent = new KeyboardEvent('keydown', { key: 'Tab', bubbles: true })
    const preventDefaultSpy = vi.spyOn(tabEvent, 'preventDefault')
    window.dispatchEvent(tabEvent)
    await nextTick()

    // preventDefault should not have been called (trap inactive)
    expect(preventDefaultSpy).not.toHaveBeenCalled()

    // Focus should remain on the outside input (not trapped)
    expect(document.activeElement).toBe(outsideInput)

    outsideInput.remove()
  })

  it('form labels are associated with inputs via for/id', async () => {
    const { wrapper } = await mountProductsPage()

    vm(wrapper).openForm(null)
    await flushPromises()
    await nextTick()
    await flushPromises()
    await nextTick()

    // Find all label elements in the form
    const labels = document.querySelectorAll('.fgrid label')
    expect(labels.length).toBeGreaterThan(0)

    // Each label (except media-uploader labels) should have a for attribute
    // pointing to an existing input id
    for (const label of Array.from(labels)) {
      const forAttr = label.getAttribute('for')
      const labelId = label.getAttribute('id')
      // Labels for media-uploader fields use id + aria-labelledby instead of for
      if (labelId && labelId.startsWith('label-field-')) {
        // Media uploader: label has id, no for; file input has aria-labelledby
        expect(forAttr).toBeNull()
        // The MediaUploader file input should have aria-labelledby pointing to this label id
        const fileInput = document.querySelector(`[aria-labelledby="${labelId}"]`)
        expect(fileInput).toBeTruthy()
      } else {
        // Regular field: label has for pointing to an input with that id
        expect(forAttr).toBeTruthy()
        expect(forAttr).toMatch(/^field-/)
        const target = document.getElementById(forAttr!)
        expect(target).toBeTruthy()
        expect(target!.tagName).toMatch(/^(INPUT|TEXTAREA|SELECT)$/)
      }
    }

    vm(wrapper).closeForm()
    await flushPromises()
    await nextTick()
  })

  it('read-only form labels omit for/id (no nonexistent control references)', async () => {
    const { wrapper } = await mountOrdersPage()

    vm(wrapper).openForm(null)
    await flushPromises()
    await nextTick()
    await flushPromises()
    await nextTick()

    // In a read-only form, all fields render as <p>, so labels must
    // NOT have for or id attributes pointing to nonexistent controls.
    const labels = document.querySelectorAll('.fgrid label')
    expect(labels.length).toBeGreaterThan(0)

    for (const label of Array.from(labels)) {
      expect(label.getAttribute('for')).toBeNull()
      expect(label.getAttribute('id')).toBeNull()
    }

    vm(wrapper).closeForm()
    await flushPromises()
    await nextTick()
  })
})

async function mountShippingPage(rows: Record<string, any>[] = []) {
  const pinia = createPinia()
  setActivePinia(pinia)
  const router = makeRouter('/res/minimal-cart-shipping')
  await router.push('/res/minimal-cart-shipping')
  await router.isReady()
  mockedApi.get.mockResolvedValue({ shipping_methods: rows })
  const wrapper = mount(ResourceListPage, {
    global: { plugins: [pinia, router] },
  })
  await flushPromises()
  await nextTick()
  return { wrapper, router, pinia }
}

describe('ResourceListPage generic shipping-method form contracts', () => {
  beforeEach(() => {
    mockedApi.get.mockResolvedValue({ shipping_methods: [] })
  })

  it('keeps method editable on create and read-only on edit', async () => {
    const row = {
      id: 'sm1',
      method: 'home_delivery',
      label: '宅配',
      description: '',
      fee: 120,
      free_threshold: 1500,
      enabled: true,
      sort_order: 1,
      version: 3,
      updated_unix: 1,
    }
    const { wrapper } = await mountShippingPage([row])

    vm(wrapper).openForm(null)
    await flushPromises()
    await nextTick()
    expect(document.querySelector('#field-method')).toBeTruthy()
    expect(document.querySelector('#field-method')?.tagName).toBe('INPUT')
    vm(wrapper).closeForm()
    await flushPromises()

    vm(wrapper).openForm(0)
    await flushPromises()
    await nextTick()
    expect(document.querySelector('#field-method')).toBeFalsy()
    const readonly = Array.from(document.querySelectorAll('.fgrid p.inp')).map((el) => el.textContent)
    expect(readonly.some((text) => text?.includes('home_delivery'))).toBe(true)
  })

  it('sends a blank nullable number as JSON null and injects expected_version from the row', async () => {
    const row = {
      id: 'sm1',
      method: 'home_delivery',
      label: '宅配',
      description: '',
      fee: 120,
      free_threshold: 1500,
      enabled: 'true',
      sort_order: 1,
      version: 4,
      updated_unix: 1,
    }
    const { wrapper } = await mountShippingPage([row])
    mockedApi.put.mockResolvedValue({})

    vm(wrapper).openForm(0)
    await flushPromises()
    await nextTick()
    vm(wrapper).formData.free_threshold = ''
    vm(wrapper).formData.label = '宅配到府'
    await vm(wrapper).saveForm()
    await flushPromises()

    expect(mockedApi.put).toHaveBeenCalled()
    const [, payload] = mockedApi.put.mock.calls[0]
    expect(payload).toMatchObject({
      free_threshold: null,
      expected_version: 4,
      label: '宅配到府',
    })
  })

  it('reloads the authoritative list after a 409 conflict', async () => {
    const row = {
      id: 'sm1',
      method: 'home_delivery',
      label: '宅配',
      description: '',
      fee: 120,
      free_threshold: null,
      enabled: 'true',
      sort_order: 1,
      version: 1,
      updated_unix: 1,
    }
    const { wrapper } = await mountShippingPage([row])
    mockedApi.put.mockRejectedValueOnce(new ApiError(409, 'stale version'))
    mockedApi.get.mockResolvedValue({
      shipping_methods: [{ ...row, version: 2, label: '宅配到府' }],
    })

    vm(wrapper).openForm(0)
    await flushPromises()
    await nextTick()
    vm(wrapper).formData.label = '新名稱'
    await vm(wrapper).saveForm()
    await flushPromises()

    expect(vm(wrapper).formOpen).toBe(false)
    expect(vm(wrapper).error).toContain('重新載入')
    expect(mockedApi.get.mock.calls.length).toBeGreaterThan(1)
    expect(vm(wrapper).rows[0].version).toBe(2)
  })
})
