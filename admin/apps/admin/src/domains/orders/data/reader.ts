import type { OrderListSummary } from '../model'

/**
 * Orders list reader port.
 *
 * The query layer depends on this interface, not on any fixture or
 * transport module. The bootstrap layer injects a concrete reader
 * (fixture or remote) at app startup via Vue provide/inject.
 */
export interface OrderListReader {
  read(signal?: AbortSignal): Promise<OrderListSummary>
}
