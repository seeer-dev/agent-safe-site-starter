<script setup lang="ts">
// 購物車頁 — 對應 reference cart-view.tsx：
//   左欄商品卡列表（圖/名/規格/單價/低庫存/stepper/小計/移除）
//   + 清空確認 dialog + 繼續購物；
//   右欄 sticky 訂單摘要（免運進度條/小計/優惠券列/預估總計/
//   優惠碼套用/前往結帳）。空態引導回商店。
import { computed, onMounted, ref } from 'vue'
import { ArrowRight, Loader2, ShoppingBag, Tag, Trash2, Truck, X } from 'lucide-vue-next'
import Dialog from '@/shared/components/Dialog.vue'
import FadeImage from '@/shared/components/FadeImage.vue'
import QuantityStepper from '@/shared/components/QuantityStepper.vue'
import { apiPost } from '@/shared/lib/api'
import { cart, cartCount, cartSubtotal, clearCart, removeItem, setQty } from '@/shared/lib/cart'
import { formatNTD } from '@/shared/lib/format'
import { bootstrap, checkoutStore, loadBootstrap } from '@/shared/lib/store'
import { navigate } from '@/shared/lib/transition'
import { toast } from '@/shared/lib/toast'
import { cn } from '@/shared/lib/utils'
import type { PromoDTO } from '@/shared/lib/types'

interface CouponValidation {
  coupon: PromoDTO
  discount: number
  free_shipping: boolean
}

const couponInput = ref(checkoutStore.coupon?.code ?? '')
const confirmingClear = ref(false)
const validating = ref(false)

const subtotal = computed(() => cartSubtotal())
const count = computed(() => cartCount())
const threshold = computed(() => {
  const v = bootstrap.data?.settings?.freeShippingThreshold
  const n = typeof v === 'number' ? v : parseInt(String(v ?? '0'), 10)
  return Number.isFinite(n) ? n : 0
})
const freeReached = computed(() => threshold.value > 0 && subtotal.value >= threshold.value)
const remaining = computed(() => Math.max(0, threshold.value - subtotal.value))
const progress = computed(() =>
  threshold.value > 0 ? Math.min(100, (subtotal.value / threshold.value) * 100) : 100,
)
const applied = computed(() => checkoutStore.coupon)
const discount = ref(0)
const estimatedTotal = computed(() => Math.max(0, subtotal.value - discount.value))

const couponDescription = computed(() => {
  const c = applied.value
  if (!c) return ''
  const base =
    c.type === 'percent' ? `${c.value}% 折扣` : c.type === 'fixed' ? `折抵 ${formatNTD(c.value)}` : '免運費優惠'
  return c.freeShipping ? `${base}（含免運）` : base
})

async function applyCoupon() {
  const code = couponInput.value.trim().toUpperCase()
  if (!code) {
    toast.error('請輸入優惠碼')
    return
  }
  validating.value = true
  try {
    const res = await apiPost<CouponValidation>('/api/coupons/validate', { code, subtotal: subtotal.value })
    checkoutStore.coupon = {
      code: res.coupon.code,
      type: res.coupon.type,
      value: res.coupon.value,
      label: res.coupon.label || res.coupon.code,
      freeShipping: res.free_shipping,
    }
    discount.value = res.discount
    couponInput.value = res.coupon.code
    toast.success('優惠券已套用', { description: `${res.coupon.code}・${couponDescription.value}` })
  } catch (e) {
    checkoutStore.coupon = null
    discount.value = 0
    toast.error('無法套用優惠券', { description: e instanceof Error ? e.message : '請確認優惠碼與使用條件' })
  } finally {
    validating.value = false
  }
}

function removeCoupon() {
  checkoutStore.coupon = null
  discount.value = 0
  couponInput.value = ''
  toast.success('已移除優惠券')
}

function confirmClear() {
  clearCart()
  removeCoupon()
  confirmingClear.value = false
  toast.success('購物車已清空')
}

onMounted(() => void loadBootstrap().catch(() => undefined))
</script>

<template>
  <!-- 空態 -->
  <div v-if="cart.items.length === 0" class="mx-auto flex min-h-[60vh] w-full max-w-7xl flex-col items-center justify-center px-4 text-center">
    <div class="flex size-20 items-center justify-center rounded-full border bg-muted/40">
      <ShoppingBag class="size-9 text-muted-foreground/50" :stroke-width="1.2" />
    </div>
    <h1 class="mt-6 font-serif text-2xl font-semibold">購物車空空如也</h1>
    <p class="mt-2 text-sm text-muted-foreground">還沒有選定的美好，去逛逛本季嚴選吧。</p>
    <button
      type="button"
      class="mt-8 inline-flex h-12 items-center gap-2 rounded-full bg-primary px-8 text-base font-medium text-primary-foreground transition-colors hover:bg-primary/90"
      @click="navigate('/shop/')"
    >
      去逛逛
      <ArrowRight class="size-4" />
    </button>
  </div>

  <div v-else class="mx-auto w-full max-w-7xl px-4 py-10 sm:px-6 sm:py-14 lg:px-8">
    <div data-reveal>
      <p class="eyebrow">Shopping Cart</p>
      <h1 class="mt-3 font-serif text-2xl font-semibold tracking-wide sm:text-3xl">購物車</h1>
      <p class="mt-2 max-w-lg text-sm leading-relaxed text-muted-foreground">
        已挑選 {{ count }} 件商品，結帳前再確認一下吧。
      </p>
    </div>

    <div class="mt-8 grid gap-10 lg:grid-cols-[1fr_380px]">
      <!-- 左：商品列表 -->
      <div class="space-y-4">
        <div
          v-for="(item, i) in cart.items"
          :key="item.key"
          v-reveal="{ delay: Math.min(i * 0.05, 0.25) }"
          class="flex gap-4 rounded-xl border bg-card p-4 sm:gap-5 sm:p-5"
        >
          <a :href="`/products/${item.productSlug}/`" class="h-28 w-24 shrink-0 overflow-hidden rounded-lg bg-muted/40 sm:w-28">
            <FadeImage :src="item.image" :alt="item.name" class="h-full w-full object-cover" />
          </a>
          <div class="flex min-w-0 flex-1 flex-col">
            <div class="flex items-start justify-between gap-3">
              <div class="min-w-0">
                <a :href="`/products/${item.productSlug}/`" class="clamp-1 block font-medium transition-colors hover:text-primary">
                  {{ item.name }}
                </a>
                <p class="mt-1 text-xs text-muted-foreground">{{ item.variantName ?? '單一規格' }}</p>
                <p class="mt-1 text-xs text-muted-foreground/70">
                  {{ formatNTD(item.unitPrice) }}／件
                  <span v-if="item.stock <= 5" class="ml-2 text-amber-700 dark:text-amber-400">庫存僅剩 {{ item.stock }} 件</span>
                </p>
              </div>
              <button
                type="button"
                :aria-label="`移除 ${item.name}`"
                class="shrink-0 rounded-full p-2 text-muted-foreground/70 transition-colors hover:bg-destructive/10 hover:text-destructive"
                @click="removeItem(item.key); toast.success('已移除商品', { description: item.name })"
              >
                <Trash2 class="size-4" />
              </button>
            </div>
            <div class="mt-auto flex flex-wrap items-center justify-between gap-3 pt-3">
              <QuantityStepper :model-value="item.qty" :max="Math.max(item.stock, 1)" @update:model-value="setQty(item.key, $event)" />
              <p class="text-base font-medium tabular-nums">{{ formatNTD(item.unitPrice * item.qty) }}</p>
            </div>
          </div>
        </div>

        <div class="flex items-center justify-between pt-2">
          <button
            type="button"
            class="inline-flex items-center gap-1.5 rounded-full px-3 py-1.5 text-sm text-muted-foreground transition-colors hover:text-destructive"
            @click="confirmingClear = true"
          >
            <Trash2 class="size-3.5" />
            清空購物車
          </button>
          <button
            type="button"
            class="inline-flex items-center rounded-full px-3 py-1.5 text-sm text-muted-foreground transition-colors hover:text-foreground"
            @click="navigate('/shop/')"
          >繼續購物</button>
        </div>

        <Dialog :open="confirmingClear" @update:open="confirmingClear = $event">
          <h2 class="font-serif text-lg font-semibold">清空購物車？</h2>
          <p class="mt-2 text-sm text-muted-foreground">這將移除購物車中的全部 {{ count }} 件商品，此動作無法復原。</p>
          <div class="mt-6 flex justify-end gap-3">
            <button type="button" class="h-10 rounded-full border px-5 text-sm" @click="confirmingClear = false">取消</button>
            <button type="button" class="h-10 rounded-full bg-destructive px-5 text-sm font-medium text-white hover:bg-destructive/90" @click="confirmClear">確認清空</button>
          </div>
        </Dialog>
      </div>

      <!-- 右：訂單摘要 -->
      <aside class="h-fit space-y-4 lg:sticky lg:top-24" data-reveal>
        <div class="rounded-xl border bg-card p-6">
          <h2 class="font-serif text-lg font-semibold tracking-wide">訂單摘要</h2>

          <div class="mt-5 space-y-2">
            <div class="flex items-center gap-2 text-xs">
              <Truck :class="cn('size-3.5', freeReached ? 'text-primary' : 'text-muted-foreground')" />
              <span v-if="freeReached" class="font-medium text-primary">已達免運門檻（滿 {{ formatNTD(threshold) }}）</span>
              <span v-else-if="threshold > 0" class="text-muted-foreground">
                再消費 <span class="font-medium text-foreground">{{ formatNTD(remaining) }}</span> 即享免運
              </span>
              <span v-else class="text-muted-foreground">運費於結帳時依配送方式計算</span>
            </div>
            <div v-if="threshold > 0" class="h-1.5 overflow-hidden rounded-full bg-muted">
              <div class="h-full rounded-full bg-primary transition-all duration-500" :style="{ width: `${progress}%` }" />
            </div>
          </div>

          <div class="my-5 h-px bg-border"></div>

          <div class="space-y-3 text-sm">
            <div class="flex justify-between">
              <span class="text-muted-foreground">小計</span>
              <span class="tabular-nums">{{ formatNTD(subtotal) }}</span>
            </div>
            <div class="flex justify-between">
              <span class="text-muted-foreground">運費</span>
              <span class="text-muted-foreground">結帳時依配送方式計算</span>
            </div>
            <div v-if="applied" class="flex justify-between">
              <span class="flex items-center gap-1.5 text-primary">
                <Tag class="size-3.5" />
                {{ applied.code }}・{{ couponDescription }}
              </span>
              <span class="tabular-nums text-primary">{{ applied.freeShipping ? '免運' : `-${formatNTD(discount)}` }}</span>
            </div>
          </div>

          <div class="my-5 h-px bg-border"></div>

          <div class="flex items-baseline justify-between">
            <span class="text-sm text-muted-foreground">預估總計</span>
            <span class="font-serif text-2xl font-semibold tabular-nums">{{ formatNTD(estimatedTotal) }}</span>
          </div>

          <!-- 優惠券 -->
          <div class="mt-6">
            <div v-if="applied" class="flex items-center justify-between rounded-lg border border-primary/25 bg-primary/[0.05] px-4 py-3">
              <div class="text-sm">
                <p class="font-medium text-primary">{{ applied.code }}</p>
                <p class="text-xs text-muted-foreground">{{ couponDescription }}</p>
              </div>
              <button type="button" aria-label="移除優惠券" class="rounded-full p-1.5 text-muted-foreground hover:bg-destructive/10 hover:text-destructive" @click="removeCoupon">
                <X class="size-4" />
              </button>
            </div>
            <div v-else class="flex gap-2">
              <input
                v-model="couponInput"
                type="text"
                placeholder="輸入優惠碼"
                aria-label="優惠碼"
                maxlength="20"
                class="h-10 flex-1 rounded-full border bg-background px-4 font-mono text-sm uppercase tracking-widest outline-none focus:border-primary"
                @keydown.enter="applyCoupon"
              />
              <button
                type="button"
                :disabled="validating || !couponInput.trim()"
                class="inline-flex h-10 items-center rounded-full border px-5 text-sm transition-colors hover:bg-muted disabled:opacity-50"
                @click="applyCoupon"
              >
                <Loader2 v-if="validating" class="size-4 animate-spin" />
                <template v-else>套用</template>
              </button>
            </div>
          </div>

          <button
            type="button"
            class="mt-6 inline-flex h-12 w-full items-center justify-center gap-2 rounded-full bg-primary text-base font-medium text-primary-foreground transition-colors hover:bg-primary/90"
            @click="navigate('/checkout/')"
          >
            前往結帳
            <ArrowRight class="size-4" />
          </button>
        </div>
      </aside>
    </div>
  </div>
</template>
