<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { Moon, Sun, LogOut } from 'lucide-vue-next'
import { useAuthStore } from '@/stores/auth'
import { useThemeStore } from '@/stores/theme'
import { useLayoutStore } from '@/stores/layout'
import { RES } from '@/config/resources'

const auth = useAuthStore()
const theme = useThemeStore()
const layout = useLayoutStore()
const route = useRoute()

const crumb = computed(() => {
  if (route.name === 'dashboard') return '總覽'
  if (route.name === 'states') return '五狀態'
  if (route.name === 'resource') {
    const key = route.path.replace('/res/', '')
    return RES[key]?.label ?? '—'
  }
  return '—'
})

const crumbPrefix = computed(() => {
  if (route.name === 'states') return '參考'
  return '後台'
})
</script>

<template>
  <header class="topbar">
    <!-- Left: sidebar toggles -->
    <div class="topbar-left" style="display:flex;align-items:center;gap:8px;flex-shrink:0">
      <!-- Mobile: hamburger to open drawer -->
      <button
        class="tbtn sidebar-toggle-mobile"
        type="button"
        title="開啟側邊欄"
        aria-label="開啟側邊欄"
        @click="layout.openMobileDrawer()"
      >
        <svg viewBox="0 0 24 24" width="17" height="17"><path d="M4 5h16"/><path d="M4 12h16"/><path d="M4 19h16"/></svg>
      </button>
      <!-- Desktop: sidebar collapse toggle (icon swaps expand/collapse) -->
      <button
        class="tbtn sidebar-toggle-desktop"
        type="button"
        title="縮放側邊欄"
        aria-label="縮放側邊欄"
        @click="layout.toggleSidebar()"
      >
        <!-- icon-expand: shown when expanded (click to collapse) -->
        <svg v-if="!layout.sidebarCollapsed" viewBox="0 0 24 24" width="17" height="17"><rect width="18" height="18" x="3" y="3" rx="2"/><path d="M9 3v18"/><path d="m16 15-3-3 3-3"/></svg>
        <!-- icon-collapse: shown when collapsed (click to expand) -->
        <svg v-else viewBox="0 0 24 24" width="17" height="17"><rect width="18" height="18" x="3" y="3" rx="2"/><path d="M9 3v18"/><path d="m14 9 3 3-3 3"/></svg>
      </button>
    </div>

    <!-- Center: breadcrumb -->
    <div class="crumb">{{ crumbPrefix }} / <b>{{ crumb }}</b></div>

    <!-- Right: actions -->
    <div class="rolebar">
      <span style="color:var(--green);font-size:12px">已連線</span>
      <span class="muted" style="font-size:12px">{{ auth.role }}</span>
      <button
        class="tbtn content-width-toggle"
        type="button"
        :title="layout.contentWidth === 'locked' ? '放寬內容寬度' : '限制內容寬度'"
        :aria-label="layout.contentWidth === 'locked' ? '放寬內容寬度' : '限制內容寬度'"
        @click="layout.toggleContentWidth()"
      >
        <!-- icon-locked: shown when locked (click to widen) -->
        <svg v-if="layout.contentWidth === 'locked'" viewBox="0 0 24 24" width="17" height="17"><path d="M16 12h6"/><path d="M8 12H2"/><path d="M12 2v2"/><path d="M12 8v2"/><path d="M12 14v2"/><path d="M12 20v2"/><path d="m19 15 3-3-3-3"/><path d="m5 9-3 3 3 3"/></svg>
        <!-- icon-fluid: shown when fluid (click to narrow) -->
        <svg v-else viewBox="0 0 24 24" width="17" height="17"><path d="M2 12h6"/><path d="M22 12h-6"/><path d="M12 2v2"/><path d="M12 8v2"/><path d="M12 14v2"/><path d="M12 20v2"/><path d="m19 9-3 3 3 3"/><path d="m5 15 3-3-3-3"/></svg>
      </button>
      <button class="tbtn" type="button" title="登出" aria-label="登出" @click="auth.logout()">
        <LogOut style="width:14px;height:14px" />
      </button>
      <button
        class="tbtn"
        type="button"
        :aria-label="theme.isDark ? '切換亮色模式' : '切換暗色模式'"
        @click="theme.toggle()"
      >
        <Moon v-if="!theme.isDark" />
        <Sun v-else />
      </button>
    </div>
  </header>
</template>
