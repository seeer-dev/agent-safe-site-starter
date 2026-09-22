// DTO types matching the Go API JSON contract (snake_case field names
// come straight from the Go json tags — see contracts/openapi.yaml).

export interface VariantDTO {
  id: string
  product_id: string
  name: string
  sku: string
  price_delta: number
  stock: number
  sort_order: number
}

export interface ProductDTO {
  id: string
  sku: string
  name: string
  slug: string
  description: string
  long_description: string
  image: string
  images: string[]
  category: string // category slug
  status: string // draft|active|low_stock|out_of_stock
  material: string
  origin: string
  price: number
  original_price: number
  stock: number
  tag: string
  rating: number
  reviews_count: number
  is_featured: boolean
  sold_count: number
  updated_unix: number
  variants: VariantDTO[]
}

export interface CategoryDTO {
  id: string
  slug: string
  name: string
  description: string
  image: string
  sort_order: number
  is_active: boolean
  product_count?: number
}

export interface ShippingMethodDTO {
  id: string // stable method key
  label: string
  available: boolean
  description: string
  // 運費刻意不公開：結帳時由 /api/quote 回傳權威金額。
}

export interface PaymentMethodDTO {
  id: string
  method: string
  label: string
  available: boolean
  fee: number
}

export interface SiteContentDTO {
  id: string
  key: string
  title: string
  body: string
  placement: string
}

export interface StoreSettings {
  storeName?: string
  storeNameEn?: string
  tagline?: string
  promoBanner?: string
  promoBannerEnabled?: boolean
  freeShippingThreshold?: number
  lowStockThreshold?: number
  notificationMaster?: boolean
  contactEmail?: string
  contactPhone?: string
  [key: string]: unknown
}

export interface BootstrapData {
  settings: StoreSettings
  categories: CategoryDTO[]
  site_content: SiteContentDTO[]
  payment_methods: PaymentMethodDTO[]
  shipping_methods: ShippingMethodDTO[]
  announcements: ArticleDTO[]
  /** Public Cloudflare Turnstile site key; empty in local development. */
  turnstile_site_key?: string
}

export interface ArticleDTO {
  id: string
  slug: string
  title: string
  excerpt: string
  body_html: string
  published: boolean
  pinned: boolean
  published_at: number
  updated_unix: number
}

export interface PromoDTO {
  id: string
  code: string
  label: string
  type: 'percent' | 'fixed' | 'freeshipping'
  value: number
  enabled: boolean
  min_subtotal: number
  usage_limit?: number | null
  used_count: number
  starts_unix: number
  expires_unix: number
}

export interface CommentDTO {
  id: string
  product_id: string
  nickname: string
  content: string
  rating?: number
  status: string // pending|approved|rejected
  admin_reply?: string
  replied_unix?: number
  created_unix: number
}

export interface OrderItemDTO {
  sku: string
  name: string
  price: number
  quantity: number
  variant_name?: string
}

export interface TimelineEventDTO {
  status: string
  at: number
  note?: string
}

export interface OrderDTO {
  id: string
  customer_name: string
  email: string
  phone: string
  items: OrderItemDTO[]
  shipping_address: string
  shipping_method: string
  payment_method: string
  tracking_number: string
  subtotal: number
  discount: number
  shipping: number
  payment_fee: number
  total: number
  status: string
  payment_status: string
  recipient_name: string
  recipient_phone: string
  city: string
  district: string
  cvs_store_name: string
  cvs_store_address: string
  invoice_type: string
  invoice_tax_id: string
  coupon_code: string
  buyer_note: string
  access_token?: string
  timeline?: TimelineEventDTO[]
  version: number
  updated_unix: number
}

export interface QuoteResult {
  items: OrderItemDTO[]
  subtotal: number
  discount: number
  shipping: number
  payment_fee: number
  total: number
  promo_code?: string
}

// ─── Display labels ─────────────────────────────────────────

export const ORDER_STATUS_LABEL: Record<string, string> = {
  pending: '待付款／處理中',
  paid: '已付款',
  processing: '處理中',
  shipped: '已出貨',
  delivered: '已送達',
  completed: '已完成',
  cancelled: '已取消',
}

export const PAYMENT_STATUS_LABEL: Record<string, string> = {
  unpaid: '待付款',
  paid: '已付款',
  refunded: '已退款',
}
