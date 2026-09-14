<script setup lang="ts">
// 訂單查詢頁 — 對應 reference track-view.tsx。
// 差異：本站採 access-token 查單（GET /api/orders/{id} +
// X-Order-Access-Token），表單為「訂單編號＋查詢碼」；本機下過單的裝置
// 會顯示「最近訂單」chips，點擊即自動帶入憑證查詢。
import { onMounted, ref } from 'vue'
import { History, Loader2, PackageSearch, Search } from 'lucide-vue-next'
import OrderDetailCard from '@/shared/components/OrderDetailCard.vue'
import { apiGet } from '@/shared/lib/api'
import { formatDate } from '@/shared/lib/format'
import { bootstrap, loadBootstrap, loadRecentOrders, type RecentOrder } from '@/shared/lib/store'
import { navigate } from '@/shared/lib/transition'
import { toast } from '@/shared/lib/toast'
import { cn } from '@/shared/lib/utils'
import type { OrderDTO } from '@/shared/lib/types'

const orderId = ref('')
const token = ref('')
const errors = ref<{ orderId?: string; token?: string }>({})
const recent = ref<RecentOrder[]>([])
const order = ref<OrderDTO | null>(null)
const pending = ref(false)
const attempted = ref(false)

onMounted(() => {
  void loadBootstrap().catch(() => undefined)
  recent.value = loadRecentOrders()
  // 若 URL 帶 id（例如從訂單頁連回來），自動用已知憑證查
  try {
    const q = new URLSearchParams(window.location.search).get('id')
    if (q) {
      orderId.value = q
      const hit = recent.value.find((r) => r.orderId === q)
      if (hit) void lookup(q, hit.token)
    }
  } catch { /* ignore */ }
})

async function lookup(id: string, tk: string) {
  pending.value = true
  attempted.value = true
  try {
    const res = await apiGet<{ order: OrderDTO } | OrderDTO>(`/api/orders/${encodeURIComponent(id)}`, {
      'X-Order-Access-Token': tk,
    })
    order.value = 'order' in res ? res.order : res
  } catch (e) {
    order.value = null
    toast.error('查無此訂單', { description: e instanceof Error ? e.message : '請確認訂單編號與查詢碼是否正確。' })
  } finally {
    pending.value = false
  }
}

function submit() {
  const e: typeof errors.value = {}
  if (!orderId.value.trim()) e.orderId = '請輸入訂單編號'
  if (!token.value.trim()) e.token = '請輸入訂單查詢碼'
  errors.value = e
  if (Object.keys(e).length > 0) return
  void lookup(orderId.value.trim(), token.value.trim())
}

function applyRecent(r: RecentOrder) {
  orderId.value = r.orderId
  token.value = r.token
  errors.value = {}
  void lookup(r.orderId, r.token)
}
</script>

<template>
  <div class="mx-auto w-full max-w-3xl px-4 py-10 sm:px-6 sm:py-14 lg:px-8">
    <header class="text-center">
      <p class="eyebrow">Order Tracking</p>
      <h1 class="mt-3 font-serif text-3xl font-semibold sm:text-4xl">訂單查詢</h1>
      <p class="mt-3 text-sm text-muted-foreground">輸入訂單編號與查詢碼，即可追蹤訂單進度。</p>
    </header>

    <!-- 查詢表單 -->
    <div class="mt-8 rounded-xl border bg-card p-6 sm:p-8">
      <form class="grid gap-5 sm:grid-cols-[1.2fr_1fr_auto] sm:items-start" @submit.prevent="submit">
        <div class="space-y-1.5">
          <label for="track-order" class="text-sm font-medium">訂單編號 <span class="text-primary">*</span></label>
          <input
            id="track-order"
            v-model="orderId"
            type="text"
            placeholder="例如 TW-xxxxxx"
            :aria-invalid="Boolean(errors.orderId)"
            :class="cn('h-11 w-full rounded-md border bg-transparent px-3 font-mono text-sm tracking-wider outline-none focus:border-primary', errors.orderId && 'border-destructive')"
          />
          <p v-if="errors.orderId" class="text-xs text-destructive" role="alert">{{ errors.orderId }}</p>
        </div>
        <div class="space-y-1.5">
          <label for="track-token" class="text-sm font-medium">訂單查詢碼 <span class="text-primary">*</span></label>
          <input
            id="track-token"
            v-model="token"
            type="text"
            placeholder="下單時取得的查詢碼"
            :aria-invalid="Boolean(errors.token)"
            :class="cn('h-11 w-full rounded-md border bg-transparent px-3 font-mono text-sm outline-none focus:border-primary', errors.token && 'border-destructive')"
          />
          <p v-if="errors.token" class="text-xs text-destructive" role="alert">{{ errors.token }}</p>
        </div>
        <button
          type="submit"
          :disabled="pending"
          class="inline-flex h-11 items-center justify-center gap-2 rounded-full bg-primary px-8 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:opacity-60 sm:mt-[26px]"
        >
          <Loader2 v-if="pending" class="size-4 animate-spin" />
          <Search v-else class="size-4" />
          查詢
        </button>
      </form>
      <p class="mt-4 text-xs text-muted-foreground/70">
        提示：訂單編號與查詢碼可在下單完成頁或訂單確認通知中找到。
      </p>
    </div>

    <!-- 最近訂單 -->
    <div v-if="recent.length > 0" class="mt-6">
      <p class="flex items-center gap-2 text-xs font-medium tracking-wider text-muted-foreground">
        <History class="size-3.5" />最近訂單（此裝置）
      </p>
      <div class="mt-3 flex flex-wrap gap-2">
        <button
          v-for="r in recent"
          :key="r.orderId"
          type="button"
          class="group flex items-center gap-2.5 rounded-full border bg-card px-4 py-2 text-sm transition-colors hover:border-primary/40 hover:text-primary"
          @click="applyRecent(r)"
        >
          <span class="font-mono text-xs tracking-wider">{{ r.orderId }}</span>
          <span class="hidden text-[10px] text-muted-foreground sm:inline">{{ formatDate(r.date) }}</span>
        </button>
      </div>
    </div>

    <!-- 查詢結果 -->
    <div class="mt-10">
      <div v-if="pending" class="space-y-5">
        <div class="h-36 w-full animate-pulse rounded-xl bg-muted" />
        <div class="h-60 w-full animate-pulse rounded-xl bg-muted" />
      </div>

      <OrderDetailCard v-else-if="order" :order="order" />

      <div v-else class="flex flex-col items-center rounded-xl border border-dashed py-14 text-center">
        <PackageSearch class="size-11 text-muted-foreground/30" :stroke-width="1.2" />
        <template v-if="attempted">
          <h3 class="mt-4 font-serif text-xl font-semibold">找不到這筆訂單</h3>
          <p class="mt-2 max-w-sm text-sm text-muted-foreground">訂單編號或查詢碼不相符，或訂單不存在。請再次確認後重試。</p>
          <button type="button" class="mt-6 inline-flex h-11 items-center rounded-full border px-7 text-sm" @click="submit">再查一次</button>
        </template>
        <template v-else>
          <h3 class="mt-4 font-serif text-lg font-semibold text-muted-foreground">還沒有查詢紀錄</h3>
          <p class="mt-2 text-sm text-muted-foreground/80">輸入上方欄位，開始追蹤您的訂單。</p>
          <button type="button" class="mt-5 inline-flex h-11 items-center rounded-full px-6 text-sm text-muted-foreground transition-colors hover:text-foreground" @click="navigate('/shop/')">
            先去逛逛
          </button>
        </template>
      </div>
    </div>
  </div>
</template>
