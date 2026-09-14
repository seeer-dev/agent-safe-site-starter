<script setup lang="ts">
// 訂單詳情卡 — 對應 reference order-detail-card.tsx，OrderPage/TrackPage 共用。
// 欄位對齊本站台 OrderDTO（小寫 snake_case、timeline: {status, at, note}）；
// 配送/付款方式名稱由 bootstrap 方法表以 method key 反查 label。
import { computed, ref } from 'vue'
import {
  ArrowRight, Banknote, Check, Copy, Mail, MapPin, Package,
  Receipt, Store, Truck, XCircle,
} from 'lucide-vue-next'
import { formatDateTime, formatNTD } from '@/shared/lib/format'
import { bootstrap } from '@/shared/lib/store'
import { navigate } from '@/shared/lib/transition'
import { toast } from '@/shared/lib/toast'
import { cn } from '@/shared/lib/utils'
import { ORDER_STATUS_LABEL, PAYMENT_STATUS_LABEL, type OrderDTO } from '@/shared/lib/types'

const props = defineProps<{ order: OrderDTO }>()
const order = computed(() => props.order)
const copied = ref(false)

const STATUS_FLOW = ['pending', 'processing', 'shipped', 'delivered', 'completed'] as const
const SHORT_STATUS: Record<string, string> = {
  pending: '訂單成立',
  processing: '處理中',
  shipped: '已出貨',
  delivered: '已送達',
  completed: '已完成',
}

const cancelled = computed(() => order.value.status === 'cancelled')
const currentIdx = computed(() => (cancelled.value ? -1 : STATUS_FLOW.indexOf(order.value.status as (typeof STATUS_FLOW)[number])))
// 貨到付款未付款但已出貨 → 「處理中」步驟改顯示「貨到付款」
const codPending = computed(
  () => /cod|cash/i.test(order.value.payment_method) && order.value.payment_status === 'unpaid' && currentIdx.value >= 2,
)

const shippingLabel = computed(
  () => bootstrap.data?.shipping_methods.find((m) => m.id === order.value.shipping_method)?.label ?? order.value.shipping_method,
)
const paymentLabel = computed(
  () => bootstrap.data?.payment_methods.find((m) => m.method === order.value.payment_method)?.label ?? order.value.payment_method,
)

const events = computed(() => [...(order.value.timeline ?? [])].sort((a, b) => b.at - a.at))
const cancelledAt = computed(() => events.value.find((e) => e.status === 'cancelled')?.at)
const paidAt = computed(() => events.value.find((e) => e.status === 'paid')?.at)

async function copyOrderNumber() {
  try {
    await navigator.clipboard.writeText(order.value.id)
    copied.value = true
    toast.success('已複製訂單編號', { description: order.value.id })
    setTimeout(() => (copied.value = false), 2000)
  } catch {
    toast.error('複製失敗，請手動選取複製')
  }
}
function eventLabel(status: string, note?: string): string {
  if (note) return note
  return ORDER_STATUS_LABEL[status] ?? status
}
</script>

<template>
  <div class="space-y-5">
    <!-- 訂單號 + 狀態 -->
    <div class="rounded-xl border bg-card p-6">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div>
          <p class="text-xs tracking-wider text-muted-foreground">訂單編號</p>
          <div class="mt-1.5 flex items-center gap-3">
            <span class="break-all font-mono text-xl font-semibold tracking-wider sm:text-2xl">{{ order.id }}</span>
            <button
              type="button"
              aria-label="複製訂單編號"
              class="shrink-0 rounded-full border p-1.5 text-muted-foreground transition-colors hover:border-primary hover:text-primary"
              @click="copyOrderNumber"
            >
              <Check v-if="copied" class="size-3.5 text-primary" />
              <Copy v-else class="size-3.5" />
            </button>
          </div>
        </div>
        <span
          :class="cn(
            'rounded-full px-3.5 py-1.5 text-xs font-medium',
            cancelled ? 'bg-destructive/10 text-destructive' : 'bg-primary/10 text-primary',
          )"
        >{{ ORDER_STATUS_LABEL[order.status] ?? order.status }}</span>
      </div>

      <!-- 狀態 stepper -->
      <div v-if="cancelled" class="mt-5 flex items-center gap-3 rounded-xl border border-destructive/30 bg-destructive/[0.05] p-4">
        <XCircle class="size-5 shrink-0 text-destructive" />
        <div>
          <p class="text-sm font-medium text-destructive">此訂單已取消</p>
          <p class="mt-0.5 text-xs text-muted-foreground">
            {{ cancelledAt ? `取消時間：${formatDateTime(cancelledAt * 1000)}` : '如有疑問請與我們聯繫' }}
            已付款項將於 3-5 個工作天內退回。
          </p>
        </div>
      </div>
      <ol v-else class="mt-6 flex items-start" aria-label="訂單進度">
        <li v-for="(s, i) in STATUS_FLOW" :key="s" class="flex flex-1 flex-col items-center">
          <div class="flex w-full items-center">
            <span :class="cn('h-px flex-1', i === 0 ? 'bg-transparent' : i <= currentIdx ? 'bg-primary/50' : 'bg-border')" aria-hidden="true" />
            <span
              :class="cn(
                'flex size-7 shrink-0 items-center justify-center rounded-full border text-[10px] transition-colors sm:size-8',
                i <= currentIdx
                  ? codPending && s === 'processing'
                    ? 'border-primary/40 bg-primary/10 text-primary/70'
                    : 'border-primary bg-primary text-primary-foreground'
                  : 'border-border text-muted-foreground',
              )"
              :title="SHORT_STATUS[s]"
            >
              <Check v-if="i <= currentIdx && !(codPending && s === 'processing')" class="size-3.5" />
              <template v-else>{{ i + 1 }}</template>
            </span>
            <span :class="cn('h-px flex-1', i === STATUS_FLOW.length - 1 ? 'bg-transparent' : i < currentIdx ? 'bg-primary/50' : 'bg-border')" aria-hidden="true" />
          </div>
          <span :class="cn('mt-2 text-center text-[10px] leading-tight sm:text-xs', i === currentIdx ? 'font-medium text-foreground' : i <= currentIdx ? 'text-foreground/70' : 'text-muted-foreground')">
            {{ codPending && s === 'processing' ? '貨到付款' : SHORT_STATUS[s] }}
          </span>
        </li>
      </ol>
    </div>

    <!-- 商品明細 + 金額 -->
    <div class="rounded-xl border bg-card p-6">
      <h3 class="flex items-center gap-2 font-serif text-base font-semibold">
        <Package class="size-4 text-primary" />商品明細
      </h3>
      <ul class="mt-4 space-y-4">
        <li v-for="item in order.items" :key="item.sku + (item.variant_name ?? '')" class="flex items-center gap-4">
          <div class="flex size-16 shrink-0 items-center justify-center overflow-hidden rounded-lg bg-muted/40 text-muted-foreground/40">
            <Package class="size-5" />
          </div>
          <div class="min-w-0 flex-1">
            <p class="clamp-1 text-sm font-medium">{{ item.name }}</p>
            <p class="mt-0.5 text-xs text-muted-foreground">
              {{ item.variant_name ? `${item.variant_name}・` : '' }}{{ formatNTD(item.price) }} × {{ item.quantity }}
            </p>
          </div>
          <span class="shrink-0 text-sm font-medium tabular-nums">{{ formatNTD(item.price * item.quantity) }}</span>
        </li>
      </ul>

      <div class="my-5 h-px bg-border"></div>

      <div class="space-y-2.5 text-sm">
        <div class="flex items-baseline justify-between gap-4"><span class="text-muted-foreground">小計</span><span class="tabular-nums">{{ formatNTD(order.subtotal) }}</span></div>
        <div v-if="order.discount > 0" class="flex items-baseline justify-between gap-4 text-primary">
          <span>優惠券 {{ order.coupon_code }}</span><span class="tabular-nums">-{{ formatNTD(order.discount) }}</span>
        </div>
        <div class="flex items-baseline justify-between gap-4"><span class="text-muted-foreground">運費（{{ shippingLabel }}）</span><span class="tabular-nums">{{ order.shipping === 0 ? '免運' : formatNTD(order.shipping) }}</span></div>
        <div v-if="order.payment_fee > 0" class="flex items-baseline justify-between gap-4"><span class="text-muted-foreground">{{ paymentLabel }}手續費</span><span class="tabular-nums">{{ formatNTD(order.payment_fee) }}</span></div>
        <div class="h-px bg-border"></div>
        <div class="flex items-baseline justify-between">
          <span class="text-sm text-muted-foreground">總計</span>
          <span class="font-serif text-2xl font-semibold tabular-nums">{{ formatNTD(order.total) }}</span>
        </div>
      </div>
    </div>

    <!-- 配送 / 付款資訊 -->
    <div class="grid gap-5 sm:grid-cols-2">
      <div class="rounded-xl border bg-card p-6">
        <h3 class="flex items-center gap-2 font-serif text-base font-semibold">
          <Truck class="size-4 text-primary" />配送資訊
        </h3>
        <div class="my-4 h-px bg-border"></div>
        <div class="space-y-2.5 text-sm">
          <div class="flex items-baseline justify-between gap-4"><span class="shrink-0 text-muted-foreground">配送方式</span><span class="min-w-0 break-words text-right">{{ shippingLabel }}</span></div>
          <div v-if="order.cvs_store_name" class="flex items-start gap-3">
            <Store class="mt-0.5 size-4 shrink-0 text-muted-foreground" />
            <div class="min-w-0">
              <p class="font-medium">{{ order.cvs_store_name }}</p>
              <p v-if="order.cvs_store_address" class="mt-0.5 text-xs text-muted-foreground">{{ order.cvs_store_address }}</p>
            </div>
          </div>
          <div v-else-if="order.city || order.shipping_address" class="flex items-start gap-3">
            <MapPin class="mt-0.5 size-4 shrink-0 text-muted-foreground" />
            <p class="min-w-0 break-words">{{ order.city || order.district ? `${order.city}${order.district}` : '' }}{{ order.shipping_address }}</p>
          </div>
          <div class="flex items-baseline justify-between gap-4"><span class="shrink-0 text-muted-foreground">收件人</span><span class="min-w-0 break-words text-right">{{ order.recipient_name }}／{{ order.recipient_phone }}</span></div>
          <div v-if="order.tracking_number" class="flex items-baseline justify-between gap-4"><span class="shrink-0 text-muted-foreground">物流追蹤</span><span class="min-w-0 break-words text-right font-mono">{{ order.tracking_number }}</span></div>
        </div>
      </div>

      <div class="rounded-xl border bg-card p-6">
        <h3 class="flex items-center gap-2 font-serif text-base font-semibold">
          <Receipt class="size-4 text-primary" />付款資訊
        </h3>
        <div class="my-4 h-px bg-border"></div>
        <div class="space-y-2.5 text-sm">
          <div class="flex items-baseline justify-between gap-4"><span class="shrink-0 text-muted-foreground">付款方式</span><span class="min-w-0 break-words text-right">{{ paymentLabel }}</span></div>
          <div class="flex items-baseline justify-between gap-4">
            <span class="shrink-0 text-muted-foreground">付款狀態</span>
            <span
              :class="cn(
                'rounded-full border px-2.5 py-0.5 text-[11px]',
                order.payment_status === 'paid' ? 'border-primary/30 text-primary'
                  : order.payment_status === 'refunded' ? 'border-destructive/30 text-destructive'
                    : 'border-border text-muted-foreground',
              )"
            >{{ PAYMENT_STATUS_LABEL[order.payment_status] ?? order.payment_status }}</span>
          </div>
          <div v-if="paidAt" class="flex items-baseline justify-between gap-4"><span class="shrink-0 text-muted-foreground">付款時間</span><span class="tabular-nums">{{ formatDateTime(paidAt * 1000) }}</span></div>
          <div v-if="order.invoice_type" class="flex items-baseline justify-between gap-4">
            <span class="shrink-0 text-muted-foreground">發票</span>
            <span class="min-w-0 break-words text-right">
              {{ order.invoice_type === 'company' ? `公司三聯式${order.invoice_tax_id ? `（統編 ${order.invoice_tax_id}）` : ''}` : '個人二聯式電子發票' }}
            </span>
          </div>
        </div>
      </div>
    </div>

    <!-- 時間軸 -->
    <div v-if="events.length > 0" class="rounded-xl border bg-card p-6">
      <h3 class="flex items-center gap-2 font-serif text-base font-semibold">
        <Banknote class="size-4 text-primary" />訂單時間軸
      </h3>
      <div class="my-4 h-px bg-border"></div>
      <ol class="relative space-y-5 border-l pl-6">
        <li v-for="(ev, i) in events" :key="i" class="relative">
          <span class="absolute -left-[31px] top-1 flex size-2.5 items-center justify-center rounded-full bg-primary" aria-hidden="true" />
          <p class="text-sm">{{ eventLabel(ev.status, ev.note) }}</p>
          <p class="mt-0.5 text-xs text-muted-foreground">{{ formatDateTime(ev.at * 1000) }}</p>
        </li>
      </ol>
    </div>

    <!-- 注意事項 -->
    <div class="rounded-xl border border-dashed bg-muted/30 p-6">
      <h3 class="flex items-center gap-2 font-serif text-base font-semibold">
        <Mail class="size-4 text-primary" />溫馨提醒
      </h3>
      <ul class="mt-3 space-y-2 text-sm leading-relaxed text-muted-foreground">
        <li>・訂單狀態變更時，我們會寄送通知至 {{ order.email }}。</li>
        <li>・超商取貨到貨後，請於 7 天內持通知簡訊取件，逾期將退回物流中心。</li>
        <li>・如對訂單有任何疑問，歡迎來信或使用「訂單查詢」隨時追蹤進度。</li>
      </ul>
    </div>

    <!-- CTA -->
    <div class="flex flex-wrap justify-center gap-3 pt-2">
      <button type="button" class="inline-flex h-11 items-center rounded-full border px-7 text-sm transition-colors hover:bg-muted" @click="navigate('/track/')">追蹤訂單</button>
      <button type="button" class="inline-flex h-11 items-center gap-2 rounded-full bg-primary px-7 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90" @click="navigate('/shop/')">
        繼續逛逛<ArrowRight class="size-4" />
      </button>
    </div>
  </div>
</template>
