/**
 * Commissions & Payouts domain model — canonical types.
 *
 * Both the 分潤明細 (commissions) and 撥款紀錄 (payouts) tabs share
 * this model. The page owns tab state; the reader returns both
 * projections in one summary so a single query feeds both tabs.
 */

export type CommissionWorkflowState = 'done' | 'current' | 'pending'

export interface CommissionWorkflowStep {
  label: string
  state: CommissionWorkflowState
}

export interface CommissionRow {
  id: string
  order: string
  influencer: string
  campaign: string
  sku: string
  base: string
  commission: string
  status: string
  influencerDisplay: string
}

export interface PayoutRow {
  id: string
  date: string
  campaign: string
  influencer: string
  amount: string
  lines: number
  method: string
  note: string
  operator: string
}

export interface CommissionPayoutSummary {
  commissionRows: CommissionRow[]
  payoutRows: PayoutRow[]
  workflowSteps: CommissionWorkflowStep[]
  commissionCount: number
}
