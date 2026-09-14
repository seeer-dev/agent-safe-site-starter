<script setup lang="ts">
// 商品列表島 — 對應 reference shop-view.tsx：
//   頁頭（標題/數量）、sticky 控制列（分類 chips + 搜尋 + 排序）、
//   商品格（skeleton/錯誤/空態/網格）。
// MPA 差異：
//   - 分類 chips 導向真實頁面 /categories/<slug>/（簾幕轉場）；
//   - 搜尋/排序為 client-side refetch + history.replaceState
//     （URL 保持可分享，不整頁重載）；
//   - ?category / ?q / ?sort 查詢字串皆支援（header 搜尋用 ?q）。
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { PackageSearch, RotateCcw, Search, X, ChevronDown } from 'lucide-vue-next'
import ProductCard from '@/shared/components/ProductCard.vue'
import Skeleton from '@/shared/components/Skeleton.vue'
import { apiGet } from '@/shared/lib/api'
import { bootstrap, loadBootstrap } from '@/shared/lib/store'
import { navigate } from '@/shared/lib/transition'
import { cn } from '@/shared/lib/utils'
import type { ProductDTO } from '@/shared/lib/types'

const props = defineProps<{ initialCategory?: string }>()

const SORT_OPTIONS = [
  { value: 'new', label: '最新上架' },
  { value: 'price_asc', label: '價格：低 → 高' },
  { value: 'price_desc', label: '價格：高 → 低' },
  { value: 'popular', label: '熱門商品' },
]

const params = new URLSearchParams(window.location.search)
const cat = ref(props.initialCategory || params.get('category') || params.get('cat') || '')
const q = ref(params.get('q') ?? '')
const sort = ref(params.get('sort') || 'new')
const searchInput = ref(q.value)
const sortOpen = ref(false)

const items = ref<ProductDTO[]>([])
const loading = ref(true)
const fetching = ref(false)
const error = ref(false)

const categories = computed(() => bootstrap.data?.categories ?? [])
const activeCat = computed(() => categories.value.find((c) => c.slug === cat.value) ?? null)
const hasFilter = computed(() => Boolean(cat.value || q.value))

const title = computed(() =>
  activeCat.value ? activeCat.value.name : q.value ? `搜尋「${q.value}」` : '全部商品',
)
const description = computed(
  () =>
    activeCat.value?.description ??
    (q.value ? `為「${q.value}」找到 ${items.value.length} 件嚴選商品` : '每一件商品，都經過編輯部親自挑選與試用。'),
)
const sortLabel = computed(() => SORT_OPTIONS.find((o) => o.value === sort.value)?.label ?? '最新上架')

async function fetchProducts() {
  fetching.value = true
  error.value = false
  const qs = new URLSearchParams()
  if (cat.value) qs.set('category', cat.value)
  if (q.value) qs.set('q', q.value)
  if (sort.value && sort.value !== 'new') qs.set('sort', sort.value)
  try {
    const res = await apiGet<{ products: ProductDTO[] }>(`/api/products?${qs.toString()}`)
    items.value = res.products ?? []
  } catch {
    error.value = true
  } finally {
    loading.value = false
    fetching.value = false
  }
}

// 狀態變更 → 更新 URL（replaceState,不整頁導航）+ refetch
function syncURL() {
  const qs = new URLSearchParams()
  if (cat.value) qs.set('category', cat.value)
  if (q.value) qs.set('q', q.value)
  if (sort.value && sort.value !== 'new') qs.set('sort', sort.value)
  const url = `${window.location.pathname}${qs.size ? '?' + qs.toString() : ''}`
  window.history.replaceState(null, '', url)
}

watch([cat, q, sort], () => {
  syncURL()
  void fetchProducts()
})

function submitSearch() {
  q.value = searchInput.value.trim()
}
function clearSearch() {
  searchInput.value = ''
  q.value = ''
}
function setSort(v: string) {
  sort.value = v
  sortOpen.value = false
}
function clearAll() {
  searchInput.value = ''
  q.value = ''
  if (cat.value) {
    navigate('/shop/')
  } else {
    void fetchProducts()
  }
}

function onPopState() {
  const p = new URLSearchParams(window.location.search)
  cat.value = props.initialCategory || p.get('category') || p.get('cat') || ''
  q.value = p.get('q') ?? ''
  sort.value = p.get('sort') || 'new'
  searchInput.value = q.value
}

function onDocClick(e: MouseEvent) {
  const t = e.target as HTMLElement
  if (!t.closest('[data-sort-menu]')) sortOpen.value = false
}

onMounted(async () => {
  window.addEventListener('popstate', onPopState)
  document.addEventListener('click', onDocClick)
  await Promise.all([loadBootstrap().catch(() => undefined), fetchProducts()])
  // 島成功載入 → 隱藏靜態基線（no-JS 使用者仍看得到 #shop-static）。
  document.getElementById('shop-header-static')?.setAttribute('hidden', '')
  document.getElementById('shop-controls-static')?.setAttribute('hidden', '')
  document.getElementById('shop-static')?.setAttribute('hidden', '')
})

onUnmounted(() => {
  window.removeEventListener('popstate', onPopState)
  document.removeEventListener('click', onDocClick)
})
</script>

<template>
  <div>
    <!-- 頁頭 -->
    <div class="mx-auto w-full max-w-7xl px-4 pt-10 sm:px-6 sm:pt-14 lg:px-8">
      <div>
        <p class="eyebrow">All Products</p>
        <h1 class="mt-3 font-serif text-2xl font-semibold tracking-wide sm:text-3xl">{{ title }}</h1>
        <p class="mt-2 max-w-lg text-sm leading-relaxed text-muted-foreground">{{ description }}</p>
        <p class="mt-4 text-sm text-muted-foreground">
          {{ loading ? '搜尋中…' : `共 ${items.length} 件商品` }}
        </p>
      </div>
    </div>

    <!-- 控制列（sticky） -->
    <div class="sticky top-14 z-30 mt-6 border-y bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/80 sm:top-16">
      <div class="mx-auto flex w-full max-w-7xl flex-col gap-3 px-4 py-3 sm:px-6 lg:flex-row lg:items-center lg:px-8">
        <!-- 分類 chips：真實頁面導航（簾幕轉場） -->
        <div class="flex flex-1 gap-2 overflow-x-auto py-0.5 no-scrollbar">
          <a
            href="/shop/"
            :class="cn(
              'shrink-0 whitespace-nowrap rounded-full border px-4 py-1.5 text-sm transition-all',
              !cat ? 'border-ink bg-ink text-cream' : 'border-border bg-card text-foreground/75 hover:border-foreground/40 hover:text-foreground',
            )"
            aria-pressed="true"
          >全部</a>
          <a
            v-for="c in categories"
            :key="c.id"
            :href="`/categories/${c.slug}/`"
            :class="cn(
              'shrink-0 whitespace-nowrap rounded-full border px-4 py-1.5 text-sm transition-all',
              cat === c.slug ? 'border-ink bg-ink text-cream' : 'border-border bg-card text-foreground/75 hover:border-foreground/40 hover:text-foreground',
            )"
            :aria-pressed="cat === c.slug"
          >{{ c.name }}</a>
        </div>

        <!-- 搜尋 + 排序 -->
        <div class="flex shrink-0 items-center gap-2">
          <form class="relative" @submit.prevent="submitSearch">
            <Search class="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-muted-foreground" />
            <input
              v-model="searchInput"
              type="text"
              placeholder="搜尋商品…"
              aria-label="搜尋商品"
              class="h-9 w-40 rounded-full border bg-card pl-9 pr-8 text-sm outline-none focus:border-primary sm:w-52"
              @keydown.esc="clearSearch"
            />
            <button
              v-if="searchInput || q"
              type="button"
              aria-label="清除搜尋"
              class="absolute right-2.5 top-1/2 -translate-y-1/2 rounded-full p-0.5 text-muted-foreground hover:text-foreground"
              @click="clearSearch"
            >
              <X class="size-3" />
            </button>
          </form>

          <!-- 排序下拉 -->
          <div class="relative" data-sort-menu>
            <button
              type="button"
              class="flex h-9 w-[132px] items-center justify-between rounded-full border bg-card px-3.5 text-sm"
              aria-label="排序方式"
              :aria-expanded="sortOpen"
              @click="sortOpen = !sortOpen"
            >
              {{ sortLabel }}
              <ChevronDown class="size-3.5 transition-transform" :class="sortOpen && 'rotate-180'" />
            </button>
            <Transition name="menu">
              <div
                v-if="sortOpen"
                class="absolute right-0 top-full z-50 mt-1 w-44 rounded-xl border bg-popover p-1.5 shadow-lg"
              >
                <button
                  v-for="o in SORT_OPTIONS"
                  :key="o.value"
                  type="button"
                  :class="cn(
                    'block w-full rounded-lg px-3 py-2 text-left text-sm transition-colors hover:bg-muted',
                    sort === o.value && 'bg-muted font-medium',
                  )"
                  @click="setSort(o.value)"
                >{{ o.label }}</button>
              </div>
            </Transition>
          </div>
        </div>
      </div>
    </div>

    <!-- 商品格 -->
    <div class="mx-auto w-full max-w-7xl px-4 py-10 sm:px-6 lg:px-8">
      <div v-if="loading" class="grid grid-cols-2 gap-x-4 gap-y-10 sm:grid-cols-3 lg:grid-cols-4 lg:gap-x-6">
        <div v-for="i in 8" :key="i">
          <Skeleton class="aspect-[4/5] w-full rounded-xl" />
          <Skeleton class="mt-3 h-3 w-1/3" />
          <Skeleton class="mt-2 h-4 w-4/5" />
          <Skeleton class="mt-2 h-4 w-1/4" />
        </div>
      </div>

      <div v-else-if="error" class="flex flex-col items-center py-20 text-center">
        <PackageSearch class="size-10 text-muted-foreground/40" :stroke-width="1.5" />
        <h3 class="mt-4 font-serif text-xl font-semibold">商品載入失敗</h3>
        <p class="mt-2 text-sm text-muted-foreground">網路似乎不太穩定，請再試一次。</p>
        <button
          type="button"
          class="mt-6 inline-flex h-10 items-center gap-2 rounded-full border px-5 text-sm transition-colors hover:bg-muted"
          @click="fetchProducts"
        >
          <RotateCcw class="size-4" />
          重新載入
        </button>
      </div>

      <div v-else-if="items.length === 0" class="flex flex-col items-center py-20 text-center">
        <PackageSearch class="size-10 text-muted-foreground/40" :stroke-width="1.5" />
        <h3 class="mt-4 font-serif text-xl font-semibold">找不到相符的商品</h3>
        <p class="mt-2 text-sm text-muted-foreground">
          {{ hasFilter ? '試試更寬鬆的關鍵字，或換個分類逛逛。' : '商品正在上架中，請稍後再回來看看。' }}
        </p>
        <button
          v-if="hasFilter"
          type="button"
          class="mt-6 inline-flex h-10 items-center rounded-full bg-primary px-5 text-sm font-medium text-primary-foreground"
          @click="clearAll"
        >清除篩選條件</button>
      </div>

      <div
        v-else
        :class="cn('grid grid-cols-2 gap-x-4 gap-y-10 sm:grid-cols-3 lg:grid-cols-4 lg:gap-x-6', fetching && 'opacity-60 transition-opacity')"
      >
        <ProductCard v-for="(p, i) in items" :key="p.id" :product="p" :index="i" />
      </div>
    </div>
  </div>
</template>

<style scoped>
.menu-enter-active,
.menu-leave-active {
  transition: opacity 0.16s ease, transform 0.16s ease;
}
.menu-enter-from,
.menu-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}
</style>
