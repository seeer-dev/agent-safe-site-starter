<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useQuery } from '@tanstack/vue-query'
import {
  PageHeader,
  Panel,
  TableViewport,
  Table,
  TableHeader,
  TableRow,
  TableHead,
  TableBody,
  TableCell,
} from '@sitecore/admin-ui'
import { merchantSettingsQuery } from '../domains/merchant-settings/data/queries'
import {
  isSettingsTab,
  type SettingsTab,
} from '../domains/merchant-settings/model'
import { useAdminRuntime } from '../app/runtime'

const route = useRoute()
const router = useRouter()
const runtime = useAdminRuntime()
const { data, isPending, isError } = useQuery(merchantSettingsQuery(runtime.settingsReader))

const SETTINGS_TAB_LABELS: Record<SettingsTab, string> = {
  site: '賣點資訊',
  brand: '品牌外觀',
  payment: '付款方式',
  shipping: '配送方式',
}

const BRAND_COLOR_TOKENS = ['#2563eb', '#16a34a', '#ea580c', '#9333ea', '#e11d48'] as const

const activeTab = computed<SettingsTab>(() => {
  const raw = route.query['tab'] as string | undefined
  return isSettingsTab(raw) ? raw : 'site'
})

function selectTab(next: SettingsTab) {
  router.replace({ query: { ...route.query, tab: next } })
}

const paymentRows = computed(() => {
  if (!data.value) return []
  return data.value.payment.items.map((item) => {
    const fee = item.feeBasis[0]
    const feeLabel = fee
      ? fee.percentageFeeBps > 0
        ? `${(fee.percentageFeeBps / 100).toFixed(1)}%`
        : `NT$${fee.fixedFee.amount}`
      : '—'
    return {
      id: item.id,
      method: SETTINGS_PROVIDER_LABELS[item.providerType] ?? item.providerType,
      fee: feeLabel,
      enabled: item.enabled,
    }
  })
})

const shippingRows = computed(() => {
  if (!data.value) return []
  return data.value.shipping.items.map((item) => ({
    id: item.id,
    method: item.name,
    fee: `NT$${item.feeAmount.amount}`,
    enabled: item.enabled,
  }))
})

const activeColorIndex = computed(() => {
  if (!data.value) return -1
  return BRAND_COLOR_TOKENS.findIndex((token) => token === data.value!.brand.brandColorToken)
})

const SETTINGS_PROVIDER_LABELS: Record<string, string> = {
  card: '信用卡',
  atm: 'ATM 轉帳',
  cvs: '超商取貨付款',
}

const READONLY_NOTICE = '此為唯讀檢視，修改功能尚未開放。'
</script>

<template>
  <div v-if="isPending" class="content"><p>載入中…</p></div>
  <div v-else-if="isError" class="content"><p>載入失敗</p></div>
  <div v-else-if="data" class="content space-y-4">
    <PageHeader variant="compact" title="賣點設定" />

    <div class="grid lg:grid-cols-[200px_1fr] gap-4 max-lg:block">
      <!-- Settings nav -->
      <nav class="panel block border border-border rounded-panel bg-surface overflow-hidden p-1.5" aria-label="設定導覽">
        <button
          v-for="tab in (['site', 'brand', 'payment', 'shipping'] as const)"
          :key="tab"
          type="button"
          class="tab-item block w-full rounded-card px-3 py-2 text-left text-fs-md"
          :class="{
            'tab-item-active bg-brand-50 text-brand-700 font-medium': activeTab === tab,
            'text-text-secondary hover:bg-surface-subtle': activeTab !== tab,
          }"
          :aria-current="activeTab === tab ? 'page' : undefined"
          @click="selectTab(tab)"
        >
          {{ SETTINGS_TAB_LABELS[tab] }}
        </button>
      </nav>

      <div class="space-y-4">
        <!-- Site tab -->
        <div v-if="activeTab === 'site'">
          <Panel class="overflow-hidden">
            <div class="px-4 py-3 border-b border-border">
              <h2 class="text-fs-xl font-semibold text-text-primary m-0">賣點資訊</h2>
            </div>
            <div class="p-4 space-y-4">
              <div>
                <label class="block text-fs-md text-text-secondary mb-1.5">賣點名稱</label>
                <input type="text" :value="data.site.siteName" disabled readonly
                  class="w-full px-3 py-2 text-fs-md border border-border rounded-input bg-surface text-text-primary opacity-70" />
              </div>
              <div>
                <label class="block text-fs-md text-text-secondary mb-1.5">客服電話</label>
                <input type="text" :value="data.site.supportPhone ?? ''" disabled readonly
                  class="w-full px-3 py-2 text-fs-md border border-border rounded-input bg-surface text-text-primary opacity-70" />
              </div>
              <div>
                <label class="block text-fs-md text-text-secondary mb-1.5">聯絡 Email</label>
                <input type="email" :value="data.site.contactEmail ?? ''" disabled readonly
                  class="w-full px-3 py-2 text-fs-md border border-border rounded-input bg-surface text-text-primary opacity-70" />
              </div>
              <div>
                <label class="block text-fs-md text-text-secondary mb-1.5">賣點描述</label>
                <textarea rows="3" :value="data.site.maintenanceMessage ?? ''" disabled readonly
                  class="w-full resize-none px-3 py-2 text-fs-md border border-border rounded-input bg-surface text-text-primary opacity-70" />
              </div>
              <p class="text-fs-base text-text-muted m-0">{{ READONLY_NOTICE }}</p>
            </div>
          </Panel>
        </div>

        <!-- Brand tab -->
        <div v-if="activeTab === 'brand'">
          <Panel class="overflow-hidden">
            <div class="px-4 py-3 border-b border-border">
              <h2 class="text-fs-xl font-semibold text-text-primary m-0">品牌外觀</h2>
            </div>
            <div class="p-4 space-y-4">
              <div>
                <label class="block text-fs-md text-text-secondary mb-1.5">品牌色</label>
                <div class="flex items-center gap-3">
                  <button
                    v-for="(token, index) in BRAND_COLOR_TOKENS"
                    :key="token"
                    type="button"
                    disabled
                    :aria-label="token"
                    class="h-8 w-8 rounded-full"
                    :class="{ 'ring-2 ring-offset-2': activeColorIndex === index }"
                    :style="{ backgroundColor: token }"
                  />
                </div>
              </div>
              <div>
                <label class="block text-fs-md text-text-secondary mb-1.5">Logo</label>
                <div class="flex items-center gap-3">
                  <div class="flex h-14 w-14 items-center justify-center rounded-card border border-dashed border-border bg-surface-subtle text-fs-sm text-text-muted">
                    Logo
                  </div>
                  <button class="px-3 py-1.5 text-fs-md border border-border rounded-btn bg-surface text-text-secondary" disabled>選擇檔案</button>
                </div>
              </div>
              <div class="flex items-center justify-between">
                <label class="text-fs-md text-text-secondary">深色模式</label>
                <button disabled role="switch" aria-checked="false"
                  class="relative h-6 w-10 rounded-full bg-surface-subtle border border-border">
                  <span class="absolute left-0.5 top-0.5 h-5 w-5 rounded-full bg-white shadow" />
                </button>
              </div>
              <p class="text-fs-base text-text-muted m-0">{{ READONLY_NOTICE }}</p>
            </div>
          </Panel>
        </div>

        <!-- Payment tab -->
        <div v-if="activeTab === 'payment'">
          <Panel class="overflow-hidden">
            <div class="px-4 py-3 border-b border-border">
              <h2 class="text-fs-xl font-semibold text-text-primary m-0">付款方式</h2>
            </div>
            <TableViewport>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>方式</TableHead>
                    <TableHead>手續費</TableHead>
                    <TableHead>啟用</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  <TableRow v-for="row in paymentRows" :key="row.id">
                    <TableCell>{{ row.method }}</TableCell>
                    <TableCell class="text-text-secondary">{{ row.fee }}</TableCell>
                    <TableCell>
                      <button
                        disabled
                        role="switch"
                        :aria-checked="row.enabled"
                        class="relative h-6 w-10 rounded-full transition-colors"
                        :class="row.enabled ? 'bg-brand' : 'bg-surface-subtle border border-border'"
                      >
                        <span
                          class="absolute top-0.5 h-5 w-5 rounded-full bg-white shadow transition-transform"
                          :class="row.enabled ? 'left-[1.125rem]' : 'left-0.5'"
                        />
                      </button>
                    </TableCell>
                  </TableRow>
                </TableBody>
              </Table>
            </TableViewport>
            <p class="p-4 text-fs-base text-text-muted m-0">{{ READONLY_NOTICE }}</p>
          </Panel>
        </div>

        <!-- Shipping tab -->
        <div v-if="activeTab === 'shipping'">
          <Panel class="overflow-hidden">
            <div class="px-4 py-3 border-b border-border">
              <h2 class="text-fs-xl font-semibold text-text-primary m-0">配送方式</h2>
            </div>
            <TableViewport>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>方式</TableHead>
                    <TableHead>運費</TableHead>
                    <TableHead>啟用</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  <TableRow v-for="row in shippingRows" :key="row.id">
                    <TableCell>{{ row.method }}</TableCell>
                    <TableCell class="text-text-secondary">{{ row.fee }}</TableCell>
                    <TableCell>
                      <button
                        disabled
                        role="switch"
                        :aria-checked="row.enabled"
                        class="relative h-6 w-10 rounded-full transition-colors"
                        :class="row.enabled ? 'bg-brand' : 'bg-surface-subtle border border-border'"
                      >
                        <span
                          class="absolute top-0.5 h-5 w-5 rounded-full bg-white shadow transition-transform"
                          :class="row.enabled ? 'left-[1.125rem]' : 'left-0.5'"
                        />
                      </button>
                    </TableCell>
                  </TableRow>
                </TableBody>
              </Table>
            </TableViewport>
            <p class="p-4 text-fs-base text-text-muted m-0">{{ READONLY_NOTICE }}</p>
          </Panel>
        </div>
      </div>
    </div>
  </div>
</template>
