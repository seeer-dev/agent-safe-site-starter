<script setup lang="ts">
import { useQuery } from '@tanstack/vue-query'
import { PageHeader, Panel, Button, StatusPill } from '@sitecore/admin-ui'
import { dashboardSummaryQuery } from '../domains/dashboard/data/queries'
import { useAdminRuntime } from '../app/runtime'

const runtime = useAdminRuntime()
const { data, isPending, isError } = useQuery(dashboardSummaryQuery(runtime.dashboardSummaryReader))
</script>

<template>
  <div v-if="isPending" class="content"><p>載入中…</p></div>
  <div v-else-if="isError" class="content"><p>載入失敗</p></div>
  <div v-else-if="data" class="content">
    <PageHeader :title="'今日待辦'" :subtitle="'把需要你處理的事集中在這裡，不用自己翻各個選單。'">
      <template #actions>
        <Button variant="secondary">自訂待辦</Button>
      </template>
    </PageHeader>

    <details class="guide" open>
      <summary class="guide-summary">這個工作台在做什麼？</summary>
      <div class="guide-body">依你的角色把待審核訂單、待出貨、可釋放與待撥款分潤集中呈現。數字全部來自 dashboard-summary 已回傳的欄位，不是估的。</div>
    </details>

    <div class="grid grid-cols-4 gap-3.5 my-4 max-lg:grid-cols-2 max-sm:grid-cols-1">
      <div v-for="card in data.cards" :key="card.id" class="card">
        <div
          class="card-icon"
          :style="{
            background: card.variant === 'warning' ? 'var(--admin-warning-bg)' : card.variant === 'info' ? 'var(--admin-info-bg)' : card.variant === 'success' ? 'var(--admin-success-bg)' : 'var(--admin-danger-bg)',
            color: card.variant === 'warning' ? 'var(--admin-warning-fg)' : card.variant === 'info' ? 'var(--admin-info-fg)' : card.variant === 'success' ? 'var(--admin-success-fg)' : 'var(--admin-danger-fg)',
          }"
        >
          <svg viewBox="0 0 24 24" width="18" height="18"><circle cx="12" cy="12" r="9"/></svg>
        </div>
        <div class="card-num">{{ card.num }}</div>
        <div class="card-label">{{ card.label }}</div>
        <a class="card-cta">{{ card.cta }}</a>
      </div>
    </div>

    <div class="grid grid-cols-[1.35fr_1fr] gap-4 max-lg:grid-cols-1">
      <Panel title="需要你決定">
        <div
          v-for="(task, index) in data.tasks"
          :key="task.id"
          class="task-row"
          :style="index < data.tasks.length - 1 ? { borderBottom: '1px solid var(--admin-border)' } : {}"
        >
          <div class="task-text flex-1 min-w-0">
            <b class="font-semibold text-fs-md-plus">{{ task.title }}</b>
            <div class="text-text-muted text-fs-base-plus">{{ task.meta }}</div>
          </div>
          <Button v-if="task.secondaryAction" variant="secondary" size="sm">{{ task.secondaryAction }}</Button>
          <Button v-if="task.primaryAction" variant="primary" size="sm">{{ task.primaryAction }}</Button>
        </div>
      </Panel>
      <Panel title="最近帶貨訂單">
        <div
          v-for="(order, index) in data.recentOrders"
          :key="order.id"
          class="row-item"
          :style="index < data.recentOrders.length - 1 ? { borderBottom: '1px solid var(--admin-border)' } : {}"
        >
          <StatusPill :variant="order.statusVariant">{{ order.statusLabel }}</StatusPill>
          <div>
            <div class="oid font-semibold text-fs-md">{{ order.orderId }}</div>
            <div class="text-text-muted text-fs-base">{{ order.customer }}</div>
          </div>
          <span class="amt ml-auto">{{ order.amount }}</span>
        </div>
      </Panel>
    </div>
  </div>
</template>
