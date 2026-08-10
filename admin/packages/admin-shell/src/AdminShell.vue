<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import {
  ChevronRight,
  Menu,
  PanelLeft,
  Search,
  User,
} from 'lucide-vue-next'
import { Sheet, SheetContent, SheetTrigger, SheetTitle } from '@sitecore/admin-ui'
import type {
  ShellNavigationModel,
  ShellBrand,
  AdminShellCopy,
} from './types'

const DEFAULT_COPY: AdminShellCopy = {
  primaryNavigation: '主要導覽',
  expandSidebar: '展開側邊欄',
  collapseSidebar: '收合側邊欄',
  openSidebar: '開啟側邊欄',
  search: '搜尋',
  mobilePrimaryNavigation: '行動版主要導覽',
  moreNavigation: '更多導覽',
  more: '更多',
}

const props = withDefaults(defineProps<{
  navigation: ShellNavigationModel
  breadcrumb: string[]
  identityLabel: string
  searchPlaceholder: string
  brand: ShellBrand
  copy?: Partial<AdminShellCopy>
  rootAttributes?: Record<string, string | undefined>
}>(), {
  copy: () => ({}),
  rootAttributes: () => ({}),
})

const emit = defineEmits<{ navigate: [href: string] }>()

const fullCopy: AdminShellCopy = { ...DEFAULT_COPY, ...props.copy }

const sidebarCollapsed = ref(false)
const mobileSheet = ref<'closed' | 'nav' | 'more'>('closed')

const crumbTrail = computed(() => props.breadcrumb.slice(0, -1))
const crumbCurrent = computed(() => props.breadcrumb.at(-1) ?? '')

function toggleSidebar() {
  sidebarCollapsed.value = !sidebarCollapsed.value
}

function closeMobileSheet(href?: string) {
  mobileSheet.value = 'closed'
  if (href) emit('navigate', href)
}

function onNavClick(href?: string) {
  if (href) emit('navigate', href)
}

/* Branch open/close state — seeded from branchOpen defaults */
const openParents = ref(new Set<string>(
  props.navigation.groups
    .flatMap((g) => g.items)
    .filter((item) => item.children && item.branchOpen)
    .map((item) => item.key)
))

function toggleParent(key: string) {
  const next = new Set(openParents.value)
  if (next.has(key)) next.delete(key)
  else next.add(key)
  openParents.value = next
}

function isBranchOpen(key: string) {
  return openParents.value.has(key)
}

/* Sync openParents when navigation prop changes — new branches that
 * declare branchOpen: true (e.g. when the current route moves into
 * a nested branch) are auto-expanded. Previously-opened branches that
 * are no longer branchOpen: true stay open (user toggle state wins). */
watch(
  () => props.navigation,
  (next) => {
    const branchOpenKeys = new Set(
      next.groups
        .flatMap((g) => g.items)
        .filter((item) => item.children && item.branchOpen)
        .map((item) => item.key),
    )
    if (branchOpenKeys.size === 0) return
    const merged = new Set(openParents.value)
    for (const key of branchOpenKeys) merged.add(key)
    openParents.value = merged
  },
  { deep: true },
)

/* Overflow navigation: mobile items removed from groups */
const overflowNavigation = computed<ShellNavigationModel>(() => {
  const mobileKeys = new Set(props.navigation.mobile.map((item) => item.key))
  return {
    groups: props.navigation.groups
      .map((group) => ({
        ...group,
        items: group.items.filter((item) => !mobileKeys.has(item.key)),
      }))
      .filter((group) => group.items.length > 0),
    footer: props.navigation.footer,
    mobile: [],
  }
})
</script>

<template>
  <div
    v-bind="rootAttributes"
    class="app bg-canvas"
    data-shell="sidebar-left"
    :data-sidebar="sidebarCollapsed ? 'collapsed' : 'expanded'"
    data-candidate-root
  >
    <!-- Desktop sidebar -->
    <aside class="sidebar" data-shell-navigation="sidebar">
      <!-- Brand -->
      <div class="brand">
        <span class="logo" aria-hidden="true">{{ brand.logo }}</span>
        <div><b>{{ brand.label }}</b><small>{{ brand.subtitle }}</small></div>
      </div>

      <!-- Navigation groups -->
      <nav :aria-label="fullCopy.primaryNavigation" class="nav">
        <div v-for="group in navigation.groups" :key="group.key">
          <p class="nav-group">{{ group.label }}<span /></p>
          <template v-for="item in group.items" :key="item.key">
            <!-- Disabled item -->
            <span
              v-if="item.disabled"
              class="nav-item cursor-not-allowed"
              :data-label="item.label"
              :aria-label="item.label"
              aria-disabled="true"
              :title="item.disabledTitle"
            >
              <component :is="item.icon" v-if="item.icon" aria-hidden="true" :size="18" />
              <span class="nav-label">{{ item.label }}</span>
              <small v-if="item.disabledNote" class="ml-auto text-[11px]">{{ item.disabledNote }}</small>
            </span>

            <!-- Branch item with children -->
            <div
              v-else-if="item.children"
              class="nav-item-wrapper relative"
              data-slot="nav-item-wrapper"
            >
              <button
                type="button"
                class="nav-item cursor-pointer"
                :class="{ active: item.current }"
                :data-label="item.label"
                data-has-children="true"
                :aria-label="item.label"
                :aria-expanded="isBranchOpen(item.key)"
                @click="toggleParent(item.key)"
              >
                <component :is="item.icon" v-if="item.icon" aria-hidden="true" :size="18" />
                <span class="nav-label">{{ item.label }}</span>
                <span v-if="item.badge" class="badge">{{ item.badge }}</span>
                <span v-if="item.sub" class="sub">{{ item.sub }}</span>
                <ChevronRight
                  aria-hidden="true"
                  :size="12"
                  class="nav-chevron ml-0.5 flex-shrink-0 text-text-muted transition-transform"
                  :style="isBranchOpen(item.key) ? { transform: 'rotate(90deg)' } : undefined"
                />
              </button>
              <div
                class="nav-children mt-0.5 mb-1 ml-3.5 border-l border-border pl-1 space-y-px"
                :class="{ hidden: !isBranchOpen(item.key) }"
              >
                <template v-for="child in item.children" :key="child.key">
                  <a
                    v-if="child.href && !child.disabled"
                    class="nav-child"
                    :class="{ 'is-current': child.current }"
                    :href="child.href"
                    @click.prevent="onNavClick(child.href)"
                  >
                    <component :is="child.icon" v-if="child.icon" aria-hidden="true" :size="14" :class="child.current ? 'text-brand-600' : 'opacity-60'" />
                    {{ child.label }}
                  </a>
                  <span
                    v-else
                    class="nav-child"
                    :class="{ 'is-current': child.current }"
                    aria-disabled="true"
                  >
                    <component :is="child.icon" v-if="child.icon" aria-hidden="true" :size="14" :class="child.current ? 'text-brand-600' : 'opacity-60'" />
                    {{ child.label }}
                  </span>
                </template>
              </div>
              <!-- Collapsed flyout: invisible bridge + submenu panel -->
              <div class="flyout-bridge" aria-hidden="true" />
              <div class="flyout" :aria-label="item.label">
                <div class="flyout-hd">
                  <component :is="item.icon" v-if="item.icon" aria-hidden="true" :size="14" class="text-text-muted" />
                  {{ item.label }}
                </div>
                <template v-for="child in item.children" :key="child.key">
                  <a
                    v-if="child.href && !child.disabled"
                    class="nav-child"
                    :class="{ 'is-current': child.current }"
                    :href="child.href"
                    @click.prevent="onNavClick(child.href)"
                  >
                    <component :is="child.icon" v-if="child.icon" aria-hidden="true" :size="14" :class="child.current ? 'text-brand-600' : 'opacity-60'" />
                    {{ child.label }}
                  </a>
                  <span
                    v-else
                    class="nav-child"
                    :class="{ 'is-current': child.current }"
                    aria-disabled="true"
                  >
                    <component :is="child.icon" v-if="child.icon" aria-hidden="true" :size="14" :class="child.current ? 'text-brand-600' : 'opacity-60'" />
                    {{ child.label }}
                  </span>
                </template>
              </div>
            </div>

            <!-- Regular leaf item -->
            <a
              v-else
              class="nav-item"
              :class="{ active: item.current }"
              :href="item.href ?? '#'"
              :data-label="item.label"
              :aria-label="item.label"
              @click.prevent="onNavClick(item.href)"
            >
              <component :is="item.icon" v-if="item.icon" aria-hidden="true" :size="18" />
              <span class="nav-label">{{ item.label }}</span>
              <span v-if="item.badge" class="badge">{{ item.badge }}</span>
              <span v-if="item.sub" class="sub">{{ item.sub }}</span>
            </a>
          </template>
        </div>
      </nav>

      <!-- Sidefoot -->
      <div class="sidefoot">
        <template v-for="item in navigation.footer" :key="item.key">
          <span
            v-if="item.disabled || !item.href"
            class="nav-item cursor-not-allowed"
            :data-label="item.label"
            :aria-label="item.label"
            aria-disabled="true"
            :title="item.disabledTitle"
          >
            <component :is="item.icon" v-if="item.icon" aria-hidden="true" :size="18" />
            <span class="nav-label">{{ item.label }}</span>
          </span>
          <a
            v-else
            class="nav-item"
            :href="item.href"
            :data-label="item.label"
            :aria-label="item.label"
            @click.prevent="onNavClick(item.href)"
          >
            <component :is="item.icon" v-if="item.icon" aria-hidden="true" :size="18" />
            <span class="nav-label">{{ item.label }}</span>
          </a>
        </template>
        <span class="nav-item" :data-label="identityLabel" :title="identityLabel">
          <User aria-hidden="true" :size="18" />
          <span class="nav-label">{{ identityLabel }}</span>
        </span>
      </div>
    </aside>

    <!-- Main content -->
    <main class="main pb-14 lg:pb-0">
      <header class="topbar" data-slot="topbar">
        <div class="topbar-left flex items-center gap-2 flex-shrink-0" data-slot="topbar-left">
          <!-- Desktop sidebar collapse toggle -->
          <button
            id="sidebar-toggle"
            class="sidebar-toggle-desktop tbtn"
            type="button"
            :aria-label="sidebarCollapsed ? fullCopy.expandSidebar : fullCopy.collapseSidebar"
            :aria-expanded="!sidebarCollapsed"
            :title="sidebarCollapsed ? fullCopy.expandSidebar : fullCopy.collapseSidebar"
            @click="toggleSidebar"
          >
            <PanelLeft aria-hidden="true" :size="17" />
          </button>

          <!-- Mobile hamburger — opens full nav sheet -->
          <Sheet
            :default-open="false"
            :model-value="mobileSheet === 'nav'"
            @update:model-value="(v: boolean) => { if (!v) closeMobileSheet() }"
          >
            <SheetTrigger>
              <button
                id="mobile-sidebar-open"
                class="sidebar-toggle-mobile tbtn"
                type="button"
                :aria-label="fullCopy.openSidebar"
                :title="fullCopy.openSidebar"
              >
                <Menu aria-hidden="true" :size="17" />
              </button>
            </SheetTrigger>
            <SheetContent side="left" class="overflow-hidden p-0" :show-close-button="false">
              <SheetTitle class="sr-only">{{ fullCopy.primaryNavigation }}</SheetTitle>
              <div class="mobile-nav-surface" data-shell-navigation="sidebar">
                <div class="brand">
                  <span class="logo" aria-hidden="true">{{ brand.logo }}</span>
                  <div><b>{{ brand.label }}</b><small>{{ brand.subtitle }}</small></div>
                </div>
                <nav :aria-label="fullCopy.primaryNavigation" class="nav">
                  <div v-for="group in navigation.groups" :key="group.key">
                    <p class="nav-group">{{ group.label }}<span /></p>
                    <template v-for="item in group.items" :key="item.key">
                      <span
                        v-if="item.disabled"
                        class="nav-item cursor-not-allowed"
                        :data-label="item.label"
                        :aria-label="item.label"
                        aria-disabled="true"
                      >
                        <component :is="item.icon" v-if="item.icon" aria-hidden="true" :size="18" />
                        <span class="nav-label">{{ item.label }}</span>
                      </span>
                      <div v-else-if="item.children" class="nav-item-wrapper relative">
                        <button
                          type="button"
                          class="nav-item cursor-pointer"
                          :class="{ active: item.current }"
                          :aria-label="item.label"
                          :aria-expanded="isBranchOpen(item.key)"
                          @click="toggleParent(item.key)"
                        >
                          <component :is="item.icon" v-if="item.icon" aria-hidden="true" :size="18" />
                          <span class="nav-label">{{ item.label }}</span>
                          <ChevronRight aria-hidden="true" :size="12" class="nav-chevron" :style="isBranchOpen(item.key) ? { transform: 'rotate(90deg)' } : undefined" />
                        </button>
                        <div class="nav-children mt-0.5 mb-1 ml-3.5 border-l border-border pl-1 space-y-px" :class="{ hidden: !isBranchOpen(item.key) }">
                          <a
                            v-for="child in item.children"
                            :key="child.key"
                            class="nav-child"
                            :class="{ 'is-current': child.current }"
                            :href="child.href ?? '#'"
                            @click.prevent="closeMobileSheet(child.href)"
                          >
                            <component :is="child.icon" v-if="child.icon" aria-hidden="true" :size="14" />
                            {{ child.label }}
                          </a>
                        </div>
                      </div>
                      <a
                        v-else
                        class="nav-item"
                        :class="{ active: item.current }"
                        :href="item.href ?? '#'"
                        @click.prevent="closeMobileSheet(item.href)"
                      >
                        <component :is="item.icon" v-if="item.icon" aria-hidden="true" :size="18" />
                        <span class="nav-label">{{ item.label }}</span>
                        <span v-if="item.badge" class="badge">{{ item.badge }}</span>
                      </a>
                    </template>
                  </div>
                </nav>
                <div class="sidefoot">
                  <template v-for="item in navigation.footer" :key="item.key">
                    <a
                      v-if="item.href && !item.disabled"
                      class="nav-item"
                      :href="item.href"
                      @click.prevent="closeMobileSheet(item.href)"
                    >
                      <component :is="item.icon" v-if="item.icon" aria-hidden="true" :size="18" />
                      <span class="nav-label">{{ item.label }}</span>
                    </a>
                  </template>
                  <span class="nav-item" :data-label="identityLabel">
                    <User aria-hidden="true" :size="18" />
                    <span class="nav-label">{{ identityLabel }}</span>
                  </span>
                </div>
              </div>
            </SheetContent>
          </Sheet>

          <!-- Mobile search trigger -->
          <button
            id="mobile-search-trigger"
            class="sidebar-toggle-mobile tbtn"
            type="button"
            :aria-label="fullCopy.search"
            :title="fullCopy.search"
          >
            <Search aria-hidden="true" :size="17" />
          </button>
        </div>

        <div class="topbar-center flex items-center gap-3 flex-1 min-w-0" data-slot="topbar-center">
          <div class="crumb flex-shrink-0" data-slot="crumb">
            <template v-for="part in crumbTrail" :key="part">{{ part }} / </template><b>{{ crumbCurrent }}</b>
          </div>
          <div class="search min-w-0" data-slot="search">
            <button type="button">
              <Search aria-hidden="true" :size="16" />
              <span class="flex-1 truncate">{{ searchPlaceholder }}</span>
              <kbd class="ml-auto flex-shrink-0 px-2 py-px border border-border rounded text-text-muted bg-surface text-xs">⌘ K</kbd>
            </button>
          </div>
        </div>

        <div class="topbar-right flex items-center gap-2 flex-shrink-0" data-slot="topbar-right">
          <slot name="topbar-actions" />
        </div>
      </header>

      <div class="content" data-slot="content">
        <slot />
      </div>
    </main>

    <!-- Mobile bottom nav -->
    <nav
      class="fixed inset-x-0 bottom-0 z-30 flex h-14 items-stretch border-t border-border bg-surface pb-[env(safe-area-inset-bottom)] lg:hidden"
      data-slot="mobile-bottom-nav"
      :aria-label="fullCopy.mobilePrimaryNavigation"
    >
      <a
        v-for="item in navigation.mobile"
        :key="item.key"
        class="flex min-h-11 flex-1 flex-col items-center justify-center gap-0.5 px-1 text-fs-xs leading-none"
        :class="item.current ? 'text-brand-600' : 'text-text-muted'"
        :href="item.href ?? '#'"
        :aria-current="item.current ? 'page' : undefined"
        @click.prevent="onNavClick(item.href)"
      >
        <component :is="item.icon" v-if="item.icon" aria-hidden="true" :size="20" />
        <span class="max-w-full truncate">{{ item.label }}</span>
      </a>

      <!-- 更多 overflow sheet -->
      <Sheet
        :default-open="false"
        :model-value="mobileSheet === 'more'"
        @update:model-value="(v: boolean) => { if (!v) closeMobileSheet() }"
      >
        <SheetTrigger>
          <button
            type="button"
            class="flex min-h-11 flex-1 flex-col items-center justify-center gap-0.5 px-1 text-fs-xs leading-none text-text-muted"
            :aria-label="fullCopy.moreNavigation"
          >
            <Menu aria-hidden="true" :size="20" />
            <span>{{ fullCopy.more }}</span>
          </button>
        </SheetTrigger>
        <SheetContent side="bottom" class="overflow-hidden p-0" :show-close-button="false">
          <SheetTitle class="sr-only">{{ fullCopy.moreNavigation }}</SheetTitle>
          <div class="mobile-nav-surface" data-shell-navigation="sidebar">
            <div class="brand">
              <span class="logo" aria-hidden="true">{{ brand.logo }}</span>
              <div><b>{{ brand.label }}</b><small>{{ brand.subtitle }}</small></div>
            </div>
            <nav :aria-label="fullCopy.moreNavigation" class="nav">
              <div v-for="group in overflowNavigation.groups" :key="group.key">
                <p class="nav-group">{{ group.label }}<span /></p>
                <template v-for="item in group.items" :key="item.key">
                  <span
                    v-if="item.disabled"
                    class="nav-item cursor-not-allowed"
                    :aria-label="item.label"
                    aria-disabled="true"
                  >
                    <component :is="item.icon" v-if="item.icon" aria-hidden="true" :size="18" />
                    <span class="nav-label">{{ item.label }}</span>
                  </span>
                  <a
                    v-else
                    class="nav-item"
                    :class="{ active: item.current }"
                    :href="item.href ?? '#'"
                    @click.prevent="closeMobileSheet(item.href)"
                  >
                    <component :is="item.icon" v-if="item.icon" aria-hidden="true" :size="18" />
                    <span class="nav-label">{{ item.label }}</span>
                    <span v-if="item.badge" class="badge">{{ item.badge }}</span>
                  </a>
                </template>
              </div>
            </nav>
            <div class="sidefoot">
              <template v-for="item in overflowNavigation.footer" :key="item.key">
                <a
                  v-if="item.href && !item.disabled"
                  class="nav-item"
                  :href="item.href"
                  @click.prevent="closeMobileSheet(item.href)"
                >
                  <component :is="item.icon" v-if="item.icon" aria-hidden="true" :size="18" />
                  <span class="nav-label">{{ item.label }}</span>
                </a>
              </template>
            </div>
          </div>
        </SheetContent>
      </Sheet>
    </nav>
  </div>
</template>
