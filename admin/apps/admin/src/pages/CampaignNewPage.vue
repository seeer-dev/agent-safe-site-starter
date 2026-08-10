<script setup lang="ts">
import { ref } from 'vue'
import { PageHeader, Panel, Button } from '@sitecore/admin-ui'

const CAMPAIGN_STEPS = [
  { label: '基本資料', state: 'current' as const },
  { label: '分潤設定', state: 'pending' as const },
  { label: '確認', state: 'pending' as const },
]

const campaignName = ref('')
const dateStart = ref('')
const dateEnd = ref('')
const description = ref('')
</script>

<template>
  <div class="content">
    <PageHeader title="新增檔期" subtitle="建立新的行銷檔期與分潤規則" />

    <div class="stepper flex items-center gap-2 py-2 mt-3.5 overflow-auto">
      <template v-for="(step, index) in CAMPAIGN_STEPS" :key="step.label">
        <div
          class="step"
          :class="{ 'step-current': step.state === 'current' }"
        >
          <span class="step-dot">{{ index + 1 }}</span>
          {{ step.label }}
        </div>
        <div v-if="index < CAMPAIGN_STEPS.length - 1" class="step-line" aria-hidden="true" />
      </template>
    </div>

    <Panel class="mt-3.5 overflow-hidden">
      <div class="panel-hd px-4 py-3 border-b border-border text-fs-lg font-semibold text-text-primary">
        基本資料
      </div>
      <div class="p-4 space-y-4">
        <div>
          <label class="block text-fs-md font-medium text-text-primary mb-1.5">檔期名稱</label>
          <input v-model="campaignName" type="text" placeholder="例如：夏季滿額贈" class="w-full px-3 py-2 text-fs-md border border-border rounded-input bg-surface text-text-primary" />
        </div>
        <div class="grid grid-cols-2 gap-3">
          <div>
            <label class="block text-fs-md font-medium text-text-primary mb-1.5">開始日期</label>
            <input v-model="dateStart" type="date" class="w-full px-3 py-2 text-fs-md border border-border rounded-input bg-surface text-text-primary" />
          </div>
          <div>
            <label class="block text-fs-md font-medium text-text-primary mb-1.5">結束日期</label>
            <input v-model="dateEnd" type="date" class="w-full px-3 py-2 text-fs-md border border-border rounded-input bg-surface text-text-primary" />
          </div>
        </div>
        <div>
          <label class="block text-fs-md font-medium text-text-primary mb-1.5">檔期描述</label>
          <textarea v-model="description" rows="3" placeholder="描述此檔期的目標與規則..." class="w-full px-3 py-2 text-fs-md border border-border rounded-input bg-surface text-text-primary" />
        </div>
      </div>
    </Panel>

    <div class="sticky bottom-0 flex items-center justify-between px-4 py-3 border-t border-border bg-surface mt-3.5">
      <span class="text-fs-base text-text-muted">步驟 1 / 3</span>
      <div class="flex items-center gap-2">
        <Button variant="secondary">取消</Button>
        <Button variant="primary">下一步</Button>
      </div>
    </div>
  </div>
</template>
