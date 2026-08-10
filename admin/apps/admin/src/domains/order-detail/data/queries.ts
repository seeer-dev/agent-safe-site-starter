import { queryOptions } from '@tanstack/vue-query'
import type { OrderDetailReader } from './reader'

export const orderDetailKeys = {
  all: ['order-detail'] as const,
  detail: (orderId: string) => [...orderDetailKeys.all, orderId] as const,
}

/**
 * Order detail query options.
 *
 * Depends only on the OrderDetailReader port.
 */
export function orderDetailQuery(reader: OrderDetailReader, orderId: string) {
  return queryOptions({
    queryKey: orderDetailKeys.detail(orderId),
    queryFn: ({ signal }) => reader.read(orderId, signal ?? undefined),
    staleTime: 60_000,
  })
}
