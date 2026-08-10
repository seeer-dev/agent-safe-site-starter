import type { ContentListSummary, ContentRow } from '../model'

const ROWS: ContentRow[] = [
  { id: 'content-home-hero', title: '夏季新品週優惠', placement: '首頁主視覺', locale: '繁體中文', updatedAt: '2025-01-12', status: 'published' },
  { id: 'content-campaign-banner', title: '夏季活動Banner設定', placement: '活動頁', locale: '繁體中文', updatedAt: '2025-01-10', status: 'published' },
  { id: 'content-footer-service', title: '退換貨政策', placement: '購物須知', locale: '繁體中文', updatedAt: '2025-01-08', status: 'draft' },
  { id: 'content-about', title: 'About Us', placement: '關於我們', locale: 'English', updatedAt: '2025-01-05', status: 'draft' },
]

export const contentListFixture: ContentListSummary = {
  items: ROWS,
  pagination: { page: 1, perPage: 100, total: 24, totalPages: 1 },
}
