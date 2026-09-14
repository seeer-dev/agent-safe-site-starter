<script setup lang="ts">
// 購物車 Drawer（右側滑出）— 對應 reference cart-drawer.tsx：
//   空態 → 商品列表（縮圖/規格/數量步進/移除）→ 免運進度條 → 小計 → CTA。
import { computed } from 'vue'
import { ShoppingBag, Trash2, Truck, ArrowRight } from 'lucide-vue-next'
import Sheet from '@/shared/components/Sheet.vue'
import FadeImage from '@/shared/components/FadeImage.vue'
import QuantityStepper from '@/shared/components/QuantityStepper.vue'
import { cart, setQty, removeItem, cartSubtotal, cartCount } from '@/shared/lib/cart'
import { cartDrawer, bootstrap } from '@/shared/lib/store'
import { formatNTD } from '@/shared/lib/format'
import { navigate } from '@/shared/lib/transition'
import { toast } from '@/shared/lib/toast'

const items = computed(() => cart.items)
const subtotal = computed(() => cartSubtotal())
const count = computed(() => cartCount())

// 免運門檻：settings.freeShippingThreshold（數字或字串皆容忍）
const threshold = computed(() => {
  const v = bootstrap.data?.settings?.freeShippingThreshold
  const n = typeof v === 'number' ? v : parseInt(String(v ?? '0'), 10)
  return Number.isFinite(n) ? n : 0
})
const remaining = computed(() => Math.max(0, threshold.value - subtotal.value))
const progress = computed(() =>
  threshold.value > 0 ? Math.min(100, (subtotal.value / threshold.value) * 100) : 100,
)
const freeReached = computed(() => threshold.value > 0 && subtotal.value >= threshold.value)

function go(path: string) {
  cartDrawer.open = false
  navigate(path)
}

function onRemove(key: string, name: string) {
  removeItem(key)
  toast.success('已移除商品', { description: name })
}
</script>

<template>
  <Sheet :open="cartDrawer.open" side="right" @update:open="cartDrawer.open = $event">
    <div class="flex h-full w-full flex-col bg-background sm:max-w-md">
      <header class="shrink-0 border-b px-5 py-4">
        <h2 class="flex items-center gap-2.5 font-serif text-lg tracking-wide">
          <ShoppingBag class="size-4 text-primary" />
          購物車
          <span v-if="count > 0" class="text-sm font-normal text-muted-foreground">（{{ count }} 件）</span>
        </h2>
      </header>

      <!-- 空態 -->
      <div v-if="items.length === 0" class="flex flex-1 flex-col items-center justify-center px-6 py-16 text-center">
        <div class="flex size-16 items-center justify-center rounded-full border bg-muted/40">
          <ShoppingBag class="size-7 text-muted-foreground/60" :stroke-width="1.5" />
        </div>
        <h3 class="mt-5 font-serif text-lg font-semibold">購物車空空如也</h3>
        <p class="mt-2 text-sm text-muted-foreground">還沒有選定的美好，去逛逛本季嚴選吧。</p>
        <button
          type="button"
          class="mt-6 inline-flex h-11 items-center rounded-full bg-primary px-6 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90"
          @click="go('/shop/')"
        >開始探索</button>
      </div>

      <template v-else>
        <!-- 商品列表 -->
        <div class="scrollbar-thin max-h-[52vh] flex-1 overflow-y-auto px-5 py-4">
          <TransitionGroup name="cart-item" tag="ul" class="space-y-5">
            <li v-for="item in items" :key="item.key" class="flex gap-4">
              <a
                :href="`/products/${item.productSlug}/`"
                class="h-24 w-20 shrink-0 cursor-pointer overflow-hidden rounded-lg bg-muted/40"
              >
                <FadeImage :src="item.image" :alt="item.name" class="h-full w-full object-cover" />
              </a>
              <div class="flex min-w-0 flex-1 flex-col">
                <div class="flex items-start justify-between gap-2">
                  <div class="min-w-0">
                    <a
                      :href="`/products/${item.productSlug}/`"
                      class="clamp-1 block cursor-pointer text-sm font-medium hover:text-primary"
                    >{{ item.name }}</a>
                    <p v-if="item.variantName" class="mt-0.5 text-[11px] text-muted-foreground">{{ item.variantName }}</p>
                  </div>
                  <button
                    type="button"
                    :aria-label="`移除 ${item.name}`"
                    class="shrink-0 rounded-full p-1.5 text-muted-foreground/70 transition-colors hover:bg-destructive/10 hover:text-destructive"
                    @click="onRemove(item.key, item.name)"
                  >
                    <Trash2 class="size-3.5" />
                  </button>
                </div>
                <div class="mt-auto flex items-center justify-between pt-2">
                  <QuantityStepper
                    :model-value="item.qty"
                    :min="1"
                    :max="Math.max(item.stock, 1)"
                    size="sm"
                    @update:model-value="setQty(item.key, $event)"
                  />
                  <div class="text-right">
                    <p class="text-[10px] text-muted-foreground">{{ formatNTD(item.unitPrice) }}／件</p>
                    <p class="text-sm font-medium tabular-nums">{{ formatNTD(item.unitPrice * item.qty) }}</p>
                  </div>
                </div>
              </div>
            </li>
          </TransitionGroup>
        </div>

        <!-- Footer：免運進度 + 小計 + CTA -->
        <footer class="shrink-0 space-y-4 border-t bg-muted/30 px-5 py-5">
          <div v-if="threshold > 0" class="space-y-2">
            <div class="flex items-center gap-2 text-xs">
              <Truck class="size-3.5" :class="freeReached ? 'text-primary' : 'text-muted-foreground'" />
              <span v-if="freeReached" class="font-medium text-primary">已達免運門檻，享有免運優惠</span>
              <span v-else class="text-muted-foreground">
                再消費 <span class="font-medium text-foreground">{{ formatNTD(remaining) }}</span>
                即享免運（滿 {{ formatNTD(threshold) }}）
              </span>
            </div>
            <div class="h-1.5 overflow-hidden rounded-full bg-muted">
              <div
                class="h-full rounded-full bg-primary transition-all duration-500"
                :style="{ width: `${progress}%` }"
              />
            </div>
          </div>

          <div class="h-px bg-border" />

          <div class="space-y-1.5 text-sm">
            <div class="flex justify-between text-muted-foreground">
              <span>小計</span>
              <span class="tabular-nums text-foreground">{{ formatNTD(subtotal) }}</span>
            </div>
            <div class="flex justify-between text-muted-foreground">
              <span>運費</span>
              <span>結帳時依配送方式計算</span>
            </div>
          </div>

          <button
            type="button"
            class="inline-flex h-12 w-full items-center justify-center gap-2 rounded-full bg-primary text-base font-medium text-primary-foreground transition-colors hover:bg-primary/90"
            @click="go('/checkout/')"
          >
            前往結帳
            <ArrowRight class="size-4" />
          </button>
          <button
            type="button"
            class="inline-flex h-10 w-full items-center justify-center rounded-full border text-sm font-medium transition-colors hover:bg-muted"
            @click="go('/cart/')"
          >查看購物車</button>
        </footer>
      </template>
    </div>
  </Sheet>
</template>

<style scoped>
.cart-item-enter-active {
  transition: opacity 0.25s ease, transform 0.25s ease;
}
.cart-item-leave-active {
  transition: opacity 0.15s ease;
}
.cart-item-enter-from {
  opacity: 0;
  transform: translateY(10px);
}
.cart-item-leave-to {
  opacity: 0;
}
@media (prefers-reduced-motion: reduce) {
  .cart-item-enter-active,
  .cart-item-leave-active {
    transition: none;
  }
}
</style>
