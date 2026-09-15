import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import Skeleton from './Skeleton.vue'

describe('Skeleton', () => {
  it('renders structural bars with a status role and no data', () => {
    const wrapper = mount(Skeleton, { props: { rows: 4 } })
    expect(wrapper.attributes('role')).toBe('status')
    expect(wrapper.findAll('.skel')).toHaveLength(4)
    // Skeletons must never carry fixture or invented content.
    expect(wrapper.text()).toBe('')
  })

  it('renders row-shaped placeholders for the table variant', () => {
    const wrapper = mount(Skeleton, { props: { variant: 'table', rows: 5 } })
    expect(wrapper.findAll('.skel-row')).toHaveLength(5)
    expect(wrapper.findAll('.skel').length).toBe(25)
    expect(wrapper.text()).toBe('')
  })
})
