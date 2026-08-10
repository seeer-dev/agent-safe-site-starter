/**
 * Orders domain model — canonical types for the orders list page.
 *
 * The fixture, reader port, query layer, and page all depend on these
 * types. This is the single source of truth for the OrderList shape.
 */

export type OrderProgressTone = 'wait' | 'ship' | 'done'

export interface OrderSavedView {
  id: 'pending-review' | 'attributed' | 'fulfillment' | 'history' | 'all'
  label: string
  count: number
}

export interface OrderRow {
  id: string
  orderNumber: string
  buyerName: string
  influencerName: string
  campaignName: string | null
  couponCode: string | null
  amount: string
  progressLabel: string
  progressTone: OrderProgressTone
  paymentLabel: string
  canReview: boolean
}

export interface OrderListSummary {
  savedViews: OrderSavedView[]
  rows: OrderRow[]
  pendingReviewCount: number
}
