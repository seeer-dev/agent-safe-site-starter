import type { OrderStatus } from '@/shared/lib/types'

const ORDER_STATUSES = new Set<OrderStatus>([
  'pending',
  'processing',
  'shipped',
  'delivered',
  'cancelled',
])

export class MalformedMemberOrdersError extends Error {
  constructor() {
    super('malformed_member_orders')
    this.name = 'MalformedMemberOrdersError'
  }
}

export interface MemberOrderItem {
  sku: string
  name: string
  price: number
  quantity: number
}

export interface MemberOrder {
  id: string
  status: OrderStatus
  subtotal: number
  discount: number
  shipping: number
  total: number
  updatedUnix: number
  items: MemberOrderItem[]
  email: string | null
  customerName: string | null
  shippingAddress: string | null
  shippingMethod: string | null
  paymentMethod: string | null
}

function requireNonEmptyString(value: unknown): string {
  if (typeof value !== 'string' || value === '') {
    throw new MalformedMemberOrdersError()
  }
  return value
}

function requireNonNegativeAmount(value: unknown): number {
  if (typeof value !== 'number' || !Number.isFinite(value) || value < 0) {
    throw new MalformedMemberOrdersError()
  }
  return value
}

function requirePositiveInteger(value: unknown): number {
  if (typeof value !== 'number' || !Number.isInteger(value) || value <= 0) {
    throw new MalformedMemberOrdersError()
  }
  return value
}

function requirePositiveUnix(value: unknown): number {
  if (typeof value !== 'number' || !Number.isInteger(value) || value <= 0) {
    throw new MalformedMemberOrdersError()
  }
  return value
}

function optionalString(value: unknown): string | null {
  return typeof value === 'string' && value !== '' ? value : null
}

export function parseMemberOrder(raw: unknown): MemberOrder {
  if (!raw || typeof raw !== 'object') {
    throw new MalformedMemberOrdersError()
  }
  const o = raw as Record<string, unknown>
  if (typeof o.status !== 'string' || !ORDER_STATUSES.has(o.status as OrderStatus)) {
    throw new MalformedMemberOrdersError()
  }
  if (!Array.isArray(o.items)) {
    throw new MalformedMemberOrdersError()
  }
  const items = o.items.map((item) => {
    if (!item || typeof item !== 'object') {
      throw new MalformedMemberOrdersError()
    }
    const row = item as Record<string, unknown>
    if (typeof row.name !== 'string') {
      throw new MalformedMemberOrdersError()
    }
    return {
      sku: requireNonEmptyString(row.sku),
      name: row.name,
      price: requireNonNegativeAmount(row.price),
      quantity: requirePositiveInteger(row.quantity),
    }
  })
  return {
    id: requireNonEmptyString(o.id),
    status: o.status as OrderStatus,
    subtotal: requireNonNegativeAmount(o.subtotal),
    discount: requireNonNegativeAmount(o.discount),
    shipping: requireNonNegativeAmount(o.shipping),
    total: requireNonNegativeAmount(o.total),
    updatedUnix: requirePositiveUnix(o.updated_unix),
    items,
    email: optionalString(o.email),
    customerName: optionalString(o.customer_name),
    shippingAddress: optionalString(o.shipping_address),
    shippingMethod: optionalString(o.shipping_method),
    paymentMethod: optionalString(o.payment_method),
  }
}

export function parseMemberOrderListEnvelope(raw: unknown): MemberOrder[] {
  if (!raw || typeof raw !== 'object' || Array.isArray(raw)) {
    throw new MalformedMemberOrdersError()
  }
  const orders = (raw as { orders?: unknown }).orders
  if (!Array.isArray(orders)) {
    throw new MalformedMemberOrdersError()
  }
  return orders.map(parseMemberOrder)
}
