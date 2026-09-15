import type { ResourceDef } from '@/lib/types';
import { formatUnix } from '@/lib/utils';

// 商品評論審核：pending → approved/rejected；approve/reject 皆可附
// 站方回覆（後端 reply 欄位，顯示於商品頁評論卡）。
export const commentsResource: ResourceDef = {
  label: '商品評論',
  desc: '顧客評論審核與站方回覆',
  pageSize: 20,
  updateCap: 'twcommerce.update',
  ops: {
    list: 'adminCommentsList',
    status: 'adminCommentsModerate',
  },
  api: {
    list: '/admin/comments',
    status: '/admin/comments/{id}',
  },
  rowMap: (raw: Record<string, any>) => ({
    ...raw,
    rating_label: raw.rating ? `${'★'.repeat(raw.rating)}${'☆'.repeat(5 - raw.rating)}` : '—',
    replied_at: formatUnix(raw.replied_unix),
    created_at: formatUnix(raw.created_unix),
  }),
  pinActions: true,
  cols: [
    { k: 'product_id', pin: 'left', l: '商品', r: 'mono' },
    { k: 'nickname', l: '暱稱', sortable: true },
    { k: 'rating_label', l: '評分' },
    { k: 'content', l: '內容' },
    { k: 'status', l: '狀態', r: 'badge' },
    { k: 'admin_reply', l: '站方回覆' },
    { k: 'created_at', l: '留言時間', r: 'datetime' },
  ],
  rowActions: [
    { k: 'detail', l: '明細', variant: 'sec', form: true },
    { k: 'approve', l: '核准', op: 'adminCommentsModerate', cap: 'twcommerce.update', payload: { status: 'approved' }, showWhen: 'status=pending', confirm: '核准此評論？可在下方附上站方回覆。', reasonOptional: true, reasonField: 'reply', reasonLabel: '站方回覆' },
    { k: 'reject', l: '駁回', op: 'adminCommentsModerate', cap: 'twcommerce.update', payload: { status: 'rejected' }, showWhen: 'status=pending|approved', confirm: '駁回此評論？駁回後不會顯示於前台。', reasonOptional: true, reasonField: 'reply', reasonLabel: '駁回說明', variant: 'danger' },
  ],
  filters: [
    { k: 'status', l: '狀態', w: 'select', opts: [['', '全部'], ['pending', '待審'], ['approved', '已核准'], ['rejected', '已駁回']] },
  ],
  form: {
    title: '評論明細',
    readOnly: true,
    sections: [
      {
        t: '評論',
        fields: [
          { k: 'product_id', l: '商品 ID', w: 'text', ro: true },
          { k: 'nickname', l: '暱稱', w: 'text', ro: true },
          { k: 'rating', l: '評分', w: 'text', ro: true },
          { k: 'status', l: '狀態', w: 'text', ro: true },
          { k: 'content', l: '內容', w: 'textarea', ro: true, span: 2 },
          { k: 'admin_reply', l: '站方回覆', w: 'textarea', ro: true, span: 2 },
        ],
      },
    ],
  },
  rows: [],
};
