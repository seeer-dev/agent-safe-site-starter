// TODO(openapi): regenerate from starter Go API (server/internal/modules/content|contact|media).
// The types and operations below are SiteCore samples kept as a skeleton;
// they MUST be replaced by generated output from the starter OpenAPI spec.

export interface ErrorEnvelope {
  code: string
  params?: Record<string, unknown>
}

export interface OperationContract {
  method: string
  path: string
  successStatus: number
  successHasJsonBody: boolean
}

// TODO(openapi): sample domain type kept so the orders domain template typechecks.
// Replace with starter domain types (content/contact/media) when regenerating.
export interface Order {
  currency: string
  id: string
  revision: number
  status: "pending" | "paid" | "cancelled" | "refund_pending" | "refunded" | "refund_failed" | "refund_unknown"
  totalMinor: number
}

// TODO(openapi): replace with starter operations once the OpenAPI spec is generated.
export const operations = {
  "orders.cancelOrder": { method: "POST", path: "/api/admin/orders/{orderId}", successStatus: 200, successHasJsonBody: false },
  "orders.getOrder": { method: "GET", path: "/api/admin/orders/{orderId}", successStatus: 200, successHasJsonBody: true },
} as const satisfies Record<string, OperationContract>

export type OperationId = keyof typeof operations
