import type { InfluencerListSummary, InfluencerDetailSummary } from '../model'

const LIST_ROWS = [
  { id: 'influencer-amy', nickname: '林小美', handle: 'amy', contactEmail: 'rainy@example.com', collaborationStatus: 'active' as const, createdAt: '2026-06-01', updatedAt: '2026-07-25', email: 'rainy@example.com', activeCampaignCount: 3, lifetimeCommission: { amount: 48620, currency: 'TWD' }, payableAmount: { amount: 0, currency: 'TWD' }, paidAmount: { amount: 48620, currency: 'TWD' }, defaultDiscountCode: 'SUMMER01' },
  { id: 'influencer-kevin', nickname: '陳冠廷', handle: 'kevin', contactEmail: 'jason@example.com', collaborationStatus: 'active' as const, createdAt: '2026-06-01', updatedAt: '2026-07-25', email: 'jason@example.com', activeCampaignCount: 2, lifetimeCommission: { amount: 36480, currency: 'TWD' }, payableAmount: { amount: 0, currency: 'TWD' }, paidAmount: { amount: 36480, currency: 'TWD' }, defaultDiscountCode: 'SUMMER02' },
  { id: 'influencer-sasa', nickname: '黃大文', handle: 'sasa', contactEmail: 'david@example.com', collaborationStatus: 'active' as const, createdAt: '2026-06-01', updatedAt: '2026-07-25', email: 'david@example.com', activeCampaignCount: 2, lifetimeCommission: { amount: 28140, currency: 'TWD' }, payableAmount: { amount: 12000, currency: 'TWD' }, paidAmount: { amount: 16140, currency: 'TWD' }, defaultDiscountCode: 'SUMMER03' },
  { id: 'influencer-jay', nickname: '王怡君', handle: 'jay', contactEmail: 'beauty@example.com', collaborationStatus: 'active' as const, createdAt: '2026-06-01', updatedAt: '2026-07-25', email: 'beauty@example.com', activeCampaignCount: 1, lifetimeCommission: { amount: 18360, currency: 'TWD' }, payableAmount: { amount: 0, currency: 'TWD' }, paidAmount: { amount: 18360, currency: 'TWD' }, defaultDiscountCode: 'SUMMER04' },
  { id: 'influencer-mia', nickname: '張雅婷', handle: 'mia', contactEmail: 'mia@example.com', collaborationStatus: 'disabled' as const, createdAt: '2026-05-15', updatedAt: '2026-07-10', email: 'mia@example.com', activeCampaignCount: 0, lifetimeCommission: { amount: 9200, currency: 'TWD' }, payableAmount: { amount: 0, currency: 'TWD' }, paidAmount: { amount: 9200, currency: 'TWD' }, defaultDiscountCode: 'SPRING05' },
  { id: 'influencer-tom', nickname: '李大衛', handle: 'tom', contactEmail: 'tom@example.com', collaborationStatus: 'active' as const, createdAt: '2026-06-10', updatedAt: '2026-07-20', email: 'tom@example.com', activeCampaignCount: 1, lifetimeCommission: { amount: 7200, currency: 'TWD' }, payableAmount: { amount: 7200, currency: 'TWD' }, paidAmount: { amount: 0, currency: 'TWD' }, defaultDiscountCode: 'SUMMER06' },
  { id: 'influencer-lisa', nickname: '劉心語', handle: 'lisa', contactEmail: 'lisa@example.com', collaborationStatus: 'disabled' as const, createdAt: '2026-04-20', updatedAt: '2026-06-30', email: 'lisa@example.com', activeCampaignCount: 0, lifetimeCommission: { amount: 5400, currency: 'TWD' }, payableAmount: { amount: 0, currency: 'TWD' }, paidAmount: { amount: 5400, currency: 'TWD' }, defaultDiscountCode: null },
]

export const influencerListFixture: InfluencerListSummary = {
  items: LIST_ROWS,
  pagination: { page: 1, perPage: 100, total: 12, totalPages: 1 },
}

export const influencerDetailFixtures: Record<string, InfluencerDetailSummary> = {
  'influencer-amy': {
    profile: { id: 'influencer-amy', nickname: '林小美', handle: 'amy', contactEmail: 'rainy@example.com', collaborationStatus: 'active', createdAt: '2026-06-01', updatedAt: '2026-07-25' },
    accountStatus: 'active',
    lifetimeCommission: { amount: 48620, currency: 'TWD' },
    payableAmount: { amount: 0, currency: 'TWD' },
    paidAmount: { amount: 48620, currency: 'TWD' },
    memberships: [
      { id: 'm1', campaign: { id: 'camp-summer', name: '夏季聯名 2026' }, status: 'active', discountCode: 'SUMMER01' },
      { id: 'm2', campaign: { id: 'camp-spring', name: '春季新品推廣' }, status: 'active', discountCode: 'SPRING01' },
      { id: 'm3', campaign: { id: 'camp-winter', name: '冬季限定' }, status: 'ended', discountCode: 'WINTER01' },
    ],
  },
  'influencer-kevin': {
    profile: { id: 'influencer-kevin', nickname: '陳冠廷', handle: 'kevin', contactEmail: 'jason@example.com', collaborationStatus: 'active', createdAt: '2026-06-01', updatedAt: '2026-07-25' },
    accountStatus: 'active',
    lifetimeCommission: { amount: 36480, currency: 'TWD' },
    payableAmount: { amount: 0, currency: 'TWD' },
    paidAmount: { amount: 36480, currency: 'TWD' },
    memberships: [
      { id: 'm4', campaign: { id: 'camp-summer', name: '夏季聯名 2026' }, status: 'active', discountCode: 'SUMMER02' },
      { id: 'm5', campaign: { id: 'camp-spring', name: '春季新品推廣' }, status: 'active', discountCode: 'SPRING02' },
    ],
  },
  'influencer-sasa': {
    profile: { id: 'influencer-sasa', nickname: '黃大文', handle: 'sasa', contactEmail: 'david@example.com', collaborationStatus: 'active', createdAt: '2026-06-01', updatedAt: '2026-07-25' },
    accountStatus: 'active',
    lifetimeCommission: { amount: 28140, currency: 'TWD' },
    payableAmount: { amount: 12000, currency: 'TWD' },
    paidAmount: { amount: 16140, currency: 'TWD' },
    memberships: [
      { id: 'm6', campaign: { id: 'camp-summer', name: '夏季聯名 2026' }, status: 'active', discountCode: 'SUMMER03' },
      { id: 'm7', campaign: { id: 'camp-winter', name: '冬季限定' }, status: 'disabled', discountCode: 'WINTER03' },
    ],
  },
  'influencer-jay': {
    profile: { id: 'influencer-jay', nickname: '王怡君', handle: 'jay', contactEmail: 'beauty@example.com', collaborationStatus: 'active', createdAt: '2026-06-01', updatedAt: '2026-07-25' },
    accountStatus: 'active',
    lifetimeCommission: { amount: 18360, currency: 'TWD' },
    payableAmount: { amount: 0, currency: 'TWD' },
    paidAmount: { amount: 18360, currency: 'TWD' },
    memberships: [
      { id: 'm8', campaign: { id: 'camp-summer', name: '夏季聯名 2026' }, status: 'active', discountCode: 'SUMMER04' },
    ],
  },
}
