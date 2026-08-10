import { inject } from 'vue'
import type { InjectionKey } from 'vue'
import type { DashboardSummaryReader } from '../domains/dashboard/data/reader'
import type { OrderListReader } from '../domains/orders/data/reader'
import type { CommissionPayoutReader } from '../domains/commissions/data/reader'
import type { OrderDetailReader } from '../domains/order-detail/data/reader'
import type { MessagingReader } from '../domains/messaging/data/reader'
import type { ContentListReader } from '../domains/content/data/reader'
import type { SettingsReader } from '../domains/merchant-settings/data/reader'
import type { InfluencerListReader, InfluencerDetailReader } from '../domains/influencers/data/reader'

/**
 * AdminRuntime — the immutable runtime contract injected at bootstrap.
 *
 * Contains typed reader ports for each domain. The bootstrap layer
 * (main.ts) imports a concrete composition (e.g. runtime.fixture.ts)
 * that creates this record and provides it via
 * app.provide(ADMIN_RUNTIME_KEY, runtime). Pages and composables
 * obtain it through useAdminRuntime() — they never import fixture
 * modules directly.
 *
 * This module MUST NOT import fixture data. Only the contract
 * (types + key + fail-fast accessor) lives here so that a future
 * live entry importing this file does not pull fixture into the
 * bundle.
 *
 * This is NOT a DI container or service locator. It is a simple
 * immutable record of typed ports, created once at startup.
 */
export interface AdminRuntime {
  readonly dashboardSummaryReader: DashboardSummaryReader
  readonly orderListReader: OrderListReader
  readonly commissionPayoutReader: CommissionPayoutReader
  readonly orderDetailReader: OrderDetailReader
  readonly messagingReader: MessagingReader
  readonly contentListReader: ContentListReader
  readonly settingsReader: SettingsReader
  readonly influencerListReader: InfluencerListReader
  readonly influencerDetailReader: InfluencerDetailReader
}

export const ADMIN_RUNTIME_KEY: InjectionKey<AdminRuntime> = Symbol('admin-runtime')

/**
 * Fail-fast accessor for the AdminRuntime. Pages and composables
 * call this instead of inject(...)! to get a typed runtime with a
 * clear error message if the bootstrap layer forgot to provide it.
 */
export function useAdminRuntime(): AdminRuntime {
  const runtime = inject(ADMIN_RUNTIME_KEY)
  if (!runtime) {
    throw new Error(
      'AdminRuntime not provided. The bootstrap layer (main.ts) must ' +
        'call app.provide(ADMIN_RUNTIME_KEY, createFixtureRuntime()) ' +
        'before mounting.',
    )
  }
  return runtime
}
