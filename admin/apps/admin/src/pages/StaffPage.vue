<script setup lang="ts">
import { ref } from 'vue'
import { Button, PageHeader, Panel, StatusPill, TabStrip, TableViewport, Table, TableHeader, TableRow, TableHead, TableBody, TableCell } from '@sitecore/admin-ui'
import { Plus } from 'lucide-vue-next'

type SysTab = 'staff' | 'roles' | 'audit'

const SYS_TABS: { id: SysTab; label: string }[] = [
  { id: 'staff', label: '員工管理' },
  { id: 'roles', label: '角色權限' },
  { id: 'audit', label: '操作紀錄' },
]

const activeTab = ref<SysTab>('staff')

const STAFF_ROWS = [
  { id: 's1', name: '陳怡君', email: 'amy@example.com', role: 'Owner', status: '啟用', statusTone: 'success' as const },
  { id: 's2', name: '林建宏', email: 'lin@example.com', role: 'Admin', status: '啟用', statusTone: 'success' as const },
  { id: 's3', name: '王芳怡', email: 'fang@example.com', role: 'Staff', status: '停用', statusTone: 'muted' as const },
]
</script>

<template>
  <div class="content">
    <PageHeader variant="compact" title="系統設定" subtitle="員工權限與操作紀錄管理">
      <template #action>
        <Button variant="primary">
          <Plus :size="15" />
          新增員工
        </Button>
      </template>
    </PageHeader>

    <TabStrip aria-label="系統設定分頁" class="mt-3.5">
      <button
        v-for="tab in SYS_TABS"
        :key="tab.id"
        type="button"
        class="tab-item px-3.25 py-2 text-fs-md-plus cursor-pointer whitespace-nowrap"
        :class="{
          'tab-item-active font-semibold text-brand-600 border-b-2 border-brand-600 -mb-px': activeTab === tab.id,
          'font-medium text-text-secondary hover:text-text-primary': activeTab !== tab.id,
        }"
        :aria-current="activeTab === tab.id ? 'page' : undefined"
        @click="activeTab = tab.id"
      >
        {{ tab.label }}
      </button>
    </TabStrip>

    <div v-if="activeTab === 'staff'" data-tab="staff" class="mt-3.5">
      <Panel class="overflow-hidden">
        <TableViewport>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>姓名</TableHead>
                <TableHead>Email</TableHead>
                <TableHead>角色</TableHead>
                <TableHead>狀態</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow v-for="row in STAFF_ROWS" :key="row.id">
                <TableCell>{{ row.name }}</TableCell>
                <TableCell>{{ row.email }}</TableCell>
                <TableCell>{{ row.role }}</TableCell>
                <TableCell>
                  <StatusPill size="compact" :variant="row.statusTone">{{ row.status }}</StatusPill>
                </TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </TableViewport>
      </Panel>
    </div>

    <div v-if="activeTab === 'roles'" data-tab="roles" class="mt-3.5">
      <Panel class="overflow-hidden">
        <TableViewport>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>角色</TableHead>
                <TableHead>權限範圍</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow>
                <TableCell>Owner</TableCell>
                <TableCell>全部模組</TableCell>
              </TableRow>
              <TableRow>
                <TableCell>Admin</TableCell>
                <TableCell>訂單、分潤、商品</TableCell>
              </TableRow>
              <TableRow>
                <TableCell>Staff</TableCell>
                <TableCell>訂單唯讀</TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </TableViewport>
      </Panel>
    </div>

    <div v-if="activeTab === 'audit'" data-tab="audit" class="mt-3.5">
      <Panel class="overflow-hidden">
        <TableViewport>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>時間</TableHead>
                <TableHead>操作者</TableHead>
                <TableHead>動作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              <TableRow>
                <TableCell class="tabular-nums">2025-01-14 10:30</TableCell>
                <TableCell>陳怡君</TableCell>
                <TableCell>審核訂單 #TW-20874</TableCell>
              </TableRow>
              <TableRow>
                <TableCell class="tabular-nums">2025-01-14 09:15</TableCell>
                <TableCell>林建宏</TableCell>
                <TableCell>撥款 $5,800</TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </TableViewport>
      </Panel>
    </div>
  </div>
</template>
