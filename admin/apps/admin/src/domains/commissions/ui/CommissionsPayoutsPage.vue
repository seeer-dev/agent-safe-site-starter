<script setup lang="ts">
import { ref } from 'vue'
import { useQuery } from '@tanstack/vue-query'
import {
  Amount,
  Button,
  Muted,
  Notice,
  PageHeader,
  Panel,
  PanelBody,
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
import { Download } from 'lucide-vue-next'
import { commissionPayoutQuery } from '../data/queries'
import { useAdminRuntime } from '../../../app/runtime'

const props = withDefaults(defineProps<{ defaultTab?: 'commissions' | 'payouts' }>(), {
  defaultTab: 'commissions',
})

const runtime = useAdminRuntime()
const { data, isPending, isError } = useQuery(commissionPayoutQuery(runtime.commissionPayoutReader))

const activeTab = ref<'commissions' | 'payouts'>(props.defaultTab)

const COMM_TABS = [
  { id: 'commissions' as const, label: '分潤明細', count: data.value?.commissionCount ?? 3 },
  { id: 'payouts' as const, label: '撥款紀錄' },
]
</script>

<template>
  <div v-if="isPending" class="content"><p>載入中…</p></div>
  <div v-else-if="isError" class="content"><p>載入失敗</p></div>
  <div v-else-if="data" class="content">
    <PageHeader
      title="分潤與撥款"
      subtitle="每一筆訂單的分潤明細與結算撥款紀錄都在這裡"
    />

    <TabStrip aria-label="分潤與撥款分頁">
      <button
        v-for="tab in COMM_TABS"
        :key="tab.id"
        type="button"
        class="tab-item"
        :class="{
          'tab-item-active': activeTab === tab.id,
          'text-brand-600': activeTab === tab.id && props.defaultTab === 'payouts',
        }"
        :aria-current="activeTab === tab.id ? 'page' : undefined"
        @click="activeTab = tab.id"
      >
        {{ tab.label }}
        <span v-if="tab.count != null" class="tab-count">{{ tab.count }}</span>
      </button>
    </TabStrip>

    <div v-if="activeTab === 'commissions'" class="space-y-3 p-4">
      <div class="flex justify-end">
        <Button variant="secondary">
          <Download :size="15" />
          匯出
        </Button>
      </div>
      <Panel>
        <PanelBody class="!p-0">
          <TableViewport aria-label="分潤明細">
            <Table striped aria-label="分潤明細">
              <TableHeader>
                <TableRow>
                  <TableHead>訂單</TableHead>
                  <TableHead>網紅</TableHead>
                  <TableHead>檔期</TableHead>
                  <TableHead>SKU</TableHead>
                  <TableHead align="end">基準</TableHead>
                  <TableHead align="end">分潤</TableHead>
                  <TableHead>狀態（後台）</TableHead>
                  <TableHead>網紅端顯示</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                <TableRow v-if="data.commissionRows.length === 0">
                  <TableCell :colspan="8" align="center">目前沒有分潤明細</TableCell>
                </TableRow>
                <TableRow v-for="row in data.commissionRows" :key="row.id" interactive>
                  <TableCell><strong>{{ row.order }}</strong></TableCell>
                  <TableCell>{{ row.influencer }}</TableCell>
                  <TableCell>{{ row.campaign }}</TableCell>
                  <TableCell>{{ row.sku }}</TableCell>
                  <TableCell align="end"><Amount>{{ row.base }}</Amount></TableCell>
                  <TableCell align="end"><Amount>{{ row.commission }}</Amount></TableCell>
                  <TableCell>
                    <StatusPill class="status-pill-block" :variant="row.status === '可撥款' ? 'wait' : 'ship'">
                      {{ row.status }}
                    </StatusPill>
                  </TableCell>
                  <TableCell><Muted>{{ row.influencerDisplay }}</Muted></TableCell>
                </TableRow>
              </TableBody>
            </Table>
          </TableViewport>
          <Notice>
            後台看七個狀態，<strong>網紅端只看兩個</strong>：待撥款 / 已撥款。
            其餘階段網紅端不顯示金額（因為預估值會被退貨改動）。
          </Notice>
        </PanelBody>
      </Panel>

      <Panel>
        <div class="panel-hd flex items-center gap-2.5 px-4 py-3 border-b border-border">
          <h3 class="m-0 text-fs-xl font-semibold">狀態流程</h3>
        </div>
        <div class="px-4 py-4">
          <div
            class="stepper flex items-center gap-2 py-2 border-none overflow-auto"
            role="list"
            aria-label="分潤狀態流程"
          >
            <template v-for="(step, index) in data.workflowSteps" :key="step.label">
              <div
                class="step"
                :class="{
                  'step-done': step.state === 'done',
                  'step-current': step.state === 'current',
                }"
                role="listitem"
                :aria-current="step.state === 'current' ? 'step' : undefined"
              >
                <span class="step-dot">{{ index + 1 }}</span>
                {{ step.label }}
              </div>
              <div
                v-if="index < data.workflowSteps.length - 1"
                class="step-line"
                aria-hidden="true"
              />
            </template>
          </div>
        </div>
      </Panel>
    </div>

    <div v-if="activeTab === 'payouts'" class="space-y-3 p-4">
      <Panel>
        <PanelBody class="!p-0">
          <TableViewport aria-label="撥款紀錄">
            <Table striped aria-label="撥款紀錄">
              <TableHeader>
                <TableRow>
                  <TableHead>日期</TableHead>
                  <TableHead>檔期</TableHead>
                  <TableHead>網紅</TableHead>
                  <TableHead align="end">金額</TableHead>
                  <TableHead align="end">明細</TableHead>
                  <TableHead>方式</TableHead>
                  <TableHead>備註</TableHead>
                  <TableHead>經辦</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                <TableRow v-if="data.payoutRows.length === 0">
                  <TableCell :colspan="8" align="center">目前沒有撥款紀錄</TableCell>
                </TableRow>
                <TableRow v-for="row in data.payoutRows" :key="row.id" interactive>
                  <TableCell>{{ row.date }}</TableCell>
                  <TableCell>{{ row.campaign }}</TableCell>
                  <TableCell>{{ row.influencer }}</TableCell>
                  <TableCell align="end"><Amount>{{ row.amount }}</Amount></TableCell>
                  <TableCell align="end">{{ row.lines }}</TableCell>
                  <TableCell><Muted>{{ row.method }}</Muted></TableCell>
                  <TableCell><Muted>{{ row.note }}</Muted></TableCell>
                  <TableCell><Muted>{{ row.operator }}</Muted></TableCell>
                </TableRow>
              </TableBody>
            </Table>
          </TableViewport>
        </PanelBody>
      </Panel>
    </div>
  </div>
</template>
