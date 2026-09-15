<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ChevronRight, LogOut } from 'lucide-vue-next'
import {
  LayoutDashboard, Package, ShoppingBag, Users, TicketPercent,
  FileText, CreditCard, UserCog, HelpCircle, FolderTree,
  MessageSquareText, Newspaper, Truck, Mail, MailCheck, Store,
  Settings, ShieldCheck,
} from 'lucide-vue-next'
import { useAuthStore } from '@/stores/auth'
import { PROFILE, hrefFor } from '@/config/profile'
import type { RouteDef } from '@/lib/types'

const auth = useAuthStore()
const route = useRoute()

const emit = defineEmits<{
  navigate: []
}>()

const ICON_MAP: Record<string, any> = {
  LayoutDashboard, Package, ShoppingBag, Users, TicketPercent,
  FileText, CreditCard, UserCog, FolderTree, MessageSquareText,
  Newspaper, Truck, Mail, MailCheck, Store, Settings, ShieldCheck,
}

// Recursive capability gating: parent caps gate the whole group; children
// are filtered individually; a group with zero visible children is dropped.
const visibleNav = computed(() =>
  PROFILE
    .filter((r) => r.caps.every((c) => auth.can(c)))
    .map((r) =>
      r.children?.length
        ? { ...r, children: r.children.filter((c) => c.caps.every((cap) => auth.can(cap))) }
        : r,
    )
    .filter((r) => !r.children || r.children.length > 0),
)

type NavItem =
  | { kind: 'divider'; label: string }
  | { kind: 'entry'; route: RouteDef }

// Emit each divider label once, before the first visible entry carrying it.
const navItems = computed<NavItem[]>(() => {
  const items: NavItem[] = []
  const seen = new Set<string>()
  for (const r of visibleNav.value) {
    if (r.dividerBefore && !seen.has(r.dividerBefore)) {
      seen.add(r.dividerBefore)
      items.push({ kind: 'divider', label: r.dividerBefore })
    }
    items.push({ kind: 'entry', route: r })
  }
  return items
})

// Track which groups are expanded (expanded sidebar mode only).
const expandedParents = ref<Set<string>>(new Set())

function toggleChildren(item: RouteDef) {
  if (expandedParents.value.has(item.key)) {
    expandedParents.value.delete(item.key)
  } else {
    expandedParents.value.add(item.key)
  }
}

function isActive(key: string): boolean {
  if (key === 'dashboard') return route.name === 'dashboard'
  if (key === 'store-settings') return route.name === 'store-settings'
  if (key === 'roles') return route.name === 'roles'
  return route.path === `/res/${key}`
}

// Auto-expand the group containing the active child so the current
// destination stays visible (mockup parity: navigate() adds _parent).
watch(
  () => route.path,
  () => {
    for (const r of PROFILE) {
      if (r.children?.some((c) => isActive(c.key))) {
        expandedParents.value.add(r.key)
      }
    }
  },
  { immediate: true },
)

const userInitial = computed(() => {
  const base = auth.email || auth.role || ''
  return base ? base.slice(0, 1).toUpperCase() : '管'
})

function onNavigate() {
  emit('navigate')
}
</script>

<template>
  <aside class="sidebar">
    <!-- Brand -->
    <div class="brand">
      <div class="logo">質</div>
      <div class="brand-text"><b>質選所後台</b><small>tw-minimal-cart</small></div>
    </div>

    <!-- Nav -->
    <nav class="nav">
      <template v-for="item in navItems" :key="item.kind === 'divider' ? `div-${item.label}` : item.route.key">
        <!-- Labeled divider between module groups and universal base groups -->
        <div
          v-if="item.kind === 'divider'"
          class="divider"
        ><span>{{ item.label }}</span></div>

        <template v-else>
          <!-- Group with children: nav-item-wrapper for flyout support -->
          <div
            v-if="item.route.children && item.route.children.length"
            class="nav-item-wrapper"
          >
            <a
              class="nav-item"
              :class="{ active: item.route.children.some((c) => isActive(c.key)) }"
              :data-label="item.route.label"
              data-has-children="true"
              href="#"
              @click.prevent="toggleChildren(item.route)"
            >
              <component :is="ICON_MAP[item.route.icon]" />
              <span class="nav-label">{{ item.route.label }}</span>
              <ChevronRight
                class="nav-chevron"
                :style="{ transform: expandedParents.has(item.route.key) ? 'rotate(90deg)' : '' }"
                style="width:12px;height:12px;flex-shrink:0;color:var(--text-3);transition:transform 200ms ease"
              />
            </a>
            <!-- Inline children (expanded mode) -->
            <div
              v-show="expandedParents.has(item.route.key)"
              class="nav-children"
              style="margin:2px 0 4px 14px;border-left:1px solid var(--border);padding-left:4px"
            >
              <RouterLink
                v-for="child in item.route.children"
                :key="child.key"
                class="nav-child"
                :class="{ 'is-current': isActive(child.key) }"
                :to="hrefFor(child.key)"
                @click="onNavigate"
              >
                <component :is="ICON_MAP[child.icon ?? item.route.icon]" />
                {{ child.label }}
              </RouterLink>
            </div>
            <!-- Flyout bridge (invisible gap filler for hover) -->
            <div class="flyout-bridge" aria-hidden="true" />
            <!-- Flyout (collapsed mode hover panel) -->
            <div class="flyout">
              <div
                style="margin-bottom:2px;display:flex;align-items:center;gap:8px;border-bottom:1px solid var(--surface-3);padding:8px 12px;font-size:13px;font-weight:600;color:var(--text)"
              >
                <component :is="ICON_MAP[item.route.icon]" :style="{ width: '14px', height: '14px' }" />
                {{ item.route.label }}
              </div>
              <RouterLink
                v-for="child in item.route.children"
                :key="child.key"
                class="nav-child"
                :class="{ 'is-current': isActive(child.key) }"
                :to="hrefFor(child.key)"
                @click="onNavigate"
              >
                <component :is="ICON_MAP[child.icon ?? item.route.icon]" />
                {{ child.label }}
              </RouterLink>
            </div>
          </div>

          <!-- Flat leaf (no children) -->
          <RouterLink
            v-else
            :to="hrefFor(item.route.key)"
            :class="{ active: isActive(item.route.key) }"
            :data-label="item.route.label"
            @click="onNavigate"
          >
            <component :is="ICON_MAP[item.route.icon]" />
            <span class="nav-label">{{ item.route.label }}</span>
          </RouterLink>
        </template>
      </template>
    </nav>

    <!-- Footer: states reference + pinned user block -->
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
      <div class="userblock">
        <div class="avatar">{{ userInitial }}</div>
        <div class="user-meta nav-label">
          <b>{{ auth.email || '—' }}</b>
          <small>{{ auth.role || '—' }}</small>
        </div>
        <button
          class="user-logout"
          type="button"
          title="登出"
          aria-label="登出"
          @click="auth.logout()"
        >
          <LogOut style="width:14px;height:14px" />
        </button>
      </div>
    </div>
  </aside>
</template>
