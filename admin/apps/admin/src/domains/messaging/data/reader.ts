import type { MessagingSummary } from '../model'

export interface MessagingReader {
  read(signal?: AbortSignal): Promise<MessagingSummary>
}
