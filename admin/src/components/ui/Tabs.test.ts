import { describe, it, expect, afterEach } from 'vitest'
import { mount, enableAutoUnmount } from '@vue/test-utils'
import Tabs from './Tabs.vue'

enableAutoUnmount(afterEach)

const TABS = [
  { key: 'info', label: '資訊' },
  { key: 'banner', label: '公告列' },
  { key: 'limits', label: '門檻' },
]

function mountTabs(modelValue = 'info') {
  return mount(Tabs, {
    props: { tabs: TABS, modelValue },
    slots: {
      'panel-info': '<div class="p-info">資訊內容</div>',
      'panel-banner': '<div class="p-banner">公告內容</div>',
      'panel-limits': '<div class="p-limits">門檻內容</div>',
    },
    attachTo: document.body,
  })
}

describe('Tabs', () => {
  it('shows only the active panel and exposes tab semantics', () => {
    const wrapper = mountTabs('info')
    const tabs = wrapper.findAll('[role="tab"]')
    expect(tabs).toHaveLength(3)
    expect(tabs[0].attributes('aria-selected')).toBe('true')
    expect(tabs[0].attributes('tabindex')).toBe('0')
    expect(tabs[1].attributes('aria-selected')).toBe('false')
    expect(tabs[1].attributes('tabindex')).toBe('-1')

    const panels = wrapper.findAll('[role="tabpanel"]')
    expect(panels[0].attributes('hidden')).toBeUndefined()
    expect(panels[0].find('.p-info').exists()).toBe(true)
    expect(panels[1].attributes('hidden')).toBeDefined()
    expect(panels[1].attributes('aria-labelledby')).toBe('tab-banner')
  })

  it('switches panels on click', async () => {
    const wrapper = mountTabs()
    await wrapper.findAll('[role="tab"]')[1].trigger('click')
    expect(wrapper.emitted('update:modelValue')).toEqual([['banner']])
  })

  it('activates the next/previous tab on arrow keys', async () => {
    const wrapper = mountTabs('info')
    const tabs = wrapper.findAll('[role="tab"]')
    await tabs[0].trigger('keydown', { key: 'ArrowRight' })
    expect(wrapper.emitted('update:modelValue')).toEqual([['banner']])

    const wrap = mountTabs('info')
    await wrap.findAll('[role="tab"]')[0].trigger('keydown', { key: 'ArrowLeft' })
    // Wraps around to the last tab.
    expect(wrap.emitted('update:modelValue')).toEqual([['limits']])
  })

  it('does not re-emit when activating the current tab', async () => {
    const wrapper = mountTabs('info')
    await wrapper.findAll('[role="tab"]')[0].trigger('click')
    expect(wrapper.emitted('update:modelValue')).toBeUndefined()
  })
})
