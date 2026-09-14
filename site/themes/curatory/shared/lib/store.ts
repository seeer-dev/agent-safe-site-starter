// 前台全域狀態：bootstrap 資料快取、購物車 drawer、結帳優惠券、最近查單。
// Reference 用 React context + zustand；這裡用 module 層級 reactive
// singleton —— 所有 island 共享同一份狀態（同一 module graph）。

import { reactive } from 'vue'
import { apiGet } from './api'
import type { BootstrapData, ShippingMethodDTO } from './types'

// ─── Bootstrap 資料 ─────────────────────────────────────────

interface BootstrapState {
  data: BootstrapData | null
  loading: boolean
  error: boolean
}

export const bootstrap = reactive<BootstrapState>({
  data: null,
  loading: false,
  error: false,
})

let bootstrapPromise: Promise<void> | null = null

/** 取得 storefront bootstrap；重複呼叫共用同一個 in-flight request。 */
export function loadBootstrap(): Promise<void> {
  if (bootstrap.data) return Promise.resolve()
  if (!bootstrapPromise) {
    bootstrap.loading = true
    bootstrapPromise = apiGet<BootstrapData>('/api/storefront/bootstrap')
      .then((data) => {
        bootstrap.data = data
        bootstrap.error = false
      })
      .catch(() => {
        bootstrap.error = true
      })
      .finally(() => {
        bootstrap.loading = false
      })
  }
  return bootstrapPromise
}

export function useStore(): BootstrapData {
  if (!bootstrap.data) throw new Error('bootstrap data not loaded')
  return bootstrap.data
}

// ─── 購物車 Drawer 開合 ──────────────────────────────────────

export const cartDrawer = reactive({ open: false })

export function openCartDrawer() {
  cartDrawer.open = true
}

// ─── 結帳共享狀態：已套用優惠券（購物車 → 結帳頁共用） ─────

export interface AppliedCoupon {
  code: string
  type: 'percent' | 'fixed' | 'freeshipping'
  value: number
  label: string
}

export const checkoutStore = reactive<{ coupon: AppliedCoupon | null }>({
  coupon: null,
})

// ─── 運費計算（顯示用；結帳金額一律以 quote 為準） ────────────

export function computeShippingFee(
  method: ShippingMethodDTO | null | undefined,
  subtotal: number,
  freeShippingCoupon = false,
): number {
  if (!method) return 0
  if (freeShippingCoupon) return 0
  const threshold = method.free_threshold
  if (threshold != null && threshold > 0 && subtotal >= threshold) return 0
  return method.fee
}

// ─── 最近查詢的訂單（localStorage） ─────────────────────────

export interface RecentOrder {
  orderId: string
  token: string
  date: string
}

const RECENT_KEY = 'curatory_recent_orders'

export function loadRecentOrders(): RecentOrder[] {
  try {
    const raw = localStorage.getItem(RECENT_KEY)
    if (raw) {
      const parsed = JSON.parse(raw) as RecentOrder[]
      if (Array.isArray(parsed)) return parsed.slice(0, 5)
    }
  } catch {
    /* ignore */
  }
  return []
}

export function rememberOrder(orderId: string, token: string) {
  try {
    const list = loadRecentOrders().filter((r) => r.orderId !== orderId)
    list.unshift({ orderId, token, date: new Date().toISOString() })
    localStorage.setItem(RECENT_KEY, JSON.stringify(list.slice(0, 5)))
  } catch {
    /* ignore */
  }
}

export function findRecentToken(orderId: string): string | null {
  return loadRecentOrders().find((r) => r.orderId === orderId)?.token ?? null
}
