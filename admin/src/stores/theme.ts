import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useThemeStore = defineStore('theme', () => {
  const isDark = ref(false)

  function init() {
    const saved = localStorage.getItem('admin-theme')
    if (saved === 'dark') isDark.value = true
    apply()
  }

  function toggle() {
    isDark.value = !isDark.value
    localStorage.setItem('admin-theme', isDark.value ? 'dark' : 'light')
    apply()
  }

  function apply() {
    document.body.dataset.theme = isDark.value ? 'dark' : ''
  }

  return { isDark, init, toggle }
})
