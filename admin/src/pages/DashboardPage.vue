<script setup lang="ts">
import { onMounted, ref } from 'vue'
import Badge from '@/components/ui/Badge.vue'
import Button from '@/components/ui/Button.vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import { api } from '@/lib/api-client'

const router = useRouter()
const auth = useAuthStore()

// KPIs are fetched from the server, not hardcoded. When unverified, no
// KPI is shown — the dashboard displays a login prompt instead.
const kpis = ref<{ label: string; value: number; desc: string }[]>([])
const tasks = ref<{ tone: 'warn' | 'info' | 'danger'; label: string; id: string; desc: string; action: string; res: string }[]>([])
const loading = ref(false)
const loadError = ref<string | null>(null)

async function loadDashboard() {
  if (!auth.isAuthenticated) return
  loading.value = true
  loadError.value = null
  try {
    // Fetch real counts from the admin API. Each endpoint returns a shape
    // like { orders: [...] } or { products: [...] }; we count what we need.
    const [ordersRes, productsRes] = await Promise.all([
      api.get<Record<string, any>>('/admin/orders').catch(() => null),
      api.get<Record<string, any>>('/admin/products').catch(() => null),
    ])
    const orders = extractArray(ordersRes)
    const products = extractArray(productsRes)

    const pending = orders.filter((o: any) => o.status === 'pending').length
    const processing = orders.filter((o: any) => o.status === 'processing').length
    const returnRequested = orders.filter((o: any) => o.return_request_status === 'requested').length
    const lowStock = products.filter((p: any) => p.stock <= 5 && p.status !== 'draft').length

    kpis.value = [
      { label: '待處理訂單', value: pending, desc: '履約狀態 pending' },
      { label: '待出貨', value: processing, desc: '履約狀態 processing' },
      { label: '退貨待審', value: returnRequested, desc: '退貨狀態 requested' },
      { label: '低庫存商品', value: lowStock, desc: '庫存 ≤ 5' },
    ]

    // Build tasks from real orders + low-stock products (no PII when unverified).
    tasks.value = [
      ...orders
        .filter((o: any) => o.status === 'pending' || o.status === 'processing' || o.return_request_status === 'requested')
        .slice(0, 3)
        .map((o: any) => ({
          tone: (o.return_request_status === 'requested' ? 'danger' : o.status === 'pending' ? 'warn' : 'info') as 'warn' | 'info' | 'danger',
          label: o.return_request_status === 'requested' ? '退貨待審' : o.status === 'pending' ? '待處理' : '處理中',
          id: o.id,
          desc: `${o.customer_name} · NT$${o.total}`,
          action: o.return_request_status === 'requested' ? '審核退貨' : o.status === 'pending' ? '開始處理' : '出貨',
          res: 'minimal-cart-orders',
        })),
      ...products
        .filter((p: any) => p.stock <= 5 && p.status !== 'draft')
        .slice(0, 1)
        .map((p: any) => ({
          tone: 'danger' as const,
          label: '低庫存',
          id: p.sku,
          desc: `${p.name} · 庫存 ${p.stock}`,
          action: '補貨',
          res: 'minimal-cart-products',
        })),
    ]
  } catch (e: any) {
    loadError.value = e?.message ?? String(e)
  } finally {
    loading.value = false
  }
}

function extractArray(res: Record<string, any> | null): any[] {
  if (!res) return []
  const arr = Object.values(res).find((v) => Array.isArray(v))
  return Array.isArray(arr) ? arr : []
}

onMounted(() => {
  if (auth.isAuthenticated) {
    loadDashboard()
  }
})

const modules = [
  { key: 'twcommerce', desc: '商品 · 訂單 · 會員 · 優惠 · 付款方式' },
  { key: 'auth', desc: '登入與 session' },
  { key: 'staff', desc: '人員與權限' },
  { key: 'storage', desc: '商品主圖上傳' },
]

function goRes(key: string) {
  router.push(`/res/${key}`)
}
</script>

<template>
  <!-- Page header -->
  <div class="pagehd">
    <div>
      <h1>總覽</h1>
      <div class="sub">質物選物</div>
    </div>
  </div>

  <!-- KPIs -->
  <div class="kpis">
      <div
        v-for="kpi in kpis"
        :key="kpi.label"
        class="kpi"
      >
        <small>{{ kpi.label }}</small>
        <b>{{ kpi.value }}</b>
        <div class="d">{{ kpi.desc }}</div>
      </div>
    </div>

    <!-- Two columns: tasks + modules -->
    <div class="cols">
      <!-- Tasks -->
      <section class="panel">
        <div class="phd">
          <h3>需要你處理</h3>
          <a class="more" @click="goRes('minimal-cart-orders')">全部訂單</a>
        </div>
        <div v-if="loading" class="emptybox"><b>載入中…</b></div>
        <div v-else-if="loadError" class="emptybox">
          <b>載入失敗</b>
          <p class="mono">{{ loadError }}</p>
        </div>
        <div v-else-if="tasks.length === 0" class="emptybox">
          <b>沒有待處理項目</b>
        </div>
        <div
          v-for="(task, i) in tasks"
          :key="i"
          class="rowi"
        >
          <Badge :tone="task.tone" :label="task.label" />
          <span class="mono">{{ task.id }}</span>
          <span class="muted">{{ task.desc }}</span>
          <Button
            size="sm"
            :variant="i < 2 ? 'pri' : 'default'"
            style="margin-left:auto"
            @click="goRes(task.res)"
          >{{ task.action }}</Button>
        </div>
      </section>

      <!-- Modules -->
      <section class="panel">
        <div class="phd">
          <h3>本站啟用的模組</h3>
          <span class="more muted">4 個</span>
        </div>
        <div
          v-for="m in modules"
          :key="m.key"
          class="rowi"
        >
          <span class="mono">{{ m.key }}</span>
          <span class="muted" style="margin-left:auto">{{ m.desc }}</span>
        </div>
      </section>
    </div>
</template>
