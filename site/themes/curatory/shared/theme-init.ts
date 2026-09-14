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
    }
  } catch {
    /* ignore */
  }
})()
