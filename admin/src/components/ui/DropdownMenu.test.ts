import { describe, it, expect, afterEach } from 'vitest'
import { mount, enableAutoUnmount, DOMWrapper } from '@vue/test-utils'
import DropdownMenu from './DropdownMenu.vue'
import type { MenuItem } from '@/lib/types'

enableAutoUnmount(afterEach)

const ITEMS: MenuItem[] = [
  { key: 'edit', label: '編輯' },
  { key: 'hide', label: '隱藏' },
  { key: 'del', label: '刪除', danger: true },
]

function mountMenu(items: MenuItem[] = ITEMS) {
  return mount(DropdownMenu, {
    props: { items, label: '更多' },
    attachTo: document.body,
  })
}

// The menu teleports to <body> (escapes table overflow clipping + sticky
// pinned-cell stacking), so all menu queries go through document.
const menu = () => document.querySelector<HTMLElement>('.ddown-menu')
const items = () =>
  Array.from(document.querySelectorAll<HTMLElement>('.ddown-item')).map(
    (el) => new DOMWrapper(el),
  )

describe('DropdownMenu', () => {
  it('opens on trigger click and exposes a menu with items', async () => {
    const wrapper = mountMenu()
    expect(menu()).toBeNull()

    await wrapper.get('.ddown-trigger').trigger('click')
    expect(menu()!.getAttribute('role')).toBe('menu')
    expect(wrapper.get('.ddown-trigger').attributes('aria-expanded')).toBe('true')
    expect(items()).toHaveLength(3)
  })

  it('teleports the menu to <body> so scroll containers cannot clip it', async () => {
    const wrapper = mountMenu()
    await wrapper.get('.ddown-trigger').trigger('click')
    const m = menu()!
    // Regression guard: an in-row menu would sit inside .rtable-wrap's
    // overflow context and inside the pinned actions cell's stacking
    // context — both clip/cover it. Body-level + fixed defeats both.
    // (jsdom does not apply stylesheet rules, so assert the inline
    // coordinates the fixed positioning is driven by instead of
    // getComputedStyle.)
    expect(m.parentElement).toBe(document.body)
    expect(m.style.top).not.toBe('')
    expect(m.style.left).not.toBe('')
  })

  it('invokes the selected action and dismisses', async () => {
    const wrapper = mountMenu()
    await wrapper.get('.ddown-trigger').trigger('click')
    await items()[2].trigger('click')

    expect(wrapper.emitted('select')).toEqual([['del']])
    expect(menu()).toBeNull()
  })

  it('dismisses on Escape without invoking an action', async () => {
    const wrapper = mountMenu()
    await wrapper.get('.ddown-trigger').trigger('click')
    await wrapper.trigger('keydown', { key: 'Escape' })

    expect(menu()).toBeNull()
    expect(wrapper.emitted('select')).toBeUndefined()
  })

  it('dismisses on outside click without invoking an action', async () => {
    const wrapper = mountMenu()
    await wrapper.get('.ddown-trigger').trigger('click')
    document.body.dispatchEvent(new MouseEvent('mousedown', { bubbles: true }))
    await wrapper.vm.$nextTick()

    expect(menu()).toBeNull()
    expect(wrapper.emitted('select')).toBeUndefined()
  })

  it('stays open when the pointer lands inside the teleported menu', async () => {
    const wrapper = mountMenu()
    await wrapper.get('.ddown-trigger').trigger('click')
    // Regression guard: a mousedown on a menu item must not count as an
    // outside click — otherwise the menu unmounts before the click lands.
    items()[0].element.dispatchEvent(new MouseEvent('mousedown', { bubbles: true }))
    await wrapper.vm.$nextTick()
    expect(menu()).not.toBeNull()
  })

  it('does not invoke a disabled item', async () => {
    const wrapper = mountMenu([
      { key: 'ok', label: '可用' },
      { key: 'no', label: '缺權限', disabled: true, hint: '需要 staff.update' },
    ])
    await wrapper.get('.ddown-trigger').trigger('click')
    const its = items()
    expect(its[1].attributes('disabled')).toBeDefined()
    await its[1].trigger('click')
    expect(wrapper.emitted('select')).toBeUndefined()
    expect(menu()).not.toBeNull()
  })

  it('navigates items with arrow keys and activates with Enter', async () => {
    const wrapper = mountMenu()
    await wrapper.get('.ddown-trigger').trigger('click')
    await wrapper.trigger('keydown', { key: 'ArrowDown' })
    await wrapper.trigger('keydown', { key: 'Enter' })

    expect(wrapper.emitted('select')).toEqual([['hide']])
    expect(menu()).toBeNull()
  })

  it('marks checked items (filter-select usage)', async () => {
    const wrapper = mountMenu([
      { key: 'a', label: '全部', checked: false },
      { key: 'b', label: '已啟用', checked: true },
    ])
    await wrapper.get('.ddown-trigger').trigger('click')
    const its = items()
    expect(its[0].find('.ddown-check').exists()).toBe(false)
    expect(its[1].find('.ddown-check').exists()).toBe(true)
  })
})
