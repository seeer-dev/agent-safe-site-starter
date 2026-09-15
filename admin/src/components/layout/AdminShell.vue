<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted } from 'vue'
import Sidebar from './Sidebar.vue'
import Topbar from './Topbar.vue'
import MobileNav from './MobileNav.vue'
import SearchPalette from './SearchPalette.vue'
import { useLayoutStore } from '@/stores/layout'

const layout = useLayoutStore()

// ⌘K / Ctrl+K toggles the global search palette. The listener lives here
// (always mounted) — the palette itself only handles in-dialog keys.
function onGlobalKeydown(e: KeyboardEvent) {
  if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
    e.preventDefault()
    layout.togglePalette()
  }
}

onMounted(() => window.addEventListener('keydown', onGlobalKeydown))
onBeforeUnmount(() => window.removeEventListener('keydown', onGlobalKeydown))

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

  <!-- Overlays live OUTSIDE .app: fixed overlays don't participate in the
       grid, and keeping them outside prevents .app[data-sidebar="collapsed"]
       rules from collapsing the drawer's nav (labels must stay visible). -->
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

  <SearchPalette v-if="layout.paletteOpen" />
</template>
