<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import {
  Star, Heart, Plus, Minus, ShoppingBag, ZoomIn, ChevronLeft, ChevronRight,
  Check, Truck, Store, Package, X,
} from 'lucide-vue-next'
import type { Product } from '@/shared/lib/types'
import { useCartStore } from '@/shared/stores/cart'
import { useUiStore } from '@/shared/stores/ui'
import { useToast } from '@/shared/composables/use-toast'
import Dialog from '@/shared/components/ui/Dialog.vue'
import Button from '@/shared/components/ui/Button.vue'
import Separator from '@/shared/components/ui/Separator.vue'
import ImageZoomModal from '@/shared/components/site/ImageZoomModal.vue'
import { formatNTD, cn } from '@/shared/lib/utils'

const ui = useUiStore()
const cart = useCartStore()
const { toast } = useToast()

const activeImage = ref(0)
const selectedSize = ref<string | undefined>(undefined)
const selectedColor = ref<string | undefined>(undefined)
const quantity = ref(1)
const zoomOpen = ref(false)
const showAllReviews = ref(false)

watch(() => ui.selectedProduct, (p) => {
  if (p) {
    activeImage.value = 0
    selectedSize.value = p.sizes?.[0]?.value
    selectedColor.value = p.colors?.[0]?.value
    quantity.value = 1
    showAllReviews.value = false
  }
}, { immediate: true })

const isFavorite = computed(() => (ui.selectedProduct ? cart.favorites.includes(ui.selectedProduct.id) : false))

const discount = computed(() => {
  if (!ui.selectedProduct?.originalPrice) return 0
  return Math.round((1 - ui.selectedProduct.price / ui.selectedProduct.originalPrice) * 100)
})

const displayedReviews = computed(() => {
  if (!ui.selectedProduct) return []
  return showAllReviews.value ? ui.selectedProduct.reviewList : ui.selectedProduct.reviewList.slice(0, 3)
})

function handleAddToCart() {
  if (!ui.selectedProduct) return
  cart.addItem(ui.selectedProduct, quantity.value, {
    size: selectedSize.value,
    color: selectedColor.value,
  })
  toast({
    title: '已加入購物車',
    description: `${ui.selectedProduct.name} · ${formatNTD(ui.selectedProduct.price * quantity.value)}`,
    image: ui.selectedProduct.image,
  })
}

function handleToggleFavorite() {
  if (!ui.selectedProduct) return
  cart.toggleFavorite(ui.selectedProduct.id)
  toast({
    title: isFavorite.value ? '已取消收藏' : '已加入收藏',
    description: ui.selectedProduct.name,
  })
}

function formatDate(dateStr: string) {
  return new Date(dateStr).toLocaleDateString('zh-TW', { year: 'numeric', month: 'short', day: 'numeric' })
}

function ratingPercent(stars: number) {
  if (!ui.selectedProduct || ui.selectedProduct.reviews === 0) return 0
  const entry = ui.selectedProduct.ratingBreakdown.find((b) => b.stars === stars)
  return entry ? Math.round((entry.count / ui.selectedProduct.reviews) * 100) : 0
}
</script>

<template>
  <Dialog
    :open="ui.productDetailOpen"
    class="max-w-4xl p-0"
    :show-close="false"
    :aria-label="ui.selectedProduct?.name ?? '商品詳情'"
    @update:open="ui.closeProductDetail()"
  >
    <div v-if="ui.selectedProduct" class="flex max-h-[85vh] flex-col overflow-hidden sm:flex-row">
      <!-- Left: Images -->
      <div class="shrink-0 overflow-y-auto border-b border-border/60 sm:w-2/5 sm:border-b-0 sm:border-r">
        <!-- Main image -->
        <div class="relative aspect-square overflow-hidden bg-muted">
          <img
            :src="ui.selectedProduct.images[activeImage]"
            :alt="ui.selectedProduct.name"
            class="h-full w-full cursor-zoom-in object-cover transition-transform duration-300 hover:scale-105"
            @click="zoomOpen = true"
          />
          <button
            @click="zoomOpen = true"
            class="absolute bottom-3 right-3 grid h-9 w-9 place-items-center rounded-full bg-background/90 text-foreground shadow-sm backdrop-blur transition-colors hover:bg-background"
            aria-label="放大檢視"
          >
            <ZoomIn class="h-4 w-4" />
          </button>

          <!-- Arrows -->
          <template v-if="ui.selectedProduct.images.length > 1">
            <button
              @click="activeImage = activeImage === 0 ? ui.selectedProduct.images.length - 1 : activeImage - 1"
              class="absolute left-2 top-1/2 grid h-8 w-8 -translate-y-1/2 place-items-center rounded-full bg-background/80 text-foreground backdrop-blur transition-colors hover:bg-background"
              aria-label="上一張"
            >
              <ChevronLeft class="h-4 w-4" />
            </button>
            <button
              @click="activeImage = activeImage === ui.selectedProduct.images.length - 1 ? 0 : activeImage + 1"
              class="absolute right-2 top-1/2 grid h-8 w-8 -translate-y-1/2 place-items-center rounded-full bg-background/80 text-foreground backdrop-blur transition-colors hover:bg-background"
              aria-label="下一張"
            >
              <ChevronRight class="h-4 w-4" />
            </button>
          </template>
        </div>

        <!-- Thumbnails -->
        <div v-if="ui.selectedProduct.images.length > 1" class="flex gap-2 p-3">
          <button
            v-for="(img, i) in ui.selectedProduct.images"
            :key="i"
            @click="activeImage = i"
            :class="cn(
              'relative h-16 w-16 shrink-0 overflow-hidden rounded-lg border-2 transition-all',
              activeImage === i ? 'border-cta' : 'border-transparent opacity-60 hover:opacity-100',
            )"
          >
            <img :src="img" :alt="`${ui.selectedProduct.name} ${i + 1}`" class="h-full w-full object-cover" />
          </button>
        </div>
      </div>

      <!-- Right: Details -->
      <div class="flex flex-1 flex-col overflow-y-auto">
        <!-- Header -->
        <div class="border-b border-border/60 px-6 py-5">
          <div class="flex items-start justify-between gap-3">
            <div class="min-w-0">
              <p class="text-[11px] font-medium uppercase tracking-wider text-muted-foreground">
                {{ ui.selectedProduct.category }}
                <span v-if="ui.selectedProduct.tag"> · {{ ui.selectedProduct.tag }}</span>
              </p>
              <h2 class="mt-1 text-lg font-semibold leading-tight tracking-tight">{{ ui.selectedProduct.name }}</h2>
            </div>
            <div class="flex shrink-0 items-center gap-2">
              <button
                @click="handleToggleFavorite"
                class="grid h-9 w-9 place-items-center rounded-full border border-border transition-colors hover:bg-muted"
                :aria-label="isFavorite ? '取消收藏' : '加入收藏'"
              >
                <Heart :class="cn('h-4 w-4 transition-all', isFavorite && 'fill-cta text-cta')" />
              </button>
              <button
                type="button"
                @click="ui.closeProductDetail()"
                class="grid h-9 w-9 place-items-center rounded-full text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
                aria-label="關閉"
              >
                <X class="h-4 w-4" />
              </button>
            </div>
          </div>

          <!-- Rating -->
          <div class="mt-3 flex items-center gap-3">
            <div class="flex items-center gap-1">
              <Star class="h-4 w-4 fill-cta text-cta" />
              <span class="text-sm font-semibold">{{ ui.selectedProduct.rating }}</span>
            </div>
            <span class="text-xs text-muted-foreground">{{ ui.selectedProduct.reviews }} 則評價</span>
            <span v-if="ui.selectedProduct.stock > 0 && ui.selectedProduct.stock <= 5" class="rounded-full bg-amber-100 px-2 py-0.5 text-[10px] font-medium text-amber-900 dark:bg-amber-900/30 dark:text-amber-200">
              剩 {{ ui.selectedProduct.stock }} 件
            </span>
          </div>

          <!-- Price -->
          <div class="mt-4 flex items-baseline gap-3">
            <span class="text-2xl font-bold tracking-tight text-cta">{{ formatNTD(ui.selectedProduct.price) }}</span>
            <span v-if="ui.selectedProduct.originalPrice" class="text-sm text-muted-foreground line-through">
              {{ formatNTD(ui.selectedProduct.originalPrice) }}
            </span>
            <span v-if="discount > 0" class="rounded-full bg-cta/10 px-2 py-0.5 text-xs font-medium text-cta">
              −{{ discount }}%
            </span>
          </div>
        </div>

        <!-- Body -->
        <div class="flex-1 px-6 py-5">
          <p class="text-sm leading-relaxed text-muted-foreground">
            {{ ui.selectedProduct.longDescription ?? ui.selectedProduct.description }}
          </p>

          <!-- Options -->
          <div class="mt-5 space-y-4">
            <!-- Sizes -->
            <div v-if="ui.selectedProduct.sizes && ui.selectedProduct.sizes.length > 0">
              <p class="text-xs font-medium uppercase tracking-wider text-muted-foreground">尺寸</p>
              <div class="mt-2 flex flex-wrap gap-2">
                <button
                  v-for="size in ui.selectedProduct.sizes"
                  :key="size.value"
                  @click="selectedSize = size.value"
                  :class="cn(
                    'inline-flex h-9 min-w-9 items-center justify-center rounded-full border px-3 text-sm font-medium transition-colors',
                    selectedSize === size.value ? 'border-cta bg-cta text-cta-foreground' : 'border-border hover:bg-muted',
                  )"
                >
                  {{ size.label }}
                </button>
              </div>
            </div>

            <!-- Colors -->
            <div v-if="ui.selectedProduct.colors && ui.selectedProduct.colors.length > 0">
              <p class="text-xs font-medium uppercase tracking-wider text-muted-foreground">顏色</p>
              <div class="mt-2 flex flex-wrap gap-2">
                <button
                  v-for="color in ui.selectedProduct.colors"
                  :key="color.value"
                  @click="selectedColor = color.value"
                  :class="cn(
                    'inline-flex h-9 items-center gap-2 rounded-full border px-3 text-sm font-medium transition-colors',
                    selectedColor === color.value ? 'border-cta bg-cta/5' : 'border-border hover:bg-muted',
                  )"
                >
                  <span
                    v-if="color.swatch"
                    class="h-4 w-4 rounded-full border border-border/40"
                    :style="{ backgroundColor: color.swatch }"
                  />
                  {{ color.label }}
                </button>
              </div>
            </div>

            <!-- Quantity -->
            <div>
              <p class="text-xs font-medium uppercase tracking-wider text-muted-foreground">數量</p>
              <div class="mt-2 inline-flex items-center rounded-full border border-border">
                <button
                  @click="quantity = Math.max(1, quantity - 1)"
                  class="grid h-9 w-9 place-items-center rounded-full text-muted-foreground transition-colors hover:text-foreground"
                  aria-label="減少"
                >
                  <Minus class="h-3.5 w-3.5" />
                </button>
                <span class="w-10 text-center text-sm font-medium tabular-nums">{{ quantity }}</span>
                <button
                  @click="quantity = Math.min(ui.selectedProduct.stock, quantity + 1)"
                  :disabled="quantity >= ui.selectedProduct.stock"
                  class="grid h-9 w-9 place-items-center rounded-full text-muted-foreground transition-colors hover:text-foreground disabled:opacity-40"
                  aria-label="增加"
                >
                  <Plus class="h-3.5 w-3.5" />
                </button>
              </div>
            </div>
          </div>

          <!-- Product info -->
          <div class="mt-5 grid grid-cols-2 gap-3 text-xs">
            <div v-if="ui.selectedProduct.material" class="rounded-lg bg-muted/30 px-3 py-2">
              <p class="text-muted-foreground">材質</p>
              <p class="mt-0.5 font-medium">{{ ui.selectedProduct.material }}</p>
            </div>
            <div v-if="ui.selectedProduct.origin" class="rounded-lg bg-muted/30 px-3 py-2">
              <p class="text-muted-foreground">產地</p>
              <p class="mt-0.5 font-medium">{{ ui.selectedProduct.origin }}</p>
            </div>
            <div class="rounded-lg bg-muted/30 px-3 py-2">
              <p class="text-muted-foreground">SKU</p>
              <p class="mt-0.5 font-mono font-medium">{{ ui.selectedProduct.sku }}</p>
            </div>
            <div class="rounded-lg bg-muted/30 px-3 py-2">
              <p class="text-muted-foreground">庫存</p>
              <p class="mt-0.5 font-medium">{{ ui.selectedProduct.stock > 0 ? `${ui.selectedProduct.stock} 件` : '已售完' }}</p>
            </div>
          </div>

          <!-- Service icons — only approved claims per GATE-007 -->
          <div class="mt-4 flex flex-wrap gap-4 text-[11px] text-muted-foreground">
            <span class="flex items-center gap-1"><Truck class="h-3.5 w-3.5 text-cta" /> 宅配到府</span>
            <span class="flex items-center gap-1"><Store class="h-3.5 w-3.5 text-cta" /> 超商取貨</span>
          </div>

          <Separator class="my-5" />

          <!-- Reviews -->
          <div>
            <div class="flex items-center justify-between">
              <h3 class="text-sm font-semibold">顧客評價</h3>
              <div class="flex items-center gap-1.5">
                <Star class="h-4 w-4 fill-cta text-cta" />
                <span class="text-sm font-semibold">{{ ui.selectedProduct.rating }}</span>
                <span class="text-xs text-muted-foreground">/ 5</span>
              </div>
            </div>

            <!-- Rating breakdown -->
            <div class="mt-3 space-y-1.5">
              <div v-for="b in ui.selectedProduct.ratingBreakdown" :key="b.stars" class="flex items-center gap-2 text-xs">
                <span class="w-6 text-muted-foreground">{{ b.stars }}★</span>
                <div class="h-1.5 flex-1 overflow-hidden rounded-full bg-muted">
                  <div class="h-full rounded-full bg-cta" :style="{ width: `${ratingPercent(b.stars)}%` }" />
                </div>
                <span class="w-8 text-right tabular-nums text-muted-foreground">{{ b.count }}</span>
              </div>
            </div>

            <Separator class="my-4" />

            <!-- Review list -->
            <ul class="space-y-4">
              <li v-for="review in displayedReviews" :key="review.id" class="space-y-1.5">
                <div class="flex items-center justify-between">
                  <div class="flex items-center gap-2">
                    <div class="grid h-7 w-7 place-items-center rounded-full bg-muted text-xs font-medium">
                      {{ review.author.charAt(0) }}
                    </div>
                    <div>
                      <p class="text-xs font-medium">{{ review.author }}</p>
                      <div class="flex items-center gap-1">
                        <div class="flex">
                          <Star v-for="s in 5" :key="s" :class="cn('h-2.5 w-2.5', s <= review.rating ? 'fill-cta text-cta' : 'text-muted-foreground/30')" />
                        </div>
                        <span v-if="review.verified" class="text-[10px] text-emerald-700 dark:text-emerald-400">已驗證購買</span>
                      </div>
                    </div>
                  </div>
                  <span class="text-[10px] text-muted-foreground">{{ formatDate(review.date) }}</span>
                </div>
                <p class="text-xs font-medium">{{ review.title }}</p>
                <p class="text-xs leading-relaxed text-muted-foreground">{{ review.body }}</p>
              </li>
            </ul>

            <button
              v-if="ui.selectedProduct.reviewList.length > 3"
              @click="showAllReviews = !showAllReviews"
              class="mt-4 text-xs font-medium text-cta underline-offset-2 hover:underline"
            >
              {{ showAllReviews ? '收起評價' : `查看全部 ${ui.selectedProduct.reviewList.length} 則評價` }}
            </button>
          </div>
        </div>

        <!-- Footer: Add to cart -->
        <div class="border-t border-border/60 px-6 py-4">
          <div class="flex items-center gap-3">
            <div class="flex-1">
              <p class="text-xs text-muted-foreground">合計</p>
              <p class="text-lg font-bold tabular-nums text-cta">{{ formatNTD(ui.selectedProduct.price * quantity) }}</p>
            </div>
            <Button
              @click="handleAddToCart"
              :disabled="ui.selectedProduct.stock === 0"
              class="h-11 rounded-full bg-cta px-6 text-cta-foreground hover:brightness-95 disabled:opacity-50"
            >
              <ShoppingBag class="mr-1.5 h-4 w-4" />
              {{ ui.selectedProduct.stock === 0 ? '已售完' : '加入購物車' }}
            </Button>
          </div>
        </div>
      </div>
    </div>

  </Dialog>

  <!-- Image zoom (outside Dialog to avoid Teleport/Escape conflicts) -->
  <ImageZoomModal
    v-if="ui.selectedProduct"
    :images="ui.selectedProduct.images"
    :alt="ui.selectedProduct.name"
    :open="zoomOpen"
    :initial-index="activeImage"
    @update:open="zoomOpen = $event"
  />
</template>
