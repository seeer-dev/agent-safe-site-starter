// Vue Islands bootstrap — curatory theme.
//
// The Go renderer emits <div data-vue-island="Name"> mount points.
// This entry scans for them and lazy-mounts each island (code-split via
// import.meta.glob — only islands actually on the page are loaded).
//
// Shared reactive state (bootstrap data, cart, toasts) lives in module
// singletons under @/shared/lib — all islands share one module graph.

import '@/shared/styles/globals.css'

import { createApp, type Component, type App as VueApp } from 'vue'
import { restoreCart, persistCart, rehydrateCart, addItem } from '@/shared/lib/cart'
import { installLinkInterceptor } from '@/shared/lib/transition'
import { getProductBySlug } from '@/shared/lib/catalog'
import { toast } from '@/shared/lib/toast'
import { vReveal } from '@/shared/composables/use-reveal'

// Lazy island registry. Each *.vue under islands/ becomes its own chunk.
const ISLANDS = import.meta.glob<{ default: Component }>('./**/*.vue')

// Props 傳遞：優先 data-props="<json>"；另支援 data-prop-*="value"
// 屬性（kebab-case → camelCase，全部為字串），模板逐欄輸出更安全
// （html/template 會逐屬性跳脫，不需手工拼 JSON）。
function mountIsland(el: HTMLElement, component: Component) {
  let props: Record<string, unknown> = {}
  const propsJson = el.getAttribute('data-props')
  if (propsJson) {
    try {
      props = JSON.parse(propsJson)
    } catch (e) {
      console.error('[islands] invalid data-props on', el, e)
      return
    }
  }
  for (const attr of el.attributes) {
    if (!attr.name.startsWith('data-prop-')) continue
    const key = attr.name
      .slice('data-prop-'.length)
      .replace(/-([a-z])/g, (_, c: string) => c.toUpperCase())
    props[key] = attr.value
  }
  const app: VueApp = createApp(component, props)
  app.directive('reveal', vReveal)
  app.mount(el)
  el.setAttribute('data-vue-island-mounted', '1')
  ;(el as unknown as { __vue_app__: VueApp }).__vue_app__ = app
}

async function scanAndMount() {
  const mounts = document.querySelectorAll<HTMLElement>('[data-vue-island]:not([data-vue-island-mounted])')
  for (const el of mounts) {
    const name = el.getAttribute('data-vue-island')
    if (!name) continue
    const candidates = [`./${name}/${name}.vue`, `./${name}.vue`]
    let loader: (() => Promise<{ default: Component }>) | undefined
    for (const path of candidates) {
      if (ISLANDS[path]) {
        loader = ISLANDS[path]
        break
      }
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

// ─── 靜態頁面的全域行為（不需掛島即可享受） ──────────────────

// [data-reveal]：滾動進場。JS 才加 reveal-init（無 JS 時內容直接可見），
// 進入視窗加 reveal-in；data-reveal-delay="0.12" 控制延遲秒數。
function setupReveals() {
  const els = document.querySelectorAll<HTMLElement>('[data-reveal]:not([data-reveal-bound])')
  if (els.length === 0) return
  const reduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches
  if (reduced || !('IntersectionObserver' in window)) {
    els.forEach((el) => el.removeAttribute('data-reveal'))
    return
  }
  const io = new IntersectionObserver(
    (entries) => {
      for (const entry of entries) {
        if (!entry.isIntersecting) continue
        entry.target.classList.add('reveal-in')
        io.unobserve(entry.target)
      }
    },
    { rootMargin: '0px 0px -40px 0px' },
  )
  // 先批次加初始態，下一幀才 observe — 讓已在視窗內的元素也能播動畫。
  els.forEach((el) => {
    el.setAttribute('data-reveal-bound', '1')
    const delay = parseFloat(el.getAttribute('data-reveal-delay') ?? '0')
    if (delay > 0) el.style.transitionDelay = `${delay}s`
    el.classList.add('reveal-init')
  })
  requestAnimationFrame(() => els.forEach((el) => io.observe(el)))
}

// [data-add-to-cart="<slug>"]：靜態商品卡上的 quick-add。
// 從目錄快取找商品（無規格商品才直接加；有規格導向商品頁）。
function setupQuickAdd() {
  document.addEventListener('click', async (e) => {
    const btn = (e.target as HTMLElement).closest<HTMLElement>('[data-add-to-cart]')
    if (!btn) return
    e.preventDefault()
    e.stopPropagation()
    const slug = btn.getAttribute('data-add-to-cart')
    if (!slug) return
    const p = await getProductBySlug(slug)
    if (!p) {
      toast.error('找不到商品資料', { description: '請稍後再試' })
      return
    }
    if (p.stock <= 0) {
      toast.message('此商品已售完', { description: p.name })
      return
    }
    if ((p.variants?.length ?? 0) > 0) {
      window.location.href = `/products/${p.slug}/`
      return
    }
    addItem({
      productId: p.id,
      productSlug: p.slug,
      sku: p.sku,
      variantId: null,
      name: p.name,
      variantName: null,
      image: p.images?.[0] ?? p.image ?? '',
      unitPrice: p.price,
      qty: 1,
      stock: p.stock,
    })
    toast.success('已加入購物車', { description: p.name })
  })
}

// [data-newsletter-form]：電子報訂閱（與 reference 相同 —— 前端提示成功）。
function setupNewsletter() {
  document.addEventListener('submit', (e) => {
    const form = (e.target as HTMLElement).closest('form[data-newsletter-form]') as HTMLFormElement | null
    if (!form) return
    e.preventDefault()
    const input = form.querySelector<HTMLInputElement>('input[type="email"]')
    const email = input?.value.trim() ?? ''
    if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email)) {
      toast.error('請輸入正確的 Email 格式')
      return
    }
    if (input) input.value = ''
    toast.success('感謝訂閱！', {
      description: '每季的嚴選故事與專屬優惠，將準時送到您的信箱。',
    })
  })
}

// [data-scroll-top]：回到頂部按鈕。
function setupScrollTop() {
  document.addEventListener('click', (e) => {
    const btn = (e.target as HTMLElement).closest('[data-scroll-top]')
    if (!btn) return
    const reduced = window.matchMedia('(prefers-reduced-motion: reduce)').matches
    window.scrollTo({ top: 0, behavior: reduced ? 'auto' : 'smooth' })
  })
}

function init() {
  restoreCart()
  // Refresh persisted cart items against the live catalog (price/stock).
  void rehydrateCart()
  window.addEventListener('pagehide', persistCart)
  // SPA 式簾幕轉場：攔截站內連結點擊播 cover 動畫。
  installLinkInterceptor()
  setupQuickAdd()
  setupNewsletter()
  setupScrollTop()

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', () => {
      setupReveals()
      void scanAndMount()
    })
  } else {
    setupReveals()
    void scanAndMount()
  }
}

init()
