<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import {
  X, Plus, Minus, ShoppingBag, Trash2, Tag, ArrowRight,
} from 'lucide-vue-next'
import { useCartStore } from '@/shared/stores/cart'
import { useUiStore } from '@/shared/stores/ui'
import { useToast } from '@/shared/composables/use-toast'
// No monetary formatting — the cart drawer shows no monetary values.
// All pricing comes from the server-authoritative /api/quote response
// in CheckoutDialog. The cart drawer shows items, quantities, and
// variant selections only.

const cart = useCartStore()
const ui = useUiStore()
const { toast } = useToast()

const promoInput = ref('')
const promoMessage = ref<{ ok: boolean; text: string } | null>(null)

const items = computed(() => cart.items)
const itemCount = computed(() => cart.itemCount)

// No local monetary computation. Subtotal, discount, shipping, tax, and
// total come from the server-authoritative /api/quote response in
// CheckoutDialog. The cart drawer shows items and quantities only.
// The free-shipping progress bar was removed because no free-shipping
// threshold has been approved (GATE-004 scope only, no fee schedule).

watch(() => ui.cartOpen, (open) => {
  if (open) document.body.style.overflow = 'hidden'
  else document.body.style.overflow = ''
})

function close() {
  ui.closeCart()
}

function handleApplyPromo() {
  const result = cart.applyPromo(promoInput.value)
  promoMessage.value = { ok: result.ok, text: result.message }
  if (result.ok) {
    toast({ title: '優惠碼已記錄', description: result.message })
    promoInput.value = ''
  }
}

function handleRemovePromo() {
  cart.removePromo()
  promoMessage.value = null
}

function handleCheckout() {
  if (items.value.length === 0) return
  close()
  ui.openCheckout()
}
</script>

<template>
  <Teleport to="body">
    <Transition name="drawer-overlay">
      <div
        v-if="ui.cartOpen"
        class="fixed inset-0 z-50 bg-foreground/40 backdrop-blur-sm"
        @click="close"
      />
    </Transition>

    <Transition name="drawer-panel">
      <aside
        v-if="ui.cartOpen"
        class="fixed right-0 top-0 z-50 flex h-full w-full max-w-md flex-col bg-background shadow-2xl"
      >
        <!-- Header -->
        <div class="flex items-center justify-between border-b border-border/60 px-5 py-4">
          <div class="flex items-center gap-2">
            <ShoppingBag class="h-4 w-4" />
            <h2 class="text-sm font-semibold tracking-tight">購物車</h2>
            <span v-if="itemCount > 0" class="rounded-full bg-muted px-2 py-0.5 text-[11px] text-muted-foreground">
              {{ itemCount }} 件
            </span>
          </div>
          <button
            @click="close"
            class="grid h-8 w-8 place-items-center rounded-full text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
            aria-label="關閉"
          >
            <X class="h-4 w-4" />
          </button>
        </div>

        <!-- Items -->
        <div class="flex-1 overflow-y-auto px-5 py-4">
          <div v-if="items.length === 0" class="flex h-full flex-col items-center justify-center gap-3 text-center">
            <div class="grid h-14 w-14 place-items-center rounded-full bg-muted">
              <ShoppingBag class="h-6 w-6 text-muted-foreground" />
            </div>
            <div>
              <p class="text-sm font-medium">購物車是空的</p>
              <p class="mt-1 text-xs text-muted-foreground">快去看看我們的商品吧！</p>
            </div>
            <button
              @click="close"
              class="mt-2 inline-flex h-9 items-center rounded-full bg-cta px-5 text-sm font-medium text-cta-foreground transition-colors hover:brightness-95"
            >
              繼續購物
            </button>
          </div>

          <ul v-else class="space-y-3">
            <li
              v-for="item in items"
              :key="`${item.product.id}-${item.selectedSize ?? ''}-${item.selectedColor ?? ''}`"
              class="flex gap-3 rounded-xl border border-border/60 bg-card p-3"
            >
              <div class="h-20 w-16 shrink-0 overflow-hidden rounded-lg bg-muted">
                <img :src="item.product.image" :alt="item.product.name" class="h-full w-full object-cover" />
              </div>

              <div class="flex flex-1 flex-col">
                <div class="flex items-start justify-between gap-2">
                  <p class="text-sm font-medium leading-snug">{{ item.product.name }}</p>
                  <button
                    @click="cart.removeItem(item.product.id, { size: item.selectedSize, color: item.selectedColor })"
                    class="shrink-0 text-muted-foreground transition-colors hover:text-destructive"
                    aria-label="移除"
                  >
                    <Trash2 class="h-3.5 w-3.5" />
                  </button>
                </div>

                <p class="mt-0.5 text-[11px] text-muted-foreground">
                  <span v-if="item.selectedSize">尺寸 {{ item.selectedSize.toUpperCase() }}</span>
                  <span v-if="item.selectedSize && item.selectedColor"> · </span>
                  <span v-if="item.selectedColor">{{ item.selectedColor }}</span>
                </p>

                <div class="mt-auto flex items-center justify-between">
                  <div class="inline-flex items-center rounded-full border border-border">
                    <button
                      @click="cart.updateQuantity(item.product.id, item.quantity - 1, { size: item.selectedSize, color: item.selectedColor })"
                      class="grid h-7 w-7 place-items-center rounded-full text-muted-foreground transition-colors hover:text-foreground"
                      aria-label="減少"
                    >
                      <Minus class="h-3 w-3" />
                    </button>
                    <span class="w-7 text-center text-xs font-medium tabular-nums">{{ item.quantity }}</span>
                    <button
                      @click="cart.updateQuantity(item.product.id, item.quantity + 1, { size: item.selectedSize, color: item.selectedColor })"
                      :disabled="item.quantity >= item.product.stock"
                      class="grid h-7 w-7 place-items-center rounded-full text-muted-foreground transition-colors hover:text-foreground disabled:opacity-40"
                      aria-label="增加"
                    >
                      <Plus class="h-3 w-3" />
                    </button>
                  </div>
                </div>
              </div>
            </li>
          </ul>
        </div>

        <!-- Footer: promo + summary + checkout -->
        <div v-if="items.length > 0" class="border-t border-border/60 px-5 py-4">
          <!-- Promo -->
          <div class="mb-3">
            <div class="flex gap-2">
              <div class="relative flex-1">
                <Tag class="pointer-events-none absolute left-3 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground" />
                <input
                  v-model="promoInput"
                  type="text"
                  placeholder="輸入優惠碼"
                  class="h-9 w-full rounded-full border border-border bg-muted/30 pl-9 pr-3 text-xs uppercase outline-none focus:border-cta"
                  @keydown.enter="handleApplyPromo"
                />
              </div>
              <button
                @click="handleApplyPromo"
                class="h-9 shrink-0 rounded-full border border-border px-4 text-xs font-medium transition-colors hover:bg-muted"
              >
                套用
              </button>
            </div>
            <div v-if="cart.appliedPromo" class="mt-2 flex items-center justify-between rounded-lg bg-cta/10 px-3 py-1.5 text-xs">
              <span class="font-medium text-cta">{{ cart.appliedPromo }} 將於結帳時驗證</span>
              <button @click="handleRemovePromo" class="text-muted-foreground hover:text-foreground">移除</button>
            </div>
            <p v-else-if="promoMessage && !promoMessage.ok" class="mt-1.5 text-[11px] text-destructive">
              {{ promoMessage.text }}
            </p>
          </div>

          <!-- Checkout -->
          <button
            @click="handleCheckout"
            class="mt-4 flex h-11 w-full items-center justify-center gap-2 rounded-full bg-cta text-sm font-medium text-cta-foreground transition-colors hover:brightness-95"
          >
            結帳
            <ArrowRight class="h-4 w-4" />
          </button>
        </div>
      </aside>
    </Transition>
  </Teleport>
</template>

<style scoped>
.drawer-overlay-enter-active, .drawer-overlay-leave-active {
  transition: opacity 0.3s ease;
}
.drawer-overlay-enter-from, .drawer-overlay-leave-to {
  opacity: 0;
}
.drawer-panel-enter-active, .drawer-panel-leave-active {
  transition: transform 0.3s ease;
}
.drawer-panel-enter-from, .drawer-panel-leave-to {
  transform: translateX(100%);
}
</style>
