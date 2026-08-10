import { queryOptions } from '@tanstack/vue-query'
import type { OrderListReader } from './reader'

export const orderKeys = {
  all: ['orders'] as const,
  list: () => [...orderKeys.all, 'list'] as const,
}

/**
 * Orders list query options.
 *
 * Depends only on the OrderListReader port — never on a fixture or
 * transport module directly.
 */
export function orderListQuery(reader: OrderListReader) {
  return queryOptions({
    queryKey: orderKeys.list(),
    queryFn: ({ signal }) => reader.read(signal ?? undefined),
    staleTime: 60_000,
  })
}
