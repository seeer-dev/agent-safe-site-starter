// 目錄快取：發布商品列表（前台目錄規模小，一次抓取後模組內快取）。
// 價格/庫存仍以 API 為準 —— 此快取只用於顯示與加入購物車前的快照；
// 結帳金額一律由 /api/quote 重新計算。

import { apiGet } from './api'
import type { ProductDTO } from './types'

let productsPromise: Promise<ProductDTO[]> | null = null

/** 取得全部發布商品（共用 in-flight request，模組層快取）。 */
export function loadProducts(): Promise<ProductDTO[]> {
  if (!productsPromise) {
    productsPromise = apiGet<{ products: ProductDTO[] }>('/api/products')
      .then((r) => r.products ?? [])
      .catch(() => [])
  }
  return productsPromise
}

export async function getProductBySlug(slug: string): Promise<ProductDTO | null> {
  const items = await loadProducts()
  return items.find((p) => p.slug === slug) ?? null
}

/** 快取失效（例如購物車 rehydrate 發現缺品時可重新拉）。 */
export function invalidateCatalog() {
  productsPromise = null
}
