<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query'
import { PageHeader, Panel, Button, Chip, ChipRow, StatusPill, TableViewport, Table, TableHeader, TableRow, TableHead, TableBody, TableCell } from '@sitecore/admin-ui'
import { Download } from 'lucide-vue-next'
import { orderListQuery } from '../data/order-list-query'
import { useAdminRuntime } from '../../../app/runtime'

const runtime = useAdminRuntime()
const { data, isPending, isError } = useQuery(orderListQuery(runtime.orderListReader))
</script>

<template>
  <div v-if="isPending" class="content"><p>載入中…</p></div>
  <div v-else-if="isError" class="content"><p>載入失敗</p></div>
  <div v-else-if="data" class="content">
    <PageHeader :title="'訂單'" :subtitle="`${data.pendingReviewCount} 筆待審核`">
      <template #action>
        <Button variant="secondary">
          <Download :size="15" />
          匯出
        </Button>
      </template>
    </PageHeader>

    <Panel class="overflow-hidden">
      <ChipRow aria-label="訂單 saved views" class="border-b border-border p-3">
        <Chip
          v-for="view in data.savedViews"
          :key="view.id"
          :active="view.id === 'pending-review'"
          :count="view.count"
        >
          {{ view.label }}
        </Chip>
        <Chip add disabled title="尚未提供個人 saved-view 持久化契約">存為新檢視</Chip>
      </ChipRow>

      <div class="toolbar flex items-center gap-1 px-3.5 py-3 border-b border-border flex-wrap">
        <span class="btn btn-ghost btn-sm inline-flex items-center gap-1 px-2 py-1 rounded-btn text-fs-base text-text-secondary cursor-pointer" style="pointer-events:none">
          搜尋訂單編號 / 買家 / 折扣碼
        </span>
        <span class="btn btn-ghost btn-sm inline-flex items-center gap-1 px-2 py-1 rounded-btn text-fs-base text-text-secondary cursor-pointer hover:bg-surface-subtle">
          狀態
        </span>
        <span class="btn btn-ghost btn-sm inline-flex items-center gap-1 px-2 py-1 rounded-btn text-fs-base text-text-secondary cursor-pointer hover:bg-surface-subtle">
          檔期
        </span>
        <span class="btn btn-ghost btn-sm inline-flex items-center gap-1 px-2 py-1 rounded-btn text-fs-base text-text-secondary cursor-pointer hover:bg-surface-subtle">
          付款狀態
        </span>
      </div>

      <TableViewport aria-label="訂單列表">
        <Table aria-label="訂單列表">
          <TableHeader>
            <TableRow>
              <TableHead style="width:34px" />
              <TableHead>訂單編號</TableHead>
              <TableHead>買家</TableHead>
              <TableHead>分潤網紅</TableHead>
              <TableHead>金額</TableHead>
              <TableHead>狀態</TableHead>
              <TableHead align="end">動作</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow v-if="data.rows.length === 0">
              <TableCell :colspan="7" align="center">目前沒有符合條件的訂單</TableCell>
            </TableRow>
            <TableRow v-for="row in data.rows" :key="row.id" interactive>
              <TableCell />
              <TableCell><strong>{{ row.orderNumber }}</strong></TableCell>
              <TableCell>{{ row.buyerName }}</TableCell>
              <TableCell>
                {{ row.influencerName }}
                <small v-if="row.couponCode" class="ml-1 text-text-muted">{{ row.couponCode }}</small>
              </TableCell>
              <TableCell align="end">{{ row.amount }}</TableCell>
              <TableCell>
                <StatusPill :variant="row.progressTone">{{ row.progressLabel }}</StatusPill>
              </TableCell>
              <TableCell align="end">
                <span class="inline-flex items-center justify-end gap-2">
                  <Button variant="ghost" size="sm">明細</Button>
                  <Button v-if="row.canReview" size="sm">審核</Button>
                </span>
              </TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </TableViewport>

      <div class="tfoot flex items-center justify-between py-2.75 px-3.5 text-text-muted text-fs-base-plus">
        <span>第 1 / 3 頁 · 共 {{ data.rows.length }} 筆</span>
        <span class="flex gap-1">
          <Button variant="ghost" size="sm" disabled>‹</Button>
          <span class="font-bold text-brand-700">1</span>
          <Button variant="ghost" size="sm">›</Button>
        </span>
      </div>
    </Panel>
  </div>
</template>
