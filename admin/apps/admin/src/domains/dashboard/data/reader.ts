import type { DashboardSummary } from '../model'

/**
 * Dashboard summary reader port.
 *
 * The query layer depends on this interface, not on any fixture or
 * transport module. The bootstrap layer injects a concrete reader
 * (fixture or remote) at app startup via Vue provide/inject. This
 * keeps the fixture/remote switch at the runtime boundary, not
 * inside the query or page.
 */
export interface DashboardSummaryReader {
  read(signal?: AbortSignal): Promise<DashboardSummary>
}
