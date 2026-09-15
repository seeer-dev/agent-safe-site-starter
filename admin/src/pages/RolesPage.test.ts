import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import RolesPage from './RolesPage.vue'
import { ROLES } from '@/config/roles'

describe('RolesPage permission matrix', () => {
  const roles = Object.values(ROLES)
  const allCaps = [...new Set(roles.flatMap((r) => r.caps))].sort((a, b) => a.localeCompare(b))

  it('lists every configured role as a column', () => {
    const wrapper = mount(RolesPage)
    const headers = wrapper.findAll('thead th.role-col')
    expect(headers).toHaveLength(roles.length)
    for (const r of roles) {
      expect(wrapper.text()).toContain(r.label)
      expect(wrapper.text()).toContain(r.key)
    }
  })

  it('lists every configured capability as a row', () => {
    const wrapper = mount(RolesPage)
    const capCells = wrapper.findAll('tbody td.cap-col')
    expect(capCells).toHaveLength(allCaps.length)
    for (const cap of allCaps) {
      expect(wrapper.text()).toContain(cap)
    }
  })

  it('shows truthful allow/deny cells per role config', () => {
    const wrapper = mount(RolesPage)
    const rows = wrapper.findAll('tbody tr')
    // staff.update: owner has it, manager/readonly/nocontent do not.
    const staffUpdateRow = rows.find((r) => r.find('td.cap-col')?.text() === 'staff.update')
    expect(staffUpdateRow).toBeTruthy()
    const cells = staffUpdateRow!.findAll('td.cell')
    expect(cells[0].classes()).toContain('allow') // owner
    expect(cells[1].classes()).not.toContain('allow') // manager
    expect(cells[2].classes()).not.toContain('allow') // readonly
    expect(cells[3].classes()).not.toContain('allow') // nocontent
  })

  it('is read-only: no editing controls on the matrix', () => {
    const wrapper = mount(RolesPage)
    expect(wrapper.find('input[type="checkbox"]').exists()).toBe(false)
    expect(wrapper.find('select').exists()).toBe(false)
    expect(wrapper.find('.matrix button').exists()).toBe(false)
  })
})
