// 購物車（免登入，localStorage 持久化 + API rehydrate）
//
// Persisted items store only identifiers + snapshot fields for instant
// paint; rehydrate() refreshes price/stock/name from the authoritative
// catalog API. Fail-closed: if the API is unavailable the cart stays
// empty — persisted price/stock are never trusted at checkout (the
// server recomputes everything anyway).

import { reactive } from 'vue'
import { apiGet } from './api'
import type { ProductDTO } from './types'

export interface CartItem {
  key: string // productSlug + variantSKU
  productId: string
  productSlug: string
  sku: string
  variantId?: string | null
  name: string
  variantName?: string | null
  image: string
  unitPrice: number // product.price + variant.price_delta
  qty: number
  stock: number
}

const STORAGE_KEY = 'curatory-cart'

function makeKey(productSlug: string, sku?: string | null) {
  return `${productSlug}::${sku ?? 'default'}`
}

interface CartState {
  items: CartItem[]
  lastAddedAt: number | null
}

export const cart = reactive<CartState>({
  items: [],
  lastAddedAt: null,
})

export function cartSubtotal(items: CartItem[] = cart.items): number {
  return items.reduce((sum, i) => sum + i.unitPrice * i.qty, 0)
}

export function cartCount(items: CartItem[] = cart.items): number {
  return items.reduce((sum, i) => sum + i.qty, 0)
}

export function addItem(item: Omit<CartItem, 'key'>) {
  const key = makeKey(item.productSlug, item.sku)
  const existing = cart.items.find((i) => i.key === key)
  if (existing) {
    existing.qty = Math.min(existing.qty + item.qty, Math.max(item.stock, 1))
    existing.unitPrice = item.unitPrice
    existing.stock = item.stock
  } else {
    cart.items.push({ ...item, key })
  }
  cart.lastAddedAt = Date.now()
  persistCart()
}

export function removeItem(key: string) {
  cart.items = cart.items.filter((i) => i.key !== key)
  persistCart()
}

export function setQty(key: string, qty: number) {
  cart.items = cart.items
    .map((i) => (i.key === key ? { ...i, qty: Math.max(0, Math.min(qty, i.stock)) } : i))
    .filter((i) => i.qty > 0)
  persistCart()
}

export function clearCart() {
  cart.items = []
  cart.lastAddedAt = null
  persistCart()
}

export function persistCart() {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify({ items: cart.items }))
  } catch {
    /* quota/private mode — cart is session-only */
  }
}

export function restoreCart() {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return
    const parsed = JSON.parse(raw) as { items?: CartItem[] }
    if (Array.isArray(parsed.items)) cart.items = parsed.items
  } catch {
    /* corrupt state — start empty */
  }
}

// Rehydrate display data (price/stock/name/image) from the catalog API.
// Items whose product/variant no longer resolves are dropped.
export async function rehydrateCart() {
  if (cart.items.length === 0) return
  let products: ProductDTO[]
  try {
    const res = await apiGet<{ products: ProductDTO[] }>('/api/products')
    products = res.products ?? []
  } catch {
    return // fail-open: keep persisted snapshot, checkout revalidates
  }
  const bySlug = new Map(products.map((p) => [p.slug, p]))
  cart.items = cart.items.flatMap((item) => {
    const p = bySlug.get(item.productSlug)
    if (!p) return []
    const variant = item.variantId ? p.variants.find((v) => v.sku === item.sku) : null
    const price = p.price + (variant?.price_delta ?? 0)
    const stock = variant ? variant.stock : p.stock
    if (stock <= 0) return []
    return [
      {
        ...item,
        name: p.name,
        variantName: variant?.name ?? item.variantName,
        image: p.images?.[0] ?? p.image,
        unitPrice: price,
        stock,
        qty: Math.min(item.qty, stock),
      },
    ]
  })
  persistCart()
}
