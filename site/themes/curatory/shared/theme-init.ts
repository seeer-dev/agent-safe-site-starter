// Pre-paint theme bootstrap. Loaded as a plain external script in <head>
// (CSP forbids inline scripts). Restores the user's dark/light choice
// from localStorage before first paint to avoid a flash.
//
// Key matches the header toggle. Values: "dark" | "light". Absent or
// unknown values fall back to the OS preference, matching the reference
// next-themes behavior.
;(function initTheme() {
  try {
    const stored = localStorage.getItem('curatory-theme')
    const dark =
      stored === 'dark' ||
      (stored == null && window.matchMedia('(prefers-color-scheme: dark)').matches)
    document.documentElement.classList.toggle('dark', dark)
  } catch {
    /* localStorage unavailable — default light */
  }

  // 轉場揭幕：上一頁若播過簾幕 cover，本頁以 reveal 動畫開場。
  try {
    if (sessionStorage.getItem('curatory-transition')) {
      sessionStorage.removeItem('curatory-transition')
      document.documentElement.classList.add('curtain-reveal')
      // 揭幕動畫 forwards 填充會把簾幕固定在 -100%，不移除 class 的話
      // 同一頁內的下一次轉場 cover 階段會失效（簾幕停在視窗外）。
      // 此腳本在 <head> 執行時 #page-curtain 尚未解析，等 DOMContentLoaded 掛接；
      // 且 animationend 會從子元素冒泡，需限定 target 為簾幕本身。
      const clear = () => document.documentElement.classList.remove('curtain-reveal')
      const attach = () => {
        const curtain = document.getElementById('page-curtain')
        if (curtain) {
          curtain.addEventListener(
            'animationend',
            (e) => {
              if (e.target === curtain) clear()
            },
            { once: false }
          )
        }
      }
      if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', attach, { once: true })
      } else {
        attach()
      }
      // 後援：animationend 未觸發時仍要歸位
      window.setTimeout(clear, 1200)
    }
  } catch {
    /* ignore */
  }
})()
