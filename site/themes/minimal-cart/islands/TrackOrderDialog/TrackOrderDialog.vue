<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import {
  PackageSearch, Search, Check, Clock, Package, Truck, Home,
  XCircle, MapPin, Copy, X, KeyRound,
} from 'lucide-vue-next'
import { useUserStore, STATUS_META } from '@/shared/stores/user'
import type { Order, OrderStatus, OrderTimelineEntry } from '@/shared/lib/types'
import { useUiStore } from '@/shared/stores/ui'
import Dialog from '@/shared/components/ui/Dialog.vue'
import Button from '@/shared/components/ui/Button.vue'
import Input from '@/shared/components/ui/Input.vue'
import Separator from '@/shared/components/ui/Separator.vue'
import { useToast } from '@/shared/composables/use-toast'
import { formatNTD } from '@/shared/lib/utils'
import { getGuestOrder, getMyOrder, ApiRequestError } from '@/shared/lib/api'
import { parseMemberOrder, type MemberOrder } from '@/shared/lib/member-orders'
import { signOut } from '@/shared/lib/auth/session'

const ui = useUiStore()
const userStore = useUserStore()
const { toast } = useToast()

const input = ref('')
const tokenInput = ref('')
const searchedId = ref<string | null>(null)
const error = ref<string | null>(null)
const loading = ref(false)
// The order returned from the API (guest order with PII masked).
const apiOrder = ref<any | null>(null)
const memberOrder = ref<MemberOrder | null>(null)

const memberBearerToken = computed(() => userStore.bearerToken.trim())
const isMemberSession = computed(() => memberBearerToken.value !== '')

const STATUS_ICONS: Record<OrderStatus, any> = {
  pending: Clock,
  processing: Package,
  shipped: Truck,
  delivered: Home,
  cancelled: XCircle,
}

const STATUS_BADGE_STYLES: Record<OrderStatus, string> = {
  pending: 'bg-amber-100 text-amber-900 dark:bg-amber-900/30 dark:text-amber-200',
  processing: 'bg-blue-100 text-blue-900 dark:bg-blue-900/30 dark:text-blue-200',
  shipped: 'bg-violet-100 text-violet-900 dark:bg-violet-900/30 dark:text-violet-200',
  delivered: 'bg-emerald-100 text-emerald-900 dark:bg-emerald-900/30 dark:text-emerald-200',
  cancelled: 'bg-red-100 text-red-900 dark:bg-red-900/30 dark:text-red-200',
}

watch(() => ui.trackOrderOpen, (open) => {
  if (open) {
    // Reset state when the dialog opens.
    input.value = ui.trackOrderId ?? ''
    tokenInput.value = ''
    searchedId.value = null
    error.value = null
    apiOrder.value = null
    memberOrder.value = null
    if (ui.trackOrderId && memberBearerToken.value) {
      input.value = ui.trackOrderId
      void loadMemberOrder(ui.trackOrderId)
    } else if (ui.trackOrderId) {
      // Guest auto-search only when a locally placed order still holds
      // the one-time X-Order-Access-Token.
      const localOrder = userStore.getOrder(ui.trackOrderId)
      if (localOrder?.accessToken) {
        tokenInput.value = localOrder.accessToken
        handleSearch()
      }
    }
  }
})

async function loadMemberOrder(orderId: string) {
  const token = memberBearerToken.value
  if (!token) return
  const id = orderId.trim().toUpperCase()
  if (!id) {
    error.value = '請輸入訂單編號'
    return
  }
  loading.value = true
  error.value = null
  memberOrder.value = null
  apiOrder.value = null
  searchedId.value = null
  try {
    const result = await getMyOrder(id, token)
    if (!result) {
      error.value = `找不到編號為「${id}」的訂單。`
      return
    }
    memberOrder.value = parseMemberOrder(result)
    searchedId.value = id
  } catch (e: unknown) {
    if (e instanceof ApiRequestError && e.status === 401) {
      userStore.logout()
      await signOut()
      error.value = '登入已過期，請重新登入。'
    } else {
      error.value = '無法載入訂單，請稍後再試。'
    }
  } finally {
    loading.value = false
  }
}

async function handleSearch() {
  if (memberBearerToken.value) {
    await loadMemberOrder(input.value)
    return
  }
  const id = input.value.trim().toUpperCase()
  if (!id) {
    error.value = '請輸入訂單編號'
    return
  }
  const token = tokenInput.value.trim()
  if (!token) {
    error.value = '請輸入訂單存取碼'
    return
  }
  loading.value = true
  error.value = null
  try {
    const result = await getGuestOrder(id, token)
    if (!result) {
      error.value = `找不到編號為「${id}」的訂單，請確認訂單編號與存取碼是否正確。`
      apiOrder.value = null
      searchedId.value = null
      return
    }
    apiOrder.value = result
    searchedId.value = id
  } catch (e: any) {
    error.value = (e as Error).message || '查詢失敗，請稍後再試'
    apiOrder.value = null
    searchedId.value = null
  } finally {
    loading.value = false
  }
}

// Build a display Order from the API response (which has snake_case fields
// and masked PII). The API returns items as an array of {sku, name, price,
// quantity}. The shipping_address is a masked string.
const foundOrder = computed<Order | null>(() => {
  if (!apiOrder.value || !searchedId.value) return null
  const o = apiOrder.value
  const status = (o.status as OrderStatus) || 'pending'
  const placedAt = o.updated_unix ? o.updated_unix * 1000 : Date.now()
  return {
    id: o.id,
    userEmail: o.email ?? null,
    status,
    placedAt,
    shippingAddress: {
      name: o.customer_name || '',
      address: o.shipping_address || '',
      city: '',
      zip: '',
      country: '台灣',
    },
    items: (o.items || []).map((i: any) => ({
      productId: i.sku,
      name: i.name,
      image: '',
      price: i.price,
      quantity: i.quantity,
    })),
    subtotal: o.subtotal ?? 0,
    discount: o.discount ?? 0,
    shipping: o.shipping ?? 0,
    tax: 0,
    total: o.total ?? 0,
    timeline: buildTimeline(status, placedAt),
    shippingMethod: o.shipping_method || '',
    paymentMethod: o.payment_method || '',
  }
})

function buildTimeline(status: OrderStatus, placedAt: number): OrderTimelineEntry[] {
  const steps: OrderStatus[] = ['pending', 'processing', 'shipped', 'delivered']
  const currentStep = STATUS_META[status].step
  return steps.map((s, i) => ({
    status: s,
    label: STATUS_META[s].label,
    description: STATUS_META[s].description,
    timestamp: placedAt + i * 86400000,
    ...(currentStep >= 0 && i <= currentStep ? {} : { future: true } as any),
  })) as OrderTimelineEntry[]
}

function handleCopyId() {
  if (!searchedId.value) return
  navigator.clipboard?.writeText(searchedId.value)
  toast({ title: '已複製訂單編號', description: searchedId.value })
}

function formatDate(ts: number) {
  return new Date(ts).toLocaleString('zh-TW', { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })
}

function isCompleted(order: Order, i: number) {
  if (order.status === 'cancelled') return false
  const currentStep = STATUS_META[order.status].step
  return currentStep >= 0 && i <= currentStep
}

function isCurrent(order: Order, entry: { status: OrderStatus }) {
  return order.status !== 'cancelled' && entry.status === order.status
}

function isFuture(order: Order, i: number) {
  return !isCompleted(order, i) && order.status !== 'cancelled'
}

function isHidden(order: Order, entry: { status: OrderStatus }) {
  return order.status === 'cancelled' && entry.status !== 'cancelled' && entry.status !== 'pending'
}
</script>

<template>
  <Dialog
    :open="ui.trackOrderOpen"
    :show-close="false"
    aria-label="訂單查詢"
    class="max-w-lg p-0"
    @update:open="ui.closeTrackOrder()"
  >
    <!-- Header -->
    <div class="border-b border-border/60 px-6 py-5">
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-2">
          <PackageSearch class="h-4 w-4" />
          <h2 class="text-base font-semibold tracking-tight">訂單查詢</h2>
        </div>
        <button
          type="button"
          class="grid h-8 w-8 place-items-center rounded-full text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
          aria-label="關閉"
          @click="ui.closeTrackOrder()"
        >
          <X class="h-4 w-4" />
        </button>
      </div>
      <p class="mt-1 text-xs text-muted-foreground">
        {{ isMemberSession ? '登入後可直接以訂單編號查詢您的訂單。' : '輸入您的訂單編號與結帳時產生的存取碼，即可查詢訂單狀態。' }}
      </p>

      <div class="mt-4 space-y-2">
        <div class="relative">
          <Search class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            v-model="input"
            @keydown.enter="handleSearch()"
            placeholder="訂單編號 (TW-XXXX...)"
            class="h-10 rounded-full pl-9 uppercase"
          />
        </div>
        <div v-if="!isMemberSession" class="relative">
          <KeyRound class="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            v-model="tokenInput"
            @keydown.enter="handleSearch()"
            placeholder="訂單存取碼"
            class="h-10 rounded-full pl-9"
          />
        </div>
        <Button
          @click="handleSearch()"
          :disabled="loading"
          class="h-10 w-full rounded-full bg-cta text-cta-foreground hover:brightness-95"
        >
          {{ loading ? '查詢中…' : '查詢' }}
        </Button>
      </div>

      <p v-if="error" class="mt-2 text-xs text-destructive">{{ error }}</p>

      <!-- Lost token notice — secure recovery not yet available -->
      <div v-if="!searchedId && !isMemberSession" class="mt-3">
        <p class="text-[11px] text-muted-foreground">
          若遺失存取碼，請聯繫客服協助。客服聯絡方式將於正式上線前公告。
        </p>
      </div>
    </div>

    <!-- Result -->
    <div class="max-h-[60vh] overflow-y-auto px-6 py-5">
      <Transition name="fade" mode="out-in">
        <div v-if="memberOrder" :key="'member-' + memberOrder.id + memberOrder.status" class="space-y-5">
          <div class="flex items-start justify-between gap-3">
            <div>
              <div class="flex items-center gap-2">
                <p class="text-sm font-semibold">{{ memberOrder.id }}</p>
                <button type="button" @click="handleCopyId" class="text-muted-foreground transition-colors hover:text-foreground" aria-label="複製訂單編號">
                  <Copy class="h-3.5 w-3.5" />
                </button>
              </div>
              <p class="mt-0.5 text-[11px] text-muted-foreground">
                更新時間：{{ formatDate(memberOrder.updatedUnix * 1000) }}
              </p>
            </div>
            <span :class="`inline-flex h-6 items-center rounded-full px-2.5 text-[11px] font-medium ${STATUS_BADGE_STYLES[memberOrder.status]}`">
              {{ STATUS_META[memberOrder.status].label }}
            </span>
          </div>

          <div v-if="memberOrder.customerName || memberOrder.shippingAddress" class="rounded-lg border border-border/60 bg-muted/20 px-4 py-3">
            <div class="flex items-start gap-2 text-xs">
              <MapPin class="mt-0.5 h-3.5 w-3.5 shrink-0 text-muted-foreground" />
              <div>
                <p v-if="memberOrder.customerName" class="font-medium text-foreground">{{ memberOrder.customerName }}</p>
                <p v-if="memberOrder.shippingAddress" class="text-muted-foreground">{{ memberOrder.shippingAddress }}</p>
                <p v-if="memberOrder.shippingMethod" class="text-muted-foreground">配送：{{ memberOrder.shippingMethod }}</p>
                <p v-if="memberOrder.paymentMethod" class="text-muted-foreground">付款：{{ memberOrder.paymentMethod }}</p>
              </div>
            </div>
          </div>

          <div>
            <h4 class="text-xs font-medium uppercase tracking-wider text-muted-foreground">
              商品 ({{ memberOrder.items.reduce((s, i) => s + i.quantity, 0) }} 件)
            </h4>
            <ul class="mt-3 space-y-2.5">
              <li v-for="item in memberOrder.items" :key="item.sku" class="flex items-center gap-3">
                <div class="flex-1 text-sm">
                  <p class="font-medium leading-tight">{{ item.name }}</p>
                  <p class="text-[11px] text-muted-foreground">{{ item.quantity }} × {{ formatNTD(item.price) }}</p>
                </div>
                <p class="text-sm font-medium tabular-nums">{{ formatNTD(item.price * item.quantity) }}</p>
              </li>
            </ul>
          </div>

          <Separator />

          <div class="space-y-1.5 text-sm">
            <div class="flex justify-between text-muted-foreground">
              <span>小計</span><span class="tabular-nums">{{ formatNTD(memberOrder.subtotal) }}</span>
            </div>
            <div v-if="memberOrder.discount > 0" class="flex justify-between text-foreground">
              <span>折扣</span><span class="tabular-nums">−{{ formatNTD(memberOrder.discount) }}</span>
            </div>
            <div class="flex justify-between text-muted-foreground">
              <span>運費</span>
              <span class="tabular-nums">{{ memberOrder.shipping === 0 ? '免運' : formatNTD(memberOrder.shipping) }}</span>
            </div>
            <Separator class="my-2" />
            <div class="flex items-baseline justify-between">
              <span class="font-semibold">合計</span>
              <span class="text-base font-semibold tabular-nums">{{ formatNTD(memberOrder.total) }}</span>
            </div>
          </div>
        </div>

        <div v-else-if="foundOrder" :key="foundOrder.id + foundOrder.status" class="space-y-5">
          <!-- Order header -->
          <div class="flex items-start justify-between gap-3">
            <div>
              <div class="flex items-center gap-2">
                <p class="text-sm font-semibold">{{ foundOrder.id }}</p>
                <button @click="handleCopyId" class="text-muted-foreground transition-colors hover:text-foreground" aria-label="複製訂單編號">
                  <Copy class="h-3.5 w-3.5" />
                </button>
              </div>
              <p class="mt-0.5 text-[11px] text-muted-foreground">
                下單時間：{{ formatDate(foundOrder.placedAt) }}
              </p>
            </div>
            <span :class="`inline-flex h-6 items-center rounded-full px-2.5 text-[11px] font-medium ${STATUS_BADGE_STYLES[foundOrder.status]}`">
              {{ STATUS_META[foundOrder.status].label }}
            </span>
          </div>

          <!-- Shipping address -->
          <div class="rounded-lg border border-border/60 bg-muted/20 px-4 py-3">
            <div class="flex items-start gap-2 text-xs">
              <MapPin class="mt-0.5 h-3.5 w-3.5 shrink-0 text-muted-foreground" />
              <div>
                <p class="font-medium text-foreground">{{ foundOrder.shippingAddress.name }}</p>
                <p class="text-muted-foreground">{{ foundOrder.shippingAddress.address }}</p>
                <p class="text-muted-foreground">{{ foundOrder.shippingAddress.city }}, {{ foundOrder.shippingAddress.zip }}</p>
                <p class="text-muted-foreground">{{ foundOrder.shippingAddress.country }}</p>
              </div>
            </div>
          </div>

          <!-- Timeline -->
          <div>
            <h4 class="text-xs font-medium uppercase tracking-wider text-muted-foreground">訂單進度</h4>
            <ol class="mt-3 space-y-0">
              <li
                v-for="(entry, i) in foundOrder.timeline"
                v-show="!isHidden(foundOrder, entry)"
                :key="entry.status"
                class="relative flex gap-3 pb-5 last:pb-0"
              >
                <div
                  v-if="i < foundOrder.timeline.length - 1"
                  :class="['absolute left-3 top-7 h-[calc(100%-1rem)] w-px', isCompleted(foundOrder, i) ? 'bg-cta' : 'bg-border']"
                />
                <div
                  :class="[
                    'relative z-10 grid h-6 w-6 shrink-0 place-items-center rounded-full transition-colors',
                    isCurrent(foundOrder, entry) ? 'bg-cta text-cta-foreground ring-4 ring-cta/10'
                    : isCompleted(foundOrder, i) ? 'bg-cta text-cta-foreground'
                    : 'bg-muted text-muted-foreground',
                  ]"
                >
                  <Check v-if="isCompleted(foundOrder, i) && !isCurrent(foundOrder, entry)" class="h-3 w-3" :stroke-width="3" />
                  <component v-else :is="STATUS_ICONS[entry.status]" class="h-3 w-3" />
                </div>
                <div class="pt-0.5">
                  <p :class="['text-sm', isFuture(foundOrder, i) ? 'text-muted-foreground' : 'font-medium text-foreground']">
                    {{ entry.label }}
                  </p>
                  <p class="text-[11px] text-muted-foreground">{{ entry.description }}</p>
                  <p class="mt-0.5 text-[11px] text-muted-foreground">
                    {{ isFuture(foundOrder, i) ? 'Estimated' : '' }} {{ formatDate(entry.timestamp) }}
                  </p>
                </div>
              </li>
            </ol>
          </div>

          <Separator />

          <!-- Items -->
          <div>
            <h4 class="text-xs font-medium uppercase tracking-wider text-muted-foreground">
              Items ({{ foundOrder.items.reduce((s, i) => s + i.quantity, 0) }})
            </h4>
            <ul class="mt-3 space-y-2.5">
              <li v-for="item in foundOrder.items" :key="item.productId" class="flex items-center gap-3">
                <div class="h-11 w-9 overflow-hidden rounded-md bg-muted">
                  <img :src="item.image" :alt="item.name" class="h-full w-full object-cover" />
                </div>
                <div class="flex-1 text-sm">
                  <p class="font-medium leading-tight">{{ item.name }}</p>
                  <p class="text-[11px] text-muted-foreground">Qty {{ item.quantity }} · ${{ item.price }}</p>
                </div>
                <p class="text-sm font-medium tabular-nums">${{ (item.price * item.quantity).toFixed(2) }}</p>
              </li>
            </ul>
          </div>

          <Separator />

          <!-- Totals -->
          <div class="space-y-1.5 text-sm">
            <div class="flex justify-between text-muted-foreground">
              <span>Subtotal</span><span class="tabular-nums">${{ foundOrder.subtotal.toFixed(2) }}</span>
            </div>
            <div v-if="foundOrder.discount > 0" class="flex justify-between text-foreground">
              <span>Discount</span><span class="tabular-nums">−${{ foundOrder.discount.toFixed(2) }}</span>
            </div>
            <div class="flex justify-between text-muted-foreground">
              <span>Shipping</span>
              <span class="tabular-nums">{{ foundOrder.shipping === 0 ? 'Free' : `$${foundOrder.shipping.toFixed(2)}` }}</span>
            </div>
            <div class="flex justify-between text-muted-foreground">
              <span>Tax</span><span class="tabular-nums">${{ foundOrder.tax.toFixed(2) }}</span>
            </div>
            <Separator class="my-2" />
            <div class="flex items-baseline justify-between">
              <span class="font-semibold">Total</span>
              <span class="text-base font-semibold tabular-nums">${{ foundOrder.total.toFixed(2) }}</span>
            </div>
          </div>
        </div>

        <div v-else key="empty" class="flex flex-col items-center justify-center gap-3 py-10 text-center">
          <div class="grid h-12 w-12 place-items-center rounded-full bg-muted">
            <PackageSearch class="h-5 w-5 text-muted-foreground" />
          </div>
          <div>
            <p class="text-sm font-medium">尚未查詢訂單</p>
            <p class="mt-1 text-xs text-muted-foreground">{{ isMemberSession ? '輸入訂單編號即可查看狀態。' : '輸入訂單編號與存取碼以查看狀態。' }}</p>
          </div>
        </div>
      </Transition>
    </div>
  </Dialog>
</template>

<style scoped>
.fade-enter-active, .fade-leave-active { transition: opacity 0.25s ease; }
.fade-enter-from, .fade-leave-to { opacity: 0; }
</style>
