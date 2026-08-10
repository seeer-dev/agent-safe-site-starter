/**
 * Dashboard domain model — canonical types.
 *
 * The fixture, reader port, query layer, and page all depend on
 * these types. This is the single source of truth for the
 * DashboardSummary shape; no other module re-declares it.
 */

export interface DashboardCard {
  id: string
  num: string
  label: string
  cta: string
  variant: 'warning' | 'info' | 'success' | 'danger'
}

export interface DashboardTask {
  id: string
  title: string
  meta: string
  primaryAction?: string
  secondaryAction?: string
}

export interface DashboardRecentOrder {
  id: string
  orderId: string
  customer: string
  amount: string
  statusVariant: 'wait' | 'ship' | 'done'
  statusLabel: string
}

export interface DashboardSummary {
  greeting: string
  cards: DashboardCard[]
  tasks: DashboardTask[]
  recentOrders: DashboardRecentOrder[]
}
