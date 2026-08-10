<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useQuery } from '@tanstack/vue-query'
import {
  Button,
  Panel,
  StatusPill,
  TableViewport,
  Table,
  TableHeader,
  TableRow,
  TableHead,
  TableBody,
  TableCell,
} from '@sitecore/admin-ui'
import { ArrowLeft } from 'lucide-vue-next'
import { influencerDetailQuery } from '../data/queries'
import { influencerStatusTone, type InfluencerTab } from '../model'
import { useAdminRuntime } from '../../../app/runtime'

const route = useRoute()
const router = useRouter()
const runtime = useAdminRuntime()
const influencerId = computed(() => route.params['influencerId'] as string)

const { data, isPending, isError } = useQuery(influencerDetailQuery(runtime.influencerDetailReader, influencerId.value))

const STATUS_LABELS: Record<string, string> = {
  active: '合作中',
  disabled: '待撥款',
}

const MEMBERSHIP_STATUS_LABELS: Record<string, string> = {
  active: '進行中',
  disabled: '已停用',
  ended: '已結束',
}

const TABS: readonly InfluencerTab[] = [
  { id: 'campaigns', label: '參與檔期', enabled: true },
  { id: 'account', label: '帳號資訊', enabled: true },
  { id: 'commissions', label: '分潤紀錄', enabled: true },
  { id: 'tags', label: '標籤分類', enabled: true },
]

const activeTab = computed<string>(() => (route.query['tab'] as string) ?? 'campaigns')

function selectTab(tabId: string) {
  router.replace({ query: { ...route.query, tab: tabId } })
}

function formatCurrency(amount: number, currency: string): string {
  if (amount === 0) return '—'
  const symbol = currency === 'TWD' ? '$' : ''
  return `${symbol}${amount.toLocaleString()}`
}

function goBack() {
  router.push('/influencers')
}
</script>

<template>
  <div v-if="isPending" class="content"><p>載入中…</p></div>
  <div v-else-if="isError" class="content">
    <Panel>
      <div class="p-4">
        <p class="m-0 text-text-muted">找不到此網紅。</p>
        <Button variant="secondary" class="mt-3" @click="goBack">返回列表</Button>
      </div>
    </Panel>
  </div>
  <div v-else-if="data" class="content space-y-4">
    <button
      type="button"
      class="back-link inline-flex items-center gap-1.5 text-text-secondary hover:text-text-primary text-fs-md mb-2.5"
      @click="goBack"
    >
      <ArrowLeft :size="15" />
      返回列表
    </button>

    <!-- Hero panel -->
    <Panel class="block overflow-hidden mb-4">
      <div class="flex items-start gap-4 p-5">
        <div class="w-12 h-12 rounded-full bg-brand-100 text-brand-700 flex items-center justify-center font-semibold text-[18px] shrink-0">
          {{ data.profile.nickname.charAt(0) }}
        </div>
        <div class="flex-1 min-w-0">
          <div class="flex items-center gap-2 mb-1">
            <StatusPill :variant="influencerStatusTone(data.profile.collaborationStatus)" class="status-pill-row rounded">
              {{ STATUS_LABELS[data.profile.collaborationStatus] ?? data.profile.collaborationStatus }}
            </StatusPill>
          </div>
          <h1 class="page-title m-0 text-fs-4xl font-semibold leading-normal text-text-primary">
            {{ data.profile.nickname }}
          </h1>
          <p class="page-subtitle m-0 text-text-muted text-fs-md leading-normal">
            {{ data.profile.contactEmail ?? '未提供 Email' }}
            <template v-if="data.memberships[0]?.discountCode">
              · 折扣碼 {{ data.memberships[0].discountCode }}
            </template>
          </p>
        </div>
      </div>
      <!-- Fact grid -->
      <div class="grid grid-cols-2 sm:grid-cols-4 border-t border-border">
        <div class="p-4 border-r border-border">
          <p class="text-fs-base text-text-muted mb-1">參與檔期</p>
          <p class="text-[18px] font-semibold text-text-primary tabular-nums">{{ data.memberships.length }}</p>
        </div>
        <div class="p-4 border-r border-border">
          <p class="text-fs-base text-text-muted mb-1">累計分潤</p>
          <p class="text-[18px] font-semibold text-text-primary tabular-nums">{{ formatCurrency(data.lifetimeCommission.amount, data.lifetimeCommission.currency) }}</p>
        </div>
        <div class="p-4 border-r border-border">
          <p class="text-fs-base text-text-muted mb-1">待撥金額</p>
          <p class="text-[18px] font-semibold tabular-nums" :style="data.payableAmount.amount > 0 ? { color: 'var(--admin-warning-fg)' } : undefined">
            {{ formatCurrency(data.payableAmount.amount, data.payableAmount.currency) }}
          </p>
        </div>
        <div class="p-4">
          <p class="text-fs-base text-text-muted mb-1">已撥金額</p>
          <p class="text-[18px] font-semibold text-text-primary tabular-nums">{{ formatCurrency(data.paidAmount.amount, data.paidAmount.currency) }}</p>
        </div>
      </div>
    </Panel>

    <!-- Tab bar -->
    <nav class="tabs-bar flex items-center gap-0 border-b border-border mb-4" aria-label="網紅詳情標籤">
      <button
        v-for="tab in TABS"
        :key="tab.id"
        type="button"
        class="tab-item"
        :class="{
          'tab-item-active': activeTab === tab.id,
          'text-text-secondary border-b-2 border-transparent': activeTab !== tab.id,
        }"
        :aria-current="activeTab === tab.id ? 'page' : undefined"
        @click="selectTab(tab.id)"
      >
        {{ tab.label }}
      </button>
    </nav>

    <!-- Campaigns tab -->
    <div v-if="activeTab === 'campaigns'">
      <Panel v-if="data.memberships.length > 0" class="block overflow-hidden">
        <TableViewport>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>檔期</TableHead>
                <TableHead align="end">狀態</TableHead>
                <TableHead align="end">折扣碼</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow v-for="m in data.memberships" :key="m.id">
                <TableCell>
                  <span class="font-semibold text-text-primary">{{ m.campaign.name }}</span>
                </TableCell>
                <TableCell align="end">
                  <StatusPill :variant="m.status === 'active' ? 'active' : m.status === 'disabled' ? 'ended' : 'draft'" class="status-pill-row rounded">
                    {{ MEMBERSHIP_STATUS_LABELS[m.status] ?? m.status }}
                  </StatusPill>
                </TableCell>
                <TableCell align="end" class="text-text-muted tabular-nums">{{ m.discountCode }}</TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </TableViewport>
      </Panel>
      <Panel v-else class="block overflow-hidden">
        <div class="p-4">
          <p class="m-0 text-text-muted">此網紅尚未參與任何檔期。</p>
        </div>
      </Panel>
    </div>

    <!-- Account tab -->
    <div v-if="activeTab === 'account'">
      <Panel class="block overflow-hidden">
        <div class="px-4 py-3 border-b border-border">
          <h2 class="text-fs-xl font-semibold text-text-primary m-0">帳號資訊</h2>
        </div>
        <div class="p-4 space-y-4">
          <div>
            <label class="block text-fs-md text-text-secondary mb-1.5">顯示名稱</label>
            <input type="text" :value="data.profile.nickname" disabled readonly
              class="w-full px-3 py-2 text-fs-md border border-border rounded-input bg-surface text-text-primary opacity-70" />
          </div>
          <div>
            <label class="block text-fs-md text-text-secondary mb-1.5">聯絡 Email</label>
            <input type="email" :value="data.profile.contactEmail ?? ''" disabled readonly
              class="w-full px-3 py-2 text-fs-md border border-border rounded-input bg-surface text-text-primary opacity-70" />
          </div>
          <div>
            <label class="block text-fs-md text-text-secondary mb-1.5">帳號狀態</label>
            <input type="text" :value="data.accountStatus" disabled readonly
              class="w-full px-3 py-2 text-fs-md border border-border rounded-input bg-surface text-text-primary opacity-70" />
          </div>
          <p class="text-fs-base text-text-muted m-0">此為唯讀檢視，修改功能尚未開放。</p>
        </div>
      </Panel>
    </div>

    <!-- Commissions tab -->
    <div v-if="activeTab === 'commissions'">
      <Panel class="block overflow-hidden">
        <div class="px-4 py-3 border-b border-border">
          <h2 class="text-fs-xl font-semibold text-text-primary m-0">分潤摘要</h2>
        </div>
        <div class="grid grid-cols-3">
          <div class="p-4 border-r border-border">
            <p class="text-fs-base text-text-muted mb-1">累計分潤</p>
            <p class="text-[18px] font-semibold text-text-primary tabular-nums">{{ formatCurrency(data.lifetimeCommission.amount, data.lifetimeCommission.currency) }}</p>
          </div>
          <div class="p-4 border-r border-border">
            <p class="text-fs-base text-text-muted mb-1">待撥金額</p>
            <p class="text-[18px] font-semibold tabular-nums" :style="data.payableAmount.amount > 0 ? { color: 'var(--admin-warning-fg)' } : undefined">
              {{ formatCurrency(data.payableAmount.amount, data.payableAmount.currency) }}
            </p>
          </div>
          <div class="p-4">
            <p class="text-fs-base text-text-muted mb-1">已撥金額</p>
            <p class="text-[18px] font-semibold text-text-primary tabular-nums">{{ formatCurrency(data.paidAmount.amount, data.paidAmount.currency) }}</p>
          </div>
        </div>
        <p class="p-4 text-fs-base text-text-muted m-0">此為唯讀檢視，修改功能尚未開放。</p>
      </Panel>
    </div>

    <!-- Tags tab -->
    <div v-if="activeTab === 'tags'">
      <Panel class="block overflow-hidden">
        <div class="px-4 py-3 border-b border-border">
          <h2 class="text-fs-xl font-semibold text-text-primary m-0">標籤分類</h2>
        </div>
        <div class="p-4">
          <p class="m-0 text-text-muted">標籤分類功能尚未開放。</p>
        </div>
      </Panel>
    </div>
  </div>
</template>
