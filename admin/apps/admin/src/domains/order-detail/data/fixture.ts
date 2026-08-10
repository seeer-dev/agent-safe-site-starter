import type {
  OrderCommissionLine,
  OrderDetailHero,
  OrderDetailSummary,
  OrderFulfillmentDoc,
  OrderProductRow,
  OrderReviewEvent,
  OrderStageStep,
} from '../model'

/* Fixture data for the Order detail cockpit. This module is imported
 * only by the bootstrap/runtime composition layer. */

const HERO: OrderDetailHero = {
  orderNumber: '#TW-20874',
  statusLabel: '已付款',
  statusTone: 'info',
  meta: '下訂 2026-07-24 09:12 · 目前第 2 / 8 階',
  facts: [
    { label: '買家', value: '陳大文' },
    { label: '分潤網紅', value: '林小美' },
    { label: '分潤來源', value: '折扣碼 AMY88' },
    { label: '金額', value: 'NT$3,480' },
  ],
}

const ORDER_STEPS: OrderStageStep[] = [
  { label: '已建立', state: 'done' },
  { label: '已付款', state: 'current' },
  { label: '已接受', state: 'pending' },
  { label: '處理中', state: 'pending' },
  { label: '揀貨中', state: 'pending' },
  { label: '配送中', state: 'pending' },
  { label: '已取貨', state: 'pending' },
  { label: '完成', state: 'pending' },
]

const COMMISSION_STEPS: OrderStageStep[] = [
  { label: '預估', state: 'done' },
  { label: '已鎖定', state: 'done' },
  { label: '結案保留', state: 'done' },
  { label: '可撥款', state: 'current' },
  { label: '已撥款', state: 'pending' },
]

const PRODUCTS: OrderProductRow[] = [
  { id: 'p1', name: '無袖小花 T-Shirt（白/M）', sku: 'TS-SU-WM', unitPrice: '$1,280', qty: 2, subtotal: '$2,560' },
  { id: 'p2', name: '托特包 BAG-01', sku: 'BAG-01', unitPrice: '$460', qty: 2, subtotal: '$920' },
]

const REVIEW_EVENTS: OrderReviewEvent[] = [
  { id: 'r1', title: '訂單建立', meta: '系統 · 09:12' },
  { id: 'r2', title: '付款完成', meta: '藍新 · 09:13' },
  { id: 'r3', title: '分潤解析：折扣碼 AMY88 → 林小美', meta: '系統 · 09:13' },
  { id: 'r4', title: '審核通過', meta: '陳老闆 · 10:05' },
]

const COMMISSION_LINES: OrderCommissionLine[] = [
  { id: 'c1', sku: 'TS-SU-WM', rule: '比例 15%', base: '$2,560', commission: '$384', status: '預估', statusTone: 'draft' },
  { id: 'c2', sku: 'BAG-01', rule: '固定 $80/件', base: '$920', commission: '$80', status: '預估', statusTone: 'draft' },
]

const FULFILLMENT_DOCS: OrderFulfillmentDoc[] = []

const COMMISSION_NOTE =
  '後台看七個狀態，網紅端只看兩個：待撥款 / 已撥款。其餘階段網紅端不顯示金額（因為預估值會被退貨改動）。'

export const orderDetailFixture: OrderDetailSummary = {
  hero: HERO,
  orderSteps: ORDER_STEPS,
  commissionSteps: COMMISSION_STEPS,
  products: PRODUCTS,
  reviewEvents: REVIEW_EVENTS,
  commissionLines: COMMISSION_LINES,
  fulfillmentDocs: FULFILLMENT_DOCS,
  commissionNote: COMMISSION_NOTE,
}
