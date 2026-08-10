import type { MerchantSettingsSummary } from '../model'

/**
 * Fixture-backed merchant settings seed.
 *
 * Matches the shape the React source adapters produce. All values are
 * read-only projections — no mutation hook exists, so the settings route
 * renders disabled controls with a readonly notice.
 */
export const merchantSettingsFixture: MerchantSettingsSummary = {
  site: {
    siteId: 'reference',
    siteName: '無袖小花示範店',
    contactEmail: 'contact@example.com',
    supportPhone: '02-1234-5678',
    maintenanceMessage: null,
    lifecycleStatus: 'active',
  },
  brand: {
    brandColorToken: '#2563eb',
    logoStorageReference: null,
  },
  payment: {
    items: [
      {
        id: 'provider-card',
        providerType: 'card',
        enabled: true,
        feeBasis: [
          {
            id: 'fee-card',
            method: 'credit_card',
            percentageFeeBps: 200,
            fixedFee: { amount: 0, currency: 'TWD' },
          },
        ],
      },
      {
        id: 'provider-atm',
        providerType: 'atm',
        enabled: true,
        feeBasis: [
          {
            id: 'fee-atm',
            method: 'atm_transfer',
            percentageFeeBps: 0,
            fixedFee: { amount: 15, currency: 'TWD' },
          },
        ],
      },
      {
        id: 'provider-cvs',
        providerType: 'cvs',
        enabled: false,
        feeBasis: [
          {
            id: 'fee-cvs',
            method: 'cvs_pay',
            percentageFeeBps: 0,
            fixedFee: { amount: 30, currency: 'TWD' },
          },
        ],
      },
    ],
  },
  shipping: {
    items: [
      {
        id: 'rule-home',
        name: '宅配',
        feeAmount: { amount: 120, currency: 'TWD' },
        enabled: true,
      },
      {
        id: 'rule-cvs',
        name: '超商取貨',
        feeAmount: { amount: 60, currency: 'TWD' },
        enabled: true,
      },
      {
        id: 'rule-intl',
        name: '國際配送',
        feeAmount: { amount: 500, currency: 'TWD' },
        enabled: false,
      },
    ],
  },
}
