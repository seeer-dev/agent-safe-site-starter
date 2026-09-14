<script setup lang="ts">
// 結帳頁 — 對應 reference checkout-view.tsx：4 步驟精靈
//   Step1 訂購資料（同收件人切換）→ Step2 配送方式（超商門市挑選 dialog /
//   宅配縣市區域地址）→ Step3 付款方式（信用卡卡面視覺表單/LINE Pay/貨到付款
//   說明）→ Step4 確認（可回跳編輯 + 發票 + 同意條款）。
// 與 reference 差異：
//   - 金額一律以 POST /api/quote 為權威（選擇配送/付款/優惠券變動即重算）；
//     公開 shipping payload 不含費率，運費只能經 quote 顯示。
//   - 信用卡欄位僅為沙箱視覺，絕不傳輸卡號（伺服器端 strict JSON decode，
//     也沒有卡號欄位）。
//   - 訂單成立後以 access_token 查單（X-Order-Access-Token），記入最近訂單。
import { computed, onMounted, ref, watch } from 'vue'
import {
  ArrowLeft, ArrowRight, Banknote, Check, ClipboardCheck, CreditCard,
  Loader2, MapPin, QrCode, Search, ShieldCheck, ShoppingCart, Store,
  Truck, UserRound,
} from 'lucide-vue-next'
import Dialog from '@/shared/components/Dialog.vue'
import FadeImage from '@/shared/components/FadeImage.vue'
import LogoMark from '@/shared/components/LogoMark.vue'
import { apiPost } from '@/shared/lib/api'
import { cart, cartCount, cartSubtotal, clearCart } from '@/shared/lib/cart'
import { TAIWAN_REGIONS, getDistricts } from '@/shared/data/taiwan-regions'
import { SEVEN_ELEVEN_STORES, FAMILYMART_STORES, type CvsStore } from '@/shared/data/cvs-stores'
import { formatNTD } from '@/shared/lib/format'
import { bootstrap, checkoutStore, loadBootstrap, rememberOrder } from '@/shared/lib/store'
import { navigate } from '@/shared/lib/transition'
import { toast } from '@/shared/lib/toast'
import { cn } from '@/shared/lib/utils'
import type { OrderDTO, PaymentMethodDTO, QuoteResult, ShippingMethodDTO } from '@/shared/lib/types'

const STEPS = [
  { title: '訂購資料', icon: UserRound },
  { title: '配送方式', icon: Truck },
  { title: '付款方式', icon: CreditCard },
  { title: '確認訂單', icon: ClipboardCheck },
]

const form = ref({
  name: '', phone: '', email: '', note: '',
  sameRecipient: true, recipientName: '', recipientPhone: '',
  shippingMethodId: '' as string,
  cvsStore: null as { name: string; address: string } | null,
  city: '', district: '', addressDetail: '', zip: '',
  paymentMethodKey: '' as string,
  cardNumber: '', cardExpiry: '', cardCvv: '', cardHolder: '',
  invoiceType: 'personal' as 'personal' | 'company',
  invoiceTaxId: '', invoiceTitle: '',
  agree: false,
})
const errors = ref<Record<string, string>>({})
const step = ref(0)
const processing = ref(false)
const storeDialogOpen = ref(false)
const storeCity = ref('台北市')
const storeSearch = ref('')
const idempotencyKey = crypto.randomUUID()

// ── 資料 ──
const shippingMethods = computed(() => bootstrap.data?.shipping_methods ?? [])
const paymentMethods = computed(() => bootstrap.data?.payment_methods ?? [])
const subtotal = computed(() => cartSubtotal())
const count = computed(() => cartCount())
const coupon = computed(() => checkoutStore.coupon)

const shippingMethod = computed<ShippingMethodDTO | null>(
  () => shippingMethods.value.find((m) => m.id === form.value.shippingMethodId) ?? null,
)
const paymentMethod = computed<PaymentMethodDTO | null>(
  () => paymentMethods.value.find((m) => m.method === form.value.paymentMethodKey) ?? null,
)

// 配送型態：方法 key 含 cvs/store/超商品牌 → 超商取貨；其餘宅配。
const isCVS = computed(() => {
  const id = (shippingMethod.value?.id ?? '').toLowerCase()
  return /cvs|store|711|seven|family|fam|hilife|okmart/.test(id)
})
const cvsBrand = computed<'7-11' | 'FamilyMart' | 'both'>(() => {
  const id = (shippingMethod.value?.id ?? '').toLowerCase()
  const label = shippingMethod.value?.label ?? ''
  if (/711|seven|7-11/.test(id) || label.includes('7-11') || label.includes('7-ELEVEN')) return '7-11'
  if (/family|fam|fm/.test(id) || label.includes('全家')) return 'FamilyMart'
  return 'both'
})
const storeList = computed<CvsStore[]>(() =>
  cvsBrand.value === 'both'
    ? [...SEVEN_ELEVEN_STORES, ...FAMILYMART_STORES]
    : cvsBrand.value === '7-11'
      ? SEVEN_ELEVEN_STORES
      : FAMILYMART_STORES,
)
const filteredStores = computed(() =>
  storeList.value
    .filter((s) => (storeCity.value ? s.city === storeCity.value : true))
    .filter((s) => {
      const kw = storeSearch.value.trim().toLowerCase()
      return !kw || s.name.toLowerCase().includes(kw) || s.address.toLowerCase().includes(kw)
    }),
)
const districts = computed(() => getDistricts(form.value.city))
const isCard = computed(() => /credit|card/.test(form.value.paymentMethodKey.toLowerCase()))
const isLinePay = computed(() => /linepay|line/.test(form.value.paymentMethodKey.toLowerCase()))
const isCOD = computed(() => /cod|cash/.test(form.value.paymentMethodKey.toLowerCase()))

// ── 權威金額（quote） ──
const quote = ref<QuoteResult | null>(null)
const quoting = ref(false)
let quoteSeq = 0
const quoteItems = computed(() => cart.items.map((i) => ({ sku: i.sku, quantity: i.qty })))

async function refreshQuote() {
  if (cart.items.length === 0 || !form.value.shippingMethodId || !form.value.paymentMethodKey) {
    quote.value = null
    return
  }
  const seq = ++quoteSeq
  quoting.value = true
  try {
    const res = await apiPost<{ quote: QuoteResult } | QuoteResult>('/api/quote', {
      items: quoteItems.value,
      shipping_method: form.value.shippingMethodId,
      payment_method: form.value.paymentMethodKey,
      promo_code: coupon.value?.code ?? '',
    })
    if (seq === quoteSeq) quote.value = 'quote' in res ? res.quote : res
  } catch {
    if (seq === quoteSeq) quote.value = null
  } finally {
    if (seq === quoteSeq) quoting.value = false
  }
}
watch(
  () => [form.value.shippingMethodId, form.value.paymentMethodKey, coupon.value?.code, subtotal.value],
  () => void refreshQuote(),
)

const shippingFee = computed(() => quote.value?.shipping ?? null)
const paymentFee = computed(() => quote.value?.payment_fee ?? paymentMethod.value?.fee ?? 0)
const discountAmt = computed(() => quote.value?.discount ?? 0)
const total = computed(() =>
  quote.value
    ? quote.value.total
    : Math.max(0, subtotal.value - discountAmt.value) + (shippingFee.value ?? 0) + paymentFee.value,
)

const recipient = computed(() =>
  form.value.sameRecipient
    ? { name: form.value.name, phone: form.value.phone }
    : { name: form.value.recipientName, phone: form.value.recipientPhone },
)

// ── 欄位工具 ──
function setField<K extends keyof typeof form.value>(key: K, value: (typeof form.value)[K]) {
  form.value[key] = value
  if (errors.value[key]) {
    const next = { ...errors.value }
    delete next[key]
    errors.value = next
  }
}
function formatCardNumber(v: string) {
  return v.replace(/\D/g, '').slice(0, 16).replace(/(\d{4})(?=\d)/g, '$1 ')
}
function formatExpiry(v: string) {
  const d = v.replace(/\D/g, '').slice(0, 4)
  return d.length > 2 ? `${d.slice(0, 2)}/${d.slice(2)}` : d
}
function cardDisplayNumber(v: string) {
  const digits = v.replace(/\s/g, '')
  return Array.from({ length: 4 }, (_, i) => digits.slice(i * 4, i * 4 + 4).padEnd(4, '•')).join(' ')
}
const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms))

// ── 驗證（逐步累加） ──
function validateStep(target: number): boolean {
  const f = form.value
  const e: Record<string, string> = {}
  if (target >= 1) {
    if (!f.name.trim()) e.name = '請輸入姓名'
    if (!/^09\d{8}$/.test(f.phone.replace(/\s|-/g, ''))) e.phone = '請輸入 09 開頭的 10 位手機號碼'
    if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(f.email.trim())) e.email = '請輸入正確的 Email 格式'
    if (!f.sameRecipient) {
      if (!f.recipientName.trim()) e.recipientName = '請輸入收件人姓名'
      if (!/^09\d{8}$/.test(f.recipientPhone.replace(/\s|-/g, '')))
        e.recipientPhone = '請輸入 09 開頭的 10 位手機號碼'
    }
  }
  if (target >= 2) {
    if (!f.shippingMethodId) {
      toast.error('請選擇配送方式')
      return false
    }
    if (isCVS.value && !f.cvsStore) {
      toast.error('請選擇取貨門市')
      return false
    }
    if (!isCVS.value) {
      if (!f.city) e.city = '請選擇縣市'
      if (!f.district) e.district = '請選擇鄉鎮市區'
      if (f.addressDetail.trim().length < 5) e.addressDetail = '請輸入完整地址（至少 5 個字）'
    }
  }
  if (target >= 3) {
    if (!f.paymentMethodKey) {
      toast.error('請選擇付款方式')
      return false
    }
    if (isCard.value) {
      if (f.cardNumber.replace(/\s/g, '').length !== 16) e.cardNumber = '卡號需為 16 位數字'
      const [mm, yy] = f.cardExpiry.split('/')
      const now = new Date()
      const expOk =
        mm && yy && Number(mm) >= 1 && Number(mm) <= 12 &&
        Number(`20${yy}`) * 100 + Number(mm) >= now.getFullYear() * 100 + now.getMonth() + 1
      if (!expOk) e.cardExpiry = '請輸入有效的到期月年（MM/YY）'
      if (!/^\d{3,4}$/.test(f.cardCvv)) e.cardCvv = '安全碼需為 3 位數'
      if (!f.cardHolder.trim()) e.cardHolder = '請輸入持卡人姓名'
    }
  }
  if (target >= 4) {
    if (f.invoiceType === 'company') {
      if (!/^\d{8}$/.test(f.invoiceTaxId)) e.invoiceTaxId = '統一編號需為 8 位數字'
      if (!f.invoiceTitle.trim()) e.invoiceTitle = '請輸入發票抬頭'
    }
    if (!f.agree) {
      toast.error('請先閱讀並同意服務條款')
      return false
    }
  }
  errors.value = e
  if (Object.keys(e).length > 0) {
    toast.error('請檢查表單欄位', { description: '尚有欄位未正確填寫。' })
    return false
  }
  return true
}
function nextStep() {
  if (!validateStep(step.value + 1)) return
  step.value = Math.min(step.value + 1, 3)
  window.scrollTo({ top: 0, behavior: 'smooth' })
}
function prevStep() {
  step.value = Math.max(step.value - 1, 0)
  window.scrollTo({ top: 0, behavior: 'smooth' })
}
function jumpStep(i: number) {
  if (i < step.value) {
    step.value = i
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }
}

// ── 送出訂單 ──
async function placeOrder() {
  const f = form.value
  processing.value = true
  try {
    if (isCard.value || isLinePay.value) await sleep(1300) // 沙箱付款儀式感
    const shippingAddress = isCVS.value
      ? `${f.cvsStore!.name}（${f.cvsStore!.address}）`
      : `${f.zip ? `(${f.zip}) ` : ''}${f.city}${f.district}${f.addressDetail.trim()}`
    const res = await apiPost<{ order: OrderDTO }>('/api/orders', {
      customer_name: f.name.trim(),
      email: f.email.trim(),
      phone: f.phone.trim(),
      items: quoteItems.value,
      shipping_method: f.shippingMethodId,
      shipping_address: shippingAddress,
      payment_method: f.paymentMethodKey,
      promo_code: coupon.value?.code ?? '',
      idempotency_key: idempotencyKey,
      recipient_name: recipient.value.name.trim(),
      recipient_phone: recipient.value.phone.trim(),
      city: isCVS.value ? '' : f.city,
      district: isCVS.value ? '' : f.district,
      cvs_store_name: isCVS.value ? f.cvsStore!.name : '',
      cvs_store_address: isCVS.value ? f.cvsStore!.address : '',
      invoice_type: f.invoiceType,
      invoice_tax_id: f.invoiceType === 'company' ? f.invoiceTaxId : '',
      buyer_note: f.note.trim(),
    })
    const order = res.order
    if (order.access_token) rememberOrder(order.id, order.access_token)
    try {
      sessionStorage.setItem('curatory_last_order', JSON.stringify({ orderId: order.id, token: order.access_token ?? '' }))
    } catch { /* ignore */ }
    clearCart()
    checkoutStore.coupon = null
    toast.success('訂單已成立！', { description: `訂單編號 ${order.id}` })
    navigate(`/order/?id=${encodeURIComponent(order.id)}`)
  } catch (e) {
    toast.error('訂單建立失敗', { description: e instanceof Error ? e.message : '請確認資料後再試一次' })
  } finally {
    processing.value = false
  }
}
function submitOrder() {
  if (validateStep(4)) void placeOrder()
}

function paymentIcon(key: string) {
  const k = key.toLowerCase()
  if (/credit|card/.test(k)) return CreditCard
  if (/linepay|line/.test(k)) return QrCode
  return Banknote
}

onMounted(() => void loadBootstrap().catch(() => undefined))
</script>

<template>
  <!-- 空購物車 -->
  <div v-if="cart.items.length === 0 && !processing" class="mx-auto flex min-h-[60vh] w-full max-w-7xl flex-col items-center justify-center px-4 text-center">
    <div class="flex size-20 items-center justify-center rounded-full border bg-muted/40">
      <ShoppingCart class="size-9 text-muted-foreground/50" :stroke-width="1.2" />
    </div>
    <h1 class="mt-6 font-serif text-2xl font-semibold">購物車是空的</h1>
    <p class="mt-2 text-sm text-muted-foreground">先挑幾件喜歡的商品，再回來完成結帳。</p>
    <button type="button" class="mt-8 inline-flex h-12 items-center gap-2 rounded-full bg-primary px-8 text-base font-medium text-primary-foreground" @click="navigate('/shop/')">
      去逛逛<ArrowRight class="size-4" />
    </button>
  </div>

  <template v-else>
    <!-- 處理中 overlay -->
    <Transition name="fade">
      <div v-if="processing" class="fixed inset-0 z-[150] flex flex-col items-center justify-center gap-8 bg-ink/95 backdrop-blur-sm" role="alertdialog" aria-label="正在處理您的訂單">
        <LogoMark :size="52" class="animate-pulse-soft text-cream" />
        <p class="text-sm tracking-[0.3em] text-cream/70">正在處理您的訂單…</p>
      </div>
    </Transition>

    <div class="mx-auto w-full max-w-6xl px-4 py-10 sm:px-6 sm:py-14 lg:px-8">
      <!-- 頂部：Logo + 步驟指示器 -->
      <div class="flex flex-col items-center gap-6">
        <a href="/" aria-label="回首頁"><LogoMark :size="40" class="text-primary" /></a>
        <ol class="flex items-center" aria-label="結帳步驟">
          <li v-for="(s, i) in STEPS" :key="s.title" class="flex items-center">
            <span v-if="i > 0" :class="cn('h-px w-6 sm:w-14', i <= step ? 'bg-primary/60' : 'bg-border')" aria-hidden="true" />
            <button
              type="button"
              :disabled="i >= step"
              class="group flex flex-col items-center gap-1.5 disabled:cursor-default"
              :aria-current="i === step ? 'step' : undefined"
              @click="jumpStep(i)"
            >
              <span
                :class="cn(
                  'flex size-9 items-center justify-center rounded-full border transition-all sm:size-10',
                  i < step && 'border-primary bg-primary text-primary-foreground',
                  i === step && 'border-primary text-primary shadow-[0_0_0_4px] shadow-primary/10',
                  i > step && 'border-border text-muted-foreground',
                )"
              >
                <Check v-if="i < step" class="size-4" />
                <component :is="s.icon" v-else class="size-4" />
              </span>
              <span :class="cn('text-[11px] sm:text-xs', i === step ? 'font-medium text-foreground' : 'text-muted-foreground')">{{ s.title }}</span>
            </button>
          </li>
        </ol>
      </div>

      <div class="mt-10 grid gap-10 lg:grid-cols-[1fr_360px]">
        <!-- 左：步驟內容 -->
        <div class="min-w-0">
          <Transition name="step" mode="out-in">
            <div :key="step">
              <!-- Step 1 訂購資料 -->
              <section v-if="step === 0" class="space-y-6">
                <div>
                  <p class="eyebrow">Step 1</p>
                  <h2 class="mt-2 font-serif text-2xl font-semibold sm:text-3xl">訂購資料</h2>
                  <p class="mt-2 text-sm text-muted-foreground">填寫您的聯絡資訊，訂單確認與通知都會寄到這裡。</p>
                </div>
                <div class="space-y-5 rounded-xl border bg-card p-6">
                  <div class="grid gap-5 sm:grid-cols-2">
                    <div class="space-y-1.5">
                      <label class="text-sm" for="co-name">姓名<span class="ml-0.5 text-primary">*</span></label>
                      <input id="co-name" v-model="form.name" type="text" placeholder="例如：王小明" autocomplete="name"
                        :class="cn('h-11 w-full rounded-md border bg-transparent px-3 text-sm outline-none focus:border-primary', errors.name && 'border-destructive')"
                        @input="setField('name', form.name)" />
                      <p v-if="errors.name" class="text-xs text-destructive" role="alert">{{ errors.name }}</p>
                    </div>
                    <div class="space-y-1.5">
                      <label class="text-sm" for="co-phone">手機號碼<span class="ml-0.5 text-primary">*</span></label>
                      <input id="co-phone" :value="form.phone" type="text" inputmode="numeric" placeholder="0912345678" autocomplete="tel"
                        :class="cn('h-11 w-full rounded-md border bg-transparent px-3 text-sm outline-none focus:border-primary', errors.phone && 'border-destructive')"
                        @input="setField('phone', ($event.target as HTMLInputElement).value.replace(/\D/g, '').slice(0, 10))" />
                      <p v-if="errors.phone" class="text-xs text-destructive" role="alert">{{ errors.phone }}</p>
                      <p v-else class="text-xs text-muted-foreground/70">格式：09xx-xxx-xxx（僅數字）</p>
                    </div>
                  </div>
                  <div class="space-y-1.5">
                    <label class="text-sm" for="co-email">Email<span class="ml-0.5 text-primary">*</span></label>
                    <input id="co-email" v-model="form.email" type="email" placeholder="your@email.com" autocomplete="email"
                      :class="cn('h-11 w-full rounded-md border bg-transparent px-3 text-sm outline-none focus:border-primary', errors.email && 'border-destructive')"
                      @input="setField('email', form.email)" />
                    <p v-if="errors.email" class="text-xs text-destructive" role="alert">{{ errors.email }}</p>
                    <p v-else class="text-xs text-muted-foreground/70">訂單通知與電子發票將寄送至此信箱</p>
                  </div>

                  <div class="h-px bg-border"></div>

                  <div class="flex items-start gap-2.5">
                    <input id="same-recipient" type="checkbox" :checked="form.sameRecipient"
                      class="mt-0.5 size-4 accent-primary"
                      @change="setField('sameRecipient', ($event.target as HTMLInputElement).checked)" />
                    <label for="same-recipient" class="cursor-pointer text-sm font-normal leading-relaxed">
                      收件人同訂購人
                      <span class="block text-xs text-muted-foreground">取消勾選後，可另行填寫收件人姓名與電話</span>
                    </label>
                  </div>

                  <Transition name="step">
                    <div v-if="!form.sameRecipient" class="grid gap-5 rounded-xl border border-dashed p-5 sm:grid-cols-2">
                      <div class="space-y-1.5">
                        <label class="text-sm" for="co-rname">收件人姓名<span class="ml-0.5 text-primary">*</span></label>
                        <input id="co-rname" :value="form.recipientName" type="text" placeholder="收件人姓名"
                          :class="cn('h-11 w-full rounded-md border bg-transparent px-3 text-sm outline-none focus:border-primary', errors.recipientName && 'border-destructive')"
                          @input="setField('recipientName', ($event.target as HTMLInputElement).value)" />
                        <p v-if="errors.recipientName" class="text-xs text-destructive" role="alert">{{ errors.recipientName }}</p>
                      </div>
                      <div class="space-y-1.5">
                        <label class="text-sm" for="co-rphone">收件人手機<span class="ml-0.5 text-primary">*</span></label>
                        <input id="co-rphone" :value="form.recipientPhone" type="text" inputmode="numeric" placeholder="0912345678"
                          :class="cn('h-11 w-full rounded-md border bg-transparent px-3 text-sm outline-none focus:border-primary', errors.recipientPhone && 'border-destructive')"
                          @input="setField('recipientPhone', ($event.target as HTMLInputElement).value.replace(/\D/g, '').slice(0, 10))" />
                        <p v-if="errors.recipientPhone" class="text-xs text-destructive" role="alert">{{ errors.recipientPhone }}</p>
                      </div>
                    </div>
                  </Transition>

                  <div class="space-y-1.5">
                    <label class="text-sm" for="co-note">訂單備註</label>
                    <textarea id="co-note" :value="form.note" rows="3" maxlength="200" placeholder="想跟我們說的話…"
                      class="field-sizing-content w-full rounded-md border bg-transparent px-3 py-2.5 text-sm outline-none placeholder:text-muted-foreground focus:border-primary"
                      @input="setField('note', ($event.target as HTMLTextAreaElement).value)"></textarea>
                    <p class="text-xs text-muted-foreground/70">選填，例如包裝偏好或指定出貨日</p>
                  </div>
                </div>
              </section>

              <!-- Step 2 配送方式 -->
              <section v-else-if="step === 1" class="space-y-6">
                <div>
                  <p class="eyebrow">Step 2</p>
                  <h2 class="mt-2 font-serif text-2xl font-semibold sm:text-3xl">配送方式</h2>
                  <p class="mt-2 text-sm text-muted-foreground">選擇最方便的收貨方式——超商取貨或宅配到府。</p>
                </div>

                <div class="space-y-3" role="radiogroup" aria-label="配送方式">
                  <button
                    v-for="m in shippingMethods"
                    :key="m.id"
                    type="button"
                    role="radio"
                    :aria-checked="form.shippingMethodId === m.id"
                    :class="cn(
                      'w-full rounded-xl border p-4 text-left transition-all sm:p-5',
                      form.shippingMethodId === m.id
                        ? 'border-primary bg-primary/[0.04] shadow-[0_0_0_1px] shadow-primary'
                        : 'border-border hover:border-foreground/30 hover:bg-muted/30',
                    )"
                    @click="setField('shippingMethodId', m.id)"
                  >
                    <div class="flex items-start gap-4">
                      <span :class="cn('flex size-10 shrink-0 items-center justify-center rounded-full', form.shippingMethodId === m.id ? 'bg-primary/10 text-primary' : 'bg-muted text-muted-foreground')">
                        <Store v-if="/cvs|store|711|seven|family|fam/.test(m.id.toLowerCase())" class="size-5" />
                        <Truck v-else class="size-5" />
                      </span>
                      <div class="min-w-0 flex-1">
                        <p class="font-medium">{{ m.label }}</p>
                        <p v-if="m.description" class="mt-1 text-xs text-muted-foreground">{{ m.description }}</p>
                        <p class="mt-2 text-sm">
                          <template v-if="form.shippingMethodId === m.id && quote">
                            <span v-if="quote.shipping === 0" class="font-medium text-primary">免運費</span>
                            <span v-else class="tabular-nums">{{ formatNTD(quote.shipping) }}</span>
                          </template>
                          <span v-else class="text-xs text-muted-foreground">運費於下一步依訂單計算</span>
                        </p>
                      </div>
                      <span :class="cn('mt-1 flex size-4.5 shrink-0 items-center justify-center rounded-full border-2', form.shippingMethodId === m.id ? 'border-primary' : 'border-muted-foreground/35')">
                        <span v-if="form.shippingMethodId === m.id" class="size-2 rounded-full bg-primary" />
                      </span>
                    </div>
                  </button>
                  <p v-if="shippingMethods.length === 0" class="rounded-xl border border-dashed p-5 text-center text-sm text-muted-foreground">
                    配送方式設定中，請稍後再試。
                  </p>
                </div>

                <!-- 超商門市 -->
                <div v-if="form.shippingMethodId && isCVS">
                  <div class="mb-5 h-px bg-border"></div>
                  <div v-if="form.cvsStore" class="flex items-center justify-between gap-4 rounded-xl border border-primary/25 bg-primary/[0.03] p-4 sm:p-5">
                    <div class="flex min-w-0 items-start gap-3">
                      <Store class="mt-0.5 size-5 shrink-0 text-primary" />
                      <div class="min-w-0">
                        <p class="font-medium">{{ form.cvsStore.name }}</p>
                        <p class="mt-1 text-xs text-muted-foreground">{{ form.cvsStore.address }}</p>
                      </div>
                    </div>
                    <button type="button" class="h-8 shrink-0 rounded-full border px-4 text-xs" @click="storeDialogOpen = true">變更</button>
                  </div>
                  <button v-else type="button" class="flex w-full items-center justify-center gap-2 rounded-xl border border-dashed py-4 text-sm text-primary transition-colors hover:bg-primary/[0.04]" @click="storeDialogOpen = true">
                    <MapPin class="size-4" />選擇取貨門市
                  </button>
                </div>

                <!-- 宅配地址 -->
                <div v-if="form.shippingMethodId && !isCVS">
                  <div class="mb-5 h-px bg-border"></div>
                  <div class="space-y-5 rounded-xl border bg-card p-6">
                    <div class="grid gap-5 sm:grid-cols-2">
                      <div class="space-y-1.5">
                        <label class="text-sm" for="co-city">縣市<span class="ml-0.5 text-primary">*</span></label>
                        <select id="co-city" :value="form.city"
                          :class="cn('h-11 w-full rounded-md border bg-background px-3 text-sm outline-none focus:border-primary', errors.city && 'border-destructive')"
                          @change="setField('city', ($event.target as HTMLSelectElement).value); setField('district', ''); setField('zip', '')">
                          <option value="" disabled>選擇縣市</option>
                          <option v-for="r in TAIWAN_REGIONS" :key="r.city" :value="r.city">{{ r.city }}</option>
                        </select>
                        <p v-if="errors.city" class="text-xs text-destructive" role="alert">{{ errors.city }}</p>
                      </div>
                      <div class="space-y-1.5">
                        <label class="text-sm" for="co-district">鄉鎮市區<span class="ml-0.5 text-primary">*</span></label>
                        <select id="co-district" :value="form.district" :disabled="!form.city"
                          :class="cn('h-11 w-full rounded-md border bg-background px-3 text-sm outline-none focus:border-primary disabled:opacity-50', errors.district && 'border-destructive')"
                          @change="setField('district', ($event.target as HTMLSelectElement).value); setField('zip', districts.find((d) => d.name === form.district)?.zip ?? '')">
                          <option value="" disabled>{{ form.city ? '選擇鄉鎮市區' : '請先選擇縣市' }}</option>
                          <option v-for="d in districts" :key="d.name" :value="d.name">{{ d.name }}</option>
                        </select>
                        <p v-if="errors.district" class="text-xs text-destructive" role="alert">{{ errors.district }}</p>
                      </div>
                    </div>
                    <div class="space-y-1.5">
                      <label class="text-sm" for="co-addr">詳細地址<span class="ml-0.5 text-primary">*</span></label>
                      <input id="co-addr" :value="form.addressDetail" type="text" placeholder="例如：復興南路一段390號10樓之2" autocomplete="street-address"
                        :class="cn('h-11 w-full rounded-md border bg-transparent px-3 text-sm outline-none focus:border-primary', errors.addressDetail && 'border-destructive')"
                        @input="setField('addressDetail', ($event.target as HTMLInputElement).value)" />
                      <p v-if="errors.addressDetail" class="text-xs text-destructive" role="alert">{{ errors.addressDetail }}</p>
                      <p v-else class="text-xs text-muted-foreground/70">路街巷弄、樓層</p>
                    </div>
                    <div class="space-y-1.5 sm:max-w-40">
                      <label class="text-sm" for="co-zip">郵遞區號</label>
                      <input id="co-zip" :value="form.zip" type="text" inputmode="numeric" placeholder="100"
                        class="h-11 w-full rounded-md border bg-transparent px-3 text-sm outline-none focus:border-primary"
                        @input="setField('zip', ($event.target as HTMLInputElement).value.replace(/\D/g, '').slice(0, 5))" />
                      <p class="text-xs text-muted-foreground/70">選擇鄉鎮市區時自動帶入</p>
                    </div>
                  </div>
                </div>
              </section>

              <!-- Step 3 付款方式 -->
              <section v-else-if="step === 2" class="space-y-6">
                <div>
                  <p class="eyebrow">Step 3</p>
                  <h2 class="mt-2 font-serif text-2xl font-semibold sm:text-3xl">付款方式</h2>
                  <p class="mt-2 text-sm text-muted-foreground">選擇付款工具——本站為沙箱環境，不會實際扣款。</p>
                </div>

                <div class="space-y-3" role="radiogroup" aria-label="付款方式">
                  <button
                    v-for="m in paymentMethods"
                    :key="m.id"
                    type="button"
                    role="radio"
                    :aria-checked="form.paymentMethodKey === m.method"
                    :class="cn(
                      'w-full rounded-xl border p-4 text-left transition-all sm:p-5',
                      form.paymentMethodKey === m.method
                        ? 'border-primary bg-primary/[0.04] shadow-[0_0_0_1px] shadow-primary'
                        : 'border-border hover:border-foreground/30 hover:bg-muted/30',
                    )"
                    @click="setField('paymentMethodKey', m.method)"
                  >
                    <div class="flex items-start gap-4">
                      <span :class="cn('flex size-10 shrink-0 items-center justify-center rounded-full', form.paymentMethodKey === m.method ? 'bg-primary/10 text-primary' : 'bg-muted text-muted-foreground')">
                        <component :is="paymentIcon(m.method)" class="size-5" />
                      </span>
                      <div class="min-w-0 flex-1">
                        <div class="flex flex-wrap items-center gap-2">
                          <p class="font-medium">{{ m.label }}</p>
                          <span v-if="m.fee > 0" class="rounded-full bg-muted px-2 py-0.5 text-[10px] text-muted-foreground">手續費 {{ formatNTD(m.fee) }}</span>
                        </div>
                      </div>
                      <span :class="cn('mt-1 flex size-4.5 shrink-0 items-center justify-center rounded-full border-2', form.paymentMethodKey === m.method ? 'border-primary' : 'border-muted-foreground/35')">
                        <span v-if="form.paymentMethodKey === m.method" class="size-2 rounded-full bg-primary" />
                      </span>
                    </div>
                  </button>
                  <p v-if="paymentMethods.length === 0" class="rounded-xl border border-dashed p-5 text-center text-sm text-muted-foreground">
                    付款方式設定中，請稍後再試。
                  </p>
                </div>

                <!-- 信用卡表單（沙箱視覺，不傳輸） -->
                <div v-if="isCard" class="space-y-6">
                  <div class="h-px bg-border"></div>
                  <div class="relative mx-auto w-full max-w-sm">
                    <div class="relative aspect-[8/5] overflow-hidden rounded-2xl bg-gradient-to-br from-[#9a5a3c] via-[#6e3a26] to-[#3a1f16] p-6 text-cream shadow-xl">
                      <div aria-hidden="true" class="pointer-events-none absolute -right-10 -top-14 size-40 rounded-full bg-white/10" />
                      <div aria-hidden="true" class="pointer-events-none absolute -bottom-16 -left-8 size-44 rounded-full bg-black/15" />
                      <div class="relative flex items-start justify-between">
                        <LogoMark :size="30" class="text-cream/90" />
                        <CreditCard class="size-5 text-cream/70" />
                      </div>
                      <p class="relative mt-5 font-mono text-base tracking-[0.14em] sm:text-lg">{{ cardDisplayNumber(form.cardNumber) }}</p>
                      <div class="relative mt-4 flex items-end justify-between text-[10px] text-cream/60">
                        <div>
                          <p class="tracking-[0.2em]">VALID THRU</p>
                          <p class="mt-0.5 font-mono text-sm text-cream/90">{{ form.cardExpiry || 'MM/YY' }}</p>
                        </div>
                        <div>
                          <p class="tracking-[0.2em]">CVV</p>
                          <p class="mt-0.5 font-mono text-sm text-cream/90">{{ form.cardCvv ? '•'.repeat(form.cardCvv.length) : '•••' }}</p>
                        </div>
                      </div>
                    </div>
                  </div>
                  <div class="grid gap-5 rounded-xl border bg-card p-6 sm:grid-cols-2">
                    <div class="space-y-1.5 sm:col-span-2">
                      <label class="text-sm" for="cc-num">卡號<span class="ml-0.5 text-primary">*</span></label>
                      <input id="cc-num" :value="form.cardNumber" type="text" inputmode="numeric" placeholder="0000 0000 0000 0000" autocomplete="cc-number"
                        :class="cn('h-11 w-full rounded-md border bg-transparent px-3 font-mono text-sm tracking-widest outline-none focus:border-primary', errors.cardNumber && 'border-destructive')"
                        @input="setField('cardNumber', formatCardNumber(($event.target as HTMLInputElement).value))" />
                      <p v-if="errors.cardNumber" class="text-xs text-destructive" role="alert">{{ errors.cardNumber }}</p>
                      <p v-else class="text-xs text-muted-foreground/70">數字每 4 位一組，共 16 碼</p>
                    </div>
                    <div class="space-y-1.5">
                      <label class="text-sm" for="cc-exp">有效期限<span class="ml-0.5 text-primary">*</span></label>
                      <input id="cc-exp" :value="form.cardExpiry" type="text" inputmode="numeric" placeholder="12/27" autocomplete="cc-exp"
                        :class="cn('h-11 w-full rounded-md border bg-transparent px-3 font-mono text-sm outline-none focus:border-primary', errors.cardExpiry && 'border-destructive')"
                        @input="setField('cardExpiry', formatExpiry(($event.target as HTMLInputElement).value))" />
                      <p v-if="errors.cardExpiry" class="text-xs text-destructive" role="alert">{{ errors.cardExpiry }}</p>
                      <p v-else class="text-xs text-muted-foreground/70">格式 MM/YY</p>
                    </div>
                    <div class="space-y-1.5">
                      <label class="text-sm" for="cc-cvv">安全碼 CVV<span class="ml-0.5 text-primary">*</span></label>
                      <input id="cc-cvv" :value="form.cardCvv" type="text" inputmode="numeric" placeholder="123" autocomplete="cc-csc"
                        :class="cn('h-11 w-full rounded-md border bg-transparent px-3 font-mono text-sm outline-none focus:border-primary', errors.cardCvv && 'border-destructive')"
                        @input="setField('cardCvv', ($event.target as HTMLInputElement).value.replace(/\D/g, '').slice(0, 3))" />
                      <p v-if="errors.cardCvv" class="text-xs text-destructive" role="alert">{{ errors.cardCvv }}</p>
                      <p v-else class="text-xs text-muted-foreground/70">卡片背面 3 位數</p>
                    </div>
                    <div class="space-y-1.5 sm:col-span-2">
                      <label class="text-sm" for="cc-holder">持卡人姓名<span class="ml-0.5 text-primary">*</span></label>
                      <input id="cc-holder" :value="form.cardHolder" type="text" placeholder="WANG XIAO MING" autocomplete="cc-name"
                        :class="cn('h-11 w-full rounded-md border bg-transparent px-3 text-sm outline-none focus:border-primary', errors.cardHolder && 'border-destructive')"
                        @input="setField('cardHolder', ($event.target as HTMLInputElement).value)" />
                      <p v-if="errors.cardHolder" class="text-xs text-destructive" role="alert">{{ errors.cardHolder }}</p>
                      <p v-else class="text-xs text-muted-foreground/70">與卡片上姓名一致</p>
                    </div>
                  </div>
                  <p class="flex items-center justify-center gap-1.5 text-xs text-muted-foreground">
                    <ShieldCheck class="size-3.5 text-primary" />
                    沙箱環境示範用，不會實際扣款、也不會真實傳輸卡號
                  </p>
                </div>

                <!-- LINE Pay 說明 -->
                <div v-if="isLinePay" class="flex items-start gap-4 rounded-xl border border-[#06C755]/30 bg-[#06C755]/[0.05] p-5">
                  <span class="flex size-10 shrink-0 items-center justify-center rounded-full bg-[#06C755]/15 text-[#059669]">
                    <QrCode class="size-5" />
                  </span>
                  <div>
                    <p class="font-medium">以 LINE Pay 完成付款</p>
                    <p class="mt-1.5 text-sm leading-relaxed text-muted-foreground">送出訂單後，付款時將顯示 LINE Pay 模擬畫面（沙箱環境），模擬完成後訂單即轉為「已付款」。</p>
                  </div>
                </div>

                <!-- 貨到付款說明 -->
                <div v-if="isCOD" class="flex items-start gap-4 rounded-xl border p-5">
                  <span class="flex size-10 shrink-0 items-center justify-center rounded-full bg-muted text-muted-foreground">
                    <Banknote class="size-5" />
                  </span>
                  <div>
                    <p class="font-medium">貨到付款</p>
                    <p class="mt-1.5 text-sm leading-relaxed text-muted-foreground">
                      商品送達時，請以現金支付給物流人員。
                      <template v-if="(paymentMethod?.fee ?? 0) > 0">貨到付款將加收 {{ formatNTD(paymentMethod!.fee) }} 手續費，</template>請備妥足額現金。
                    </p>
                  </div>
                </div>
              </section>

              <!-- Step 4 確認訂單 -->
              <section v-else class="space-y-6">
                <div>
                  <p class="eyebrow">Step 4</p>
                  <h2 class="mt-2 font-serif text-2xl font-semibold sm:text-3xl">確認訂單</h2>
                  <p class="mt-2 text-sm text-muted-foreground">送出前，最後確認一次資訊。</p>
                </div>

                <div class="rounded-xl border bg-card p-6">
                  <div class="flex items-center justify-between">
                    <h3 class="font-serif text-base font-semibold">訂購資料</h3>
                    <button type="button" class="link-underline text-sm text-primary" @click="jumpStep(0)">編輯</button>
                  </div>
                  <div class="my-4 h-px bg-border"></div>
                  <div class="grid gap-x-8 gap-y-2 text-sm sm:grid-cols-2">
                    <div class="flex gap-3"><span class="w-20 shrink-0 text-muted-foreground">姓名</span><span class="min-w-0 flex-1 break-words">{{ form.name || '—' }}</span></div>
                    <div class="flex gap-3"><span class="w-20 shrink-0 text-muted-foreground">手機</span><span class="min-w-0 flex-1 break-words">{{ form.phone || '—' }}</span></div>
                    <div class="flex gap-3 sm:col-span-2"><span class="w-20 shrink-0 text-muted-foreground">Email</span><span class="min-w-0 flex-1 break-words">{{ form.email || '—' }}</span></div>
                    <div v-if="form.note" class="flex gap-3 sm:col-span-2"><span class="w-20 shrink-0 text-muted-foreground">備註</span><span class="min-w-0 flex-1 break-words">{{ form.note }}</span></div>
                  </div>
                </div>

                <div class="rounded-xl border bg-card p-6">
                  <div class="flex items-center justify-between">
                    <h3 class="font-serif text-base font-semibold">配送資訊</h3>
                    <button type="button" class="link-underline text-sm text-primary" @click="jumpStep(1)">編輯</button>
                  </div>
                  <div class="my-4 h-px bg-border"></div>
                  <div class="grid gap-x-8 gap-y-2 text-sm sm:grid-cols-2">
                    <div class="flex gap-3"><span class="w-20 shrink-0 text-muted-foreground">配送方式</span><span class="min-w-0 flex-1 break-words">{{ shippingMethod?.label ?? '—' }}</span></div>
                    <div class="flex gap-3"><span class="w-20 shrink-0 text-muted-foreground">運費</span><span class="min-w-0 flex-1 break-words">{{ shippingFee == null ? '—' : shippingFee === 0 ? '免運' : formatNTD(shippingFee) }}</span></div>
                    <template v-if="form.cvsStore">
                      <div class="flex gap-3 sm:col-span-2"><span class="w-20 shrink-0 text-muted-foreground">取貨門市</span><span class="min-w-0 flex-1 break-words">{{ form.cvsStore.name }}</span></div>
                      <div class="flex gap-3 sm:col-span-2"><span class="w-20 shrink-0 text-muted-foreground">門市地址</span><span class="min-w-0 flex-1 break-words">{{ form.cvsStore.address }}</span></div>
                    </template>
                    <div v-else class="flex gap-3 sm:col-span-2"><span class="w-20 shrink-0 text-muted-foreground">收件地址</span><span class="min-w-0 flex-1 break-words">{{ `${form.zip ? `(${form.zip}) ` : ''}${form.city}${form.district}${form.addressDetail}` || '—' }}</span></div>
                    <div class="flex gap-3 sm:col-span-2"><span class="w-20 shrink-0 text-muted-foreground">收件人</span><span class="min-w-0 flex-1 break-words">{{ recipient.name || '—' }}／{{ recipient.phone || '—' }}</span></div>
                  </div>
                </div>

                <div class="rounded-xl border bg-card p-6">
                  <div class="flex items-center justify-between">
                    <h3 class="font-serif text-base font-semibold">付款資訊</h3>
                    <button type="button" class="link-underline text-sm text-primary" @click="jumpStep(2)">編輯</button>
                  </div>
                  <div class="my-4 h-px bg-border"></div>
                  <div class="grid gap-x-8 gap-y-2 text-sm sm:grid-cols-2">
                    <div class="flex gap-3"><span class="w-20 shrink-0 text-muted-foreground">付款方式</span><span class="min-w-0 flex-1 break-words">{{ paymentMethod?.label ?? '—' }}</span></div>
                    <div class="flex gap-3"><span class="w-20 shrink-0 text-muted-foreground">手續費</span><span class="min-w-0 flex-1 break-words">{{ paymentMethod ? (paymentFee > 0 ? formatNTD(paymentFee) : '免手續費') : '—' }}</span></div>
                    <div v-if="isCard && form.cardNumber" class="flex gap-3 sm:col-span-2"><span class="w-20 shrink-0 text-muted-foreground">卡片</span><span class="min-w-0 flex-1 break-words font-mono">•••• •••• •••• {{ form.cardNumber.replace(/\s/g, '').slice(-4) }}</span></div>
                  </div>
                </div>

                <!-- 發票 -->
                <div class="rounded-xl border bg-card p-6">
                  <h3 class="font-serif text-base font-semibold">發票開立</h3>
                  <div class="mt-4 grid gap-3 sm:grid-cols-2" role="radiogroup" aria-label="發票類型">
                    <label :class="cn('flex cursor-pointer items-start gap-3 rounded-xl border p-4 transition-colors', form.invoiceType === 'personal' ? 'border-primary bg-primary/[0.03]' : 'hover:bg-muted/30')">
                      <input type="radio" name="invoice-type" value="personal" :checked="form.invoiceType === 'personal'" class="mt-1 size-3.5 accent-primary" @change="setField('invoiceType', 'personal')" />
                      <span>
                        <span class="block text-sm font-medium">個人二聯式電子發票</span>
                        <span class="mt-1 block text-xs text-muted-foreground">發票將寄送至您的 Email</span>
                      </span>
                    </label>
                    <label :class="cn('flex cursor-pointer items-start gap-3 rounded-xl border p-4 transition-colors', form.invoiceType === 'company' ? 'border-primary bg-primary/[0.03]' : 'hover:bg-muted/30')">
                      <input type="radio" name="invoice-type" value="company" :checked="form.invoiceType === 'company'" class="mt-1 size-3.5 accent-primary" @change="setField('invoiceType', 'company')" />
                      <span>
                        <span class="block text-sm font-medium">公司三聯式電子發票</span>
                        <span class="mt-1 block text-xs text-muted-foreground">需填寫統一編號與發票抬頭</span>
                      </span>
                    </label>
                  </div>

                  <Transition name="step">
                    <div v-if="form.invoiceType === 'company'" class="mt-4 grid gap-5 rounded-xl border border-dashed p-4 sm:grid-cols-2">
                      <div class="space-y-1.5">
                        <label class="text-sm" for="inv-taxid">統一編號<span class="ml-0.5 text-primary">*</span></label>
                        <input id="inv-taxid" :value="form.invoiceTaxId" type="text" inputmode="numeric" placeholder="12345678"
                          :class="cn('h-11 w-full rounded-md border bg-transparent px-3 text-sm outline-none focus:border-primary', errors.invoiceTaxId && 'border-destructive')"
                          @input="setField('invoiceTaxId', ($event.target as HTMLInputElement).value.replace(/\D/g, '').slice(0, 8))" />
                        <p v-if="errors.invoiceTaxId" class="text-xs text-destructive" role="alert">{{ errors.invoiceTaxId }}</p>
                        <p v-else class="text-xs text-muted-foreground/70">8 位數字</p>
                      </div>
                      <div class="space-y-1.5">
                        <label class="text-sm" for="inv-title">發票抬頭<span class="ml-0.5 text-primary">*</span></label>
                        <input id="inv-title" :value="form.invoiceTitle" type="text" placeholder="公司全名"
                          :class="cn('h-11 w-full rounded-md border bg-transparent px-3 text-sm outline-none focus:border-primary', errors.invoiceTitle && 'border-destructive')"
                          @input="setField('invoiceTitle', ($event.target as HTMLInputElement).value)" />
                        <p v-if="errors.invoiceTitle" class="text-xs text-destructive" role="alert">{{ errors.invoiceTitle }}</p>
                      </div>
                    </div>
                  </Transition>
                </div>

                <!-- 同意條款 -->
                <div class="flex items-start gap-3 rounded-xl border bg-muted/30 p-5">
                  <input id="agree" type="checkbox" :checked="form.agree" class="mt-0.5 size-4 accent-primary" @change="setField('agree', ($event.target as HTMLInputElement).checked)" />
                  <label for="agree" class="cursor-pointer text-sm font-normal leading-relaxed">
                    我已閱讀並同意<span class="mx-1 text-primary link-underline">服務條款</span>與<span class="mx-1 text-primary link-underline">退換貨政策</span>，並確認以上訂單資訊無誤。
                  </label>
                </div>
              </section>
            </div>
          </Transition>

          <!-- 步驟導覽 -->
          <div class="mt-10 flex items-center justify-between gap-4">
            <button v-if="step > 0" type="button" :disabled="processing" class="inline-flex h-12 items-center gap-2 rounded-full border px-7 text-sm transition-colors hover:bg-muted disabled:opacity-50" @click="prevStep">
              <ArrowLeft class="size-4" />上一步
            </button>
            <button v-else type="button" :disabled="processing" class="inline-flex h-12 items-center gap-2 rounded-full px-5 text-sm text-muted-foreground transition-colors hover:text-foreground disabled:opacity-50" @click="navigate('/cart/')">
              <ArrowLeft class="size-4" />回購物車
            </button>

            <button v-if="step < 3" type="button" :disabled="processing" class="inline-flex h-12 items-center gap-2 rounded-full bg-primary px-9 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:opacity-50" @click="nextStep">
              下一步<ArrowRight class="size-4" />
            </button>
            <button v-else type="button" :disabled="processing" class="inline-flex h-12 items-center gap-2 rounded-full bg-primary px-9 text-base font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:opacity-50" @click="submitOrder">
              <Loader2 v-if="processing" class="size-4 animate-spin" />
              <Check v-else class="size-4" />
              {{ processing ? '處理中…' : `完成訂單（${formatNTD(total)}）` }}
            </button>
          </div>
        </div>

        <!-- 右：訂單摘要 -->
        <aside class="h-fit lg:sticky lg:top-24">
          <div class="rounded-xl border bg-card p-6">
            <h2 class="font-serif text-lg font-semibold tracking-wide">訂單摘要</h2>
            <ul class="scrollbar-thin mt-4 max-h-64 space-y-3 overflow-y-auto pr-1">
              <li v-for="item in cart.items" :key="item.key" class="flex items-center gap-3">
                <div class="relative size-12 shrink-0 overflow-hidden rounded-lg bg-muted/40">
                  <FadeImage :src="item.image" :alt="item.name" class="h-full w-full object-cover" />
                  <span class="absolute -right-1.5 -top-1.5 flex size-4.5 items-center justify-center rounded-full bg-ink text-[10px] font-medium text-cream">{{ item.qty }}</span>
                </div>
                <div class="min-w-0 flex-1">
                  <p class="clamp-1 text-xs font-medium">{{ item.name }}</p>
                  <p v-if="item.variantName" class="text-[10px] text-muted-foreground">{{ item.variantName }}</p>
                </div>
                <span class="shrink-0 text-xs tabular-nums text-muted-foreground">{{ formatNTD(item.unitPrice * item.qty) }}</span>
              </li>
            </ul>

            <div class="my-4 h-px bg-border"></div>

            <div class="space-y-2.5 text-sm">
              <div class="flex justify-between"><span class="text-muted-foreground">小計（{{ count }} 件）</span><span class="tabular-nums">{{ formatNTD(subtotal) }}</span></div>
              <div v-if="coupon" class="flex justify-between text-primary">
                <span>優惠券 {{ coupon.code }}</span>
                <span class="tabular-nums">{{ quote ? `-${formatNTD(discountAmt)}` : '已套用' }}</span>
              </div>
              <div class="flex justify-between">
                <span class="text-muted-foreground">運費</span>
                <span :class="cn('tabular-nums', !shippingMethod && 'text-muted-foreground/60')">
                  <Loader2 v-if="quoting && form.shippingMethodId" class="inline size-3 animate-spin" />
                  <template v-else>{{ shippingMethod ? (shippingFee === 0 ? '免運' : shippingFee == null ? '計算中…' : formatNTD(shippingFee)) : '尚未選擇' }}</template>
                </span>
              </div>
              <div class="flex justify-between">
                <span class="text-muted-foreground">{{ paymentMethod ? `${paymentMethod.label}${paymentFee > 0 ? '手續費' : ''}` : '付款手續費' }}</span>
                <span :class="cn('tabular-nums', !paymentMethod && 'text-muted-foreground/60')">{{ paymentMethod ? (paymentFee > 0 ? formatNTD(paymentFee) : 'NT$0') : '尚未選擇' }}</span>
              </div>
            </div>

            <div class="my-4 h-px bg-border"></div>

            <div class="flex items-baseline justify-between">
              <span class="text-sm text-muted-foreground">應付總額</span>
              <span class="font-serif text-2xl font-semibold tabular-nums">{{ formatNTD(total) }}</span>
            </div>

            <p class="mt-4 flex items-center gap-1.5 text-[11px] text-muted-foreground/70">
              <ShieldCheck class="size-3.5 text-primary" />
              沙箱環境・不會實際扣款，資料僅用於示範
            </p>
          </div>
        </aside>
      </div>
    </div>

    <!-- 門市選擇 Dialog -->
    <Dialog :open="storeDialogOpen" panel-class="max-w-md gap-0 p-0" @update:open="storeDialogOpen = $event">
      <div class="px-5 pt-5">
        <h2 class="font-serif text-lg font-semibold">選擇取貨門市</h2>
        <p class="mt-1 text-sm text-muted-foreground">選擇縣市後挑選門市，到貨時將以簡訊通知</p>
      </div>
      <div class="grid grid-cols-2 gap-2.5 px-5 pt-4">
        <select v-model="storeCity" aria-label="選擇縣市" class="h-10 w-full rounded-md border bg-background px-3 text-sm outline-none focus:border-primary" @change="storeSearch = ''">
          <option v-for="r in TAIWAN_REGIONS" :key="r.city" :value="r.city">{{ r.city }}</option>
        </select>
        <div class="relative">
          <Search class="pointer-events-none absolute left-2.5 top-1/2 size-3.5 -translate-y-1/2 text-muted-foreground" />
          <input v-model="storeSearch" type="text" placeholder="搜尋門市／地址" aria-label="搜尋門市" class="h-10 w-full rounded-md border bg-background pl-8 pr-3 text-sm outline-none focus:border-primary" />
        </div>
      </div>
      <div class="scrollbar-thin mx-5 mt-3 max-h-72 overflow-y-auto rounded-lg border">
        <div class="divide-y">
          <p v-if="filteredStores.length === 0" class="py-10 text-center text-sm text-muted-foreground">找不到相符的門市，試試其他搜尋條件</p>
          <button
            v-for="s in filteredStores"
            :key="s.id"
            type="button"
            class="flex w-full items-start gap-3 px-4 py-3 text-left transition-colors hover:bg-muted/50"
            @click="setField('cvsStore', { name: s.name, address: s.address }); storeDialogOpen = false; toast.success('已選擇門市', { description: s.name })"
          >
            <Store class="mt-0.5 size-4 shrink-0 text-primary" />
            <span class="min-w-0">
              <span class="block text-sm font-medium">{{ s.name }}</span>
              <span class="mt-0.5 block text-xs text-muted-foreground">{{ s.address }}</span>
            </span>
          </button>
        </div>
      </div>
      <div class="h-5"></div>
    </Dialog>
  </template>
</template>

<style scoped>
.step-enter-active,
.step-leave-active {
  transition: opacity 0.3s cubic-bezier(0.22, 1, 0.36, 1), transform 0.3s cubic-bezier(0.22, 1, 0.36, 1);
}
.step-enter-from {
  opacity: 0;
  transform: translateX(28px);
}
.step-leave-to {
  opacity: 0;
  transform: translateX(-28px);
}
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.25s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
@media (prefers-reduced-motion: reduce) {
  .step-enter-active,
  .step-leave-active,
  .fade-enter-active,
  .fade-leave-active {
    transition: none;
  }
}
</style>
