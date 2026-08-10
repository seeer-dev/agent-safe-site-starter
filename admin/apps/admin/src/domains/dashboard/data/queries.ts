import { queryOptions } from '@tanstack/vue-query'
import type { DashboardSummaryReader } from './reader'

export const dashboardKeys = {
  all: ['dashboard'] as const,
  summary: () => [...dashboardKeys.all, 'summary'] as const,
}

/**
 * Dashboard summary query options.
 *
 * Depends only on the DashboardSummaryReader port — never on a
 * fixture or transport module directly. The bootstrap layer
 * injects the reader via Vue provide/inject.
 */
export function dashboardSummaryQuery(reader: DashboardSummaryReader) {
  return queryOptions({
    queryKey: dashboardKeys.summary(),
    queryFn: ({ signal }) => reader.read(signal ?? undefined),
    staleTime: 60_000,
  })
}
