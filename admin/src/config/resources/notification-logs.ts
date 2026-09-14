import type { ResourceDef } from '@/lib/types';
import { formatUnix } from '@/lib/utils';

// 通知發送日誌（唯讀）：sent/skipped/failed 的完整決策軌跡。
export const notificationLogsResource: ResourceDef = {
  label: '通知日誌',
  desc: '通知發送與略過紀錄（唯讀）',
  pageSize: 30,
  ops: {
    list: 'adminNotificationLogsList',
  },
  api: {
    list: '/admin/notification-logs',
  },
  rowMap: (raw: Record<string, any>) => ({
    ...raw,
    created_at: formatUnix(raw.created_unix),
  }),
  cols: [
    { k: 'created_at', l: '時間', r: 'datetime' },
    { k: 'code', l: '事件', r: 'badge' },
    { k: 'order_id', l: '訂單', r: 'mono' },
    { k: 'recipient', l: '收件人' },
    { k: 'subject', l: '主旨' },
    { k: 'status', l: '結果', r: 'badge' },
    { k: 'error', l: '錯誤' },
  ],
  rowActions: [
    { k: 'detail', l: '明細', variant: 'sec', form: true },
  ],
  filters: [
    { k: 'code', l: '事件', w: 'select', opts: [['', '全部'], ['order_placed', '訂單成立'], ['order_paid', '已付款'], ['order_shipped', '已出貨'], ['order_completed', '已完成'], ['order_cancelled', '已取消']] },
    { k: 'status', l: '結果', w: 'select', opts: [['', '全部'], ['sent', '已發送'], ['skipped', '已略過'], ['failed', '失敗']] },
    { k: 'order_id', l: '訂單編號', w: 'text' },
  ],
  form: {
    title: '通知日誌',
    readOnly: true,
    sections: [
      {
        t: '日誌',
        fields: [
          { k: 'code', l: '事件', w: 'text', ro: true },
          { k: 'order_id', l: '訂單', w: 'text', ro: true },
          { k: 'recipient', l: '收件人', w: 'text', ro: true },
          { k: 'status', l: '結果', w: 'text', ro: true },
          { k: 'subject', l: '主旨', w: 'text', ro: true, span: 2 },
          { k: 'body', l: '內容', w: 'textarea', ro: true, span: 2 },
          { k: 'error', l: '錯誤', w: 'textarea', ro: true, span: 2 },
        ],
      },
    ],
  },
  rows: [],
};
