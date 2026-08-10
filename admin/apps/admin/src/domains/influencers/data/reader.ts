import type { InfluencerListSummary, InfluencerDetailSummary } from '../model'

export interface InfluencerListReader {
  read(signal?: AbortSignal): Promise<InfluencerListSummary>
}

export interface InfluencerDetailReader {
  read(influencerId: string, signal?: AbortSignal): Promise<InfluencerDetailSummary>
}
