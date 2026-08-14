import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { CartItem, Product } from '@/shared/lib/types'
import { fetchProducts } from '@/shared/lib/api'

// The cart store retains identifiers, quantity, and selections only.
// Monetary values (subtotal, discount, shipping, tax, total) are NOT
// computed here — they come from the server-authoritative /api/quote
// response in CheckoutDialog. The browser is not the pricing authority.
// This is the browser-authority fix: no local pricing constants.
//
// PERSISTENCE: localStorage stores ONLY product/variant identifiers,
// quantity, and selections — never the full Product object (which
// contains price, stock, name that could be stale or tampered). On
// restore, the store rehydrates display data from the current catalog
// API response. If rehydration fails (API down, product removed), the
// cart is fail-closed: affected items are dropped.

// PersistedCartItem is the minimal shape stored in localStorage. It
// contains NO monetary or display data — only identifiers needed to
// rehydrate from the authoritative catalog.
interface PersistedCartItem {
  productId: string
  sku: string
  quantity: number
  selectedSize?: string
  selectedColor?: string
}

function sameLine(
  item: CartItem,
  productId: string,
  options?: { size?: string; color?: string }
) {
  return (
    item.product.id === productId &&
    (item.selectedSize ?? '') === (options?.size ?? '') &&
    (item.selectedColor ?? '') === (options?.color ?? '')
  )
}

export const useCartStore = defineStore('cart', () => {
  const items = ref<CartItem[]>([])
  const isOpen = ref(false)
  const appliedPromo = ref<string | null>(null)
  const favorites = ref<string[]>([])

  function openCart() { isOpen.value = true }
  function closeCart() { isOpen.value = false }
  function toggleCart() { isOpen.value = !isOpen.value }

  function addItem(product: Product, quantity = 1, options?: { size?: string; color?: string }) {
    const existing = items.value.find((i) => sameLine(i, product.id, options))
    if (existing) {
      items.value = items.value.map((i) =>
        sameLine(i, product.id, options)
          ? { ...i, quantity: Math.min(i.quantity + quantity, product.stock) }
          : i
      )
    } else {
      items.value = [
        ...items.value,
        { product, quantity, selectedSize: options?.size, selectedColor: options?.color },
      ]
    }
    isOpen.value = true
  }

  function removeItem(productId: string, options?: { size?: string; color?: string }) {
    items.value = items.value.filter((i) => !sameLine(i, productId, options))
  }

  function updateQuantity(productId: string, quantity: number, options?: { size?: string; color?: string }) {
    if (quantity <= 0) {
      items.value = items.value.filter((i) => !sameLine(i, productId, options))
      return
    }
    items.value = items.value.map((i) =>
      sameLine(i, productId, options)
        ? { ...i, quantity: Math.min(quantity, i.product.stock) }
        : i
    )
  }

  function clear() {
    items.value = []
    appliedPromo.value = null
  }

  function toggleFavorite(productId: string) {
    if (favorites.value.includes(productId)) {
      favorites.value = favorites.value.filter((id) => id !== productId)
    } else {
      favorites.value = [...favorites.value, productId]
    }
  }

  function isFavorite(productId: string) {
    return favorites.value.includes(productId)
  }

  function applyPromo(code: string): { ok: boolean; message: string } {
    const normalized = code.trim().toUpperCase()
    if (!normalized) return { ok: false, message: '請輸入優惠碼' }
    appliedPromo.value = normalized
    return { ok: true, message: '優惠碼將於結帳報價時由伺服器驗證' }
  }

  function removePromo() { appliedPromo.value = null }

  // itemCount is the only computed value — it is a count of physical
  // items, not a monetary value. It does not require server authority.
  const itemCount = computed(() =>
    items.value.reduce((sum, i) => sum + i.quantity, 0)
  )

  // persist stores ONLY identifiers, quantity, and selections. The full
  // Product object (price, stock, name, image) is NEVER persisted — it
  // is rehydrated from the authoritative catalog on restore.
  function persist() {
    const persistedItems: PersistedCartItem[] = items.value.map((i) => ({
      productId: i.product.id,
      sku: i.product.sku,
      quantity: i.quantity,
      selectedSize: i.selectedSize,
      selectedColor: i.selectedColor,
    }))
    localStorage.setItem('minimal-cart', JSON.stringify({
      items: persistedItems,
      favorites: favorites.value,
      appliedPromo: appliedPromo.value,
    }))
  }

  // restore reads persisted identifiers and marks the cart as needing
  // rehydration. The actual Product data is NOT restored from localStorage
  // — it must be fetched from the catalog API via rehydrate(). Until
  // rehydration completes, items is empty (fail-closed).
  function restore() {
    try {
      const raw = localStorage.getItem('minimal-cart')
      if (raw) {
        const data = JSON.parse(raw)
        favorites.value = data.favorites ?? []
        appliedPromo.value = data.appliedPromo ?? null
        // Do NOT restore items directly — they need rehydration.
        // Store pending identifiers for rehydrate() to process.
        pendingRestore = (data.items ?? []) as PersistedCartItem[]
      }
    } catch { /* ignore */ }
  }

  // pendingRestore holds persisted identifiers between restore() and
  // rehydrate(). It is module-scoped so rehydrate can access it.
  let pendingRestore: PersistedCartItem[] = []

  // rehydrate fetches the current product catalog from the server and
  // rebuilds CartItem objects with authoritative Product data. Items
  // whose product no longer exists in the catalog are dropped. If the
  // API call fails entirely, the cart stays empty (fail-closed) — we
  // do NOT trust persisted price/stock/name.
  async function rehydrate() {
    if (pendingRestore.length === 0) return
    const pending = pendingRestore
    pendingRestore = []
    try {
      const products = await fetchProducts()
      const byId = new Map(products.map((p) => [p.id, p]))
      const bySku = new Map(products.map((p) => [p.sku, p]))
      const rehydrated: CartItem[] = []
      for (const p of pending) {
        // Prefer productId match, fall back to sku match.
        const product = byId.get(p.productId) ?? bySku.get(p.sku)
        if (!product) continue // product removed from catalog — drop
        if (product.stock <= 0) continue // out of stock — drop
        const quantity = Math.min(p.quantity, product.stock)
        if (quantity <= 0) continue
        rehydrated.push({
          product,
          quantity,
          selectedSize: p.selectedSize,
          selectedColor: p.selectedColor,
        })
      }
      items.value = rehydrated
    } catch {
      // API failure — fail closed. Do NOT use persisted Product data.
      items.value = []
    }
  }

  return {
    items, isOpen, appliedPromo, favorites,
    openCart, closeCart, toggleCart,
    addItem, removeItem, updateQuantity, clear,
    toggleFavorite, isFavorite,
    applyPromo, removePromo,
    itemCount,
    persist, restore, rehydrate,
  }
})
