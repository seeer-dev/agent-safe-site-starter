// 深淺色切換 — 與 theme-init.ts 使用同一 storage key 與預設規則。
// theme-init 是純 JS（預繪執行），此檔提供 Vue reactive 介面。

import { ref } from 'vue'

const KEY = 'curatory-theme'

export const isDark = ref(
  typeof document !== 'undefined' && document.documentElement.classList.contains('dark'),
)

export function toggleTheme() {
  const next = !document.documentElement.classList.contains('dark')
  document.documentElement.classList.toggle('dark', next)
  isDark.value = next
  try {
    localStorage.setItem(KEY, next ? 'dark' : 'light')
  } catch {
    /* private mode */
  }
}
