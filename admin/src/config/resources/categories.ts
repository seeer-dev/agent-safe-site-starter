import type { ResourceDef } from '@/lib/types';
import { formatUnix } from '@/lib/utils';

// 商品分類：curatory 前台的分類實體（slug-keyed，供商品關聯與
// /categories/{slug}/ 靜態頁）。
export const categoriesResource: ResourceDef = {
  label: '商品分類',
  desc: '前台分類導覽與商品歸屬',
  pageSize: 20,
  createCap: 'twcommerce.create',
  updateCap: 'twcommerce.update',
  ops: {
    list: 'adminCategoriesList',
    create: 'adminCategoriesCreate',
    update: 'adminCategoriesUpdate',
    del: 'adminCategoriesDelete',
  },
  api: {
    list: '/admin/categories',
    create: '/admin/categories',
    update: '/admin/categories/{id}',
    delete: '/admin/categories/{id}',
  },
  rowMap: (raw: Record<string, any>) => ({
    ...raw,
    is_active_label: raw.is_active ? 'true' : 'false',
    updated_at: formatUnix(raw.updated_unix),
  }),
  pinActions: true,
  cols: [
    { k: 'slug', pin: 'left', l: 'Slug', r: 'mono' },
    { k: 'name', l: '名稱', sortable: true },
    { k: 'sort_order', l: '排序', r: 'number' },
    { k: 'is_active_label', l: '啟用', r: 'badge' },
    { k: 'updated_at', l: '更新', r: 'datetime' },
  ],
  rowActions: [
    { k: 'edit', l: '編輯', cap: 'twcommerce.update', variant: 'sec', form: true },
    { k: 'delete', l: '刪除', op: 'adminCategoriesDelete', cap: 'twcommerce.delete', confirm: '確認刪除此分類？其下商品的分類欄位需另行調整。', variant: 'danger' },
  ],
  filters: [],
  form: {
    title: '商品分類',
    sections: [
      {
        t: '分類資料',
        fields: [
          { k: 'slug', l: 'Slug', w: 'text', req: true, help: 'URL 友善代碼，例如 ceramics' },
          { k: 'name', l: '名稱', w: 'text', req: true },
          { k: 'description', l: '說明', w: 'textarea', span: 2 },
          { k: 'image', l: '分類圖 URL', w: 'text' },
          { k: 'sort_order', l: '排序', w: 'number' },
          { k: 'is_active', l: '啟用', w: 'switch' },
        ],
      },
    ],
  },
  rows: [],
};
