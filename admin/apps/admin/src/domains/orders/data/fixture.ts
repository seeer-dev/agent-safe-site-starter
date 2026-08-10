import type { OrderListSummary, OrderRow, OrderSavedView } from '../model'

/* Fixture data for the Orders list. This module is imported only by
 * the bootstrap/runtime composition layer, never by the query layer
 * or the page. */
const SAVED_VIEWS: OrderSavedView[] = [
  { id: 'pending-review', label: '待審核', count: 12 },
  { id: 'attributed', label: '帶貨訂單', count: 146 },
  { id: 'fulfillment', label: '待出貨', count: 8 },
  { id: 'history', label: '確認紀錄', count: 231 },
  { id: 'all', label: '全部', count: 231 },
]

const ROWS: OrderRow[] = [
  { id: 'ord-20874', orderNumber: '#TW-20874', buyerName: '林小美', influencerName: '莎莎', campaignName: '夏季聯名', couponCode: 'AMY88', amount: '$3,480', progressLabel: '待審核', progressTone: 'wait', paymentLabel: '已付款', canReview: true },
  { id: 'ord-20871', orderNumber: '#TW-20871', buyerName: 'Kevin Lu', influencerName: 'Kevin Lu', campaignName: '限時折扣', couponCode: null, amount: '$2,200', progressLabel: '待審核', progressTone: 'wait', paymentLabel: '已付款', canReview: true },
  { id: 'ord-20866', orderNumber: '#TW-20866', buyerName: '莎莎', influencerName: 'Amy', campaignName: '春季限定', couponCode: 'SPRING20', amount: '$6,010', progressLabel: '配送中', progressTone: 'ship', paymentLabel: '已付款', canReview: false },
  { id: 'ord-20852', orderNumber: '#TW-20852', buyerName: 'Kevin Lu', influencerName: 'Kevin Lu', campaignName: null, couponCode: null, amount: '$4,320', progressLabel: '已完成', progressTone: 'done', paymentLabel: '已付款', canReview: false },
]

export const orderListFixture: OrderListSummary = {
  savedViews: SAVED_VIEWS,
  rows: ROWS,
  pendingReviewCount: 12,
}
