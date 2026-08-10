/**
 * Messaging domain model — SMTP settings, template list, and send
 * history (attempts). Three tabs share one page shell.
 */

export interface MessagingSmtpSettings {
  host: string
  port: string
  username: string
  fromName: string
  fromEmail: string
  encryption: 'ssl' | 'tls' | 'none'
}

export interface MessagingTemplate {
  id: string
  event: string
  name: string
  sender: string
  enabled: boolean
}

export interface MessagingAttempt {
  id: string
  time: string
  event: string
  recipient: string
  status: string
  statusTone: 'success' | 'wait' | 'danger'
}

export interface MessagingSummary {
  smtp: MessagingSmtpSettings
  templates: MessagingTemplate[]
  attempts: MessagingAttempt[]
}
