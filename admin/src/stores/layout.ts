import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useLayoutStore = defineStore('layout', () => {
  const sidebarCollapsed = ref(false)
  const contentWidth = ref<'locked' | 'fluid'>('locked')
  const mobileDrawerOpen = ref(false)

  function init() {
    try {
      const s = localStorage.getItem('sidebar_state')
      sidebarCollapsed.value = s === 'false'
    } catch (_) { /* noop */ }
    try {
      const c = localStorage.getItem('admin.contentWidth')
      contentWidth.value = c === 'fluid' ? 'fluid' : 'locked'
    } catch (_) { /* noop */ }
  }

  function toggleSidebar() {
    sidebarCollapsed.value = !sidebarCollapsed.value
    try {
      localStorage.setItem('sidebar_state', String(!sidebarCollapsed.value))
    } catch (_) { /* noop */ }
  }

  function toggleContentWidth() {
    contentWidth.value = contentWidth.value === 'fluid' ? 'locked' : 'fluid'
    try {
      localStorage.setItem('admin.contentWidth', contentWidth.value)
    } catch (_) { /* noop */ }
  }

  function openMobileDrawer() {
    mobileDrawerOpen.value = true
  }

  function closeMobileDrawer() {
    mobileDrawerOpen.value = false
  }

  return {
    sidebarCollapsed,
    contentWidth,
    mobileDrawerOpen,
    init,
    toggleSidebar,
    toggleContentWidth,
    openMobileDrawer,
    closeMobileDrawer,
  }
})
