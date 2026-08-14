import type { ResourceDef } from '@/lib/types';
import { formatUnix } from '@/lib/utils';

export const productsResource: ResourceDef = {
  label: '商品',
  desc: 'tw-minimal-cart 商品目錄與庫存',
  pageSize: 20,
  ops: {
    list: 'adminMinimalCartProductsList',
    create: 'adminMinimalCartProductsCreate',
    update: 'adminMinimalCartProductsUpdate',
    del: 'adminMinimalCartProductsDelete',
    status: 'adminMinimalCartProductsUpdateStatus',
    bulk: 'adminMinimalCartProductsBulkUpdate',
  },
  api: {
    list: '/admin/products',
    get: '/admin/products/{id}',
    create: '/admin/products',
    update: '/admin/products/{id}',
    delete: '/admin/products/{id}',
    status: '/admin/products/{id}/status',
  },
  rowMap: (raw: Record<string, any>) => ({
    ...raw,
    updated_at: formatUnix(raw.updated_unix),
  }),
  createCap: 'twcommerce.create',
  updateCap: 'twcommerce.update',
  cols: [
    { k: 'sku', l: 'SKU', r: 'mono' },
    { k: 'name', l: '商品' },
    { k: 'category', l: '分類', r: 'badge' },
    { k: 'price', l: '售價', r: 'number' },
    { k: 'stock', l: '庫存', r: 'number' },
    { k: 'status', l: '狀態', r: 'badge' },
    { k: 'updated_at', l: '更新', r: 'datetime' },
  ],
  rowActions: [
    { k: 'edit', l: '編輯', cap: 'twcommerce.update', variant: 'sec', form: true },
    { k: 'publish', l: '上架', op: 'adminMinimalCartProductsUpdateStatus', cap: 'twcommerce.update', payload: { status: 'active' }, showWhen: 'status=draft', confirm: '確認上架？' },
    { k: 'draft', l: '轉草稿', op: 'adminMinimalCartProductsUpdateStatus', cap: 'twcommerce.update', payload: { status: 'draft' }, showWhen: 'status=active|low_stock|out_of_stock', confirm: '確認下架？', variant: 'danger' },
    { k: 'archive', l: '封存', op: 'adminMinimalCartProductsDelete', cap: 'twcommerce.delete', confirm: '確認封存商品？', reason: true, variant: 'danger' },
  ],
  bulkActions: [
    { k: 'bulk-active', l: '批次上架', op: 'adminMinimalCartProductsBulkUpdate', cap: 'twcommerce.update', payload: { status: 'active' }, confirm: '確認批次上架？' },
    { k: 'bulk-draft', l: '批次草稿', op: 'adminMinimalCartProductsBulkUpdate', cap: 'twcommerce.update', payload: { status: 'draft' }, confirm: '確認批次轉為草稿？', reason: true, variant: 'danger' },
  ],
  filters: [
    { k: 'status', l: '狀態', w: 'select', opts: [['', '全部'], ['active', '上架'], ['draft', '草稿']] },
    { k: 'category', l: '分類', w: 'text' },
  ],
  form: {
    title: '商品',
    sections: [
      {
        t: '商品資料',
        fields: [
          { k: 'sku', l: 'SKU', w: 'text', req: true },
          { k: 'name', l: '名稱', w: 'text', req: true },
          { k: 'slug', l: 'Slug', w: 'text', req: true },
          { k: 'product_images', l: '商品圖片', w: 'media-uploader', span: 2,
            help: '上傳圖片後系統會驗證並關聯。每張圖片需通過 R2 驗證（magic byte、尺寸、解碼）。' },
          { k: 'description', l: '描述', w: 'textarea', span: 2 },
          { k: 'category', l: '分類', w: 'select', opts: ['apparel', 'accessories', 'home', 'stationery'] },
          { k: 'status', l: '狀態', w: 'select', opts: ['draft', 'active'] },
          { k: 'material', l: '材質', w: 'text' },
          { k: 'origin', l: '產地', w: 'text' },
          { k: 'price', l: '售價', w: 'number', req: true },
          { k: 'original_price', l: '原價', w: 'number' },
          { k: 'stock', l: '庫存', w: 'number', req: true },
          { k: 'tag', l: '標籤', w: 'text' },
        ],
      },
    ],
  },
  rows: [],
};
