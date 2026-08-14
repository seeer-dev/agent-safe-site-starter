// Vue Islands bootstrap.
//
// This is the single entry point Vite builds. It:
//   1. Creates one shared Pinia instance (all islands share state).
//   2. Restores persisted stores (cart, user, theme).
//   3. Scans the document for <div data-vue-island="Name"> mount points.
//   4. For each mount point, lazy-loads the corresponding island component,
//      reads its props from the data-props JSON attribute, and mounts it.
//
// Islands are code-split: only the islands actually on a page are loaded.
// The Go renderer emits one <script type="module" src="/assets/islands/islands.js">
// at the bottom of each page that contains islands.

// Import global styles (Tailwind base/components/utilities + CSS variables).
// This ensures the main CSS bundle is emitted from the entry chunk.
import '@/shared/styles/globals.css'

import { createApp, type Component, type App as VueApp } from 'vue'
import { createPinia, type Pinia } from 'pinia'
import { useCartStore } from '@/shared/stores/cart'
import { useUserStore } from '@/shared/stores/user'
import { useThemeStore } from '@/shared/stores/theme'
import { initPublicAuth } from '@/shared/lib/auth/session'

// Lazy island registry. Vite code-splits each glob entry into its own chunk.
const ISLANDS = import.meta.glob<{ default: Component }>('./**/*.vue')

// One shared Pinia for all islands on the page.
let pinia: Pinia

const SAFE_CATEGORY = /^[a-z0-9-]+$/

function mountIsland(el: HTMLElement, component: Component) {
  const propsJson = el.getAttribute('data-props')
  let props: Record<string, unknown> = {}
  if (propsJson) {
    try {
      props = JSON.parse(propsJson)
    } catch (e) {
      console.error('[islands] invalid data-props on', el, e)
      return
    }
  }
  // Category pages pass a validated slug token in data-category instead of
  // constructing JSON in the HTML attribute.
  const category = el.getAttribute('data-category')
  if (category && SAFE_CATEGORY.test(category) && props.initialCategory == null) {
    props.initialCategory = category
  }
  const app: VueApp = createApp(component, props)
  app.use(pinia)
  app.mount(el)
  // Mark so we don't double-mount on re-scan.
  el.setAttribute('data-vue-island-mounted', '1')
  // Keep a reference for HMR / teardown if ever needed.
  ;(el as any).__vue_app__ = app
}

async function scanAndMount() {
  const mounts = document.querySelectorAll<HTMLElement>('[data-vue-island]:not([data-vue-island-mounted])')
  for (const el of mounts) {
    const name = el.getAttribute('data-vue-island')
    if (!name) continue
    // Look up by flexible path: ./Header/Header.vue or ./Header.vue
    const candidates = [
      `./${name}/${name}.vue`,
      `./${name}.vue`,
    ]
    let loader: (() => Promise<{ default: Component }>) | undefined
    for (const path of candidates) {
      if (ISLANDS[path]) { loader = ISLANDS[path]; break }
    }
    if (!loader) {
      console.error(`[islands] no component registered for "${name}"`)
      continue
    }
    try {
      const mod = await loader()
      mountIsland(el, mod.default)
    } catch (e) {
      console.error(`[islands] failed to mount "${name}"`, e)
    }
  }
}

async function init() {
  pinia = createPinia()

  // Restore persisted stores before mounting so islands render with real state.
  const cart = useCartStore(pinia)
  const user = useUserStore(pinia)
  const theme = useThemeStore(pinia)
  theme.restore()
  cart.restore()
  user.restore()

  try {
    await initPublicAuth((session) => user.syncFromSession(session))
  } catch {
    user.syncFromSession(null)
  }

  // Rehydrate cart display data from the authoritative catalog API.
  // restore() only reads persisted identifiers; rehydrate() fetches
  // current product data (price, stock, name). Fail-closed: if the API
  // is unavailable, the cart stays empty — persisted price/stock/name
  // are never trusted.
  cart.rehydrate()

  // Persist on pagehide so we don't lose cart identifiers on navigation.
  // Auth session persistence is owned by Supabase, not this store.
  window.addEventListener('pagehide', () => {
    cart.persist()
  })

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', scanAndMount)
  } else {
    scanAndMount()
  }
}

void init()
