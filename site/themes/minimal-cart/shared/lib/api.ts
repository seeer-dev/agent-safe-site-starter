import type { Product } from '@/shared/lib/types'

export class ApiRequestError extends Error {
  constructor(public readonly status: number, message: string) {
    super(message)
    this.name = 'ApiRequestError'
  }
}

async function readErrorMessage(res: Response, fallback: string): Promise<string> {
  const err = await res.json().catch(() => ({ error: res.statusText }))
  return err.error || `${fallback}: ${res.status}`
}

// Get API base from the <html data-api-base="..."> attribute.
// In dev the Go server serves the API at the same origin; in production on
// Cloudflare Pages the API is also same-origin. The data-api-base attribute
// lets the renderer override the base if needed (e.g. a different subdomain).
function apiBase(): string {
  const html = document.documentElement
  return html.getAttribute('data-api-base') || ''
}

function apiUrl(path: string): string {
  const base = apiBase()
  return base ? `${base}${path}` : path
}

// Map a Go API product shape to the frontend Product type.
// The Go API returns snake_case fields and `images` as a JSON string array.
function mapProduct(raw: any): Product {
  if (!Array.isArray(raw.images) || !raw.images.every((image: unknown) => typeof image === 'string')) {
    throw new Error('產品回應的 images 必須是字串陣列')
  }
  let images: string[] = raw.images
  if (images.length === 0 && raw.image) {
    images = [raw.image]
  }
  return {
    id: raw.id,
    name: raw.name,
    slug: raw.slug,
    description: raw.description || '',
    longDescription: raw.long_description || '',
    price: raw.price || 0,
    originalPrice: raw.original_price || undefined,
    image: raw.image || '',
    images,
    category: raw.category || 'apparel',
    tag: raw.tag || undefined,
    rating: raw.rating || 0,
    reviews: raw.reviews_count || 0,
    ratingBreakdown: [],
    reviewList: [],
    stock: raw.stock || 0,
    sku: raw.sku || '',
    material: raw.material || undefined,
    origin: raw.origin || undefined,
  }
}

// Fetch all published products (status=active).
export async function fetchProducts(): Promise<Product[]> {
  const res = await fetch(apiUrl('/api/products'))
  if (!res.ok) throw new Error(`fetchProducts: ${res.status}`)
  const data = await res.json()
  return (data.products || []).map(mapProduct)
}

// Fetch a single product by slug. Returns null on 404.
export async function fetchProductBySlug(slug: string): Promise<Product | null> {
  const res = await fetch(apiUrl(`/api/products/${encodeURIComponent(slug)}`))
  if (res.status === 404) return null
  if (!res.ok) throw new Error(`fetchProductBySlug: ${res.status}`)
  const data = await res.json()
  return mapProduct(data.product)
}

// Server-authoritative shipping method. The browser must NOT hardcode
// shipping methods, fees, or availability — it must use this endpoint.
// Available is false when no fee schedule is approved (GATE-004 scope only).
export interface ShippingMethodResult {
  id: string
  label: string
  available: boolean
  description: string
}

export async function fetchShippingMethods(): Promise<ShippingMethodResult[]> {
  const res = await fetch(apiUrl('/api/shipping-methods'))
  if (!res.ok) throw new Error(`fetchShippingMethods: ${res.status}`)
  const data = await res.json()
  return (data.shipping_methods || []).map((m: any) => ({
    id: m.id,
    label: m.label || '',
    available: !!m.available,
    description: m.description || '',
  }))
}

// Server-authoritative payment method. The browser must NOT hardcode
// payment methods — it must use this endpoint. Only enabled and ready
// methods are returned.
export interface PaymentMethodResult {
  id: string
  method: string
  label: string
  available: boolean
}

export async function fetchPaymentMethods(): Promise<PaymentMethodResult[]> {
  const res = await fetch(apiUrl('/api/payment-methods'))
  if (!res.ok) throw new Error(`fetchPaymentMethods: ${res.status}`)
  const data = await res.json()
  return (data.payment_methods || []).map((m: any) => ({
    id: m.id,
    method: m.method || '',
    label: m.label || '',
    available: !!m.available,
  }))
}

export interface SiteContentBlock {
  id: string
  key: string
  placement: string
  title: string
  body: string
  sortOrder: number
}

// Fetch published site content blocks (CMS-managed content).
export async function fetchPublishedSiteContent(): Promise<SiteContentBlock[]> {
  const res = await fetch(apiUrl('/api/site-content/published'))
  if (!res.ok) throw new Error(`fetchPublishedSiteContent: ${res.status}`)
  const data = await res.json()
  return (data.items || []).map((c: any) => ({
    id: c.id,
    key: c.key,
    placement: c.placement,
    title: c.title || '',
    body: c.body || '',
    sortOrder: c.sort_order || 0,
  }))
}

// --- Order creation -------------------------------------------------------

export interface OrderInputItem {
  sku: string
  name?: string
  price?: number
  quantity: number
}

export interface OrderInput {
  items: OrderInputItem[]
  customerName: string
  email: string
  phone: string
  shippingAddress: string
  shippingMethod: string
  paymentMethod: string
  promoCode?: string
  idempotencyKey: string
}

// Server-authoritative quote result item. The server always returns
// name and price — these are required in the response.
export interface QuoteResultItem {
  sku: string
  name: string
  price: number
  quantity: number
}

// Server-authoritative quote result. The checkout UI must display these
// totals, not client-computed ones. The Go server always emits items,
// shipping_method, and payment_method in a successful response (Quote
// validates and rejects empty shipping_method/payment_method before
// returning), so they are required in this type.
export interface QuoteResult {
  items: QuoteResultItem[]
  subtotal: number
  discount: number
  shipping: number
  total: number
  promo_code?: string
  shipping_method: string
  payment_method: string
}

// Request a server-authoritative quote. The browser must call this before
// showing the final checkout total and must display these totals. The
// payment_method is validated server-side against enabled+ready methods.
export async function fetchQuote(
  items: { sku: string; quantity: number }[],
  shippingMethod: string,
  paymentMethod: string,
  promoCode?: string,
): Promise<QuoteResult> {
  const res = await fetch(apiUrl('/api/quote'), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      items: items.map((i) => ({ sku: i.sku, quantity: i.quantity })),
      shipping_method: shippingMethod,
      payment_method: paymentMethod,
      promo_code: promoCode || '',
    }),
  })
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }))
    throw new Error(err.error || `fetchQuote: ${res.status}`)
  }
  return res.json()
}

// Create an order via the Go API. Returns the created order object.
// The server recalculates totals from the catalog; client prices, shipping,
// and discount are ignored. The response includes a one-time plaintext
// access_token that the customer must save to look up their order later.
// Member identity is NOT sent — guest checkout has no MemberID. For
// member checkout, use createOrderForMember which derives MemberID from
// the authenticated principal.
export async function createOrder(input: OrderInput): Promise<{ order: any }> {
  // Convert camelCase to snake_case for the Go API
  const payload = {
    items: input.items,
    customer_name: input.customerName,
    email: input.email,
    phone: input.phone,
    shipping_address: input.shippingAddress,
    shipping_method: input.shippingMethod,
    payment_method: input.paymentMethod,
    promo_code: input.promoCode || '',
    idempotency_key: input.idempotencyKey,
  }
  const res = await fetch(apiUrl('/api/orders'), {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  })
  if (!res.ok) {
    throw new ApiRequestError(res.status, await readErrorMessage(res, 'createOrder'))
  }
  return res.json()
}

// Create an order as an authenticated member. The MemberID is derived
// from the Bearer token (the authenticated principal) — it is NOT sent
// in the request body. Requires a Bearer token.
export async function createOrderForMember(
  input: OrderInput,
  bearerToken: string,
): Promise<{ order: any }> {
  const payload = {
    items: input.items,
    customer_name: input.customerName,
    email: input.email,
    phone: input.phone,
    shipping_address: input.shippingAddress,
    shipping_method: input.shippingMethod,
    payment_method: input.paymentMethod,
    promo_code: input.promoCode || '',
    idempotency_key: input.idempotencyKey,
  }
  const res = await fetch(apiUrl('/api/orders/mine'), {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${bearerToken}`,
    },
    body: JSON.stringify(payload),
  })
  if (!res.ok) {
    throw new ApiRequestError(res.status, await readErrorMessage(res, 'createOrderForMember'))
  }
  const data = await res.json()
  // Go returns the Order object directly. Normalize to the same
  // { order } envelope the checkout consumer already requires.
  if (data && typeof data === 'object' && data.order && typeof data.order === 'object') {
    return data
  }
  return { order: data }
}

export interface ECPayLaunchForm {
  action: string
  fields: Record<string, string>
}

export async function prepareECPayPayment(orderId: string, accessToken: string): Promise<ECPayLaunchForm> {
  const res = await fetch(apiUrl(`/api/orders/${encodeURIComponent(orderId)}/payments/ecpay`), {
    method: 'POST',
    headers: { 'X-Order-Access-Token': accessToken },
  })
  if (!res.ok) {
    throw new ApiRequestError(res.status, await readErrorMessage(res, 'prepareECPayPayment'))
  }
  const data = await res.json()
  if (!data || typeof data.action !== 'string' || !data.action || !data.fields || typeof data.fields !== 'object') {
    throw new Error('綠界付款初始化回應格式不完整')
  }
  return data as ECPayLaunchForm
}

export function submitHostedPayment(form: ECPayLaunchForm): void {
  const element = document.createElement('form')
  element.method = 'POST'
  element.action = form.action
  element.style.display = 'none'
  for (const [name, value] of Object.entries(form.fields)) {
    const input = document.createElement('input')
    input.type = 'hidden'
    input.name = name
    input.value = value
    element.appendChild(input)
  }
  document.body.appendChild(element)
  element.submit()
}

// Look up a guest order by ID + opaque access token. The token is sent
// via the X-Order-Access-Token header (NOT the query string) to avoid
// logging it in URLs, browser history, and Referer headers. Returns the
// order with PII fields masked. Returns null on 404.
export async function getGuestOrder(
  orderId: string,
  accessToken: string,
): Promise<any | null> {
  const res = await fetch(apiUrl(`/api/orders/${encodeURIComponent(orderId)}`), {
    headers: { 'X-Order-Access-Token': accessToken },
  })
  if (res.status === 404) return null
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }))
    throw new Error(err.error || `getGuestOrder: ${res.status}`)
  }
  return res.json()
}

// List the authenticated member's own orders. PII is masked. Requires a
// Bearer token. Throws on 401/403/network/server error so the caller can
// show an appropriate error state and clear the session if needed.
export async function listMyOrders(
  bearerToken: string,
): Promise<unknown> {
  const res = await fetch(apiUrl('/api/orders/mine'), {
    headers: { Authorization: `Bearer ${bearerToken}` },
  })
  if (!res.ok) {
    throw new ApiRequestError(res.status, await readErrorMessage(res, 'listMyOrders'))
  }
  return res.json()
}

// Get a single order for the authenticated member by ID. Cross-user
// access returns null (404). PII is masked.
export async function getMyOrder(
  orderId: string,
  bearerToken: string,
): Promise<unknown | null> {
  const res = await fetch(apiUrl(`/api/orders/mine/${encodeURIComponent(orderId)}`), {
    headers: { Authorization: `Bearer ${bearerToken}` },
  })
  if (res.status === 404) return null
  if (!res.ok) {
    throw new ApiRequestError(res.status, await readErrorMessage(res, 'getMyOrder'))
  }
  return res.json()
}
