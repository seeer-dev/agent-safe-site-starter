/**
 * Merchant settings domain model — canonical read projections for the
 * four settings sections (site, brand, payment, shipping).
 *
 * Mirrors the React source adapters' output shapes. All sections are
 * read-only: no mutation hook exists in the domain layer yet.
 */

export type SiteLifecycleStatus = 'active' | 'maintenance' | 'disabled'

export interface SiteLifecycleSettings {
  siteId: string
  siteName: string
  contactEmail: string | null
  supportPhone: string | null
  maintenanceMessage: string | null
  lifecycleStatus: SiteLifecycleStatus
}

export interface StorefrontBrandSettings {
  brandColorToken: string
  logoStorageReference: string | null
}

export interface PaymentFeeBasis {
  id: string
  method: string
  percentageFeeBps: number
  fixedFee: { amount: number; currency: string }
}

export interface PaymentMethodItem {
  id: string
  providerType: string
  enabled: boolean
  feeBasis: PaymentFeeBasis[]
}

export interface PaymentMethodSettings {
  items: PaymentMethodItem[]
}

export interface ShippingRuleItem {
  id: string
  name: string
  feeAmount: { amount: number; currency: string }
  enabled: boolean
}

export interface ShippingRuleSettings {
  items: ShippingRuleItem[]
}

export interface MerchantSettingsSummary {
  site: SiteLifecycleSettings
  brand: StorefrontBrandSettings
  payment: PaymentMethodSettings
  shipping: ShippingRuleSettings
}

export type SettingsTab = 'site' | 'brand' | 'payment' | 'shipping'

export const SETTINGS_TABS: readonly SettingsTab[] = Object.freeze([
  'site',
  'brand',
  'payment',
  'shipping',
])

export function isSettingsTab(value: string | null | undefined): value is SettingsTab {
  return value !== null && value !== undefined && SETTINGS_TABS.includes(value as SettingsTab)
}
