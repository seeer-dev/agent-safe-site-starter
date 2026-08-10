<script setup lang="ts">
import { computed } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useQuery } from '@tanstack/vue-query'
import {
  Button,
  PageHeader,
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
import { messagingQuery } from '../data/queries'
import { useAdminRuntime } from '../../../app/runtime'

const runtime = useAdminRuntime()
const route = useRoute()
const router = useRouter()

const { data, isPending, isError } = useQuery(messagingQuery(runtime.messagingReader))

type MsgTab = 'smtp' | 'templates' | 'attempts'

const MSG_TABS = [
  { id: 'smtp' as const, label: 'SMTP 設定' },
  { id: 'templates' as const, label: '範本管理' },
  { id: 'attempts' as const, label: '發送紀錄' },
]

const activeTab = computed<MsgTab>(() => {
  const segments = route.path.split('/')
  const last = segments[segments.length - 1]
  if (last === 'templates' || last === 'attempts') return last
  return 'smtp'
})

function switchTab(tab: MsgTab) {
  if (tab === 'smtp') router.push('/messaging')
  else router.push(`/messaging/${tab}`)
}
</script>

<template>
  <div v-if="isPending" class="content"><p>載入中…</p></div>
  <div v-else-if="isError" class="content"><p>載入失敗</p></div>
  <div v-else-if="data" class="content">
    <PageHeader variant="compact" title="通知" subtitle="Email 通知設定與範本管理">
      <template #action>
        <Button variant="secondary">寄測試信</Button>
      </template>
    </PageHeader>

    <TabStrip aria-label="通知分頁">
      <button
        v-for="tab in MSG_TABS"
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

    <!-- SMTP tab -->
    <div v-if="activeTab === 'smtp'" data-tab="smtp" class="mt-3.5">
      <Panel class="max-w-[560px] overflow-hidden">
        <div class="px-5 py-3.5 border-b border-border">
          <h3 class="text-fs-lg font-semibold text-text-primary m-0">SMTP 設定</h3>
        </div>
        <div class="p-5 space-y-4">
          <div>
            <label class="block text-fs-md font-medium text-text-primary mb-1.5">伺服器</label>
            <input type="text" :value="data.smtp.host" class="w-full px-3 py-2 text-fs-md rounded-admin-lg border border-border bg-surface text-text-primary" />
          </div>
          <div class="grid grid-cols-2 gap-3">
            <div>
              <label class="block text-fs-md font-medium text-text-primary mb-1.5">Port</label>
              <input type="text" :value="data.smtp.port" class="w-full px-3 py-2 text-fs-md rounded-admin-lg border border-border bg-surface text-text-primary" />
            </div>
            <div>
              <label class="block text-fs-md font-medium text-text-primary mb-1.5">加密</label>
              <select class="w-full px-3 py-2 text-fs-md rounded-admin-lg border border-border bg-surface text-text-primary">
                <option>SSL</option>
                <option selected>TLS</option>
                <option>無</option>
              </select>
            </div>
          </div>
          <div>
            <label class="block text-fs-md font-medium text-text-primary mb-1.5">帳號</label>
            <input type="text" :value="data.smtp.username" class="w-full px-3 py-2 text-fs-md rounded-admin-lg border border-border bg-surface text-text-primary" />
          </div>
          <div>
            <label class="block text-fs-md font-medium text-text-primary mb-1.5">寄件人名稱</label>
            <input type="text" :value="data.smtp.fromName" class="w-full px-3 py-2 text-fs-md rounded-admin-lg border border-border bg-surface text-text-primary" />
          </div>
          <div>
            <label class="block text-fs-md font-medium text-text-primary mb-1.5">寄件人 Email</label>
            <input type="text" :value="data.smtp.fromEmail" class="w-full px-3 py-2 text-fs-md rounded-admin-lg border border-border bg-surface text-text-primary" />
          </div>
        </div>
      </Panel>
    </div>

    <!-- Templates tab -->
    <div v-if="activeTab === 'templates'" data-tab="templates" class="mt-3.5">
      <Panel class="overflow-hidden">
        <TableViewport>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>事件</TableHead>
                <TableHead>範本名稱</TableHead>
                <TableHead>寄件人</TableHead>
                <TableHead>啟用</TableHead>
                <TableHead align="end">編輯/新增</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow v-for="tpl in data.templates" :key="tpl.id">
                <TableCell>{{ tpl.event }}</TableCell>
                <TableCell>{{ tpl.name }}</TableCell>
                <TableCell>{{ tpl.sender }}</TableCell>
                <TableCell>
                  <button
                    class="relative inline-flex h-5 w-9 items-center rounded-full"
                    :class="tpl.enabled ? 'bg-brand' : 'bg-surface-200'"
                    role="switch"
                    :aria-checked="tpl.enabled"
                  >
                    <span
                      class="inline-block h-4 w-4 transform rounded-full bg-white transition-transform"
                      :class="tpl.enabled ? 'translate-x-4' : 'translate-x-0.5'"
                    />
                  </button>
                </TableCell>
                <TableCell align="end">
                  <button class="text-brand-700 hover:underline">{{ tpl.enabled ? '編輯' : '新增' }}</button>
                </TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </TableViewport>
      </Panel>
    </div>

    <!-- Attempts tab -->
    <div v-if="activeTab === 'attempts'" data-tab="history" class="mt-3.5">
      <Panel class="overflow-hidden">
        <TableViewport>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>時間</TableHead>
                <TableHead>事件</TableHead>
                <TableHead>收件人</TableHead>
                <TableHead>狀態</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow v-for="att in data.attempts" :key="att.id">
                <TableCell class="tabular-nums">{{ att.time }}</TableCell>
                <TableCell>{{ att.event }}</TableCell>
                <TableCell>{{ att.recipient }}</TableCell>
                <TableCell>
                  <StatusPill :variant="att.statusTone">{{ att.status }}</StatusPill>
                </TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </TableViewport>
      </Panel>
    </div>
  </div>
</template>
