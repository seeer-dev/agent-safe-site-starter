import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { Product, FooterPageKey } from '@/shared/lib/types'

// UI store — coordinates cross-island dialog state.
//
// In the original SPA, App.vue held all dialog open/close state and passed
// it via props/emits. In the islands architecture there is no shared App
// component, so islands communicate through this Pinia store instead:
//   - Header island can call ui.openCart() / ui.openAccount() etc.
//   - CartDrawer island watches ui.cartOpen.
//   - ProductGrid island calls ui.openProductDetail(product).
//   - ProductDetailDialog island watches ui.productDetail.
//
// This keeps the islands decoupled: no island imports another island.

export const useUiStore = defineStore('ui', () => {
  const cartOpen = ref(false)
  const accountOpen = ref(false)
  const trackOrderOpen = ref(false)
  const checkoutOpen = ref(false)
  const footerPageOpen = ref(false)
  const productDetailOpen = ref(false)

  const footerPage = ref<FooterPageKey | null>(null)
  const selectedProduct = ref<Product | null>(null)
  const trackOrderId = ref<string | null>(null)

  function openCart() { cartOpen.value = true }
  function closeCart() { cartOpen.value = false }

  function openAccount() { accountOpen.value = true }
  function closeAccount() { accountOpen.value = false }

  function openTrackOrder(orderId?: string) {
    trackOrderId.value = orderId ?? null
    trackOrderOpen.value = true
  }
  function closeTrackOrder() { trackOrderOpen.value = false }

  function openCheckout() { checkoutOpen.value = true }
  function closeCheckout() { checkoutOpen.value = false }

  function openFooterPage(page: FooterPageKey) {
    footerPage.value = page
    footerPageOpen.value = true
  }
  function closeFooterPage() { footerPageOpen.value = false }

  function openProductDetail(product: Product) {
    selectedProduct.value = product
    productDetailOpen.value = true
  }
  function closeProductDetail() { productDetailOpen.value = false }

  return {
    cartOpen, accountOpen, trackOrderOpen, checkoutOpen,
    footerPageOpen, productDetailOpen,
    footerPage, selectedProduct, trackOrderId,
    openCart, closeCart,
    openAccount, closeAccount,
    openTrackOrder, closeTrackOrder,
    openCheckout, closeCheckout,
    openFooterPage, closeFooterPage,
    openProductDetail, closeProductDetail,
  }
})
