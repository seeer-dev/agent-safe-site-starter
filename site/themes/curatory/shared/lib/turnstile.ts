// Managed Cloudflare Turnstile integration for public side-effecting
// forms (contact / comment / order). The Go origin verifies the token
// against Siteverify before any persistence, mail, or stock mutation —
// this file only mints the token and attaches it to the request.
//
// The public site key arrives via /api/storefront/bootstrap
// (turnstile_site_key). When it is empty — local development without a
// configured widget — the composable stays disabled and the server-side
// permissive verifier accepts the submission. In production the key is
// always configured, so a missing token blocks the submit client-side
// and the server would refuse it anyway.
import { onBeforeUnmount, onMounted, ref, type ComponentPublicInstance, type Ref } from 'vue'
import { bootstrap, loadBootstrap } from './store'

const SCRIPT_URL = 'https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit'

interface TurnstileRenderOptions {
  sitekey: string
  action?: string
  theme?: 'light' | 'dark' | 'auto'
  callback?: (token: string) => void
  'expired-callback'?: () => void
  'error-callback'?: () => void
}

interface TurnstileApi {
  render(el: HTMLElement, opts: TurnstileRenderOptions): string
  reset(id?: string): void
  remove(id?: string): void
}

declare global {
  interface Window {
    turnstile?: TurnstileApi
  }
}

let scriptPromise: Promise<void> | null = null

function ensureScript(): Promise<void> {
  if (window.turnstile) return Promise.resolve()
  if (!scriptPromise) {
    scriptPromise = new Promise((resolve, reject) => {
      const el = document.createElement('script')
      el.src = SCRIPT_URL
      el.async = true
      el.defer = true
      el.onload = () => resolve()
      el.onerror = () => {
        scriptPromise = null
        reject(new Error('turnstile script failed to load'))
      }
      document.head.appendChild(el)
    })
  }
  return scriptPromise
}

export interface TurnstileSlot {
  /** Function ref for the mount target: `<div :ref="ts.setEl" />`. Always
   *  render the element; the widget appears inside only when a site key
   *  is configured. */
  setEl(el: Element | ComponentPublicInstance | null): void
  /** true once bootstrap/script resolved — safe to attempt submit. */
  ready: Ref<boolean>
  /** true when a site key exists and the widget is live. */
  enabled: Ref<boolean>
  /**
   * Token for the next submit. Returns the minted token, or null when the
   * widget is enabled but not yet solved — the caller must block. Returns
   * "" when the widget is disabled (dev mode); the field is then omitted.
   */
  tokenForSubmit(): string | null
  /** Reset after every submit attempt that consumed a token — Turnstile
   *  tokens are single-use, so a follow-up submit needs a fresh one. */
  reset(): void
}

export function useTurnstile(action: string): TurnstileSlot {
  const widgetEl = ref<HTMLElement | null>(null)
  const token = ref('')
  const ready = ref(false)
  const enabled = ref(false)
  let widgetId: string | null = null

  onMounted(async () => {
    try {
      await loadBootstrap()
    } catch {
      // Bootstrap failure leaves the widget disabled; the server still
      // enforces its side, so the submission simply refuses upstream.
      ready.value = true
      return
    }
    const sitekey = (bootstrap.data?.turnstile_site_key ?? '').trim()
    if (!sitekey || !widgetEl.value) {
      ready.value = true
      return
    }
    try {
      await ensureScript()
      if (!widgetEl.value || !window.turnstile) return
      widgetId = window.turnstile.render(widgetEl.value, {
        sitekey,
        action,
        theme: 'auto',
        callback: (t) => {
          token.value = t
        },
        'expired-callback': () => {
          token.value = ''
        },
        'error-callback': () => {
          token.value = ''
        },
      })
      enabled.value = true
    } finally {
      ready.value = true
    }
  })

  onBeforeUnmount(() => {
    if (widgetId !== null && window.turnstile) {
      try {
        window.turnstile.remove(widgetId)
      } catch {
        /* widget already gone */
      }
    }
  })

  function tokenForSubmit(): string | null {
    if (!enabled.value) return ''
    return token.value || null
  }

  function reset(): void {
    token.value = ''
    if (widgetId !== null && window.turnstile) {
      try {
        window.turnstile.reset(widgetId)
      } catch {
        /* widget gone */
      }
    }
  }

  function setEl(el: Element | ComponentPublicInstance | null): void {
    widgetEl.value = el instanceof HTMLElement ? el : null
  }

  return { setEl, ready, enabled, tokenForSubmit, reset }
}
