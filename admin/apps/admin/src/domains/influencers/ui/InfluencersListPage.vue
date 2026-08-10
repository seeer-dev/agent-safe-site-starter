<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useQuery } from '@tanstack/vue-query'
import {
  Button,
  PageHeader,
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
import { UserPlus, Search } from 'lucide-vue-next'
import { influencerListQuery } from '../data/queries'
import { INFLUENCER_SAVED_VIEWS, influencerStatusTone } from '../model'
import { useAdminRuntime } from '../../../app/runtime'

const router = useRouter()
const runtime = useAdminRuntime()
const { data, isPending, isError } = useQuery(influencerListQuery(runtime.influencerListReader))

const activeView = ref<string>('all')
const searchQuery = ref('')

const STATUS_LABELS: Record<string, string> = {
  active: '合作中',
  disabled: '待撥款',
}

const VIEW_LABELS: Record<string, string> = {
  all: '全部',
  active: '合作中',
  disabled: '待撥款',
}

const filteredItems = computed(() => {
  if (!data.value) return []
  let rows = data.value.items
  const view = INFLUENCER_SAVED_VIEWS.find(v => v.id === activeView.value)
  if (view?.status) rows = rows.filter(r => r.collaborationStatus === view.status)
  const q = searchQuery.value.trim().toLowerCase()
  if (q) rows = rows.filter(r => r.nickname.toLowerCase().includes(q) || (r.email ?? '').toLowerCase().includes(q))
  return rows
})

const totalCount = computed(() => data.value?.pagination.total ?? 0)
const activeCount = computed(() => data.value?.items.filter(r => r.collaborationStatus === 'active').length ?? 0)
const disabledCount = computed(() => data.value?.items.filter(r => r.collaborationStatus === 'disabled').length ?? 0)

function chipCount(id: string): number {
  if (id === 'all') return totalCount.value
  if (id === 'active') return activeCount.value
  return disabledCount.value
}

function formatCurrency(amount: number, currency: string): string {
  if (amount === 0) return '—'
  const symbol = currency === 'TWD' ? '$' : ''
  return `${symbol}${amount.toLocaleString()}`
}

function goToDetail(id: string) {
  router.push(`/influencers/${id}`)
}

function goToCreate() {
  router.push('/influencers/new')
}
</script>

<template>
  <div v-if="isPending" class="content"><p>載入中…</p></div>
  <div v-else-if="isError" class="content"><p>載入失敗</p></div>
  <div v-else-if="data" class="content space-y-4">
    <PageHeader :title="'網紅列表'" :subtitle="`管理所有合作網紅與分潤帳戶`">
      <template #action>
        <Button variant="primary" @click="goToCreate">
          <template #default>
            <span class="inline-flex items-center gap-1">
              <UserPlus :size="15" />
              新增網紅
            </span>
          </template>
        </Button>
      </template>
    </PageHeader>

    <Panel class="overflow-hidden">
      <!-- Chips -->
      <div class="px-4 pt-3 flex items-center gap-2 flex-wrap">
        <button
          v-for="view in INFLUENCER_SAVED_VIEWS"
          :key="view.id"
          type="button"
          class="chip inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full text-fs-md font-medium cursor-pointer"
          :class="{
            'bg-brand-50 text-brand-700 border border-brand-100': activeView === view.id,
            'bg-surface text-text-secondary border border-border hover:bg-surface-subtle': activeView !== view.id,
          }"
          @click="activeView = view.id"
        >
          {{ VIEW_LABELS[view.id] }}
          <span class="tabular-nums">{{ chipCount(view.id) }}</span>
        </button>
      </div>

      <!-- Toolbar -->
      <div class="px-4 py-3 flex items-center justify-between gap-3 flex-wrap">
        <div class="relative">
          <Search :size="15" class="absolute left-2.5 top-1/2 -translate-y-1/2 text-text-muted" />
          <input
            v-model="searchQuery"
            type="text"
            class="w-[240px] pl-8 pr-3 py-1.5 rounded-input bg-surface border border-border text-fs-md text-text-primary focus:outline-none focus:border-brand-500"
            placeholder="搜尋網紅名稱或 Email"
          />
        </div>
      </div>

      <!-- Table -->
      <TableViewport v-if="filteredItems.length > 0">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>網紅</TableHead>
              <TableHead>折扣碼</TableHead>
              <TableHead>參與檔期</TableHead>
              <TableHead align="end">累計分潤</TableHead>
              <TableHead>狀態</TableHead>
              <TableHead align="end">查看</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow
              v-for="row in filteredItems"
              :key="row.id"
              class="cursor-pointer"
              @click="goToDetail(row.id)"
            >
              <TableCell>
                <div class="flex items-center gap-2.5">
                  <span class="w-7 h-7 rounded-full bg-brand-50 text-brand-700 text-fs-base font-semibold inline-flex items-center justify-center shrink-0">
                    {{ row.nickname.charAt(0) }}
                  </span>
                  <div class="flex flex-col">
                    <span class="text-text-primary">{{ row.nickname }}</span>
                    <span v-if="row.email" class="text-text-muted text-fs-base">{{ row.email }}</span>
                  </div>
                </div>
              </TableCell>
              <TableCell class="text-text-muted">{{ row.defaultDiscountCode ?? '—' }}</TableCell>
              <TableCell class="tabular-nums">{{ row.activeCampaignCount }}</TableCell>
              <TableCell align="end" class="tabular-nums font-semibold whitespace-nowrap">{{ formatCurrency(row.lifetimeCommission.amount, row.lifetimeCommission.currency) }}</TableCell>
              <TableCell>
                <StatusPill :variant="influencerStatusTone(row.collaborationStatus)">
                  <span class="w-1.5 h-1.5 rounded-full bg-current" aria-hidden="true" />
                  {{ STATUS_LABELS[row.collaborationStatus] ?? row.collaborationStatus }}
                </StatusPill>
              </TableCell>
              <TableCell align="end">
                <span class="text-brand-700 font-medium">查看</span>
              </TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </TableViewport>

      <!-- Empty state -->
      <div v-else class="flex flex-col items-center gap-2.5 py-8 text-center">
        <div class="w-11 h-11 rounded-kpi bg-surface-subtle text-text-muted grid place-items-center">
          <Search :size="22" />
        </div>
        <b class="text-fs-xl">沒有符合條件的網紅</b>
        <p class="m-0 text-text-muted text-fs-md">調整搜尋條件或新增網紅。</p>
      </div>

      <!-- Footer -->
      <div class="flex items-center justify-between px-4 py-3 border-t border-border">
        <span class="text-fs-base-plus text-text-muted">
          共 <span class="tabular-nums">{{ totalCount }}</span> 位網紅，顯示第
          <span class="tabular-nums">1–{{ filteredItems.length }}</span> 位
        </span>
      </div>
    </Panel>
  </div>
</template>
