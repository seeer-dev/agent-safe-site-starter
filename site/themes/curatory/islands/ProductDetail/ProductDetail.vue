<script setup lang="ts">
// 商品詳情島 — 對應 reference product-view.tsx：
//   圖庫（主圖 + 縮圖）、規格選擇（價差/售完禁用）、數量、
//   加入購物車（toast + drawer）/ 直接購買、低庫存提示、
//   配送/付款/退換 accordion、已售數。
// 掛上前模板已輸出靜態基線；此島以 slug 重新拉取權威資料後接管。
import { computed, onMounted, ref, watch } from 'vue'
import {
  Check, ChevronDown, ChevronRight, Flame, PackageSearch,
  RotateCcw, ShieldCheck, ShoppingBag, Truck,
} from 'lucide-vue-next'
import FadeImage from '@/shared/components/FadeImage.vue'
import PriceDisplay from '@/shared/components/PriceDisplay.vue'
import QuantityStepper from '@/shared/components/QuantityStepper.vue'
import Skeleton from '@/shared/components/Skeleton.vue'
import { apiGet } from '@/shared/lib/api'
import { addItem } from '@/shared/lib/cart'
import { bootstrap, loadBootstrap, openCartDrawer } from '@/shared/lib/store'
import { formatNTD } from '@/shared/lib/format'
import { navigate } from '@/shared/lib/transition'
import { toast } from '@/shared/lib/toast'
import { cn } from '@/shared/lib/utils'
import type { ProductDTO, VariantDTO } from '@/shared/lib/types'

const props = defineProps<{ slug: string }>()

const product = ref<ProductDTO | null>(null)
const loading = ref(true)
const error = ref(false)

const selectedImage = ref<string | null>(null)
const pickedVariantId = ref<string | null>(null)
const qty = ref(1)
const justAdded = ref(false)
const openSections = ref<Record<string, boolean>>({ shipping: false, payment: false, returns: false })
let addedTimer: number | null = null

// 規格：預設第一個有庫存者（對應 reference render 期間推導）
const variant = computed<VariantDTO | null>(() => {
  const p = product.value
  if (!p || (p.variants?.length ?? 0) === 0) return null
  if (pickedVariantId.value) {
    const found = p.variants.find((v) => v.id === pickedVariantId.value)
    if (found) return found
  }
  return p.variants.find((v) => v.stock > 0) ?? null
})

const soldOut = computed(() => !product.value || product.value.stock <= 0)
const variantSoldOut = computed(() => Boolean(variant.value && variant.value.stock <= 0))
const cannotBuy = computed(() => soldOut.value || variantSoldOut.value)
const stock = computed(() => (variant.value ? variant.value.stock : (product.value?.stock ?? 0)))
const unitPrice = computed(() => (product.value?.price ?? 0) + (variant.value?.price_delta ?? 0))

const gallery = computed(() => {
  const p = product.value
  if (!p) return [] as string[]
  const imgs = p.images?.length ? [...p.images] : p.image ? [p.image] : []
  return imgs
})
const currentImage = computed(() => selectedImage.value ?? gallery.value[0] ?? '')

const categoryName = computed(() => {
  const p = product.value
  if (!p) return '嚴選商品'
  return bootstrap.data?.categories.find((c) => c.slug === p.category)?.name ?? '嚴選商品'
})
const shippingMethods = computed(() => bootstrap.data?.shipping_methods ?? [])
const paymentMethods = computed(() => bootstrap.data?.payment_methods ?? [])
const freeThreshold = computed(() => {
  const v = bootstrap.data?.settings?.freeShippingThreshold
  const n = typeof v === 'number' ? v : parseInt(String(v ?? '0'), 10)
  return Number.isFinite(n) ? n : 0
})

async function load() {
  loading.value = true
  error.value = false
  try {
    const [p] = await Promise.all([
      apiGet<{ product: ProductDTO } | ProductDTO>(`/api/products/${props.slug}`),
      loadBootstrap().catch(() => undefined),
    ])
    product.value = ('product' in p ? p.product : p) as ProductDTO
  } catch {
    error.value = true
    product.value = null
  } finally {
    loading.value = false
  }
}

function pickVariant(v: VariantDTO) {
  if (v.stock <= 0) return
  pickedVariantId.value = v.id
  qty.value = 1
}

function handleAdd(goCheckout = false) {
  const p = product.value
  if (!p) return
  if (cannotBuy.value) {
    toast.error(soldOut.value ? '此商品已售完' : '此規格已售完')
    return
  }
  if ((p.variants?.length ?? 0) > 0 && !variant.value) {
    toast.error('請先選擇商品規格')
    return
  }
  addItem({
    productId: p.id,
    productSlug: p.slug,
    sku: variant.value?.sku ?? p.sku,
    variantId: variant.value?.id ?? null,
    name: p.name,
    variantName: variant.value?.name ?? null,
    image: gallery.value[0] ?? '',
    unitPrice: unitPrice.value,
    qty: qty.value,
    stock: stock.value,
  })
  if (goCheckout) {
    navigate('/checkout/')
    return
  }
  toast.success('已加入購物車', {
    description: `${p.name}${variant.value ? `・${variant.value.name}` : ''}`,
  })
  justAdded.value = true
  if (addedTimer) window.clearTimeout(addedTimer)
  addedTimer = window.setTimeout(() => (justAdded.value = false), 1600)
  openCartDrawer()
}

function toggleSection(key: string) {
  openSections.value[key] = !openSections.value[key]
}

watch(() => props.slug, () => {
  selectedImage.value = null
  pickedVariantId.value = null
  qty.value = 1
  void load()
})

onMounted(load)
</script>

<template>
  <div>
    <!-- 載入骨架 -->
    <div v-if="loading" class="mt-6 grid gap-10 pb-16 lg:grid-cols-2 lg:gap-16">
      <div>
        <Skeleton class="aspect-[4/5] w-full rounded-2xl" />
        <div class="mt-4 flex gap-3">
          <Skeleton v-for="i in 3" :key="i" class="aspect-[4/5] w-16 rounded-lg sm:w-20" />
        </div>
      </div>
      <div class="space-y-4">
        <Skeleton class="h-3 w-24" />
        <Skeleton class="h-9 w-2/3" />
        <Skeleton class="h-7 w-32" />
        <Skeleton class="h-4 w-full" />
        <Skeleton class="h-4 w-5/6" />
        <Skeleton class="mt-6 h-12 w-full rounded-full" />
      </div>
    </div>

    <!-- 錯誤 -->
    <div v-else-if="error || !product" class="mx-auto flex min-h-[60vh] w-full max-w-7xl flex-col items-center justify-center px-4 text-center">
      <PackageSearch class="size-12 text-muted-foreground/40" :stroke-width="1.5" />
      <h1 class="mt-5 font-serif text-2xl font-semibold">找不到這件商品</h1>
      <p class="mt-2 text-sm text-muted-foreground">商品可能已下架或連結有誤，歡迎回到商店繼續探索。</p>
      <div class="mt-7 flex gap-3">
        <button type="button" class="inline-flex h-10 items-center gap-2 rounded-full border px-5 text-sm hover:bg-muted" @click="load">
          <RotateCcw class="size-4" />重新載入
        </button>
        <button type="button" class="inline-flex h-10 items-center rounded-full bg-primary px-5 text-sm font-medium text-primary-foreground" @click="navigate('/shop/')">
          回到商店
        </button>
      </div>
    </div>

    <!-- 商品主體 -->
    <div v-else class="mt-6 grid gap-10 pb-16 lg:grid-cols-2 lg:gap-16">
      <!-- 左：圖片 -->
      <div>
        <div class="overflow-hidden rounded-2xl bg-muted/40">
          <FadeImage :key="currentImage" :src="currentImage" :alt="product.name" eager class="aspect-[4/5] w-full object-cover" />
        </div>
        <div v-if="gallery.length > 1" class="mt-4 flex gap-3">
          <button
            v-for="img in gallery"
            :key="img"
            type="button"
            :aria-label="`查看圖片`"
            :class="cn(
              'overflow-hidden rounded-lg bg-muted/40 transition-all',
              currentImage === img ? 'ring-2 ring-primary ring-offset-2 ring-offset-background' : 'opacity-70 hover:opacity-100',
            )"
            @click="selectedImage = img"
          >
            <FadeImage :src="img" :alt="product.name" class="aspect-[4/5] w-16 object-cover sm:w-20" />
          </button>
        </div>
      </div>

      <!-- 右：資訊 -->
      <div class="flex flex-col">
        <p class="eyebrow">{{ categoryName }}</p>
        <h1 class="mt-3 font-serif text-3xl font-semibold leading-snug sm:text-4xl">{{ product.name }}</h1>

        <div class="mt-5 flex flex-wrap items-center gap-3">
          <PriceDisplay :price="unitPrice" :original-price="variant ? null : product.original_price" size="xl" />
          <span v-if="product.sold_count > 0" class="text-xs text-muted-foreground">已售出 {{ product.sold_count }} 件</span>
        </div>

        <div v-if="product.tag" class="mt-4 flex flex-wrap gap-2">
          <span
            v-for="t in product.tag.split(',').map((s) => s.trim()).filter(Boolean)"
            :key="t"
            class="rounded-full border border-primary/25 px-3 py-1 text-[11px] font-normal text-primary"
          >{{ t }}</span>
        </div>

        <p class="mt-6 leading-relaxed text-muted-foreground">{{ product.description }}</p>

        <div class="my-7 h-px bg-border"></div>

        <!-- 規格 -->
        <div v-if="(product.variants?.length ?? 0) > 0">
          <div class="flex items-baseline gap-2">
            <p class="text-sm font-medium">規格</p>
            <p class="text-xs text-muted-foreground">{{ variant ? `已選：${variant.name}` : '請選擇一個規格' }}</p>
          </div>
          <div class="mt-3 flex flex-wrap gap-2.5" role="radiogroup" aria-label="商品規格">
            <button
              v-for="v in product.variants"
              :key="v.id"
              type="button"
              role="radio"
              :aria-checked="variant?.id === v.id"
              :disabled="v.stock <= 0"
              :class="cn(
                'rounded-full border px-4 py-2 text-sm transition-all',
                variant?.id === v.id ? 'border-ink bg-ink text-cream' : 'border-border bg-card hover:border-foreground/40',
                v.stock <= 0 && 'cursor-not-allowed text-muted-foreground/50 line-through opacity-60 hover:border-border',
              )"
              @click="pickVariant(v)"
            >
              {{ v.name }}
              <span
                v-if="v.price_delta !== 0"
                :class="cn('ml-1.5 text-xs', variant?.id === v.id ? 'text-cream/70' : 'text-primary', v.stock <= 0 && 'text-muted-foreground/50')"
              >{{ v.price_delta > 0 ? `+${formatNTD(v.price_delta, false)}` : `-${formatNTD(-v.price_delta, false)}` }}</span>
            </button>
          </div>
        </div>

        <!-- 數量 -->
        <div class="mt-7 flex flex-wrap items-center gap-4">
          <QuantityStepper v-model="qty" :max="Math.max(stock, 1)" :disabled="cannotBuy" />
          <p v-if="cannotBuy" class="text-sm text-muted-foreground">目前無法購買</p>
          <p v-else-if="stock <= 5" class="flex items-center gap-1.5 text-sm text-amber-700 dark:text-amber-400">
            <Flame class="size-3.5" />
            {{ variant ? '此規格' : '商品' }}僅剩 {{ stock }} 件
          </p>
          <p v-else class="text-sm text-muted-foreground">庫存充足</p>
        </div>

        <!-- CTA -->
        <div class="mt-7 flex flex-col gap-3 sm:flex-row">
          <button
            type="button"
            :disabled="cannotBuy"
            class="inline-flex h-13 flex-1 items-center justify-center gap-2 rounded-full bg-primary px-8 text-base font-medium tracking-wide text-primary-foreground transition-colors hover:bg-primary/90 disabled:opacity-50"
            @click="handleAdd(false)"
          >
            <template v-if="justAdded"><Check class="size-4.5" />已加入購物車</template>
            <template v-else-if="cannotBuy">已售完，無法購買</template>
            <template v-else><ShoppingBag class="size-4.5" />加入購物車</template>
          </button>
          <button
            type="button"
            :disabled="cannotBuy"
            class="inline-flex h-13 items-center justify-center rounded-full border px-8 text-base tracking-wide transition-colors hover:bg-muted disabled:opacity-50"
            @click="handleAdd(true)"
          >直接購買</button>
        </div>

        <p class="mt-4 flex items-center gap-2 text-xs text-muted-foreground">
          <ShieldCheck class="size-3.5 text-primary" />
          7 天鑑賞期<template v-if="freeThreshold > 0">・{{ formatNTD(freeThreshold) }} 起享指定物流免運</template>
        </p>

        <!-- 配送與付款 Accordion -->
        <div class="mt-9">
          <div class="border-t">
            <button
              type="button"
              class="flex w-full items-center justify-between py-4 text-sm font-medium"
              :aria-expanded="openSections.shipping"
              @click="toggleSection('shipping')"
            >
              <span class="flex items-center gap-2"><Truck class="size-4 text-primary" />配送方式與運費</span>
              <ChevronDown class="size-4 transition-transform" :class="openSections.shipping && 'rotate-180'" />
            </button>
            <div v-show="openSections.shipping" class="pb-4 text-sm">
              <ul class="space-y-2.5">
                <li v-for="m in shippingMethods" :key="m.id" class="flex items-baseline justify-between gap-4">
                  <span>
                    {{ m.label }}
                    <span v-if="m.description" class="ml-2 text-xs text-muted-foreground">{{ m.description }}</span>
                  </span>
                </li>
              </ul>
              <p v-if="freeThreshold > 0" class="mt-3 text-xs text-muted-foreground">
                全館滿 {{ formatNTD(freeThreshold) }} 享免運；實際運費於結帳時依訂單金額計算。
              </p>
              <p v-else class="mt-3 text-xs text-muted-foreground">實際運費於結帳時依訂單金額計算。</p>
            </div>
          </div>
          <div class="border-t">
            <button
              type="button"
              class="flex w-full items-center justify-between py-4 text-sm font-medium"
              :aria-expanded="openSections.payment"
              @click="toggleSection('payment')"
            >
              <span>付款方式</span>
              <ChevronDown class="size-4 transition-transform" :class="openSections.payment && 'rotate-180'" />
            </button>
            <div v-show="openSections.payment" class="pb-4 text-sm">
              <ul class="space-y-2.5">
                <li v-for="m in paymentMethods" :key="m.id" class="flex items-baseline justify-between gap-4">
                  <span>{{ m.label }}</span>
                  <span class="shrink-0 tabular-nums text-muted-foreground">{{ m.fee > 0 ? `手續費 ${formatNTD(m.fee)}` : '免手續費' }}</span>
                </li>
              </ul>
            </div>
          </div>
          <div class="border-y">
            <button
              type="button"
              class="flex w-full items-center justify-between py-4 text-sm font-medium"
              :aria-expanded="openSections.returns"
              @click="toggleSection('returns')"
            >
              <span>退換貨與鑑賞期</span>
              <ChevronDown class="size-4 transition-transform" :class="openSections.returns && 'rotate-180'" />
            </button>
            <div v-show="openSections.returns" class="pb-4 text-sm leading-relaxed text-muted-foreground">
              依消費者保護法提供 7 天鑑賞期（非試用期）。商品須保持全新狀態與完整包裝，貼身物品、食品及客製化商品除外。退貨運費由買方負擔，若有瑕疵我們無償換貨或退款。
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
