/**
 * Order detail domain model — canonical types for the order detail
 * cockpit (hero, stepper, tabs: 明細/審核歷史/分潤/出貨文件).
 */

export type OrderStageState = 'done' | 'current' | 'pending'

export interface OrderStageStep {
  label: string
  state: OrderStageState
}

export interface OrderHeroFact {
  label: string
  value: string
}

export interface OrderDetailHero {
  orderNumber: string
  statusLabel: string
  statusTone: 'wait' | 'ship' | 'done' | 'info'
  meta: string
  facts: OrderHeroFact[]
}

export interface OrderProductRow {
  id: string
  name: string
  sku: string
  unitPrice: string
  qty: number
  subtotal: string
}

export interface OrderReviewEvent {
  id: string
  title: string
  meta: string
}

export interface OrderCommissionLine {
  id: string
  sku: string
  rule: string
  base: string
  commission: string
  status: string
  statusTone: 'wait' | 'ship' | 'done' | 'draft'
}

export interface OrderFulfillmentDoc {
  id: string
  name: string
  date: string
  type: string
}

export interface OrderDetailSummary {
  hero: OrderDetailHero
  orderSteps: OrderStageStep[]
  commissionSteps: OrderStageStep[]
  products: OrderProductRow[]
  reviewEvents: OrderReviewEvent[]
  commissionLines: OrderCommissionLine[]
  fulfillmentDocs: OrderFulfillmentDoc[]
  commissionNote: string
}
