<script setup lang="ts">
import { ref, computed, watch, reactive } from 'vue'
import {
  Truck, Store, Banknote, Check, ChevronLeft,
  Package, MapPin, ArrowRight, PartyPopper, X, AlertCircle,
} from 'lucide-vue-next'
import { useCartStore } from '@/shared/stores/cart'
import { useUserStore } from '@/shared/stores/user'
import { useUiStore } from '@/shared/stores/ui'
import { useToast } from '@/shared/composables/use-toast'
import {
  createOrder, createOrderForMember, fetchQuote, fetchShippingMethods, fetchPaymentMethods,
  ApiRequestError,
  type QuoteResult, type ShippingMethodResult, type PaymentMethodResult,
} from '@/shared/lib/api'
import { signOut } from '@/shared/lib/auth/session'
import type { Order, OrderStatus, Address, OrderItem } from '@/shared/lib/types'
import Dialog from '@/shared/components/ui/Dialog.vue'
import Button from '@/shared/components/ui/Button.vue'
import Input from '@/shared/components/ui/Input.vue'
import Separator from '@/shared/components/ui/Separator.vue'
import { formatNTD, cn } from '@/shared/lib/utils'

const ui = useUiStore()
const cart = useCartStore()
const userStore = useUserStore()
const { toast } = useToast()

type Step = 'shipping' | 'payment' | 'review' | 'done'
const step = ref<Step>('shipping')
const placedOrder = ref<Order | null>(null)
const submitting = ref(false)
const submitError = ref<string | null>(null)

// Server-authoritative quote. The checkout displays these totals, not
// client-computed ones. The quote is fetched when the user reaches the
// review step.
const quote = ref<QuoteResult | null>(null)
const quoteLoading = ref(false)
const quoteError = ref<string | null>(null)

// Server-authoritative shipping and payment methods. Fetched from the
// Go API, not hardcoded. Fail-closed: starts empty, populated only from
// the API. If the API is unavailable, no methods are shown.
const shippingMethods = ref<ShippingMethodResult[]>([])
const shippingMethodsLoading = ref(false)
const shippingMethodsError = ref<string | null>(null)
const paymentMethods = ref<PaymentMethodResult[]>([])
const paymentMethodsLoading = ref(false)
const paymentMethodsError = ref<string | null>(null)

const shippingForm = reactive({
  name: userStore.user?.name ?? '',
  email: userStore.user?.email ?? '',
  phone: '',
  address: userStore.user?.defaultAddress?.address ?? '',
  city: userStore.user?.defaultAddress?.city ?? '',
  zip: userStore.user?.defaultAddress?.zip ?? '',
  country: '台灣',
})

const selectedShipping = ref('')
const selectedPayment = ref('')

// Stable idempotency key for the current checkout submission intent.
// Generated once via crypto.randomUUID() when the user first clicks
// Confirm, and reused across transport retries. If the server commits
// but the response is lost, the retry uses the SAME key so the server
// returns the original order instead of creating a duplicate. The key
// rotates ONLY after confirmed success (in placeOrder) or when a
// material order-defining input changes (via the checkoutFingerprint
// watcher below). Opening/closing or retrying an unchanged checkout
// does NOT rotate the key — this is critical for replay safety after
// a committed-but-lost response.
const idempotencyKey = ref<string | null>(null)

// checkoutFingerprint captures the material order-defining inputs that
// determine the created order. When any of these change, the
// idempotency key is invalidated so the next submission gets a fresh
// key. This prevents a stale key from being reused after the customer
// changes their cart, shipping info, or payment method.
//
// ENCODING: Uses JSON.stringify over a typed object with sorted cart
// entries. JSON is length-delimited and escapes special characters
// (newlines, pipes, quotes) inside values, so two materially different
// inputs can never produce the same fingerprint string. The previous
// delimiter-based concatenation (sku:qty.join(',') + '|' + fields)
// was collision-prone: a field value containing the delimiter character
// could merge with the next field and leave the old key active after a
// material change.
//
// DECLARATION ORDER: this computed and its watcher MUST be declared
// after shippingForm, selectedShipping, and selectedPayment are
// initialized. Vue's watch evaluates the computed getter immediately
// to collect dependencies; accessing a const binding before its
// initializer runs is a TDZ violation.
const checkoutFingerprint = computed(() => {
  const cartItems = cart.items
    .map((i) => ({ sku: i.product.sku, quantity: i.quantity }))
    .sort((a, b) => a.sku < b.sku ? -1 : a.sku > b.sku ? 1 : a.quantity - b.quantity)
  return JSON.stringify({
    cart: cartItems,
    name: shippingForm.name,
    email: shippingForm.email,
    phone: shippingForm.phone,
    address: shippingForm.address,
    city: shippingForm.city,
    zip: shippingForm.zip,
    shippingMethod: selectedShipping.value,
    paymentMethod: selectedPayment.value,
    promoCode: cart.appliedPromo ?? '',
  })
})

// Invalidate the idempotency key when material inputs change.
watch(checkoutFingerprint, () => {
  idempotencyKey.value = null
})

const steps: { key: Step; label: string }[] = [
  { key: 'shipping', label: '配送資訊' },
  { key: 'payment', label: '付款方式' },
  { key: 'review', label: '確認訂單' },
]

const stepIndex = computed(() => steps.findIndex((s) => s.key === step.value))

const SHIP_ICONS: Record<string, any> = { truck: Truck, store: Store }
const PAY_ICONS: Record<string, any> = { banknote: Banknote, store: Store }

const availableShippingMethods = computed(() => shippingMethods.value.filter((m) => m.available))
const availablePaymentMethods = computed(() => paymentMethods.value.filter((m) => m.available))

const selectedShippingMethod = computed(() => shippingMethods.value.find((m) => m.id === selectedShipping.value))
const selectedPaymentMethod = computed(() => paymentMethods.value.find((m) => m.id === selectedPayment.value))

// Server-authoritative totals from the quote. Fall back to 0 when no quote
// has been fetched yet (the review step triggers the fetch).
const subtotal = computed(() => quote.value?.subtotal ?? 0)
const discount = computed(() => quote.value?.discount ?? 0)
const baseShipping = computed(() => quote.value?.shipping ?? 0)
const total = computed(() => quote.value?.total ?? 0)

const isShippingValid = computed(() =>
  shippingForm.name.trim() && shippingForm.email.trim() && shippingForm.phone.trim() &&
  shippingForm.address.trim() && shippingForm.city.trim() && shippingForm.zip.trim() &&
  selectedShipping.value !== ''
)

// Checkout is unavailable when no shipping methods are available (no
// approved fee schedule) or no payment methods are ready.
const checkoutUnavailable = computed(() =>
  availableShippingMethods.value.length === 0 || availablePaymentMethods.value.length === 0
)

watch(() => ui.checkoutOpen, (open) => {
  if (open) {
    step.value = 'shipping'
    placedOrder.value = null
    quote.value = null
    quoteError.value = null
    submitError.value = null
    // NOTE: idempotencyKey is NOT reset here. The key must survive
    // dialog close/reopen so that a committed-but-lost response can
    // be retried with the same key after reopening. The key rotates
    // only after confirmed success (in placeOrder) or when a material
    // input changes (via the checkoutFingerprint watcher). Resetting
    // here would allow duplicate orders after transport failure.
    if (userStore.user) {
      shippingForm.name = userStore.user.name
      shippingForm.email = userStore.user.email
    }
    loadShippingMethods()
    loadPaymentMethods()
  }
})

async function loadShippingMethods() {
  shippingMethodsLoading.value = true
  shippingMethodsError.value = null
  try {
    shippingMethods.value = await fetchShippingMethods()
    // Only auto-select an AVAILABLE method. If no method is available
    // (no approved fee schedule), keep selectedShipping empty so the
    // checkout stays blocked at the shipping step with the truthful
    // unavailable state. Do NOT auto-select an unavailable method.
    const firstAvailable = shippingMethods.value.find((m) => m.available)
    selectedShipping.value = firstAvailable?.id ?? ''
  } catch (e: any) {
    shippingMethodsError.value = (e as Error).message || '無法取得配送方式'
    shippingMethods.value = []
    selectedShipping.value = ''
  } finally {
    shippingMethodsLoading.value = false
  }
}

async function loadPaymentMethods() {
  paymentMethodsLoading.value = true
  paymentMethodsError.value = null
  try {
    paymentMethods.value = await fetchPaymentMethods()
    const firstAvailable = paymentMethods.value.find((m) => m.available)
    selectedPayment.value = firstAvailable?.id ?? ''
  } catch (e: any) {
    paymentMethodsError.value = (e as Error).message || '無法取得付款方式'
    paymentMethods.value = []
    selectedPayment.value = ''
  } finally {
    paymentMethodsLoading.value = false
  }
}

async function loadQuote() {
  quoteLoading.value = true
  quoteError.value = null
  try {
    quote.value = await fetchQuote(
      cart.items.map((item) => ({ sku: item.product.sku, quantity: item.quantity })),
      selectedShipping.value,
      selectedPayment.value,
      cart.appliedPromo || undefined,
    )
  } catch (e: any) {
    quoteError.value = (e as Error).message || '無法取得報價'
    quote.value = null
  } finally {
    quoteLoading.value = false
  }
}

async function nextStep() {
  if (step.value === 'shipping') {
    if (!isShippingValid.value) {
      toast.error('請填寫完整配送資訊')
      return
    }
    step.value = 'payment'
  } else if (step.value === 'payment') {
    if (selectedPayment.value === '') {
      toast.error('請選擇付款方式')
      return
    }
    step.value = 'review'
    // Fetch the server-authoritative quote before showing the review.
    await loadQuote()
  }
}

function prevStep() {
  if (step.value === 'payment') step.value = 'shipping'
  else if (step.value === 'review') step.value = 'payment'
}

async function placeOrder() {
  if (submitting.value) return
  submitting.value = true
  submitError.value = null

  const address: Address = {
    name: shippingForm.name,
    email: shippingForm.email,
    phone: shippingForm.phone,
    address: shippingForm.address,
    city: shippingForm.city,
    zip: shippingForm.zip,
    country: shippingForm.country,
  }

  const items: OrderItem[] = cart.items.map((item) => ({
    productId: item.product.id,
    name: item.product.name,
    image: item.product.image,
    price: item.product.price,
    quantity: item.quantity,
    size: item.selectedSize,
    color: item.selectedColor,
  }))

  try {
    // Stable idempotency key: generated once per checkout submission
    // intent and reused across transport retries. If the server commits
    // but the response is lost, clicking Confirm again uses the SAME key
    // so the server returns the original order instead of creating a
    // duplicate. The key rotates only after confirmed success (in the
    // success path below) or when a material order-defining input
    // changes (via the checkoutFingerprint watcher). Opening/closing
    // or retrying an unchanged checkout does NOT create a new key.
    //
    // crypto.randomUUID provides a cryptographically strong 128-bit
    // UUIDv4. If the browser does not support it (insecure context),
    // we fail closed — no order is placed with a weak key.
    if (!idempotencyKey.value) {
      if (typeof crypto === 'undefined' || typeof crypto.randomUUID !== 'function') {
        throw new Error('瀏覽器不支援安全金鑰產生 (crypto.randomUUID)，無法下單')
      }
      idempotencyKey.value = crypto.randomUUID()
    }

    // Items send only sku and quantity — the server is the pricing
    // authority and recalculates from the catalog. Client prices are
    // ignored. Payment method is validated server-side against
    // enabled+ready methods. MemberID is never sent from the browser.
    const orderPayload = {
      items: cart.items.map((item) => ({
        sku: item.product.sku,
        quantity: item.quantity,
      })),
      customerName: shippingForm.name,
      email: shippingForm.email,
      phone: shippingForm.phone,
      shippingAddress: `${address.address}, ${address.city} ${address.zip}`,
      shippingMethod: selectedShipping.value,
      paymentMethod: selectedPayment.value,
      promoCode: cart.appliedPromo || '',
      idempotencyKey: idempotencyKey.value,
    }
    const token = userStore.bearerToken.trim()
    const apiOrder = token
      ? await createOrderForMember(orderPayload, token)
      : await createOrder(orderPayload)

    // Fail-closed validation of the server response: if required
    // server-authoritative fields are missing from a 2xx response, we
    // must NOT fabricate fallback values. The Go server always emits
    // id, status, subtotal, discount, shipping, total, shipping_method,
    // and payment_method as non-omitempty JSON fields. A missing field
    // means the response is malformed — we surface an error instead of
    // synthesizing a fallback. No nullish coalescing (??) or default
    // (||) on these required fields.
    const o = apiOrder.order
    if (!o.id || typeof o.id !== 'string' || o.id === '') {
      throw new Error('伺服器回應缺少訂單編號，無法完成下單')
    }
    if (typeof o.status !== 'string' || o.status === '') {
      throw new Error('伺服器回應缺少訂單狀態，無法完成下單')
    }
    if (typeof o.subtotal !== 'number') {
      throw new Error('伺服器回應缺少小計金額，無法完成下單')
    }
    if (typeof o.discount !== 'number') {
      throw new Error('伺服器回應缺少折扣金額，無法完成下單')
    }
    if (typeof o.shipping !== 'number') {
      throw new Error('伺服器回應缺少運費金額，無法完成下單')
    }
    if (typeof o.total !== 'number') {
      throw new Error('伺服器回應缺少總金額，無法完成下單')
    }
    if (typeof o.shipping_method !== 'string' || o.shipping_method === '') {
      throw new Error('伺服器回應缺少配送方式，無法完成下單')
    }
    if (typeof o.payment_method !== 'string' || o.payment_method === '') {
      throw new Error('伺服器回應缺少付款方式，無法完成下單')
    }
    const order: Order = {
      id: o.id,
      userEmail: userStore.user?.email ?? null,
      items,
      subtotal: o.subtotal,
      discount: o.discount,
      shipping: o.shipping,
      tax: 0, // tax-inclusive TWD per GATE-003
      total: o.total,
      status: o.status as OrderStatus,
      shippingAddress: address,
      shippingMethod: selectedShippingMethod.value?.label ?? o.shipping_method,
      paymentMethod: selectedPaymentMethod.value?.label ?? o.payment_method,
      placedAt: Date.now(),
      timeline: [],
      accessToken: o.access_token || undefined,
    }

    placedOrder.value = order
    step.value = 'done'
    // Rotate the idempotency key after confirmed success so a future
    // checkout gets a fresh key.
    idempotencyKey.value = null
    cart.clear()
    toast({ title: '訂單已成立', description: `訂單編號 ${order.id}` })
  } catch (e: unknown) {
    if (e instanceof ApiRequestError && e.status === 401) {
      userStore.logout()
      await signOut()
    }
    submitError.value = e instanceof Error ? e.message : '訂單成立失敗，請稍後再試'
    toast({ title: '訂單成立失敗', description: submitError.value ?? undefined, variant: 'destructive' })
  } finally {
    submitting.value = false
  }
}

function close() {
  ui.closeCheckout()
}
</script>

<template>
  <Dialog
    :open="ui.checkoutOpen"
    :show-close="false"
    :aria-label="step === 'done' ? '訂單完成' : '結帳'"
    class="max-w-2xl p-0"
    @update:open="ui.closeCheckout()"
  >
    <div class="flex max-h-[85vh] flex-col">
      <!-- Header -->
      <div class="flex items-center justify-between border-b border-border/60 px-6 py-4">
        <h2 class="text-lg font-semibold tracking-tight">{{ step === 'done' ? '訂單完成' : '結帳' }}</h2>
        <button
          type="button"
          class="grid h-8 w-8 place-items-center rounded-full text-muted-foreground transition-colors hover:bg-muted hover:text-foreground"
          aria-label="關閉"
          @click="close"
        >
          <X class="h-4 w-4" />
        </button>
      </div>

      <!-- Stepper -->
      <div v-if="step !== 'done'" class="border-b border-border/60 px-6 py-4">
        <div class="flex items-center gap-2">
          <template v-for="(s, i) in steps" :key="s.key">
            <div class="flex items-center gap-2">
              <div
                :class="[
                  'grid h-7 w-7 place-items-center rounded-full text-xs font-medium transition-colors',
                  i <= stepIndex ? 'bg-cta text-cta-foreground' : 'bg-muted text-muted-foreground',
                ]"
              >
                <Check v-if="i < stepIndex" class="h-3.5 w-3.5" :stroke-width="3" />
                <span v-else>{{ i + 1 }}</span>
              </div>
              <span :class="['text-xs', i <= stepIndex ? 'font-medium text-foreground' : 'text-muted-foreground']">
                {{ s.label }}
              </span>
            </div>
            <div v-if="i < steps.length - 1" :class="['h-px flex-1', i < stepIndex ? 'bg-cta' : 'bg-border']" />
          </template>
        </div>
      </div>

      <!-- Body -->
      <div class="flex-1 overflow-y-auto px-6 py-5">
        <!-- Step: Shipping -->
        <div v-if="step === 'shipping'" class="space-y-5">
          <div>
            <h3 class="text-sm font-semibold">收件人資訊</h3>
            <div class="mt-3 grid grid-cols-2 gap-3">
              <div class="col-span-2 sm:col-span-1">
                <label class="text-xs font-medium">姓名 *</label>
                <Input v-model="shippingForm.name" placeholder="林美玲" class="mt-1 h-9" />
              </div>
              <div class="col-span-2 sm:col-span-1">
                <label class="text-xs font-medium">電話 *</label>
                <Input v-model="shippingForm.phone" placeholder="0912-345-678" class="mt-1 h-9" />
              </div>
              <div class="col-span-2">
                <label class="text-xs font-medium">Email *</label>
                <Input v-model="shippingForm.email" type="email" placeholder="you@example.com" class="mt-1 h-9" />
              </div>
              <div class="col-span-2">
                <label class="text-xs font-medium">地址 *</label>
                <Input v-model="shippingForm.address" placeholder="忠孝東路四段 100 號 5F" class="mt-1 h-9" />
              </div>
              <div>
                <label class="text-xs font-medium">城市 *</label>
                <Input v-model="shippingForm.city" placeholder="台北市" class="mt-1 h-9" />
              </div>
              <div>
                <label class="text-xs font-medium">郵遞區號 *</label>
                <Input v-model="shippingForm.zip" placeholder="106" class="mt-1 h-9" />
              </div>
            </div>
          </div>

          <Separator />

          <div>
            <h3 class="text-sm font-semibold">配送方式</h3>
            <!-- Loading state -->
            <div v-if="shippingMethodsLoading" class="mt-3 flex items-center gap-2 text-sm text-muted-foreground">
              <div class="h-4 w-4 animate-spin rounded-full border-2 border-muted border-t-cta" />
              正在從伺服器取得配送方式…
            </div>
            <!-- Error state -->
            <div v-else-if="shippingMethodsError" class="mt-3 rounded-lg border border-destructive/30 bg-destructive/10 p-3 text-sm text-destructive">
              無法取得配送方式：{{ shippingMethodsError }}
              <button class="ml-2 underline" @click="loadShippingMethods">重試</button>
            </div>
            <!-- Empty state -->
            <div v-else-if="shippingMethods.length === 0" class="mt-3 rounded-lg border border-border bg-muted/20 p-3 text-sm text-muted-foreground">
              目前沒有可用的配送方式。
            </div>
            <!-- Methods list -->
            <div v-else class="mt-3 space-y-2">
              <label
                v-for="method in shippingMethods"
                :key="method.id"
                :class="[
                  'flex items-center gap-3 rounded-xl border p-3 transition-colors',
                  method.available
                    ? selectedShipping === method.id
                      ? 'border-cta bg-cta/5 cursor-pointer'
                      : 'border-border hover:bg-muted/30 cursor-pointer'
                    : 'border-border opacity-50 cursor-not-allowed',
                ]"
              >
                <input
                  type="radio"
                  :value="method.id"
                  v-model="selectedShipping"
                  :disabled="!method.available"
                  class="sr-only"
                />
                <div :class="['grid h-9 w-9 place-items-center rounded-full', selectedShipping === method.id ? 'bg-cta text-cta-foreground' : 'bg-muted text-muted-foreground']">
                  <Truck class="h-4 w-4" />
                </div>
                <div class="flex-1">
                  <p class="text-sm font-medium">{{ method.label }}</p>
                  <p class="text-[11px] text-muted-foreground">{{ method.description }}</p>
                </div>
                <span v-if="method.available" class="text-sm font-medium text-cta">可用</span>
                <span v-else class="text-xs text-muted-foreground">尚未開放</span>
              </label>
            </div>
            <!-- Checkout unavailable notice -->
            <div v-if="!shippingMethodsLoading && !shippingMethodsError && availableShippingMethods.length === 0 && shippingMethods.length > 0" class="mt-3 flex items-start gap-2 rounded-lg border border-amber-300/60 bg-amber-50 p-3 text-xs text-amber-800 dark:border-amber-700/60 dark:bg-amber-950/30 dark:text-amber-200">
              <AlertCircle class="h-4 w-4 shrink-0 mt-0.5" />
              <span>結帳功能尚未開放。配送費率需經產品決策核准後才能啟用。</span>
            </div>
          </div>
        </div>

        <!-- Step: Payment -->
        <div v-else-if="step === 'payment'" class="space-y-5">
          <div>
            <h3 class="text-sm font-semibold">付款方式</h3>
            <!-- Loading state -->
            <div v-if="paymentMethodsLoading" class="mt-3 flex items-center gap-2 text-sm text-muted-foreground">
              <div class="h-4 w-4 animate-spin rounded-full border-2 border-muted border-t-cta" />
              正在從伺服器取得付款方式…
            </div>
            <!-- Error state -->
            <div v-else-if="paymentMethodsError" class="mt-3 rounded-lg border border-destructive/30 bg-destructive/10 p-3 text-sm text-destructive">
              無法取得付款方式：{{ paymentMethodsError }}
              <button class="ml-2 underline" @click="loadPaymentMethods">重試</button>
            </div>
            <!-- Empty state -->
            <div v-else-if="availablePaymentMethods.length === 0" class="mt-3 rounded-lg border border-border bg-muted/20 p-3 text-sm text-muted-foreground">
              目前沒有可用的付款方式。
            </div>
            <!-- Methods list -->
            <div v-else class="mt-3 space-y-2">
              <label
                v-for="method in availablePaymentMethods"
                :key="method.id"
                :class="[
                  'flex cursor-pointer items-center gap-3 rounded-xl border p-3 transition-colors',
                  selectedPayment === method.id ? 'border-cta bg-cta/5' : 'border-border hover:bg-muted/30',
                ]"
              >
                <input type="radio" :value="method.id" v-model="selectedPayment" class="sr-only" />
                <div :class="['grid h-9 w-9 place-items-center rounded-full', selectedPayment === method.id ? 'bg-cta text-cta-foreground' : 'bg-muted text-muted-foreground']">
                  <Banknote class="h-4 w-4" />
                </div>
                <div class="flex-1">
                  <p class="text-sm font-medium">{{ method.label }}</p>
                  <p class="text-[11px] text-muted-foreground">{{ method.method }}</p>
                </div>
              </label>
            </div>
          </div>
        </div>

        <!-- Step: Review -->
        <div v-else-if="step === 'review'" class="space-y-5">
          <!-- Error message -->
          <div v-if="submitError" class="rounded-lg border border-destructive/30 bg-destructive/10 p-3 text-sm text-destructive">
            {{ submitError }}
          </div>
          <!-- Quote error -->
          <div v-if="quoteError" class="rounded-lg border border-destructive/30 bg-destructive/10 p-3 text-sm text-destructive">
            無法取得伺服器報價：{{ quoteError }}
            <button class="ml-2 underline" @click="loadQuote">重試</button>
          </div>
          <!-- Quote loading -->
          <div v-if="quoteLoading" class="flex items-center gap-2 text-sm text-muted-foreground">
            <div class="h-4 w-4 animate-spin rounded-full border-2 border-muted border-t-cta" />
            正在從伺服器取得報價…
          </div>
          <!-- Shipping info -->
          <div class="rounded-xl border border-border/60 bg-muted/20 p-4">
            <div class="flex items-center gap-2 text-sm font-semibold">
              <MapPin class="h-4 w-4" />
              收件資訊
            </div>
            <div class="mt-2 text-xs text-muted-foreground">
              <p>{{ shippingForm.name }} · {{ shippingForm.phone }}</p>
              <p>{{ shippingForm.address }}</p>
              <p>{{ shippingForm.city }}, {{ shippingForm.zip }} {{ shippingForm.country }}</p>
              <p class="mt-1">配送：{{ selectedShippingMethod?.label ?? selectedShipping }}</p>
              <p>付款：{{ selectedPaymentMethod?.label ?? selectedPayment }}</p>
            </div>
          </div>

          <!-- Items — server-authoritative names and prices from quote.items -->
          <div v-if="quote && quote.items.length > 0">
            <h4 class="text-xs font-medium uppercase tracking-wider text-muted-foreground">
              商品 ({{ quote.items.reduce((s, i) => s + i.quantity, 0) }} 件)
            </h4>
            <ul class="mt-3 space-y-2">
              <li v-for="(item, idx) in quote.items" :key="idx" class="flex items-center gap-3">
                <div class="flex-1 text-sm">
                  <p class="font-medium leading-tight">{{ item.name }}</p>
                  <p class="text-[11px] text-muted-foreground">
                    {{ item.quantity }} × {{ formatNTD(item.price) }}
                  </p>
                </div>
                <p class="text-sm font-medium tabular-nums">{{ formatNTD(item.price * item.quantity) }}</p>
              </li>
            </ul>
          </div>

          <Separator />

          <!-- Summary — server-authoritative totals from /api/quote -->
          <div v-if="quote" class="space-y-1.5 text-sm">
            <div class="flex justify-between text-muted-foreground">
              <span>小計</span><span class="tabular-nums">{{ formatNTD(subtotal) }}</span>
            </div>
            <div v-if="discount > 0" class="flex justify-between text-foreground">
              <span>折扣</span><span class="tabular-nums">−{{ formatNTD(discount) }}</span>
            </div>
            <div class="flex justify-between text-muted-foreground">
              <span>運費 ({{ selectedShippingMethod?.label ?? selectedShipping }})</span>
              <span class="tabular-nums">{{ baseShipping === 0 ? '免運' : formatNTD(baseShipping) }}</span>
            </div>
            <Separator class="my-2" />
            <div class="flex items-baseline justify-between">
              <span class="font-semibold">合計</span>
              <span class="text-lg font-semibold tabular-nums text-cta">{{ formatNTD(total) }}</span>
            </div>
            <p class="text-[11px] text-muted-foreground">含稅價 · 伺服器計算</p>
          </div>
        </div>

        <!-- Step: Done -->
        <div v-else-if="step === 'done' && placedOrder" class="flex flex-col items-center justify-center py-8 text-center">
          <div class="grid h-16 w-16 place-items-center rounded-full bg-cta/10">
            <PartyPopper class="h-7 w-7 text-cta" />
          </div>
          <h2 class="mt-4 text-lg font-semibold">訂單已成立！</h2>
          <p class="mt-1 text-sm text-muted-foreground">感謝您的購買，我們已收到您的訂單。</p>

          <div class="mt-5 w-full max-w-xs rounded-xl border border-border/60 bg-muted/20 p-4 text-left">
            <div class="flex items-center justify-between text-xs">
              <span class="text-muted-foreground">訂單編號</span>
              <span class="font-mono font-semibold">{{ placedOrder.id }}</span>
            </div>
            <Separator class="my-2" />
            <div class="flex items-center justify-between text-xs">
              <span class="text-muted-foreground">合計</span>
              <span class="font-semibold tabular-nums">{{ formatNTD(placedOrder.total) }}</span>
            </div>
            <div class="mt-2 flex items-center justify-between text-xs">
              <span class="text-muted-foreground">配送方式</span>
              <span>{{ placedOrder.shippingMethod }}</span>
            </div>
          </div>

          <!-- One-time access token display: the customer must save this
               to look up their guest order later. It is never shown again. -->
          <div v-if="placedOrder.accessToken" class="mt-3 w-full max-w-xs rounded-xl border border-amber-300/60 bg-amber-50 p-4 text-left dark:border-amber-700/60 dark:bg-amber-950/30">
            <p class="text-xs font-medium text-amber-800 dark:text-amber-200">請保存您的訂單查詢代碼</p>
            <p class="mt-1 text-xs text-amber-700 dark:text-amber-300">此代碼僅顯示一次，請複製保存以利後續查詢訂單。</p>
            <div class="mt-2 break-all rounded bg-white/60 px-2 py-1 font-mono text-xs dark:bg-black/20">
              {{ placedOrder.accessToken }}
            </div>
          </div>

          <Button @click="close" class="mt-6 h-10 rounded-full bg-cta px-6 text-cta-foreground hover:brightness-95">
            完成
          </Button>
        </div>
      </div>

      <!-- Footer nav -->
      <div v-if="step !== 'done'" class="flex items-center justify-between border-t border-border/60 px-6 py-4">
        <Button
          v-if="step !== 'shipping'"
          variant="outline"
          @click="prevStep"
          class="h-9 rounded-full"
        >
          <ChevronLeft class="mr-1 h-4 w-4" />
          上一步
        </Button>
        <div v-else />

        <Button
          v-if="step === 'review'"
          @click="placeOrder"
          :disabled="submitting || checkoutUnavailable"
          class="h-9 rounded-full bg-cta px-6 text-cta-foreground hover:brightness-95"
        >
          {{ submitting ? '提交中…' : '確認下單' }}
          <ArrowRight v-if="!submitting" class="ml-1 h-4 w-4" />
        </Button>
        <Button
          v-else
          @click="nextStep"
          :disabled="(step === 'shipping' && (!isShippingValid || checkoutUnavailable)) || (step === 'payment' && selectedPayment === '')"
          class="h-9 rounded-full bg-cta px-6 text-cta-foreground hover:brightness-95 disabled:opacity-50"
        >
          下一步
          <ArrowRight class="ml-1 h-4 w-4" />
        </Button>
      </div>
    </div>
  </Dialog>
</template>
