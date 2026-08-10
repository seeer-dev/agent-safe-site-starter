import { queryOptions } from '@tanstack/vue-query'
import type { SettingsReader } from './reader'

export const merchantSettingsKeys = {
  all: ['merchant-settings'] as const,
  summary: () => [...merchantSettingsKeys.all, 'summary'] as const,
}

export function merchantSettingsQuery(reader: SettingsReader) {
  return queryOptions({
    queryKey: merchantSettingsKeys.summary(),
    queryFn: ({ signal }) => reader.read(signal ?? undefined),
    staleTime: 60_000,
  })
}
