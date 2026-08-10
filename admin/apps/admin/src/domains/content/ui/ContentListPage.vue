<script setup lang="ts">
import { computed, ref } from 'vue'
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
import { Plus, FileText } from 'lucide-vue-next'
import { contentListQuery } from '../data/queries'
import { useAdminRuntime } from '../../../app/runtime'

const runtime = useAdminRuntime()
const { data, isPending, isError } = useQuery(contentListQuery(runtime.contentListReader))

type StatusFilter = 'all' | 'published' | 'draft'

const STATUS_VIEWS: { id: StatusFilter; label: string }[] = [
  { id: 'all', label: '全部' },
  { id: 'published', label: '已發佈' },
  { id: 'draft', label: '草稿' },
]

const activeStatus = ref<StatusFilter>('all')
const localeFilter = ref('')

const filteredItems = computed(() => {
  if (!data.value) return []
  let rows = data.value.items
  if (activeStatus.value === 'published') rows = rows.filter(r => r.status === 'published')
  if (activeStatus.value === 'draft') rows = rows.filter(r => r.status === 'draft')
  if (localeFilter.value === 'zh-TW') rows = rows.filter(r => r.locale === '繁體中文')
  if (localeFilter.value === 'en') rows = rows.filter(r => r.locale === 'English')
  return rows
})

const totalCount = computed(() => data.value?.pagination.total ?? 0)
const publishedCount = computed(() => data.value?.items.filter(r => r.status === 'published').length ?? 0)
const draftCount = computed(() => data.value?.items.filter(r => r.status === 'draft').length ?? 0)

function chipCount(id: StatusFilter): number {
  if (id === 'all') return totalCount.value
  if (id === 'published') return publishedCount.value
  return draftCount.value
}
</script>

<template>
  <div v-if="isPending" class="content"><p>載入中…</p></div>
  <div v-else-if="isError" class="content"><p>載入失敗</p></div>
  <div v-else-if="data" class="content space-y-4">
    <PageHeader variant="compact" :title="'內容'" :subtitle="`共 ${totalCount} 篇 · ${draftCount} 篇草稿`">
      <template #action>
        <Button variant="primary" disabled title="編輯功能尚未開放">
          <template #default>
            <span class="inline-flex items-center gap-1">
              <Plus :size="15" />
              新增內容
            </span>
          </template>
        </Button>
      </template>
    </PageHeader>

    <Panel class="overflow-hidden">
      <!-- Chips -->
      <div class="flex items-center gap-2 px-4 py-3 border-b border-border">
        <button
          v-for="view in STATUS_VIEWS"
          :key="view.id"
          type="button"
          class="inline-flex items-center px-2.5 py-1 rounded-full text-fs-base font-medium cursor-pointer"
          :class="{
            'bg-brand-50 text-brand-700 border border-brand-100': activeStatus === view.id,
            'bg-surface-subtle text-text-secondary border border-border hover:bg-surface': activeStatus !== view.id,
          }"
          @click="activeStatus = view.id"
        >
          {{ view.label }} {{ chipCount(view.id) }}
        </button>
      </div>

      <!-- Toolbar -->
      <div class="flex items-center gap-3 px-4 py-3 border-b border-border">
        <select v-model="localeFilter" class="px-3 py-1.5 text-fs-md rounded-admin-lg border border-border bg-surface text-text-primary">
          <option value="">版位：全部</option>
          <option value="zh-TW">繁體中文</option>
          <option value="en">English</option>
        </select>
      </div>

      <!-- Table -->
      <TableViewport v-if="filteredItems.length > 0">
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>標題</TableHead>
              <TableHead>版位</TableHead>
              <TableHead>語系</TableHead>
              <TableHead>最後更新</TableHead>
              <TableHead>狀態</TableHead>
              <TableHead align="end">編輯</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            <TableRow v-for="row in filteredItems" :key="row.id">
              <TableCell>{{ row.title }}</TableCell>
              <TableCell class="text-text-secondary">{{ row.placement }}</TableCell>
              <TableCell class="text-text-secondary">{{ row.locale }}</TableCell>
              <TableCell class="text-text-secondary tabular-nums">{{ row.updatedAt }}</TableCell>
              <TableCell>
                <StatusPill size="compact" :variant="row.status === 'published' ? 'success' : 'wait'">
                  {{ row.status === 'published' ? '已發佈' : '草稿' }}
                </StatusPill>
              </TableCell>
              <TableCell align="end">
                <button disabled class="text-brand-700 hover:underline disabled:opacity-50 disabled:no-underline" title="編輯功能尚未開放">編輯</button>
              </TableCell>
            </TableRow>
          </TableBody>
        </Table>
      </TableViewport>

      <!-- Empty state -->
      <div v-else class="flex flex-col items-center gap-2.5 py-8 text-center">
        <div class="w-11 h-11 rounded-kpi bg-surface-subtle text-text-muted grid place-items-center">
          <FileText :size="22" />
        </div>
        <b class="text-fs-xl">沒有符合條件的內容</b>
        <p class="m-0 text-text-muted text-fs-md">調整篩選條件或新增內容。</p>
      </div>

      <!-- Pagination footer -->
      <div class="flex items-center justify-between px-4 py-3 border-t border-border">
        <p class="text-fs-base text-text-muted m-0">顯示 {{ filteredItems.length === 0 ? 0 : 1 }}-{{ filteredItems.length }} / 共 {{ totalCount }} 篇</p>
      </div>
    </Panel>
  </div>
</template>
