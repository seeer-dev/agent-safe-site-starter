import { queryOptions } from '@tanstack/vue-query'
import type { InfluencerListReader, InfluencerDetailReader } from './reader'

export const influencerKeys = {
  all: ['influencers'] as const,
  list: () => [...influencerKeys.all, 'list'] as const,
  detail: (id: string) => [...influencerKeys.all, 'detail', id] as const,
}

export function influencerListQuery(reader: InfluencerListReader) {
  return queryOptions({
    queryKey: influencerKeys.list(),
    queryFn: ({ signal }) => reader.read(signal ?? undefined),
    staleTime: 30_000,
  })
}

export function influencerDetailQuery(reader: InfluencerDetailReader, influencerId: string) {
  return queryOptions({
    queryKey: influencerKeys.detail(influencerId),
    queryFn: ({ signal }) => reader.read(influencerId, signal ?? undefined),
    staleTime: 30_000,
  })
}
