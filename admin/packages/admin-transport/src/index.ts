// TODO(transport): stub skeleton. Methods throw `transport.not_wired` until
// wired to the starter Go API (server/internal/modules/content|contact|media).
// The TransportError + request() helpers are kept as a reference implementation;
// replace adminTransport methods with real operations when api-client is regenerated.

import type { ErrorEnvelope as ApiError, OperationContract, Order } from '@sitecore/api-client'

export type OrderDTO = Order

export class TransportError extends Error {
  constructor(
    public status: number,
    public code: string,
  ) {
    super(code)
  }
}

export function resolvePath(operation: OperationContract, params: Record<string, string>): string {
  return operation.path.replace(/\{([^}]+)\}/g, (_, name: string) => {
    const value = params[name]
    if (value === undefined) throw new Error(`missing path parameter: ${name}`)
    return encodeURIComponent(value)
  })
}

export async function request<T>(
  operation: OperationContract,
  pathParams: Record<string, string>,
  init: RequestInit = {},
): Promise<T> {
  const response = await fetch(resolvePath(operation, pathParams), {
    ...init,
    method: operation.method,
    headers: {
      Accept: 'application/json',
      ...init.headers,
    },
  })
  if (!response.ok) {
    const body = await response.json().catch((): ApiError => ({ code: 'network.http_error' })) as ApiError
    throw new TransportError(response.status, body.code ?? 'network.http_error')
  }
  if (!operation.successHasJsonBody) return undefined as T
  return response.json() as Promise<T>
}

export const adminTransport = {
  // TODO(transport): wire to starter Go API. Currently stubs so the orders
  // domain template typechecks and fixture mode runs without a backend.
  getOrder: (_: { siteId: string; id: string; signal?: AbortSignal }): Promise<Order> => {
    throw new TransportError(501, 'transport.not_wired')
  },
  cancelOrder: (_: {
    siteId: string
    id: string
    expectedRevision: number
    commandId: string
  }): Promise<void> => {
    throw new TransportError(501, 'transport.not_wired')
  },
}

// Reference implementation kept for reuse when wiring real operations:
// request<Order>(operations['orders.getOrder'], { orderId: id }, { signal, headers: { 'X-Effective-Site': siteId } })
// request<void>(operations['orders.cancelOrder'], { orderId: id }, { headers: { 'Content-Type': 'application/json', 'X-Effective-Site': siteId }, body: JSON.stringify({ expectedRevision, commandId }) })
