import type { ResourceDef } from '@/lib/types';
import { formatUnix } from '@/lib/utils';

// 公告／新聞文章：POST /admin/articles 為 slug-keyed upsert（建立或
// 更新即發布），pinned 控制前台置頂排序。
export const articlesResource: ResourceDef = {
  label: '公告文章',
  desc: '前台「最新公告」文章（發布即上線）',
  pageSize: 20,
  createCap: 'content.publish',
  updateCap: 'content.publish',
  ops: {
    list: 'adminArticlesList',
    create: 'adminArticlesUpsert',
    update: 'adminArticlesUpsert',
  },
  api: {
    list: '/admin/articles',
    create: '/admin/articles',
    update: '/admin/articles',
  },
  rowMap: (raw: Record<string, any>) => ({
    ...raw,
    published_label: raw.published ? 'true' : 'false',
    pinned_label: raw.pinned ? 'true' : 'false',
    published_at_label: formatUnix(raw.published_at),
    updated_at: formatUnix(raw.updated_unix),
  }),
  cols: [
    { k: 'slug', l: 'Slug', r: 'mono' },
    { k: 'title', l: '標題' },
    { k: 'pinned_label', l: '置頂', r: 'badge' },
    { k: 'published_label', l: '已發布', r: 'badge' },
    { k: 'published_at_label', l: '發布時間', r: 'datetime' },
    { k: 'updated_at', l: '更新', r: 'datetime' },
  ],
  rowActions: [
    { k: 'edit', l: '編輯', cap: 'content.publish', variant: 'sec', form: true },
  ],
  filters: [
    { k: 'published', l: '狀態', w: 'select', opts: [['', '全部'], ['true', '已發布'], ['false', '草稿']] },
  ],
  form: {
    title: '公告文章',
    sections: [
      {
        t: '文章',
        fields: [
          { k: 'slug', l: 'Slug', w: 'text', req: true, roOnEdit: true, help: 'URL 友善代碼；同 slug 再存即更新' },
          { k: 'title', l: '標題', w: 'text', req: true },
          { k: 'published', l: '發布', w: 'switch', help: '關閉則僅存草稿，前台不可見' },
          { k: 'pinned', l: '置頂', w: 'switch' },
          { k: 'published_at', l: '發布時間', w: 'datetime' },
          { k: 'excerpt', l: '摘要', w: 'textarea', span: 2 },
          { k: 'body_html', l: '內容（HTML）', w: 'textarea', req: true, span: 2 },
        ],
      },
    ],
  },
  rows: [],
};
