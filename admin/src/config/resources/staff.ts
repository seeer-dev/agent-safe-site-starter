import type { ResourceDef } from '@/lib/types';

export const staffResource: ResourceDef = {
  label: '員工',
  desc: '後台使用者與角色',
  pageSize: 20,
  updateCap: 'staff.update',
  ops: {
    list: 'adminStaffMembersList',
    create: 'adminStaffMembersCreate',
    update: 'adminStaffMembersUpdate',
    status: 'adminStaffMembersStatusUpdate',
    del: 'adminStaffMembersDelete',
  },
  api: {
    list: '/admin/staff',
    create: '/admin/staff',
    update: '/admin/staff/{id}',
    status: '/admin/staff/{id}/status',
    delete: '/admin/staff/{id}',
  },
  cols: [
    { k: 'id', l: 'ID', r: 'mono' },
    { k: 'display_name', l: '姓名' },
    { k: 'role_label', l: '角色', r: 'badge' },
    { k: 'email', l: 'Email' },
    { k: 'status', l: '狀態', r: 'badge' },
  ],
  rowActions: [
    { k: 'edit', l: '編輯', cap: 'staff.update', variant: 'sec', form: true },
    { k: 'disable', l: '停用', op: 'adminStaffMembersStatusUpdate', cap: 'staff.update', payload: { status: 'disabled' }, showWhen: 'status=active', confirm: '確認停用？', variant: 'danger' },
    { k: 'enable', l: '啟用', op: 'adminStaffMembersStatusUpdate', cap: 'staff.update', payload: { status: 'active' }, showWhen: 'status=disabled', confirm: '確認啟用？' },
  ],
  filters: [],
  form: {
    title: '員工',
    sections: [
      {
        t: '帳號',
        fields: [
          { k: 'display_name', l: '姓名', w: 'text', req: true },
          { k: 'email', l: 'Email', w: 'text', req: true },
          { k: 'role_label', l: '角色', w: 'select', opts: ['owner', 'manager', 'readonly'] },
          { k: 'supabase_user_id', l: 'Supabase User ID', w: 'text' },
          { k: 'status', l: '狀態', w: 'select', opts: ['active', 'disabled'] },
        ],
      },
    ],
  },
  rows: [],
};
