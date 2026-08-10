import type { CommissionPayoutSummary, CommissionRow, CommissionWorkflowStep, PayoutRow } from '../model'

/* Fixture data for Commissions & Payouts. This module is imported
 * only by the bootstrap/runtime composition layer. */

const COMMISSION_ROWS: CommissionRow[] = [
  { id: 'cm-1', order: '#TW-20874', influencer: '林小美', campaign: '夏季聯名', sku: 'TS-SU-WM', base: '$2,560', commission: '$384', status: '可撥款', influencerDisplay: '待撥款' },
  { id: 'cm-2', order: '#TW-20871', influencer: 'Kevin Lu', campaign: '限時折扣', sku: 'BAG-01', base: '$920', commission: '$80', status: '可撥款', influencerDisplay: '待撥款' },
  { id: 'cm-3', order: '#TW-20866', influencer: 'SASA', campaign: '夏季聯名', sku: 'TS-SU-WM', base: '$6,010', commission: '$902', status: '結案保留', influencerDisplay: '結案保留中' },
]

const PAYOUT_ROWS: PayoutRow[] = [
  { id: 'pay-1', date: '07-24', campaign: '夏季聯名', influencer: 'Kevin Lu', amount: '$5,800', lines: 9, method: '銀行轉帳', note: '末五碼 34112', operator: '陳老闆' },
  { id: 'pay-2', date: '07-20', campaign: '限時折扣', influencer: '林小美', amount: '$3,200', lines: 5, method: '銀行轉帳', note: '末五碼 88201', operator: '陳老闆' },
]

const WORKFLOW_STEPS: CommissionWorkflowStep[] = [
  { label: '預估', state: 'done' },
  { label: '已鎖定', state: 'done' },
  { label: '結案保留', state: 'done' },
  { label: '可撥款', state: 'current' },
  { label: '已撥款', state: 'pending' },
]

export const commissionPayoutFixture: CommissionPayoutSummary = {
  commissionRows: COMMISSION_ROWS,
  payoutRows: PAYOUT_ROWS,
  workflowSteps: WORKFLOW_STEPS,
  commissionCount: COMMISSION_ROWS.length,
}
