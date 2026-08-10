import type { DashboardSummary } from '../model'

/* Fixture data for the Dashboard. This module is imported only by
 * the bootstrap/runtime composition layer, never by the query layer
 * or the page. The fixture/remote switch happens at that boundary —
 * a future remote provider replaces the reader without changing
 * the query or page. */
export const dashboardFixture: DashboardSummary = {
  greeting: '早安，怡君 · 週五 7/25',
  cards: [
    { id: 'pending-review', num: '12', label: '待審核訂單', cta: '前往審核 →', variant: 'warning' },
    { id: 'pending-ship', num: '8', label: '待理貨 / 待出貨', cta: '準備出貨 →', variant: 'info' },
    { id: 'releasable', num: '6', label: '分潤可釋放', cta: '釋放為待撥款 →', variant: 'success' },
    { id: 'pending-payout', num: '$19.6k', label: '待撥款分潤', cta: '記一筆撥款 →', variant: 'danger' },
  ],
  tasks: [
    { id: 't1', title: '#TW-20874 待審核', meta: '林小美 · 夏季聯名 · NT$3,480 · 可審核通過', secondaryAction: '明細', primaryAction: '審核通過' },
    { id: 't2', title: '#TW-20871 待審核，但缺物流單號', meta: 'Kevin Lu · 建立物流單後才能審核通過', primaryAction: '建立物流單' },
    { id: 't3', title: '夏季聯名 · 2 位網紅待撥款', meta: '合計 NT$19,600 · 撥款後網紅端顯示「已撥款」' },
  ],
  recentOrders: [
    { id: 'r1', orderId: '#TW-20874', customer: '林小美', amount: '$3,480', statusVariant: 'wait', statusLabel: '待審核' },
    { id: 'r2', orderId: '#TW-20866', customer: '莎莎', amount: '$6,010', statusVariant: 'ship', statusLabel: '配送中' },
    { id: 'r3', orderId: '#TW-20852', customer: 'Kevin Lu', amount: '$4,320', statusVariant: 'done', statusLabel: '已完成' },
  ],
}
