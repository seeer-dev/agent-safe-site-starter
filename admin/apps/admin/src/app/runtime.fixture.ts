import type { AdminRuntime } from './runtime'
import type { DashboardSummaryReader } from '../domains/dashboard/data/reader'
import type { OrderListReader } from '../domains/orders/data/reader'
import type { CommissionPayoutReader } from '../domains/commissions/data/reader'
import type { OrderDetailReader } from '../domains/order-detail/data/reader'
import type { MessagingReader } from '../domains/messaging/data/reader'
import type { ContentListReader } from '../domains/content/data/reader'
import type { SettingsReader } from '../domains/merchant-settings/data/reader'
import type { InfluencerListReader, InfluencerDetailReader } from '../domains/influencers/data/reader'
import type { DashboardSummary } from '../domains/dashboard/model'
import type { OrderListSummary } from '../domains/orders/model'
import type { CommissionPayoutSummary } from '../domains/commissions/model'
import type { OrderDetailSummary } from '../domains/order-detail/model'
import type { MessagingSummary } from '../domains/messaging/model'
import type { ContentListSummary } from '../domains/content/model'
import type { MerchantSettingsSummary } from '../domains/merchant-settings/model'
import type { InfluencerListSummary, InfluencerDetailSummary } from '../domains/influencers/model'
import { dashboardFixture } from '../domains/dashboard/data/fixture'
import { orderListFixture } from '../domains/orders/data/fixture'
import { commissionPayoutFixture } from '../domains/commissions/data/fixture'
import { orderDetailFixture } from '../domains/order-detail/data/fixture'
import { messagingFixture } from '../domains/messaging/data/fixture'
import { contentListFixture } from '../domains/content/data/fixture'
import { merchantSettingsFixture } from '../domains/merchant-settings/data/fixture'
import { influencerListFixture, influencerDetailFixtures } from '../domains/influencers/data/fixture'

/**
 * Fixture-backed AdminRuntime composition.
 *
 * This is the only module that imports fixture data. main.ts
 * imports this to create the runtime for the fixture/preview entry.
 * A future live entry imports a different composition (e.g.
 * runtime.live.ts) that provides real transport-backed readers,
 * without changing any downstream consumer.
 */
export function createFixtureRuntime(): AdminRuntime {
  const dashboardReader: DashboardSummaryReader = {
    async read(): Promise<DashboardSummary> {
      return dashboardFixture
    },
  }
  const orderReader: OrderListReader = {
    async read(): Promise<OrderListSummary> {
      return orderListFixture
    },
  }
  const commissionReader: CommissionPayoutReader = {
    async read(): Promise<CommissionPayoutSummary> {
      return commissionPayoutFixture
    },
  }
  const orderDetailReader: OrderDetailReader = {
    async read(): Promise<OrderDetailSummary> {
      return orderDetailFixture
    },
  }
  const messagingReader: MessagingReader = {
    async read(): Promise<MessagingSummary> {
      return messagingFixture
    },
  }
  const contentListReader: ContentListReader = {
    async read(): Promise<ContentListSummary> {
      return contentListFixture
    },
  }
  const settingsReader: SettingsReader = {
    async read(): Promise<MerchantSettingsSummary> {
      return merchantSettingsFixture
    },
  }
  const influencerListReader: InfluencerListReader = {
    async read(): Promise<InfluencerListSummary> {
      return influencerListFixture
    },
  }
  const influencerDetailReader: InfluencerDetailReader = {
    async read(influencerId: string): Promise<InfluencerDetailSummary> {
      const fixture = influencerDetailFixtures[influencerId]
      if (!fixture) throw new Error(`not_found: influencer ${influencerId}`)
      return fixture
    },
  }
  return {
    dashboardSummaryReader: dashboardReader,
    orderListReader: orderReader,
    commissionPayoutReader: commissionReader,
    orderDetailReader,
    messagingReader,
    contentListReader,
    settingsReader,
    influencerListReader,
    influencerDetailReader,
  }
}
