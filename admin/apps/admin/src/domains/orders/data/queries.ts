import { queryOptions } from '@tanstack/vue-query'
import { adminTransport, TransportError } from '@sitecore/admin-transport'
import { orderKeys } from './query-keys'

export function orderQuery(siteId: string, id: string) {
  return queryOptions({
    queryKey: orderKeys.detail(siteId, id),
    queryFn: ({ signal }) => adminTransport.getOrder({ siteId, id, signal }),
    staleTime: 30_000,
    retry: (count, error) => error instanceof TransportError && error.status >= 500 && count < 2,
  })
}
