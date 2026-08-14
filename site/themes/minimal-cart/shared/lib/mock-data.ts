import type { Category } from '@/shared/lib/types'

// This file contains ONLY structural category labels. All pricing,
// shipping, payment, and product data has been removed per the browser
// authority fix — monetary values and method availability must come from
// current Go API responses, not hardcoded browser constants.
//
// FREE_SHIPPING_THRESHOLD, SHIPPING_FLAT_RATE, SHIPPING_METHODS, and
// PAYMENT_METHODS were removed because:
// - No approval artifact authorizes a fee schedule (GATE-004 approved
//   Taiwan-main-island scope only, not fees or a free-shipping threshold).
// - The browser must not be the authority for pricing, shipping, or
//   payment availability — it must fetch from /api/shipping-methods,
//   /api/payment-methods, and /api/quote.
// - Cart storage retains identifiers, quantity, and selections only.

export const CATEGORIES: { value: Category; label: string }[] = [
  { value: 'all', label: '全部' },
  { value: 'apparel', label: '服飾' },
  { value: 'accessories', label: '配件' },
  { value: 'home', label: '家居' },
  { value: 'stationery', label: '文具' },
]

// HERO_SLIDES and HERO_STATS are empty — the Hero island receives its
// content via data-props from the Go renderer, which reads from the CMS.
// No fabricated hero content is bundled.
export const HERO_SLIDES: any[] = []
export const HERO_STATS: any[] = []

// ANNOUNCEMENTS is empty — the AnnouncementBar island receives its
// content via data-props from the Go renderer, which reads from the CMS.
// No fabricated announcements are bundled.
export const ANNOUNCEMENTS: any[] = []
