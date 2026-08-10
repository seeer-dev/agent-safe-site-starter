import { queryOptions } from '@tanstack/vue-query'
import type { MessagingReader } from './reader'

export const messagingKeys = {
  all: ['messaging'] as const,
  summary: () => [...messagingKeys.all, 'summary'] as const,
}

export function messagingQuery(reader: MessagingReader) {
  return queryOptions({
    queryKey: messagingKeys.summary(),
    queryFn: ({ signal }) => reader.read(signal ?? undefined),
    staleTime: 60_000,
  })
}
