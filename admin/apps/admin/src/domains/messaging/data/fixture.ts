import type { MessagingAttempt, MessagingSummary, MessagingSmtpSettings, MessagingTemplate } from '../model'

const SMTP: MessagingSmtpSettings = {
  host: 'smtp.example.com',
  port: '587',
  username: 'noreply@example.com',
  fromName: '網紅分潤後台',
  fromEmail: 'noreply@example.com',
  encryption: 'tls',
}

const TEMPLATES: MessagingTemplate[] = [
  { id: 't1', event: '新訂單成立', name: '訂單確認信', sender: '系統', enabled: true },
  { id: 't2', event: '訂單出貨', name: '出貨通知', sender: '系統', enabled: true },
  { id: 't3', event: '分潤撥款', name: '撥款通知', sender: '系統', enabled: false },
]

const ATTEMPTS: MessagingAttempt[] = [
  { id: 'a1', time: '2025-01-14 10:22', event: '新訂單成立', recipient: 'amy@example.com', status: '已送達', statusTone: 'success' },
  { id: 'a2', time: '2025-01-14 09:15', event: '退款完成', recipient: 'linmei@example.com', status: '已送達', statusTone: 'success' },
  { id: 'a3', time: '2025-01-13 18:40', event: '訂單出貨', recipient: 'ken@example.com', status: '已送達', statusTone: 'success' },
]

export const messagingFixture: MessagingSummary = {
  smtp: SMTP,
  templates: TEMPLATES,
  attempts: ATTEMPTS,
}
