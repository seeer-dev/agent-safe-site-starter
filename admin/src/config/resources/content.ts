import type { ResourceDef } from '@/lib/types';
import { formatUnix } from '@/lib/utils';

export const contentResource: ResourceDef = {
  label: '前台內容',
  desc: 'Hero、公告、Popup、Footer、政策版位',
  pageSize: 20,
  createCap: 'content.create',
  updateCap: 'content.update',
  ops: {
    list: 'adminMinimalCartContentList',
    create: 'adminMinimalCartContentCreate',
    update: 'adminMinimalCartContentUpdate',
    approve: 'adminMinimalCartContentApprove',
    publish: 'adminMinimalCartContentPublish',
  },
  api: {
    list: '/admin/site-content',
    create: '/admin/site-content',
    update: '/admin/site-content/{id}',
    delete: '/admin/site-content/{id}',
    approve: '/admin/site-content/{id}/approve',
    publish: '/admin/site-content/{id}/publish',
  },
  rowMap: (raw: Record<string, any>) => ({
    ...raw,
    approved_at: formatUnix(raw.approved_unix),
    approved_expiry_at: formatUnix(raw.approved_expiry_unix),
    published_approved_at: formatUnix(raw.published_approved_unix),
    published_approval_expiry_at: formatUnix(raw.published_approval_expiry_unix),
  }),
  cols: [
    { k: 'key', l: 'Key', r: 'mono' },
    { k: 'placement', l: '位置', r: 'badge' },
    { k: 'title', l: '標題' },
    { k: 'status', l: '狀態', r: 'badge' },
    { k: 'draft_version', l: '草稿版', r: 'number' },
    { k: 'approved_version', l: '已核可版', r: 'number' },
    { k: 'approver_user_id', l: '核可人' },
    { k: 'approved_at', l: '核可時間' },
    { k: 'approved_expiry_at', l: '核可到期' },
    { k: 'published_version', l: '發布版', r: 'number' },
    { k: 'published_approver_user_id', l: '發布核可人' },
    { k: 'published_approved_at', l: '發布核可時間' },
    { k: 'published_approval_expiry_at', l: '發布快照到期' },
    { k: 'sort_order', l: '排序', r: 'number' },
  ],
  rowActions: [
    { k: 'edit', l: '編輯', cap: 'content.update', variant: 'sec', form: true },
    {
      k: 'approve',
      l: '核可',
      cap: 'content.approve',
      variant: 'pri',
      op: 'adminMinimalCartContentApprove',
      expect: 'draft_version',
      expiryInput: true,
      confirm: '核可此草稿版本？請設定核可到期時間，過期後需重新核可。',
      // Always show approve to users with content.approve. An approval
      // can be missing, stale (version mismatch), or expired (version
      // matches but expiry has passed). Hiding it would trap operators.
      showWhen: 'approve_always',
    },
    {
      k: 'publish',
      l: '發布',
      cap: 'content.publish',
      variant: 'pri',
      op: 'adminMinimalCartContentPublish',
      expect: 'draft_version',
      confirm: '發布此內容？需有目前有效的核可（版本相符且未過期）。',
      // Only show publish when there is a current, non-expired approval.
      // The server still enforces all conditions; this is UX guidance.
      showWhen: 'publish_ready',
    },
  ],
  filters: [
    { k: 'placement', l: '位置', w: 'select', opts: [['', '全部'], ['hero', 'Hero'], ['announcement', '公告'], ['popup', 'Popup'], ['footer', 'Footer'], ['policy', '政策']] },
    { k: 'status', l: '狀態', w: 'select', opts: [['', '全部'], ['draft', '草稿'], ['published', '已發布']] },
  ],
  form: {
    title: '前台內容',
    sections: [
      {
        t: '內容',
        fields: [
          { k: 'key', l: 'Key', w: 'text', req: true },
          { k: 'placement', l: '位置', w: 'select', req: true, opts: ['hero', 'announcement', 'popup', 'footer', 'policy'] },
          { k: 'title', l: '標題', w: 'text', req: true },
          { k: 'body', l: '內容', w: 'textarea', span: 2 },
          { k: 'sort_order', l: '排序', w: 'number' },
        ],
      },
    ],
  },
  rows: [],
};
