<script setup lang="ts">
// 黏置頁首 — 對應 reference header.tsx：
//   滾動 >10px 切換玻璃擬態；桌面中欄導覽（全部商品/分類下拉/公告/故事）；
//   右側 search → 訂單查詢 → 深色切換 → 購物車徽標；手機 hamburger + 面板。
// MPA 差異：導覽為真實 <a href>（no-JS 可用），轉場由全域攔截器播簾幕。
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { Menu, Search, Package, Sun, Moon, ShoppingBag, X, ChevronRight, Package2 } from 'lucide-vue-next'
import LogoLockup from '@/shared/components/LogoLockup.vue'
import { bootstrap, loadBootstrap, openCartDrawer } from '@/shared/lib/store'
import { cartCount } from '@/shared/lib/cart'
import { isDark, toggleTheme } from '@/shared/lib/theme'

const props = defineProps<{ current?: string }>()

const scrolled = ref(false)
const mobileNavOpen = ref(false)
const catMenuOpen = ref(false)
const searchOpen = ref(false)
const searchQuery = ref('')
const catBtn = ref<HTMLElement | null>(null)
const catMenu = ref<HTMLElement | null>(null)
const searchInput = ref<HTMLInputElement | null>(null)

const count = computed(() => cartCount())
const categories = computed(() => bootstrap.data?.categories ?? [])
const storeName = computed(() => bootstrap.data?.settings?.storeName ?? '質選所')
const settings = computed(() => bootstrap.data?.settings)

const path = computed(() => props.current ?? window.location.pathname)
const NAV = [
  { href: '/shop/', label: '全部商品', active: (p: string) => p.startsWith('/shop') || p.startsWith('/products') || p.startsWith('/categories') },
  { href: '/news/', label: '最新消息', active: (p: string) => p.startsWith('/news') || p.startsWith('/articles') },
  { href: '/about/', label: '品牌故事', active: (p: string) => p.startsWith('/about') },
]
const isActive = (n: (typeof NAV)[number]) => n.active(path.value)

function onScroll() {
  scrolled.value = window.scrollY > 10
}

function onKey(e: KeyboardEvent) {
  if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
    e.preventDefault()
    openSearch()
  }
}

function onDocClick(e: MouseEvent) {
  if (!catMenuOpen.value) return
  const t = e.target as Node
  if (catMenu.value?.contains(t) || catBtn.value?.contains(t)) return
  catMenuOpen.value = false
}

function openSearch() {
  searchOpen.value = true
  setTimeout(() => searchInput.value?.focus(), 80)
}

function submitSearch() {
  const q = searchQuery.value.trim()
  if (!q) return
  searchOpen.value = false
  window.location.href = `/shop/?q=${encodeURIComponent(q)}`
}

function openCart() {
  openCartDrawer()
}

onMounted(async () => {
  window.addEventListener('scroll', onScroll, { passive: true })
  window.addEventListener('keydown', onKey)
  document.addEventListener('click', onDocClick)
  await loadBootstrap().catch(() => undefined)
})

onUnmounted(() => {
  window.removeEventListener('scroll', onScroll)
  window.removeEventListener('keydown', onKey)
  document.removeEventListener('click', onDocClick)
})
</script>

<template>
  <header
    class="sticky top-0 z-50 w-full transition-all duration-300"
    :class="scrolled ? 'border-b bg-background/80 shadow-sm backdrop-blur-xl' : 'bg-transparent'"
  >
    <div class="mx-auto flex h-16 w-full max-w-7xl items-center justify-between px-4 md:px-6">
      <!-- 左：手機 hamburger + 桌面 logo -->
      <div class="flex items-center gap-1">
        <button
          type="button"
          class="mr-1 inline-flex size-10 items-center justify-center rounded-full hover:bg-muted md:hidden"
          aria-label="開啟選單"
          @click="mobileNavOpen = true"
        >
          <Menu class="size-5" />
        </button>
        <a href="/" class="hidden items-center gap-2.5 md:flex" aria-label="回到首頁">
          <LogoLockup :store-name="storeName" />
        </a>
      </div>

      <!-- 手機置中 logo -->
      <a href="/" class="absolute left-1/2 -translate-x-1/2 md:hidden" aria-label="回到首頁">
        <LogoLockup :store-name="storeName" compact />
      </a>

      <!-- 桌面導覽 -->
      <nav class="hidden items-center gap-1 md:flex" aria-label="主選單">
        <a
          v-for="n in NAV.slice(0, 1)"
          :key="n.href"
          :href="n.href"
          class="rounded-full px-3.5 py-2 text-sm transition-colors hover:text-primary"
          :class="isActive(n) ? 'font-medium text-primary' : 'text-foreground/80'"
        >{{ n.label }}</a>

        <!-- 分類下拉 -->
        <div class="relative">
          <button
            ref="catBtn"
            type="button"
            class="flex items-center gap-1 rounded-full px-3.5 py-2 text-sm text-foreground/80 transition-colors hover:text-primary"
            :class="{ 'font-medium text-primary': path.startsWith('/categories') }"
            @click="catMenuOpen = !catMenuOpen"
          >
            商品分類
            <ChevronRight class="size-3.5 transition-transform" :class="catMenuOpen ? 'rotate-90' : 'rotate-0'" />
          </button>
          <Transition name="menu">
            <div
              v-if="catMenuOpen"
              ref="catMenu"
              class="absolute left-1/2 top-full z-50 mt-1 w-44 -translate-x-1/2 rounded-xl border bg-popover p-1.5 shadow-lg"
            >
              <a
                v-for="c in categories"
                :key="c.id"
                :href="`/categories/${c.slug}/`"
                class="block rounded-lg px-3 py-2 text-sm transition-colors hover:bg-muted"
                @click="catMenuOpen = false"
              >{{ c.name }}</a>
              <p v-if="categories.length === 0" class="px-3 py-2 text-xs text-muted-foreground">尚無分類</p>
            </div>
          </Transition>
        </div>

        <a
          v-for="n in NAV.slice(1)"
          :key="n.href"
          :href="n.href"
          class="rounded-full px-3.5 py-2 text-sm transition-colors hover:text-primary"
          :class="isActive(n) ? 'font-medium text-primary' : 'text-foreground/80'"
        >{{ n.label }}</a>
      </nav>

      <!-- 右側操作 -->
      <div class="flex items-center">
        <button
          type="button"
          class="inline-flex size-10 items-center justify-center rounded-full hover:bg-muted"
          aria-label="搜尋商品（Ctrl+K）"
          @click="openSearch"
        >
          <Search class="size-[18px]" />
        </button>
        <a
          href="/track/"
          class="hidden size-10 items-center justify-center rounded-full hover:bg-muted sm:inline-flex"
          :class="path.startsWith('/track') ? 'text-primary' : ''"
          aria-label="訂單查詢"
        >
          <Package class="size-[18px]" />
        </a>
        <button
          type="button"
          class="inline-flex size-10 items-center justify-center rounded-full hover:bg-muted"
          :aria-label="isDark ? '切換為淺色模式' : '切換為深色模式'"
          @click="toggleTheme"
        >
          <Sun v-if="isDark" class="size-[18px]" />
          <Moon v-else class="size-[18px]" />
        </button>
        <button
          type="button"
          class="relative inline-flex size-10 items-center justify-center rounded-full hover:bg-muted"
          aria-label="開啟購物車"
          @click="openCart"
        >
          <ShoppingBag class="size-[18px]" />
          <span
            v-if="count > 0"
            class="absolute right-0.5 top-0.5 flex size-4 items-center justify-center rounded-full bg-primary text-[10px] font-medium text-primary-foreground"
          >{{ count > 99 ? '99+' : count }}</span>
        </button>
      </div>
    </div>

    <!-- 手機導覽面板 -->
    <Transition name="mobile-nav">
      <div v-if="mobileNavOpen" class="fixed inset-0 z-[60] md:hidden">
        <div class="absolute inset-0 bg-ink/40" @click="mobileNavOpen = false" />
        <div class="absolute left-0 top-0 flex h-full w-[78%] max-w-xs flex-col bg-background shadow-xl">
          <div class="flex items-center justify-between border-b px-4 py-3.5">
            <LogoLockup :store-name="storeName" compact />
            <button type="button" class="inline-flex size-9 items-center justify-center rounded-full hover:bg-muted" aria-label="關閉選單" @click="mobileNavOpen = false">
              <X class="size-5" />
            </button>
          </div>
          <nav class="flex-1 overflow-y-auto px-3 py-4" aria-label="手機主選單">
            <a
              v-for="n in NAV"
              :key="n.href"
              :href="n.href"
              class="mb-1 flex items-center justify-between rounded-xl px-4 py-3 text-base transition-colors hover:bg-muted"
              :class="isActive(n) ? 'bg-muted font-medium text-primary' : ''"
            >
              {{ n.label }}
              <ChevronRight class="size-4 text-muted-foreground" />
            </a>
            <a href="/track/" class="mb-1 flex items-center justify-between rounded-xl px-4 py-3 text-base transition-colors hover:bg-muted" :class="path.startsWith('/track') ? 'bg-muted font-medium text-primary' : ''">
              訂單查詢
              <ChevronRight class="size-4 text-muted-foreground" />
            </a>
            <p class="mt-5 px-4 text-xs tracking-wider text-muted-foreground">商品分類</p>
            <a
              v-for="c in categories"
              :key="c.id"
              :href="`/categories/${c.slug}/`"
              class="flex items-center justify-between rounded-xl px-4 py-2.5 text-sm transition-colors hover:bg-muted"
            >
              {{ c.name }}
              <span class="text-xs text-muted-foreground">{{ c.product_count ?? '' }}</span>
            </a>
          </nav>
          <div class="border-t px-5 py-4 text-xs text-muted-foreground">
            <p v-if="settings?.contactEmail">客服信箱：{{ settings.contactEmail }}</p>
            <p class="mt-1">{{ settings?.footerTagline ?? '日常器物的質選' }}</p>
          </div>
        </div>
      </div>
    </Transition>

    <!-- 搜尋彈層（簡化版 — 即時結果在 /shop/ 頁面呈現） -->
    <Transition name="search">
      <div v-if="searchOpen" class="fixed inset-0 z-[70] flex items-start justify-center px-4 pt-[12vh]">
        <div class="absolute inset-0 bg-ink/40" @click="searchOpen = false" />
        <div class="relative w-full max-w-xl overflow-hidden rounded-2xl border bg-popover shadow-xl">
          <div class="flex items-center gap-3 border-b px-4 py-3.5">
            <Search class="size-4 shrink-0 text-muted-foreground" />
            <input
              ref="searchInput"
              v-model="searchQuery"
              type="text"
              placeholder="搜尋商品名稱、分類或關鍵字…"
              class="flex-1 bg-transparent text-sm outline-none placeholder:text-muted-foreground"
              maxlength="40"
              aria-label="搜尋關鍵字"
              @keydown.enter="submitSearch"
              @keydown.esc="searchOpen = false"
            />
            <button v-if="searchQuery" type="button" class="shrink-0 text-xs text-muted-foreground hover:text-foreground" @click="searchQuery = ''">清除</button>
            <kbd class="hidden shrink-0 rounded border bg-muted px-1.5 py-0.5 font-mono text-[10px] text-muted-foreground sm:block">ESC</kbd>
          </div>
          <div class="px-5 py-5 text-center text-sm text-muted-foreground">
            <Package2 class="mx-auto mb-2 size-5 text-muted-foreground/50" />
            輸入關鍵字後按 Enter 前往商品搜尋結果
          </div>
        </div>
      </div>
    </Transition>
  </header>
</template>

<style scoped>
.menu-enter-active,
.menu-leave-active {
  transition: opacity 0.16s ease, transform 0.16s ease;
}
.menu-enter-from,
.menu-leave-to {
  opacity: 0;
  transform: translate(-50%, -4px);
}
.mobile-nav-enter-active > div:first-child,
.mobile-nav-leave-active > div:first-child {
  transition: opacity 0.24s ease;
}
.mobile-nav-enter-active > div:last-child {
  transition: transform 0.28s cubic-bezier(0.22, 1, 0.36, 1);
}
.mobile-nav-leave-active > div:last-child {
  transition: transform 0.2s ease-in;
}
.mobile-nav-enter-from > div:first-child,
.mobile-nav-leave-to > div:first-child {
  opacity: 0;
}
.mobile-nav-enter-from > div:last-child,
.mobile-nav-leave-to > div:last-child {
  transform: translateX(-100%);
}
.search-enter-active {
  transition: opacity 0.2s ease;
}
.search-enter-active > div:last-child {
  transition: transform 0.24s cubic-bezier(0.22, 1, 0.36, 1), opacity 0.2s ease;
}
.search-leave-active {
  transition: opacity 0.16s ease;
}
.search-enter-from,
.search-leave-to {
  opacity: 0;
}
.search-enter-from > div:last-child {
  transform: translateY(12px) scale(0.98);
}
@media (prefers-reduced-motion: reduce) {
  .menu-enter-active,
  .menu-leave-active,
  .mobile-nav-enter-active > div,
  .mobile-nav-leave-active > div,
  .search-enter-active,
  .search-leave-active {
    transition: none;
  }
}
</style>
