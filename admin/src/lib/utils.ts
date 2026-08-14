/** Format number as NT$ — e.g. NT$1,880 */
export function formatNT(n: number | string): string {
  return 'NT$' + Number(n).toLocaleString()
}

/** Format a unix timestamp (seconds) as a zh-TW locale string. */
export function formatUnix(unix: number | string | null | undefined): string {
  if (unix == null || unix === '' || Number(unix) === 0) return ''
  return new Date(Number(unix) * 1000).toLocaleString('zh-TW')
}

/** Escape HTML — not needed in Vue but kept for parity */
export function escapeHtml(s: unknown): string {
  return String(s == null ? '' : s)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}

/**
 * Evaluate a showWhen expression. Supports:
 *   - "key=val1|val2" — row[key] matches any of the pipe-separated values.
 *   - "approve_always" — special: always returns true. The approve action
 *     is always shown to users with content.approve capability, because
 *     an approval can be missing, stale (version mismatch), OR expired
 *     (version matches but approved_expiry_unix has passed). Hiding
 *     approve when the version matches would trap the operator: they
 *     could see publish, get a 409 (expired), and have no way to re-approve.
 *     The server is the final authority on whether the approve is a no-op
 *     or a re-approval.
 *   - "publish_ready" — special: draft has a current, non-expired approval
 *     (approved_version === draft_version AND approver_user_id is non-empty
 *     AND approved_expiry_unix > now). This avoids offering an obviously-
 *     failing publish when no valid approval exists. The server still
 *     enforces all conditions; this is purely UX guidance.
 */
export function evalShowWhen(expr: string | undefined, row: Record<string, any>): boolean {
  if (!expr) return true
  // Special compound expressions for approval workflow.
  if (expr === 'approve_always') {
    return true
  }
  if (expr === 'publish_ready') {
    const dv = Number(row.draft_version ?? 0)
    const av = Number(row.approved_version ?? 0)
    const approver = String(row.approver_user_id ?? '').trim()
    const expiry = Number(row.approved_expiry_unix ?? 0)
    const now = Math.floor(Date.now() / 1000)
    return dv === av && dv > 0 && approver !== '' && expiry > now
  }
  // Standard key=value|value expression.
  const i = expr.indexOf('=')
  if (i < 0) return true
  const k = expr.slice(0, i)
  const vals = expr.slice(i + 1).split('|')
  return vals.indexOf(String(row[k] == null ? '' : row[k])) >= 0
}

/** Simple class name combiner (no dependency on clsx) */
export function cn(...classes: (string | false | null | undefined)[]): string {
  return classes.filter(Boolean).join(' ')
}
