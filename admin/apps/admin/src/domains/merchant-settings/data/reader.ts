import type { MerchantSettingsSummary } from '../model'

export interface SettingsReader {
  read(signal?: AbortSignal): Promise<MerchantSettingsSummary>
}
