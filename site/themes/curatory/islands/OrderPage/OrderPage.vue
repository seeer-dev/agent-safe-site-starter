<script setup lang="ts">
// 訂單成功／詳情頁 — 對應 reference order-view.tsx。
// 差異：本站查單採 access-token 模型（GET /api/orders/{id} +
// X-Order-Access-Token），不以「單號＋Email」查詢。憑證來源依序：
//   1. sessionStorage curatory_last_order（剛下單跳轉）
//   2. localStorage curatory_recent_orders（rememberOrder 寫入）
//   3. 手動輸入查詢碼（訂單完成頁／通知信中提供）
import { computed, onMounted, ref } from 'vue'
import { KeyRound, Loader2, MailSearch, PackageSearch, SearchX } from 'lucide-vue-next'
import OrderDetailCard from '@/shared/components/OrderDetailCard.vue'
import { apiGet } from '@/shared/lib/api'
import { bootstrap, findRecentToken, loadBootstrap } from '@/shared/lib/store'
import { navigate } from '@/shared/lib/transition'
import type { OrderDTO } from '@/shared/lib/types'

const props = defineProps<{ orderId?: string }>()

const orderId = computed(() => {
  if (props.orderId) return props.orderId
  try {
    return new URLSearchParams(window.location.search).get('id') ?? ''
  } catch {
    return ''
  }
})

const token = ref<string | null>(null)
const tokenInput = ref('')
const tokenError = ref('')
const isFreshOrder = ref(false)
const order = ref<OrderDTO | null>(null)
const loading = ref(false)
const loadError = ref('')

onMounted(() => {
  void loadBootstrap().catch(() => undefined)
  if (!orderId.value) return
  try {
    const raw = sessionStorage.getItem('curatory_last_order')
    if (raw) {
      const parsed = JSON.parse(raw) as { orderId?: string; token?: string }
      if (parsed.orderId === orderId.value && parsed.token) {
        token.value = parsed.token
        isFreshOrder.value = true
      }
    }
  } catch { /* ignore */ }
  token.value ??= findRecentToken(orderId.value)
  if (token.value) void fetchOrder()
})

async function fetchOrder() {
  if (!orderId.value || !token.value) return
  loading.value = true
  loadError.value = ''
  try {
    const res = await apiGet<{ order: OrderDTO } | OrderDTO>(`/api/orders/${encodeURIComponent(orderId.value)}`, {
      'X-Order-Access-Token': token.value,
    })
    order.value = 'order' in res ? res.order : res
  } catch (e) {
    loadError.value = e instanceof Error ? e.message : '查無此訂單'
    order.value = null
  } finally {
    loading.value = false
  }
}

function submitToken() {
  const t = tokenInput.value.trim()
  if (!t) {
    tokenError.value = '請輸入訂單查詢碼'
    return
  }
  tokenError.value = ''
  token.value = t
  void fetchOrder()
}
</script>

<template>
  <!-- 無訂單編號 → 導向查詢頁 -->
  <div v-if="!orderId" class="mx-auto flex min-h-[60vh] w-full max-w-md flex-col items-center justify-center px-4 text-center">
    <PackageSearch class="size-12 text-muted-foreground/40" :stroke-width="1.5" />
    <h1 class="mt-5 font-serif text-2xl font-semibold">缺少訂單編號</h1>
    <p class="mt-2 text-sm leading-relaxed text-muted-foreground">請從訂單查詢頁輸入訂單編號追蹤您的訂單。</p>
    <button type="button" class="mt-7 inline-flex h-11 items-center rounded-full bg-primary px-7 text-sm font-medium text-primary-foreground" @click="navigate('/track/')">
      前往訂單查詢
    </button>
  </div>

  <!-- 需要查詢碼 -->
  <div v-else-if="!token" class="mx-auto flex min-h-[70vh] w-full max-w-md flex-col items-center justify-center px-4 py-12">
    <div class="w-full rounded-xl border bg-card p-8 text-center">
      <div class="mx-auto flex size-14 items-center justify-center rounded-full border bg-muted/40">
        <KeyRound class="size-6 text-muted-foreground/60" :stroke-width="1.5" />
      </div>
      <h1 class="mt-5 font-serif text-2xl font-semibold">查詢訂單</h1>
      <p class="mt-2 text-sm leading-relaxed text-muted-foreground">
        為保護您的訂單隱私，<br class="sm:hidden" />請輸入訂單查詢碼以查看訂單。
      </p>
      <p class="mt-2 font-mono text-xs tracking-wider text-muted-foreground/70">{{ orderId }}</p>
      <div class="mt-6 space-y-3 text-left">
        <input
          v-model="tokenInput"
          type="text"
          placeholder="訂單查詢碼"
          aria-label="訂單查詢碼"
          :aria-invalid="Boolean(tokenError)"
          :class="['h-11 w-full rounded-md border bg-transparent px-3 font-mono text-sm outline-none focus:border-primary', tokenError && 'border-destructive']"
          @keydown.enter="submitToken"
          @input="tokenError = ''"
        />
        <p v-if="tokenError" class="text-xs text-destructive" role="alert">{{ tokenError }}</p>
        <p v-else class="text-xs text-muted-foreground/70">查詢碼見於訂單完成頁或訂單確認通知</p>
        <button type="button" class="h-11 w-full rounded-full bg-primary text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90" @click="submitToken">
          查看訂單
        </button>
      </div>
      <button type="button" class="mt-5 text-sm text-primary link-underline" @click="navigate('/track/')">
        或使用訂單查詢頁（可查最近訂單）
      </button>
    </div>
  </div>

  <!-- 載入中 -->
  <div v-else-if="loading" class="mx-auto w-full max-w-3xl px-4 py-12 sm:px-6 lg:px-8">
    <div class="space-y-5">
      <div class="mx-auto size-16 animate-pulse rounded-full bg-muted" />
      <div class="mx-auto h-8 w-56 animate-pulse rounded bg-muted" />
      <div class="h-40 w-full animate-pulse rounded-xl bg-muted" />
      <div class="h-64 w-full animate-pulse rounded-xl bg-muted" />
    </div>
    <p class="mt-6 flex items-center justify-center gap-2 text-sm text-muted-foreground">
      <Loader2 class="size-4 animate-spin" />載入訂單中…
    </p>
  </div>

  <!-- 查無訂單 -->
  <div v-else-if="loadError || !order" class="mx-auto flex min-h-[60vh] w-full max-w-md flex-col items-center justify-center px-4 text-center">
    <SearchX class="size-12 text-muted-foreground/40" :stroke-width="1.5" />
    <h1 class="mt-5 font-serif text-2xl font-semibold">查無此訂單</h1>
    <p class="mt-2 text-sm leading-relaxed text-muted-foreground">{{ loadError || '訂單編號或查詢碼不相符，或訂單不存在。' }}</p>
    <div class="mt-7 flex gap-3">
      <button type="button" class="inline-flex h-11 items-center rounded-full border px-6 text-sm" @click="token = null; tokenInput = ''">重新輸入</button>
      <button type="button" class="inline-flex h-11 items-center rounded-full bg-primary px-6 text-sm font-medium text-primary-foreground" @click="navigate('/track/')">前往訂單查詢</button>
    </div>
  </div>

  <!-- 成功畫面 -->
  <div v-else class="mx-auto w-full max-w-3xl px-4 py-12 sm:px-6 sm:py-16 lg:px-8">
    <div class="mb-8 text-center">
      <div class="mx-auto flex size-18 items-center justify-center rounded-full bg-primary/10 text-primary">
        <PackageSearch class="size-8" :stroke-width="1.6" />
      </div>
      <h1 class="mt-6 font-serif text-3xl font-semibold sm:text-4xl">{{ isFreshOrder ? '感謝您的訂購！' : '訂單詳情' }}</h1>
      <p class="mt-3 text-sm leading-relaxed text-muted-foreground">
        <template v-if="isFreshOrder">
          我們已寄送訂單確認通知至 {{ order.email }}，<br class="hidden sm:block" />出貨時也會再次通知您。
        </template>
        <template v-else>以下是此筆訂單的完整資訊。</template>
      </p>
      <!-- 新訂單：顯示查詢碼供留存 -->
      <p v-if="isFreshOrder && token" class="mx-auto mt-4 max-w-md break-all rounded-lg border border-dashed bg-muted/40 px-4 py-2.5 font-mono text-[11px] leading-relaxed text-muted-foreground">
        訂單查詢碼：{{ token }}<br />
        <span class="text-[10px]">請妥善保存，日後可用於訂單查詢頁追蹤進度</span>
      </p>
    </div>
    <OrderDetailCard :order="order" />
  </div>
</template>
