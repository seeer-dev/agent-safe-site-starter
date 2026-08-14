import type { ResourceDef } from '@/lib/types';
import { formatUnix } from '@/lib/utils';

export const promosResource: ResourceDef = {
  label: '優惠活動',
  desc: '折扣碼與檔期',
  pageSize: 20,
  createCap: 'twcommerce.create',
  updateCap: 'twcommerce.update',
  ops: {
    list: 'adminMinimalCartPromosList',
    create: 'adminMinimalCartPromosCreate',
    update: 'adminMinimalCartPromosUpdate',
    del: 'adminMinimalCartPromosDelete',
  },
  api: {
    list: '/admin/promos',
    create: '/admin/promos',
    update: '/admin/promos/{id}',
    delete: '/admin/promos/{id}',
  },
  rowMap: (raw: Record<string, any>) => ({
    ...raw,
    enabled: String(raw.enabled),
    starts_at: formatUnix(raw.starts_unix),
    expires_at: formatUnix(raw.expires_unix),
  }),
  cols: [
    { k: 'code', l: 'Code', r: 'mono' },
    { k: 'type', l: '類型', r: 'badge' },
    { k: 'value', l: '數值', r: 'number' },
    { k: 'enabled', l: '啟用', r: 'badge' },
    { k: 'starts_at', l: '開始', r: 'datetime' },
    { k: 'expires_at', l: '到期', r: 'datetime' },
  ],
  rowActions: [
    { k: 'edit', l: '編輯', cap: 'twcommerce.update', variant: 'sec', form: true },
    { k: 'disable', l: '停用', op: 'adminMinimalCartPromosDelete', cap: 'twcommerce.delete', showWhen: 'enabled=true', variant: 'danger', confirm: '確認停用？' },
  ],
  filters: [],
  form: {
    title: '優惠',
    sections: [
      {
        t: '檔期',
        fields: [
          { k: 'code', l: 'Code', w: 'text', req: true },
          { k: 'label', l: '名稱', w: 'text', req: true },
          { k: 'type', l: '類型', w: 'select', opts: ['percent', 'fixed'] },
          { k: 'value', l: '數值', w: 'number' },
          { k: 'enabled', l: '啟用', w: 'switch' },
          { k: 'starts_unix', l: '開始', w: 'datetime' },
          { k: 'expires_unix', l: '到期（空白為無限）', w: 'datetime', span: 2 },
        ],
      },
    ],
  },
  rows: [],
};
