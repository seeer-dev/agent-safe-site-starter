import type { ResourceDef } from '@/lib/types';

export const membersResource: ResourceDef = {
  label: '會員',
  desc: '會員狀態、標籤與累計',
  pageSize: 20,
  updateCap: 'twcommerce.update',
  ops: {
    list: 'adminMinimalCartMembersList',
    update: 'adminMinimalCartMembersUpdate',
    status: 'adminMinimalCartMembersStatusUpdate',
  },
  api: {
    list: '/admin/members',
    update: '/admin/members/{id}',
    status: '/admin/members/{id}/status',
  },
  cols: [
    { k: 'email', l: 'Email', r: 'mono' },
    { k: 'name', l: '姓名' },
    { k: 'status', l: '狀態', r: 'badge' },
    { k: 'tier', l: '等級', r: 'badge' },
    { k: 'total_orders', l: '訂單', r: 'number' },
    { k: 'total_spent', l: '累計', r: 'number' },
  ],
  rowActions: [
    { k: 'edit', l: '編輯', cap: 'twcommerce.update', variant: 'sec', form: true },
    { k: 'lock', l: '鎖定', op: 'adminMinimalCartMembersStatusUpdate', cap: 'twcommerce.update', payload: { status: 'locked' }, showWhen: 'status=active', reason: true, variant: 'danger' },
    { k: 'unlock', l: '解鎖', op: 'adminMinimalCartMembersStatusUpdate', cap: 'twcommerce.update', payload: { status: 'active' }, showWhen: 'status=locked', reason: true },
  ],
  filters: [
    { k: 'status', l: '狀態', w: 'text' },
    { k: 'tag', l: '標籤', w: 'text' },
  ],
  form: {
    title: '會員',
    sections: [
      {
        t: '營運欄位',
        fields: [
          { k: 'email', l: 'Email', w: 'text', ro: true },
          { k: 'tags', l: '標籤', w: 'tags' },
          { k: 'tier', l: '等級', w: 'text' },
          { k: 'notes', l: '備註', w: 'textarea', span: 2 },
        ],
      },
    ],
  },
  rows: [],
};
