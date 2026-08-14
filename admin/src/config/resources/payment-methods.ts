import type { ResourceDef } from '@/lib/types';
import { formatUnix } from '@/lib/utils';

export const paymentMethodsResource: ResourceDef = {
  label: '付款方式',
  desc: '付款與相依設定',
  pageSize: 10,
  updateCap: 'twcommerce.admin',
  ops: {
    list: 'adminTwCommerceMethodsList',
    update: 'adminTwCommerceMethodsUpdate',
  },
  api: {
    list: '/admin/payment-methods',
    update: '/admin/payment-methods/{id}',
  },
  rowMap: (raw: Record<string, any>) => ({
    ...raw,
    enabled: String(raw.enabled),
    updated_at: formatUnix(raw.updated_unix),
  }),
  cols: [
    { k: 'method', l: '付款方式', r: 'mono' },
    { k: 'provider_label', l: '服務商' },
    { k: 'environment', l: '環境', r: 'badge' },
    { k: 'readiness_status', l: 'Readiness', r: 'badge' },
    { k: 'enabled', l: '啟用', r: 'badge' },
    { k: 'updated_at', l: '更新時間', r: 'datetime' },
  ],
  rowActions: [
    { k: 'edit', l: '編輯', cap: 'twcommerce.admin', variant: 'sec', form: true },
  ],
  filters: [],
  form: {
    title: '付款方式',
    sections: [
      {
        t: '付款與相依設定',
        fields: [
          { k: 'method', l: '付款方式', w: 'text', req: true, ro: true },
          { k: 'provider_label', l: '服務商', w: 'text', req: true },
          { k: 'environment', l: '環境', w: 'select', opts: ['sandbox', 'production'], req: true },
          { k: 'readiness_status', l: 'Readiness', w: 'select', opts: ['pending_setup', 'ready'], req: true },
          { k: 'enabled', l: '啟用', w: 'switch' },
        ],
      },
    ],
  },
  rows: [],
};
