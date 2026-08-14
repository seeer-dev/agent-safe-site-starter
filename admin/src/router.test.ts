import { describe, it, expect } from 'vitest'
import { authCapabilityGuard } from './router'

const dashboard = {
  path: '/',
  name: 'dashboard',
  meta: { caps: ['twcommerce.read'] },
}

const resource = {
  path: '/res/products',
  name: 'resource',
  meta: { caps: ['twcommerce.read'] },
}

const open = {
  path: '/states',
  name: 'states',
  meta: { caps: [] as string[] },
}

describe('authCapabilityGuard', () => {
  it('does not redirect `/` to `/` while auth is connecting or unverified', () => {
    const noCaps = { status: 'connecting' as const, can: () => false }
    expect(authCapabilityGuard(dashboard, noCaps)).toBe(true)
    expect(authCapabilityGuard(dashboard, { status: 'unverified', can: () => false })).toBe(true)
    expect(authCapabilityGuard(dashboard, { status: 'failed', can: () => false })).toBe(true)
    expect(authCapabilityGuard(dashboard, { status: 'unavailable', can: () => false })).toBe(true)
  })

  it('does not self-redirect the dashboard after verification when caps are missing', () => {
    expect(authCapabilityGuard(dashboard, { status: 'verified', can: () => false })).toBe(true)
  })

  it('hides a protected non-root route only after Go-confirmed capabilities', () => {
    expect(authCapabilityGuard(resource, { status: 'connecting', can: () => false })).toBe(true)
    expect(authCapabilityGuard(resource, { status: 'verified', can: () => false })).toBe('/')
    expect(authCapabilityGuard(resource, {
      status: 'verified',
      can: (cap) => cap === 'twcommerce.read',
    })).toBe(true)
  })

  it('leaves routes with no required caps open', () => {
    expect(authCapabilityGuard(open, { status: 'verified', can: () => false })).toBe(true)
  })
})
