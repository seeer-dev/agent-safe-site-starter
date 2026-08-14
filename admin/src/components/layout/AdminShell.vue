<script setup lang="ts">
import { computed } from 'vue'
import Sidebar from './Sidebar.vue'
import Topbar from './Topbar.vue'
import MobileNav from './MobileNav.vue'
import { useLayoutStore } from '@/stores/layout'

const layout = useLayoutStore()

const sidebarAttr = computed(() => layout.sidebarCollapsed ? 'collapsed' : 'expanded')
const contentWidthAttr = computed(() => layout.contentWidth)
</script>

<template>
  <div
    class="app"
    :data-sidebar="sidebarAttr"
    :data-content-width="contentWidthAttr"
  >
    <Sidebar />
    <div class="main">
      <Topbar />
      <main class="content">
        <slot />
      </main>
    </div>
  </div>

  <!-- Mobile drawer -->
  <div
    class="mobile-drawer"
    :class="{ open: layout.mobileDrawerOpen }"
    @click.self="layout.closeMobileDrawer()"
  >
    <div class="drawer-panel">
      <Sidebar @navigate="layout.closeMobileDrawer()" />
    </div>
  </div>

  <MobileNav />
</template>
