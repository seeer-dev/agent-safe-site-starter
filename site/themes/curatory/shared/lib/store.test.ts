// REQ-005 / AC-006: 訂單查詢碼不得持久化於 localStorage；舊版含 token 的
// 紀錄在載入時剝除並回寫；剛成立的訂單憑證只留在 sessionStorage。
import { beforeEach, describe, expect, it } from 'vitest'

class MemoryStorage {
  private m = new Map<string, string>()
  getItem(k: string) { return this.m.get(k) ?? null }
  setItem(k: string, v: string) { this.m.set(k, String(v)) }
  removeItem(k: string) { this.m.delete(k) }
  clear() { this.m.clear() }
  get size() { return this.m.size }
}

const local = new MemoryStorage()
const session = new MemoryStorage()

Object.defineProperty(globalThis, 'localStorage', { value: local, configurable: true })
Object.defineProperty(globalThis, 'sessionStorage', { value: session, configurable: true })

const { loadRecentOrders, rememberOrder } = await import('./store')

describe('recent orders storage hardening', () => {
  beforeEach(() => {
    local.clear()
    session.clear()
  })

  it('strips legacy token fields on load and re-saves de-identified list', () => {
    local.setItem('curatory_recent_orders', JSON.stringify([
      { orderId: 'TW-1', token: 'SECRET-A', date: '2026-01-01T00:00:00Z' },
      { orderId: 'TW-2', token: 'SECRET-B', date: '2026-01-02T00:00:00Z' },
    ]))

    const list = loadRecentOrders()
    expect(list).toEqual([
      { orderId: 'TW-1', date: '2026-01-01T00:00:00Z' },
      { orderId: 'TW-2', date: '2026-01-02T00:00:00Z' },
    ])
    expect(JSON.stringify(list)).not.toContain('SECRET')

    const resaved = local.getItem('curatory_recent_orders') ?? ''
    expect(resaved).not.toContain('SECRET-A')
    expect(resaved).not.toContain('SECRET-B')
    expect(resaved).not.toContain('token')
  })

  it('rememberOrder writes order metadata only — no token parameter exists', () => {
    rememberOrder('TW-9')
    const raw = local.getItem('curatory_recent_orders') ?? ''
    expect(raw).toContain('TW-9')
    expect(raw).not.toContain('token')
  })

  it('rememberOrder dedupes and caps history at five entries', () => {
    for (let i = 0; i < 7; i++) rememberOrder(`TW-${i}`)
    rememberOrder('TW-3') // re-remember an existing id
    const list = loadRecentOrders()
    expect(list).toHaveLength(5)
    expect(list[0].orderId).toBe('TW-3')
    expect(list.filter((r) => r.orderId === 'TW-3')).toHaveLength(1)
  })

  it('returns empty list for malformed payloads without throwing', () => {
    local.setItem('curatory_recent_orders', '{not json')
    expect(loadRecentOrders()).toEqual([])
    local.setItem('curatory_recent_orders', '"a string"')
    expect(loadRecentOrders()).toEqual([])
  })

  it('drops entries missing an orderId', () => {
    local.setItem('curatory_recent_orders', JSON.stringify([
      { token: 'SECRET-C', date: 'x' },
      { orderId: 'TW-ok', token: 'SECRET-D', date: 'y' },
    ]))
    const list = loadRecentOrders()
    expect(list).toEqual([{ orderId: 'TW-ok', date: 'y' }])
    expect(local.getItem('curatory_recent_orders')).not.toContain('SECRET')
  })
})
