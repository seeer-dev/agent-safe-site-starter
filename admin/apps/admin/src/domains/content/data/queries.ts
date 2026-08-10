import { queryOptions } from '@tanstack/vue-query'
import type { ContentListReader } from './reader'

export const contentKeys = {
  all: ['content'] as const,
  list: () => [...contentKeys.all, 'list'] as const,
}

export function contentListQuery(reader: ContentListReader) {
  return queryOptions({
    queryKey: contentKeys.list(),
    queryFn: ({ signal }) => reader.read(signal ?? undefined),
    staleTime: 60_000,
  })
}
