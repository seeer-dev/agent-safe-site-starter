import type { ResourceDef } from '@/lib/types';
import { formatUnix } from '@/lib/utils';

export const ordersResource: ResourceDef = {
  label: '訂單',
  desc: '履約與付款狀態分離的訂單操作',
  pageSize: 20,
  updateCap: 'twcommerce.update',
  ops: {
    list: 'adminMinimalCartOrdersList',
    get: 'adminMinimalCartOrdersGet',
    status: 'adminMinimalCartOrdersStatusUpdate',
    returnStatus: 'adminMinimalCartOrdersReturnStatusUpdate',
    restock: 'adminMinimalCartOrdersRestock',
  },
  api: {
    list: '/admin/orders',
    get: '/admin/orders/{id}',
    status: '/admin/orders/{id}/status',
    returnStatus: '/admin/orders/{id}/return',
    restock: '/admin/orders/{id}/restock',
  },
  rowMap: (raw: Record<string, any>) => ({
    ...raw,
    customer: raw.customer_name ?? raw.customer,
    items_detail: Array.isArray(raw.items)
      ? raw.items.map((i: any) => `${i.name} ×${i.quantity}`).join('\n')
      : raw.items_detail ?? '',
    timeline: Array.isArray(raw.timeline)
      ? raw.timeline.map((t: any) => `${new Date(t.at * 1000).toLocaleString('zh-TW')} ${t.status}${t.note ? ' ' + t.note : ''}`).join('\n')
      : raw.timeline ?? '',
    updated_at: formatUnix(raw.updated_unix),
  }),
  pinActions: true,
  cols: [
    { k: 'id', pin: 'left', l: '訂單', r: 'mono' },
    { k: 'customer', l: '顧客', sortable: true },
    { k: 'total', l: '金額', r: 'number', sortable: true },
    { k: 'status', l: '履約', r: 'badge' },
    { k: 'payment_status', l: '付款', r: 'badge' },
    { k: 'return_request_status', l: '退貨', r: 'badge' },
    { k: 'updated_at', l: '更新', r: 'datetime' },
  ],
  rowActions: [
    { k: 'detail', l: '明細', variant: 'sec', form: true },
    { k: 'processing', l: '處理中', op: 'adminMinimalCartOrdersStatusUpdate', cap: 'twcommerce.update', payload: { status: 'processing' }, expect: 'version', showWhen: 'status=pending', confirm: '確認開始處理？' },
    { k: 'shipped', l: '已出貨', op: 'adminMinimalCartOrdersStatusUpdate', cap: 'twcommerce.update', payload: { status: 'shipped' }, expect: 'version', showWhen: 'status=processing', confirm: '確認出貨？' },
    { k: 'delivered', l: '已送達', op: 'adminMinimalCartOrdersStatusUpdate', cap: 'twcommerce.update', payload: { status: 'delivered' }, expect: 'version', showWhen: 'status=shipped', confirm: '確認送達？' },
    { k: 'completed', l: '結案', op: 'adminMinimalCartOrdersStatusUpdate', cap: 'twcommerce.update', payload: { status: 'completed' }, expect: 'version', showWhen: 'status=delivered', confirm: '確認結案？' },
    { k: 'cancel', l: '取消', op: 'adminMinimalCartOrdersStatusUpdate', cap: 'twcommerce.update', payload: { status: 'cancelled' }, expect: 'version', showWhen: 'status=pending|processing', confirm: '確認取消？', reason: true, variant: 'danger' },
    { k: 'approve-return', l: '核准退貨', op: 'adminMinimalCartOrdersReturnStatusUpdate', cap: 'twcommerce.update', payload: { status: 'approved' }, expect: 'version', showWhen: 'return_request_status=requested', confirm: '確認核准退貨？' },
    { k: 'reject-return', l: '駁回退貨', op: 'adminMinimalCartOrdersReturnStatusUpdate', cap: 'twcommerce.update', payload: { status: 'rejected' }, expect: 'version', showWhen: 'return_request_status=requested', confirm: '確認駁回退貨？', reason: true, variant: 'danger' },
    { k: 'receive-return', l: '確認收件', op: 'adminMinimalCartOrdersReturnStatusUpdate', cap: 'twcommerce.update', payload: { status: 'received' }, expect: 'version', showWhen: 'return_request_status=approved', confirm: '確認已收到退貨？' },
    { k: 'restock', l: '驗收回補', op: 'adminMinimalCartOrdersRestock', allCaps: ['orders.returns', 'inventory.adjust'], expect: 'version', showWhen: 'return_request_status=received', confirm: '確認驗收回補可售庫存？', reason: true, variant: 'sec', restockItems: true },
  ],
  filters: [
    { k: 'status', l: '履約狀態', w: 'text' },
    { k: 'payment_status', l: '付款狀態', w: 'text' },
  ],
  form: {
    title: '訂單詳情',
    readOnly: true,
    sections: [
      {
        t: '營運資料',
        fields: [
          { k: 'id', l: '訂單', w: 'text', ro: true },
          { k: 'customer', l: '顧客', w: 'text', ro: true },
          { k: 'email', l: 'Email', w: 'text', ro: true },
          { k: 'phone', l: '電話', w: 'text', ro: true },
          { k: 'items_detail', l: '品項', w: 'textarea', ro: true, span: 2 },
          { k: 'shipping_address', l: '地址', w: 'textarea', ro: true, span: 2 },
          { k: 'recipient_name', l: '收件人', w: 'text', ro: true },
          { k: 'recipient_phone', l: '收件人電話', w: 'text', ro: true },
          { k: 'recipient_city', l: '縣市', w: 'text', ro: true },
          { k: 'recipient_district', l: '鄉鎮市區', w: 'text', ro: true },
          { k: 'cvs_store_name', l: '超商門市', w: 'text', ro: true },
          { k: 'shipping_method', l: '配送方式', w: 'text', ro: true },
          { k: 'payment_method', l: '付款方式', w: 'text', ro: true },
          { k: 'subtotal', l: '小計', w: 'text', ro: true },
          { k: 'discount', l: '折扣', w: 'text', ro: true },
          { k: 'shipping_fee', l: '運費', w: 'text', ro: true },
          { k: 'payment_fee', l: '金流手續費', w: 'text', ro: true },
          { k: 'coupon_code', l: '優惠券', w: 'text', ro: true },
          { k: 'invoice_type', l: '發票類型', w: 'text', ro: true },
          { k: 'invoice_company_tax_id', l: '統編', w: 'text', ro: true },
          { k: 'buyer_note', l: '買家備註', w: 'text', ro: true, span: 2 },
          { k: 'payment_intent_id', l: '付款 intent', w: 'text', ro: true },
          { k: 'payment_status', l: '付款狀態', w: 'text', ro: true },
          { k: 'timeline', l: '時間軸', w: 'textarea', ro: true, span: 2 },
        ],
      },
    ],
  },
  rows: [],
};
