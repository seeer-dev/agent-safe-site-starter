<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRoute } from 'vue-router'
import { ChevronRight } from 'lucide-vue-next'
import {
  LayoutDashboard, Package, ShoppingBag, Users, TicketPercent,
  FileText, CreditCard, UserCog, HelpCircle,
} from 'lucide-vue-next'
import { useAuthStore } from '@/stores/auth'
import { PROFILE, SECTION_LABEL } from '@/config/profile'
import type { SectionKey, RouteDef } from '@/lib/types'

const auth = useAuthStore()
const route = useRoute()

const emit = defineEmits<{
  navigate: []
}>()

const ICON_MAP: Record<string, any> = {
  LayoutDashboard, Package, ShoppingBag, Users, TicketPercent,
  FileText, CreditCard, UserCog,
}

const grouped = computed(() => {
  const groups: Record<SectionKey, typeof PROFILE> = { primary: [], secondary: [], settings: [] }
  for (const r of PROFILE) {
    if (!r.caps.every((c) => auth.can(c))) continue
    groups[r.section].push(r)
  }
  return groups
})

// Track which parent items have children expanded (expanded sidebar mode only).
// Currently no PROFILE entry has children, so this stays empty.
const expandedParents = ref<Set<string>>(new Set())

function toggleChildren(item: RouteDef) {
  if (expandedParents.value.has(item.key)) {
    expandedParents.value.delete(item.key)
  } else {
    expandedParents.value.add(item.key)
  }
}

function hrefFor(key: string): string {
  if (key === 'dashboard') return '/'
  return `/res/${key}`
}

function isActive(key: string): boolean {
  if (key === 'dashboard') return route.name === 'dashboard'
  return route.path === `/res/${key}`
}

function onNavigate() {
  emit('navigate')
}
</script>

<template>
  <aside class="sidebar">
    <!-- Brand -->
    <div class="brand">
      <div class="logo">質</div>
      <div class="brand-text"><b>質物選物後台</b><small>tw-minimal-cart</small></div>
    </div>

    <!-- Nav -->
    <nav class="nav">
      <template v-for="sec in (['primary','secondary','settings'] as SectionKey[])" :key="sec">
        <div v-if="grouped[sec].length" class="group">
          {{ SECTION_LABEL[sec] }}
        </div>
        <template v-for="r in grouped[sec]" :key="r.key">
          <!-- Parent with children: nav-item-wrapper for flyout support -->
          <div
            v-if="r.children && r.children.length"
            class="nav-item-wrapper"
          >
            <a
              class="nav-item"
              :class="{ active: isActive(r.key) }"
              :data-label="r.label"
              data-has-children="true"
              href="#"
              @click.prevent="toggleChildren(r)"
            >
              <component :is="ICON_MAP[r.icon]" />
              <span class="nav-label">{{ r.label }}</span>
              <span
                v-if="r.key === 'minimal-cart-orders'"
                class="badge"
              >6</span>
              <ChevronRight
                class="nav-chevron"
                :style="{ transform: expandedParents.has(r.key) ? 'rotate(90deg)' : '' }"
                style="width:12px;height:12px;flex-shrink:0;color:var(--text-3);transition:transform 200ms ease"
              />
            </a>
            <!-- Inline children (expanded mode) -->
            <div
              v-show="expandedParents.has(r.key)"
              class="nav-children"
              style="margin:2px 0 4px 14px;border-left:1px solid var(--border);padding-left:4px"
            >
              <a
                v-for="child in r.children"
                :key="child.key"
                class="nav-child"
                :class="{ 'is-current': isActive(child.key) }"
                :href="hrefFor(child.key)"
                @click="onNavigate"
              >
                {{ child.label }}
              </a>
            </div>
            <!-- Flyout bridge (invisible gap filler for hover) -->
            <div class="flyout-bridge" aria-hidden="true" />
            <!-- Flyout (collapsed mode hover panel) -->
            <div class="flyout">
              <div
                style="margin-bottom:2px;display:flex;align-items:center;gap:8px;border-bottom:1px solid var(--surface-3);padding:8px 12px;font-size:13px;font-weight:600;color:var(--text)"
              >
                <component :is="ICON_MAP[r.icon]" :style="{ width: '14px', height: '14px' }" />
                {{ r.label }}
              </div>
              <a
                v-for="child in r.children"
                :key="child.key"
                class="nav-child"
                :class="{ 'is-current': isActive(child.key) }"
                :href="hrefFor(child.key)"
                @click="onNavigate"
              >
                {{ child.label }}
              </a>
            </div>
          </div>

          <!-- Flat leaf (no children) -->
          <RouterLink
            v-else
            :to="hrefFor(r.key)"
            :class="{ active: isActive(r.key) }"
            :data-label="r.label"
            @click="onNavigate"
          >
            <component :is="ICON_MAP[r.icon]" />
            <span class="nav-label">{{ r.label }}</span>
            <span
              v-if="r.key === 'minimal-cart-orders'"
              class="badge"
            >6</span>
          </RouterLink>
        </template>
      </template>
    </nav>

    <!-- Footer -->
    <div class="sidefoot">
      <RouterLink
        to="/states"
        :class="{ active: route.name === 'states' }"
        data-label="五狀態參考"
        @click="onNavigate"
      >
        <HelpCircle />
        <span class="nav-label">五狀態參考</span>
      </RouterLink>
    </div>
  </aside>
</template>
