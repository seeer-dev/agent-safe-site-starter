/**
 * Influencer domain model — list and detail viewmodels.
 *
 * Mirrors the React source adapters' canonical read projections.
 * All data is read-only; no mutation hook exists in the Vue port yet.
 */

export type InfluencerStatus = 'active' | 'disabled'
export type MembershipStatus = 'active' | 'disabled' | 'ended'

export interface InfluencerProfile {
  id: string
  nickname: string
  handle: string
  contactEmail: string | null
  collaborationStatus: InfluencerStatus
  createdAt: string
  updatedAt: string
}

export interface InfluencerListRow extends InfluencerProfile {
  email: string | null
  activeCampaignCount: number
  lifetimeCommission: { amount: number; currency: string }
  payableAmount: { amount: number; currency: string }
  paidAmount: { amount: number; currency: string }
  defaultDiscountCode: string | null
}

export interface InfluencerListSummary {
  items: InfluencerListRow[]
  pagination: { page: number; perPage: number; total: number; totalPages: number }
}

export interface InfluencerMembership {
  id: string
  campaign: { id: string; name: string }
  status: MembershipStatus
  discountCode: string
}

export interface InfluencerDetailSummary {
  profile: InfluencerProfile
  accountStatus: string
  lifetimeCommission: { amount: number; currency: string }
  payableAmount: { amount: number; currency: string }
  paidAmount: { amount: number; currency: string }
  memberships: InfluencerMembership[]
}

export function influencerStatusTone(status: string): 'active' | 'draft' | 'ended' {
  if (status === 'active') return 'active'
  if (status === 'disabled') return 'ended'
  return 'draft'
}

export const INFLUENCER_SAVED_VIEWS = Object.freeze([
  { id: 'all' as const },
  { id: 'active' as const, status: 'active' as const },
  { id: 'disabled' as const, status: 'disabled' as const },
])

export interface InfluencerTab {
  readonly id: string
  readonly label: string
  readonly enabled: boolean
}
