<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useQuery } from '@tanstack/vue-query'
import {
  Amount,
  Notice,
  Panel,
  StatusPill,
  TabStrip,
  TableViewport,
  Table,
  TableHeader,
  TableRow,
  TableHead,
  TableBody,
  TableCell,
} from '@sitecore/admin-ui'
import { ArrowLeft, Package } from 'lucide-vue-next'
import { orderDetailQuery } from '../data/queries'
import { useAdminRuntime } from '../../../app/runtime'

const props = defineProps<{ orderId: string }>()

const runtime = useAdminRuntime()
const route = useRoute()
const router = useRouter()

const { data, isPending, isError } = useQuery(orderDetailQuery(runtime.orderDetailReader, props.orderId))

type DetailTab = 'detail' | 'review-history' | 'commission' | 'fulfillment-documents'

const DETAIL_TABS = [
  { id: 'detail' as const, label: '明細' },
  { id: 'review-history' as const, label: '審核歷史' },
  { id: 'commission' as const, label: '分潤' },
  { id: 'fulfillment-documents' as const, label: '出貨文件' },
]

const activeTab = computed<DetailTab>(() => {
  const segments = route.path.split('/')
  const last = segments[segments.length - 1]
  if (last === 'review-history' || last === 'commission' || last === 'fulfillment-documents') return last
  return 'detail'
})

function switchTab(tab: DetailTab) {
  if (tab === 'detail') router.push(`/orders/${props.orderId}`)
  else router.push(`/orders/${props.orderId}/${tab}`)
}
</script>

<template>
  <div v-if="isPending" class="content"><p>載入中…</p></div>
  <div v-else-if="isError" class="content"><p>載入失敗</p></div>
  <div v-else-if="data" class="content">
    <a class="back-link inline-flex items-center gap-1.5 text-text-primary text-fs-md mb-2.5 cursor-pointer" @click="router.push('/orders')">
      <ArrowLeft :size="15" />
      訂單
    </a>

    <Panel class="overflow-hidden">
      <!-- Hero -->
      <div class="hero flex gap-4 items-start flex-wrap px-4.5 py-4.5 border-b border-border">
        <div>
          <div class="flex items-center gap-2.5">
            <h2 class="hero-title m-0 text-fs-3xl font-bold">{{ data.hero.orderNumber }}</h2>
            <StatusPill :variant="data.hero.statusTone">{{ data.hero.statusLabel }}</StatusPill>
          </div>
          <div class="hero-meta text-text-muted text-fs-md mt-0.75 flex gap-3 flex-wrap">{{ data.hero.meta }}</div>
        </div>
        <div class="hero-facts flex gap-5.5 ml-auto flex-wrap">
          <div v-for="fact in data.hero.facts" :key="fact.label">
            <span class="fact-label block text-text-muted text-fs-sm-plus">{{ fact.label }}</span>
            <span class="fact-value text-fs-2xl font-semibold">{{ fact.value }}</span>
          </div>
        </div>
      </div>

      <!-- Order progress stepper -->
      <div class="px-4.5 pt-3.5">
        <div class="text-text-muted text-fs-base font-semibold mb-2">訂單進度</div>
      </div>
      <div class="stepper flex items-center px-4.5 pt-0 pb-3.5 border-b border-border overflow-auto">
        <template v-for="(step, index) in data.orderSteps" :key="step.label">
          <div
            class="step"
            :class="{ 'step-done': step.state === 'done', 'step-current': step.state === 'current' }"
          >
            <span class="step-dot">{{ index + 1 }}</span>
            {{ step.label }}
          </div>
          <div v-if="index < data.orderSteps.length - 1" class="step-line" aria-hidden="true" />
        </template>
      </div>

      <!-- Detail tabs -->
      <TabStrip aria-label="訂單明細分頁">
        <button
          v-for="tab in DETAIL_TABS"
          :key="tab.id"
          type="button"
          class="tab-item px-3.25 py-2 text-fs-md-plus cursor-pointer whitespace-nowrap"
          :class="{
            'tab-item-active font-semibold text-brand-600 border-b-2 border-brand-600 -mb-px': activeTab === tab.id,
            'font-medium text-text-secondary hover:text-text-primary': activeTab !== tab.id,
          }"
          :aria-current="activeTab === tab.id ? 'page' : undefined"
          @click="switchTab(tab.id)"
        >
          {{ tab.label }}
        </button>
      </TabStrip>

      <!-- d1: 明細 -->
      <div v-if="activeTab === 'detail'" id="d1">
        <div class="px-4.5 py-3.5">
          <div
            v-for="(row, index) in data.products"
            :key="row.id"
            class="product-row flex items-center gap-3 py-3"
            :class="{ 'border-b border-border': index < data.products.length - 1 }"
          >
            <div class="product-thumb w-12 h-12 rounded-card bg-surface-100 flex-none grid place-items-center text-text-muted">
              <Package :size="20" />
            </div>
            <div class="flex-1 min-w-0">
              <b class="text-fs-md-plus font-semibold text-text-primary block truncate">{{ row.name }}</b>
              <div class="text-text-muted text-fs-base mt-0.5">SKU {{ row.sku }} · 單價 {{ row.unitPrice }}</div>
            </div>
            <div class="text-right flex-none">
              <div class="text-text-secondary text-fs-base">x{{ row.qty }}</div>
              <div class="text-text-primary text-fs-lg font-semibold tabular-nums mt-0.5">{{ row.subtotal }}</div>
            </div>
          </div>
        </div>
      </div>

      <!-- d2: 審核歷史 -->
      <div v-if="activeTab === 'review-history'" id="d2">
        <div class="px-4.5 py-3.5">
          <div class="timeline pl-1.5">
            <div
              v-for="event in data.reviewEvents"
              :key="event.id"
              class="timeline-event flex gap-3 pb-3.75 relative"
            >
              <div class="timeline-dot w-[11px] h-[11px] rounded-full bg-brand-600 border-2 border-surface flex-none mt-0.75" />
              <div>
                <b class="text-fs-md font-semibold">{{ event.title }}</b>
                <div class="timeline-meta mt-[1.5px] text-text-muted text-fs-base">{{ event.meta }}</div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- d3: 分潤 -->
      <div v-if="activeTab === 'commission'" id="d3">
        <TableViewport>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>SKU</TableHead>
                <TableHead>規則</TableHead>
                <TableHead align="end">基準</TableHead>
                <TableHead align="end">分潤</TableHead>
                <TableHead>狀態</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow v-for="line in data.commissionLines" :key="line.id">
                <TableCell>{{ line.sku }}</TableCell>
                <TableCell>{{ line.rule }}</TableCell>
                <TableCell align="end"><Amount>{{ line.base }}</Amount></TableCell>
                <TableCell align="end"><Amount>{{ line.commission }}</Amount></TableCell>
                <TableCell>
                  <StatusPill :variant="line.statusTone">{{ line.status }}</StatusPill>
                </TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </TableViewport>
        <Notice>
          {{ data.commissionNote }}
        </Notice>
      </div>

      <!-- d4: 出貨文件 -->
      <div v-if="activeTab === 'fulfillment-documents'" id="d4">
        <div v-if="data.fulfillmentDocs.length === 0" class="px-4.5 py-3.5">
          <div class="empty-state flex flex-col items-center gap-2.5 py-8 text-center">
            <div class="empty-state-icon w-11 h-11 rounded-kpi bg-surface-subtle text-text-muted grid place-items-center">
              <Package :size="22" />
            </div>
            <b class="text-fs-xl">還沒有出貨文件</b>
            <p class="m-0 text-text-muted text-fs-md">審核通過後會自動產生出貨單與出貨標籤。</p>
          </div>
        </div>
      </div>
    </Panel>
  </div>
</template>
