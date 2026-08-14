export type Category = 'all' | 'apparel' | 'accessories' | 'home' | 'stationery'

export interface ProductReview {
  id: string
  author: string
  rating: number
  title: string
  body: string
  date: string
  verified: boolean
}

export interface ProductOption {
  label: string
  value: string
  swatch?: string
  stock?: number
}

export interface Product {
  id: string
  name: string
  slug: string
  description: string
  longDescription?: string
  price: number
  originalPrice?: number
  image: string
  images: string[]
  category: Category
  tag?: string
  rating: number
  reviews: number
  ratingBreakdown: { stars: number; count: number }[]
  reviewList: ProductReview[]
  sizes?: ProductOption[]
  colors?: ProductOption[]
  stock: number
  sku: string
  material?: string
  origin?: string
}

export interface CartItem {
  product: Product
  quantity: number
  selectedSize?: string
  selectedColor?: string
}

export type OrderStatus = 'pending' | 'processing' | 'shipped' | 'delivered' | 'cancelled'

export interface OrderItem {
  productId: string
  name: string
  image: string
  price: number
  quantity: number
  size?: string
  color?: string
}

export interface OrderTimelineEntry {
  status: OrderStatus
  label: string
  description: string
  timestamp: number
  future?: boolean
}

export interface Address {
  name: string
  email?: string
  phone?: string
  address: string
  city: string
  zip: string
  country: string
}

export interface Order {
  id: string
  userEmail: string | null
  items: OrderItem[]
  subtotal: number
  discount: number
  shipping: number
  tax: number
  total: number
  status: OrderStatus
  shippingAddress: Address
  shippingMethod: string
  paymentMethod: string
  trackingNumber?: string
  placedAt: number
  timeline: OrderTimelineEntry[]
  // One-time plaintext access token returned at order creation. The
  // customer must save this to look up their guest order later via
  // X-Order-Access-Token header. Not present in subsequent responses.
  accessToken?: string
}

export interface User {
  id?: string
  email: string
  name: string
  phone?: string
  joinedAt: number | null
  defaultAddress?: Address
  stats?: {
    totalOrders: number
    totalSpent: number
    rewardPoints: number
    memberLevel: 'bronze' | 'silver' | 'gold'
  }
}

export type FooterPageKey = 'privacy' | 'terms' | 'shipping' | 'contact' | 'about' | 'faq'
