<script setup lang="ts">
// 商品橫列島 — 首頁「本季嚴選」「最新商品」與商品頁「相關商品」共用。
// 掛上後改走 live API：後台改精選/上下架/售完即時反映，不需等 render。
// 靜態基線（props.staticId 指定的區塊）保留給 no-JS 與 API 失敗降級。
import { onMounted, ref } from 'vue'
import ProductCard from '@/shared/components/ProductCard.vue'
import Skeleton from '@/shared/components/Skeleton.vue'
import { apiGet } from '@/shared/lib/api'
import { loadBootstrap } from '@/shared/lib/store'
import type { ProductDTO } from '@/shared/lib/types'

const props = defineProps<{
  /** featured=精選(後台 is_featured) / latest=最新 / related=同分類 */
  mode: string
  /** related 模式：要排除的商品 slug（商品頁自己） */
  exclude?: string
  /** related 模式：分類 slug */
  category?: string
  /** 要隱藏的靜態基線區塊 id */
  staticId?: string
  limit?: string
}>()

const items = ref<ProductDTO[]>([])
const loading = ref(true)
const failed = ref(false)

onMounted(async () => {
  const limit = Math.max(1, parseInt(props.limit ?? '8', 10) || 8)
  try {
    const qs = new URLSearchParams()
    if (props.mode === 'featured') qs.set('featured', '1')
    if (props.mode === 'related' && props.category) qs.set('category', props.category)
    const res = await apiGet<{ products: ProductDTO[] }>(`/api/products?${qs.toString()}`)
    let list = res.products ?? []
    if (props.mode === 'related' && props.exclude) {
      list = list.filter((p) => p.slug !== props.exclude)
    }
    items.value = list.slice(0, limit)
    await loadBootstrap().catch(() => undefined)
    // 成功取得即時資料 → 隱藏靜態基線，改由本島渲染。
    if (props.staticId) document.getElementById(props.staticId)?.setAttribute('hidden', '')
  } catch {
    // API 失敗 → 保留靜態基線（舊快照總比空白好）。
    failed.value = true
  } finally {
    loading.value = false
  }
})
</script>

<template>
  <!-- 載入中骨架（靜態基線仍在上方顯示，骨架只佔位避免版面跳動） -->
  <div
    v-if="loading"
    aria-hidden="true"
    class="grid grid-cols-2 gap-x-4 gap-y-10 sm:grid-cols-3 lg:grid-cols-4 lg:gap-x-6"
  >
    <div v-for="i in 4" :key="i">
      <Skeleton class="aspect-[4/5] w-full rounded-xl" />
      <Skeleton class="mt-3 h-3 w-1/3" />
      <Skeleton class="mt-2 h-4 w-4/5" />
      <Skeleton class="mt-2 h-4 w-1/4" />
    </div>
  </div>

  <!-- API 失敗：不渲染，靜態基線繼續顯示 -->
  <template v-else-if="failed" />

  <div
    v-else-if="items.length"
    class="grid grid-cols-2 gap-x-4 gap-y-10 sm:grid-cols-3 lg:grid-cols-4 lg:gap-x-6"
  >
    <ProductCard v-for="(p, i) in items" :key="p.id" :product="p" :index="i" />
  </div>
  <p v-else class="py-10 text-center text-sm text-muted-foreground">商品準備中…</p>
</template>
