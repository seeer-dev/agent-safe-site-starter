<script setup lang="ts">
// 商品卡 — 對應 reference product-card.tsx：
//   精選金色徽章、售完章（-12° 傾斜）、hover 底部浮出 quick-add、
//   圖片 hover 微放大、分類小標 + clamp-1 名稱 + PriceDisplay。
// quick-add：無 variants 直接加入購物車；有 variants 跳商品頁選規格。
import { computed } from 'vue'
import { Plus } from 'lucide-vue-next'
import FadeImage from './FadeImage.vue'
import PriceDisplay from './PriceDisplay.vue'
import { addItem } from '@/shared/lib/cart'
import { bootstrap } from '@/shared/lib/store'
import { toast } from '@/shared/lib/toast'
import { navigate } from '@/shared/lib/transition'
import { cn } from '@/shared/lib/utils'
import type { ProductDTO } from '@/shared/lib/types'

const props = defineProps<{ product: ProductDTO; index?: number }>()

const soldOut = computed(() => props.product.stock <= 0)
const canQuickAdd = computed(() => (props.product.variants?.length ?? 0) === 0)
const image = computed(() => props.product.images?.[0] ?? props.product.image ?? '')
const categoryName = computed(
  () =>
    bootstrap.data?.categories.find((c) => c.slug === props.product.category)?.name ?? '嚴選商品',
)

function openProduct() {
  navigate(`/products/${props.product.slug}/`)
}

function handleQuickAdd(e: MouseEvent) {
  e.stopPropagation()
  if (soldOut.value) return
  if (!canQuickAdd.value) {
    openProduct()
    return
  }
  const p = props.product
  addItem({
    productId: p.id,
    productSlug: p.slug,
    sku: p.sku,
    variantId: null,
    name: p.name,
    variantName: null,
    image: image.value,
    unitPrice: p.price,
    qty: 1,
    stock: p.stock,
  })
  toast.success('已加入購物車', { description: p.name })
}
</script>

<template>
  <article
    v-reveal="{ delay: Math.min((index ?? 0) * 0.06, 0.36) }"
    :class="cn('group lift cursor-pointer rounded-xl')"
    role="link"
    tabindex="0"
    :aria-label="`查看商品 ${product.name}`"
    @click="openProduct"
    @keydown.enter="openProduct"
  >
    <!-- 圖面 -->
    <div class="relative overflow-hidden rounded-xl bg-muted/40">
      <FadeImage
        :src="image"
        :alt="product.name"
        class="aspect-[4/5] w-full object-cover transition-transform duration-500 ease-out group-hover:scale-[1.04]"
      />

      <!-- 精選徽章：奶油底+金邊，在圖片上保持可讀 -->
      <span
        v-if="product.is_featured && !soldOut"
        class="absolute left-3 top-3 rounded-full border border-gold/60 bg-cream/90 px-2.5 py-1 text-[10px] font-medium tracking-[0.22em] text-ink/80 backdrop-blur-sm"
      >精選</span>

      <!-- 售完章 -->
      <div
        v-if="soldOut"
        class="absolute inset-0 flex items-center justify-center bg-background/55 backdrop-blur-[1px]"
      >
        <span class="rotate-[-12deg] rounded border-2 border-foreground/50 bg-background/80 px-4 py-1.5 font-serif text-sm tracking-[0.32em] text-foreground/70">
          已售完
        </span>
      </div>

      <!-- quick-add：hover 從底部上滑浮現 -->
      <button
        type="button"
        :class="cn(
          'absolute inset-x-0 bottom-0 flex translate-y-full items-center justify-center gap-1.5 bg-ink/90 py-3 text-xs font-medium tracking-[0.25em] text-cream backdrop-blur-sm transition-all duration-300 ease-out hover:bg-ink',
          'group-hover:translate-y-0 group-focus-within:translate-y-0',
          soldOut && 'pointer-events-none opacity-0',
        )"
        :aria-label="canQuickAdd ? `將 ${product.name} 加入購物車` : `選擇 ${product.name} 規格`"
        @click="handleQuickAdd"
      >
        <Plus class="size-3.5" />
        {{ canQuickAdd ? '加入購物車' : '選擇規格' }}
      </button>
    </div>

    <!-- 資訊 -->
    <div class="mt-3 space-y-1.5 px-0.5">
      <p class="text-[11px] tracking-wider text-muted-foreground">{{ categoryName }}</p>
      <h3 class="clamp-1 text-sm font-medium leading-snug transition-colors group-hover:text-primary">
        {{ product.name }}
      </h3>
      <PriceDisplay :price="product.price" :original-price="product.original_price" size="sm" />
    </div>
  </article>
</template>
