import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { Order, OrderStatus, OrderTimelineEntry, User } from '@/shared/lib/types'
import type { PublicAuthSession } from '@/shared/lib/auth/session'

const DAY = 24 * 60 * 60 * 1000

export const STATUS_META: Record<
  OrderStatus,
  { label: string; description: string; step: number }
> = {
  pending: { label: '訂單成立', description: '我們已收到您的訂單。', step: 0 },
  processing: { label: '處理中', description: '倉庫正在理貨包裝。', step: 1 },
  shipped: { label: '已出貨', description: '商品正在配送途中。', step: 2 },
  delivered: { label: '已送達', description: '商品已送達，祝您使用愉快！', step: 3 },
  cancelled: { label: '已取消', description: '訂單已取消。', step: -1 },
}

function buildTimeline(status: OrderStatus, placedAt: number): OrderTimelineEntry[] {
  const steps: OrderStatus[] = ['pending', 'processing', 'shipped', 'delivered']
  const currentStep = STATUS_META[status].step
  return steps.map((s, i) => ({
    status: s,
    label: STATUS_META[s].label,
    description: STATUS_META[s].description,
    timestamp: placedAt + i * DAY,
    ...(currentStep >= 0 && i <= currentStep ? {} : { future: true } as any),
  })) as OrderTimelineEntry[]
}

void buildTimeline

export const useUserStore = defineStore('user', () => {
  const user = ref<User | null>(null)
  const orders = ref<Order[]>([])
  const lastPlacedOrderId = ref<string | null>(null)
  const bearerToken = ref('')

  function syncFromSession(session: PublicAuthSession | null) {
    if (!session) {
      user.value = null
      bearerToken.value = ''
      return
    }
    bearerToken.value = session.accessToken
    user.value = {
      id: session.user.id,
      email: session.user.email,
      name: session.user.name,
      joinedAt: session.user.joinedAt,
    }
  }

  function logout() {
    user.value = null
    bearerToken.value = ''
  }

  function recordOrder(order: Order) {
    orders.value = [order, ...orders.value]
    lastPlacedOrderId.value = order.id
  }

  function getOrder(id: string): Order | undefined {
    const normalized = id.trim().toUpperCase()
    return orders.value.find((o) => o.id === normalized)
  }

  function getUserOrders(email?: string): Order[] {
    const targetEmail = (email ?? user.value?.email)?.toLowerCase()
    if (!targetEmail) return []
    return orders.value
      .filter((o) => o.userEmail?.toLowerCase() === targetEmail)
      .sort((a, b) => b.placedAt - a.placedAt)
  }

  function resetLastPlaced() { lastPlacedOrderId.value = null }

  function persist() {
    try {
      localStorage.removeItem('minimal-user')
    } catch { /* ignore */ }
  }

  function restore() {
    try {
      localStorage.removeItem('minimal-user')
    } catch { /* ignore */ }
  }

  return {
    user, orders, lastPlacedOrderId, bearerToken,
    syncFromSession, logout,
    recordOrder, getOrder, getUserOrders,
    resetLastPlaced, persist, restore,
  }
})
