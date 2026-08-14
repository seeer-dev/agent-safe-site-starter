import type { ResourceDef } from '@/lib/types';
import { formatUnix } from '@/lib/utils';

export const shippingMethodsResource: ResourceDef = {
  label: '配送方式',
  desc: '配送方式與門檻',
  pageSize: 20,
  createCap: 'twcommerce.create',
  updateCap: 'twcommerce.update',
  expectedVersionField: 'version',
  ops: {
    list: 'adminMinimalCartShippingMethodsList',
    create: 'adminMinimalCartShippingMethodsCreate',
    update: 'adminMinimalCartShippingMethodsUpdate',
  },
  api: {
    list: '/admin/shipping-methods',
    create: '/admin/shipping-methods',
    update: '/admin/shipping-methods/{id}',
  },
  rowMap: (raw: Record<string, any>) => ({
    ...raw,
    enabled: String(raw.enabled),
    updated_at: formatUnix(raw.updated_unix),
  }),
  cols: [
    { k: 'method', l: '代碼', r: 'mono' },
    { k: 'label', l: '名稱' },
    { k: 'fee', l: '運費', r: 'number' },
    { k: 'free_threshold', l: '免運門檻', r: 'number' },
    { k: 'enabled', l: '啟用', r: 'badge' },
    { k: 'sort_order', l: '排序', r: 'number' },
    { k: 'version', l: '版本', r: 'number' },
    { k: 'updated_at', l: '更新', r: 'datetime' },
  ],
  rowActions: [
    { k: 'edit', l: '編輯', cap: 'twcommerce.update', variant: 'sec', form: true },
  ],
  filters: [],
  form: {
    title: '配送方式',
    sections: [
      {
        t: '配送設定',
        fields: [
          { k: 'method', l: '代碼', w: 'text', req: true, roOnEdit: true, help: '建立後不可變更' },
          { k: 'label', l: '名稱', w: 'text', req: true },
          { k: 'description', l: '說明', w: 'textarea' },
          { k: 'fee', l: '運費（TWD）', w: 'number', req: true },
          { k: 'free_threshold', l: '免運門檻（空白為無）', w: 'number', nullable: true },
          { k: 'enabled', l: '啟用', w: 'switch' },
          { k: 'sort_order', l: '排序', w: 'number', req: true },
        ],
      },
    ],
  },
  rows: [],
};
