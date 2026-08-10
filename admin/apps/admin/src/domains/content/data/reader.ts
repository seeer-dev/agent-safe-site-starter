import type { ContentListSummary } from '../model'

export interface ContentListReader {
  read(signal?: AbortSignal): Promise<ContentListSummary>
}
