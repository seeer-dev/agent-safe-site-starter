<script setup lang="ts">
// 黏置頁首 — 對應 reference header.tsx：
//   滾動 >10px 切換玻璃擬態；桌面中欄導覽（全部商品/分類下拉/公告/故事）；
//   右側 search → 訂單查詢 → 深色切換 → 購物車徽標；手機 hamburger + 面板。
// MPA 差異：導覽為真實 <a href>（no-JS 可用），轉場由全域攔截器播簾幕。
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { Loader2, Menu, Search, SearchX, Sun, Moon, ShoppingBag, X, ChevronDown } from 'lucide-vue-next'
import FadeImage from '@/shared/components/FadeImage.vue'
import LogoLockup from '@/shared/components/LogoLockup.vue'
import PriceDisplay from '@/shared/components/PriceDisplay.vue'
import Skeleton from '@/shared/components/Skeleton.vue'
import { apiGet } from '@/shared/lib/api'
import { bootstrap, loadBootstrap, openCartDrawer } from '@/shared/lib/store'
import { cartCount } from '@/shared/lib/cart'
import { isDark, toggleTheme } from '@/shared/lib/theme'
import { navigate } from '@/shared/lib/transition'
import type { ProductDTO } from '@/shared/lib/types'

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
  { href: '/news/', label: '最新公告', active: (p: string) => p.startsWith('/news') || p.startsWith('/articles') },
  { href: '/about/', label: '品牌故事', active: (p: string) => p.startsWith('/about') },
  { href: '/track/', label: '訂單查詢', active: (p: string) => p.startsWith('/track') || p.startsWith('/order') },
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

// ─── 搜尋彈層（reference search-dialog.tsx）：300ms debounce 即時結果 ──
const debouncedQuery = ref('')
const searchResults = ref<ProductDTO[]>([])
const searchFetching = ref(false)
let debounceTimer: number | null = null
let searchSeq = 0

watch(searchQuery, (v) => {
  if (debounceTimer) window.clearTimeout(debounceTimer)
  debounceTimer = window.setTimeout(() => {
    debouncedQuery.value = v.trim()
  }, 300)
})

watch(debouncedQuery, async (q) => {
  if (!q) {
    searchResults.value = []
    return
  }
  const seq = ++searchSeq
  searchFetching.value = true
  try {
    const res = await apiGet<{ products: ProductDTO[] }>(`/api/products?q=${encodeURIComponent(q)}&sort=popular`)
    if (seq === searchSeq) searchResults.value = res.products ?? []
  } catch {
    if (seq === searchSeq) searchResults.value = []
  } finally {
    if (seq === searchSeq) searchFetching.value = false
  }
})

watch(searchOpen, (open) => {
  if (!open) {
    searchQuery.value = ''
    debouncedQuery.value = ''
    searchResults.value = []
  }
})

function openProduct(p: ProductDTO) {
  searchOpen.value = false
  navigate(`/products/${p.slug}/`)
}

function submitSearch() {
  const first = searchResults.value[0]
  if (first) {
    openProduct(first)
    return
  }
  const q = searchQuery.value.trim()
  if (!q) return
  searchOpen.value = false
  navigate(`/shop/?q=${encodeURIComponent(q)}`)
}

function goShop() {
  searchOpen.value = false
  navigate('/shop/')
}

function categoryName(p: ProductDTO) {
  return categories.value.find((c) => c.slug === p.category)?.name ?? '嚴選商品'
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
      <nav class="hidden items-center gap-7 md:flex" aria-label="主選單">
        <a
          v-for="n in NAV"
          :key="n.href"
          :href="n.href"
          class="link-underline py-1 text-sm tracking-wide transition-colors"
          :class="isActive(n) ? 'font-medium text-foreground' : 'text-muted-foreground hover:text-foreground'"
          :aria-current="isActive(n) ? 'page' : undefined"
        >{{ n.label }}</a>

        <!-- 分類下拉 -->
        <div class="relative">
          <button
            ref="catBtn"
            type="button"
            class="flex items-center gap-1 py-1 text-sm tracking-wide outline-none transition-colors"
            :class="path.startsWith('/categories') ? 'font-medium text-foreground' : 'text-muted-foreground hover:text-foreground'"
            @click="catMenuOpen = !catMenuOpen"
          >
            分類探索
            <ChevronDown class="size-3.5 opacity-60 transition-transform" :class="catMenuOpen ? 'rotate-180' : ''" />
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
              <hr class="my-1 border-border" />
              <a
                href="/shop/"
                class="block rounded-lg px-3 py-2 text-sm text-primary transition-colors hover:bg-muted"
                @click="catMenuOpen = false"
              >查看全部商品</a>
            </div>
          </Transition>
        </div>
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

    <!-- 搜尋彈層 — 300ms debounce 即時結果（reference search-dialog.tsx） -->
    <Transition name="search">
      <div v-if="searchOpen" class="fixed inset-0 z-[70] flex items-start justify-center px-4 pt-[12vh]" role="dialog" aria-modal="true" aria-label="商品搜尋">
        <div class="absolute inset-0 bg-ink/40" @click="searchOpen = false" />
        <div class="relative max-h-[76vh] w-full max-w-xl overflow-hidden rounded-2xl border bg-popover shadow-xl">
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
            <Loader2 v-if="searchFetching" class="size-4 shrink-0 animate-spin text-muted-foreground" />
            <button v-else-if="searchQuery" type="button" class="shrink-0 text-xs text-muted-foreground hover:text-foreground" @click="searchQuery = ''">清除</button>
            <kbd class="hidden shrink-0 rounded border bg-muted px-1.5 py-0.5 font-mono text-[10px] text-muted-foreground sm:block">ESC</kbd>
          </div>

          <div class="scrollbar-thin max-h-[60vh] overflow-y-auto">
            <!-- 未輸入 -->
            <div v-if="!debouncedQuery" class="px-6 py-10 text-center">
              <p class="text-sm text-muted-foreground">輸入關鍵字，立即探索店內嚴選商品</p>
              <p class="mt-1 text-xs text-muted-foreground/70">小提示：試試「馬克杯」「亞麻」「蠟燭」</p>
            </div>
            <!-- 載入中 -->
            <div v-else-if="searchFetching && searchResults.length === 0" class="space-y-3 p-4">
              <div v-for="i in 4" :key="i" class="flex items-center gap-3">
                <Skeleton class="h-14 w-12 rounded-lg" />
                <div class="flex-1 space-y-2">
                  <Skeleton class="h-4 w-2/3" />
                  <Skeleton class="h-3 w-16" />
                </div>
              </div>
            </div>
            <!-- 無結果 -->
            <div v-else-if="searchResults.length === 0" class="px-6 py-10 text-center">
              <SearchX class="mx-auto size-6 text-muted-foreground/50" />
              <p class="mt-3 text-sm font-medium">找不到「{{ debouncedQuery }}」的結果</p>
              <p class="mt-1 text-xs text-muted-foreground">換個關鍵字試試，或到全部商品逛逛</p>
              <button type="button" class="link-underline mt-4 text-sm text-primary" @click="goShop">前往全部商品</button>
            </div>
            <!-- 結果 -->
            <ul v-else class="divide-y">
              <li v-for="p in searchResults.slice(0, 8)" :key="p.id">
                <button type="button" class="flex w-full items-center gap-3.5 px-4 py-3 text-left transition-colors hover:bg-muted/50" @click="openProduct(p)">
                  <div class="h-14 w-12 shrink-0 overflow-hidden rounded-lg bg-muted/40">
                    <FadeImage :src="p.images?.[0] ?? p.image ?? ''" :alt="p.name" class="h-full w-full object-cover" />
                  </div>
                  <div class="min-w-0 flex-1">
                    <p class="clamp-1 text-sm font-medium">{{ p.name }}</p>
                    <p class="mt-0.5 text-[11px] text-muted-foreground">
                      {{ categoryName(p) }}{{ p.stock <= 0 ? '・已售完' : '' }}
                    </p>
                  </div>
                  <PriceDisplay :price="p.price" :original-price="p.original_price" size="sm" class="shrink-0" />
                </button>
              </li>
            </ul>
          </div>

          <div class="border-t px-4 py-2.5 text-[10px] tracking-wider text-muted-foreground/70">
            按下 Enter 開啟第一個結果<template v-if="debouncedQuery">・共 {{ searchResults.length }} 筆相關商品</template>
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
