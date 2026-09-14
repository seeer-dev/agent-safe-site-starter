// 頁面轉場（MPA 版）— 對應 reference page-transition.tsx 的單層墨色簾幕。
//
// Reference 是 SPA：cover 0.34s → 遮罩下換頁 → reveal 0.4s。
// 靜態站無法在遮罩下換頁，做法：
//   1. 點擊站內連結 → 攔截 → 播 cover（簾幕由下方升起蓋住頁面）
//      → sessionStorage 標記 → location 導航。
//   2. 新頁面 <head> 的 theme-init.js 偵測標記 → <html> 加
//      class="curtain-reveal" → 簾幕 div（模板常駐）從覆蓋狀態向上滑出。
// 全程 CSS 動畫，prefers-reduced-motion 時降級為直接導航。

const FLAG = 'curatory-transition'

/** 本站內部導航（帶簾幕轉場）。path 例：'/shop/'、'/products/xxx/' */
export function navigate(path: string) {
  if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
    window.location.href = path
    return
  }
  const curtain = document.getElementById('page-curtain')
  try {
    sessionStorage.setItem(FLAG, '1')
  } catch {
    /* private mode — transition flag optional */
  }
  if (curtain) {
    // 防禦：上一頁 reveal 的 forwards 填充若仍在，先清掉再播 cover。
    document.documentElement.classList.remove('curtain-reveal')
    curtain.classList.add('curtain-cover')
    // 覆蓋期間預取目標頁，縮短新頁到達前的停頓。
    try {
      void fetch(path, { credentials: 'same-origin' }).catch(() => undefined)
    } catch {
      /* prefetch is best-effort */
    }
    window.setTimeout(() => {
      window.location.href = path
    }, 420)
  } else {
    window.location.href = path
  }
}

// 全域攔截站內 <a> 點擊 → 播 cover 再導航。
// 跳過：修飾鍵、非左鍵、target、download、hash-only、外部連結、
// 或標記 data-no-transition 的元素。
export function installLinkInterceptor() {
  document.addEventListener('click', (e) => {
    if (e.defaultPrevented || e.button !== 0 || e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) return
    const anchor = (e.target as HTMLElement).closest('a[href]') as HTMLAnchorElement | null
    if (!anchor) return
    if (anchor.target || anchor.hasAttribute('download') || anchor.hasAttribute('data-no-transition')) return
    const href = anchor.getAttribute('href') ?? ''
    if (!href.startsWith('/') || href.startsWith('//')) return
    if (href === window.location.pathname + window.location.search) return
    e.preventDefault()
    navigate(href)
  })
}
