import type { CommissionPayoutSummary } from '../model'

/**
 * Commissions & Payouts reader port.
 *
 * The query layer depends on this interface, not on any fixture or
 * transport module. The bootstrap layer injects a concrete reader
 * (fixture or remote) at app startup via Vue provide/inject.
 */
export interface CommissionPayoutReader {
  read(signal?: AbortSignal): Promise<CommissionPayoutSummary>
}
