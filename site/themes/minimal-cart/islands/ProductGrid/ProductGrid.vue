<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { fetchProducts } from '@/shared/lib/api'
import type { Category, Product } from '@/shared/lib/types'
import { CATEGORIES } from '@/shared/lib/mock-data'
import ProductCard from '@/shared/components/site/ProductCard.vue'
import CategoryFilter from '@/shared/components/site/CategoryFilter.vue'
import Pagination from '@/shared/components/site/Pagination.vue'
import Skeleton from '@/shared/components/ui/Skeleton.vue'
import { useUiStore } from '@/shared/stores/ui'

const ui = useUiStore()

const props = defineProps<{
  initialCategory?: string
}>()

const ALLOWED_CATEGORIES: Category[] = ['all', 'apparel', 'accessories', 'home', 'stationery']

function normalizeCategory(raw?: string): Category {
  if (raw && (ALLOWED_CATEGORIES as string[]).includes(raw)) return raw as Category
  return 'all'
}

const category = ref<Category>(normalizeCategory(props.initialCategory))
const searchQuery = ref('')

const heading = computed(() => {
  if (category.value === 'all') return '全部商品'
  return CATEGORIES.find((c) => c.value === category.value)?.label ?? '全部商品'
})

// Products are fetched from the Go API on mount. The server-rendered
// static product grid (#shop-static) is the no-JS baseline and remains
// visible during loading and error. Only on successful load does this
// island hide the static section and show the interactive grid.
const products = ref<Product[]>([])
const isLoading = ref(true)
const error = ref<string | null>(null)
const isFiltering = ref(false)
let loadingTimer: ReturnType<typeof setTimeout> | null = null

// Hide the server-rendered static product section once the interactive
// grid has successfully loaded. During loading/error the static section
// stays visible so no-JS users and failed-fetch users still see products.
function hideStaticSection() {
  document.getElementById('shop-static')?.setAttribute('hidden', '')
}
function showStaticSection() {
  document.getElementById('shop-static')?.removeAttribute('hidden')
}

// Listen for search events from the Header island.
// (Search is page-scoped ephemeral state — not worth a store.)
window.addEventListener('minimal-cart:search', (e: Event) => {
  searchQuery.value = (e as CustomEvent<string>).detail ?? ''
})

onMounted(async () => {
  await loadProducts()
})

async function loadProducts() {
  error.value = null
  isLoading.value = true
  try {
    products.value = await fetchProducts()
    // Only hide the static baseline after a successful fetch.
    hideStaticSection()
  } catch {
    error.value = '無法載入商品，請稍後再試。'
    // Keep static section visible on error — truthful products remain.
    showStaticSection()
  } finally {
    isLoading.value = false
  }
}

const ITEMS_PER_PAGE = 8
const page = ref(1)

const filtered = computed(() => {
  const q = searchQuery.value.trim().toLowerCase()
  return products.value.filter((p) => {
    const matchCat = category.value === 'all' || p.category === category.value
    const matchSearch = !q || p.name.toLowerCase().includes(q) || p.description.toLowerCase().includes(q) || p.category.toLowerCase().includes(q)
    return matchCat && matchSearch
  })
})

const totalPages = computed(() => Math.max(1, Math.ceil(filtered.value.length / ITEMS_PER_PAGE)))

watch(() => `${category.value}-${searchQuery.value}`, () => {
  page.value = 1
  isFiltering.value = true
  if (loadingTimer) clearTimeout(loadingTimer)
  loadingTimer = setTimeout(() => {
    isFiltering.value = false
  }, 400)
})

const safePage = computed(() => Math.min(Math.max(1, page.value), totalPages.value))

const paged = computed(() => {
  const start = (safePage.value - 1) * ITEMS_PER_PAGE
  return filtered.value.slice(start, start + ITEMS_PER_PAGE)
})

function onPageChange(p: number) {
  page.value = p
  document.getElementById('shop')?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}
</script>

<template>
  <section id="shop" class="mx-auto max-w-7xl px-4 py-16 sm:px-6 lg:px-8">
    <div class="flex flex-col gap-6 sm:flex-row sm:items-end sm:justify-between">
      <div>
        <p class="text-xs font-medium uppercase tracking-[0.18em] text-muted-foreground">線上商品</p>
        <h2 class="mt-2 text-2xl font-semibold tracking-tight sm:text-3xl">{{ heading }}</h2>
      </div>
      <CategoryFilter :active="category" @change="category = $event" />
    </div>

    <!-- Initial load keeps the static #shop-static baseline visible. -->
    <div v-if="isLoading" class="sr-only">載入商品中</div>

    <!-- Skeleton loading (filter change only; initial load uses static baseline) -->
    <div v-else-if="!isLoading && isFiltering" class="mt-10 grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
      <div
        v-for="i in ITEMS_PER_PAGE"
        :key="`skel-${i}`"
        class="flex flex-col overflow-hidden rounded-2xl border border-border/60"
      >
        <Skeleton class="aspect-[4/5] w-full rounded-none" />
        <div class="space-y-2 p-4">
          <Skeleton class="h-4 w-3/4" />
          <Skeleton class="h-3 w-full" />
          <Skeleton class="h-3 w-2/3" />
          <div class="flex items-center justify-between pt-2">
            <Skeleton class="h-5 w-20" />
            <Skeleton class="h-3 w-12" />
          </div>
        </div>
      </div>
    </div>

    <!-- Error state -->
    <div v-else-if="error" class="mt-16 flex flex-col items-center justify-center rounded-2xl border border-dashed border-border/60 py-20 text-center">
      <p class="text-sm font-medium">{{ error }}</p>
      <button
        @click="loadProducts"
        class="mt-4 inline-flex h-9 items-center rounded-full bg-cta px-5 text-sm font-medium text-cta-foreground transition-colors hover:brightness-95"
      >
        重試
      </button>
    </div>

    <!-- Actual grid -->
    <template v-else-if="filtered.length > 0">
      <div :key="`${category}-${searchQuery}-${safePage}`" class="mt-10 grid grid-cols-1 gap-5 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
        <ProductCard
          v-for="(p, i) in paged"
          :key="p.id"
          :product="p"
          :index="i"
        />
      </div>

      <div v-if="totalPages > 1" class="mt-10 flex flex-col items-center gap-2">
        <Pagination
          :current-page="safePage"
          :total-pages="totalPages"
          @page-change="onPageChange"
        />
        <p class="text-xs text-muted-foreground">
          第 {{ safePage }} / {{ totalPages }} 頁 · 每頁 {{ ITEMS_PER_PAGE }} 件
        </p>
      </div>
    </template>

    <div v-else class="mt-16 flex flex-col items-center justify-center rounded-2xl border border-dashed border-border/60 py-20 text-center">
      <p class="text-sm font-medium">找不到符合的商品</p>
      <p class="mt-1 text-xs text-muted-foreground">換個關鍵字或分類試試。</p>
    </div>

    <p v-if="!isLoading && !error" class="mt-8 text-center text-xs text-muted-foreground">
      共 {{ filtered.length }} / {{ products.length }} 件商品
    </p>
  </section>
</template>
