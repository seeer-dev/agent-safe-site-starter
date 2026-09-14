import type { ResourceDef } from '@/lib/types';
import { formatUnix } from '@/lib/utils';

// 訂單通知信模板：code 對應訂單事件（order_placed / paid / shipped /
// delivered / cancelled），subject/body 支援 {{變數}} 佔位。
export const notificationTemplatesResource: ResourceDef = {
  label: '通知模板',
  desc: '訂單生命週期通知信模板',
  pageSize: 20,
  createCap: 'twcommerce.update',
  updateCap: 'twcommerce.update',
  ops: {
    list: 'adminNotificationTemplatesList',
    create: 'adminNotificationTemplatesCreate',
    update: 'adminNotificationTemplatesUpdate',
    del: 'adminNotificationTemplatesDelete',
  },
  api: {
    list: '/admin/notification-templates',
    create: '/admin/notification-templates',
    update: '/admin/notification-templates/{id}',
    delete: '/admin/notification-templates/{id}',
  },
  rowMap: (raw: Record<string, any>) => ({
    ...raw,
    is_enabled_label: raw.is_enabled ? 'true' : 'false',
    updated_at: formatUnix(raw.updated_unix),
  }),
  cols: [
    { k: 'code', l: '事件代碼', r: 'mono' },
    { k: 'name', l: '名稱' },
    { k: 'subject', l: '主旨' },
    { k: 'is_enabled_label', l: '啟用', r: 'badge' },
    { k: 'updated_at', l: '更新', r: 'datetime' },
  ],
  rowActions: [
    { k: 'edit', l: '編輯', cap: 'twcommerce.update', variant: 'sec', form: true },
    { k: 'delete', l: '刪除', op: 'adminNotificationTemplatesDelete', cap: 'twcommerce.delete', confirm: '確認刪除此模板？對應事件將不再寄送通知。', variant: 'danger' },
  ],
  filters: [
    { k: 'code', l: '事件', w: 'select', opts: [['', '全部'], ['order_placed', '訂單成立'], ['order_paid', '已付款'], ['order_shipped', '已出貨'], ['order_completed', '已完成'], ['order_cancelled', '已取消']] },
  ],
  form: {
    title: '通知模板',
    sections: [
      {
        t: '模板',
        fields: [
          { k: 'code', l: '事件代碼', w: 'select', req: true, opts: ['order_placed', 'order_paid', 'order_shipped', 'order_completed', 'order_cancelled'], help: '對應訂單狀態事件' },
          { k: 'name', l: '名稱', w: 'text', req: true },
          { k: 'is_enabled', l: '啟用', w: 'switch' },
          { k: 'subject', l: '主旨', w: 'text', req: true, span: 2, help: '支援 {{order_id}} {{customer_name}} {{total}} 等佔位' },
          { k: 'body', l: '內容', w: 'textarea', req: true, span: 2 },
        ],
      },
    ],
  },
  rows: [],
};
