import type { OrderDetailSummary } from '../model'

/**
 * Order detail reader port.
 *
 * The query layer depends on this interface, not on any fixture or
 * transport module. The bootstrap layer injects a concrete reader
 * (fixture or remote) at app startup via Vue provide/inject.
 */
export interface OrderDetailReader {
  read(orderId: string, signal?: AbortSignal): Promise<OrderDetailSummary>
}
