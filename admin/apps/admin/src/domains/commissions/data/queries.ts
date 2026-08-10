import { queryOptions } from '@tanstack/vue-query'
import type { CommissionPayoutReader } from './reader'

export const commissionKeys = {
  all: ['commissions'] as const,
  summary: () => [...commissionKeys.all, 'summary'] as const,
}

/**
 * Commissions & Payouts summary query options.
 *
 * Depends only on the CommissionPayoutReader port.
 */
export function commissionPayoutQuery(reader: CommissionPayoutReader) {
  return queryOptions({
    queryKey: commissionKeys.summary(),
    queryFn: ({ signal }) => reader.read(signal ?? undefined),
    staleTime: 60_000,
  })
}
