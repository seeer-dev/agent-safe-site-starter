<script setup lang="ts">
import { ref } from 'vue'
import { Button, PageHeader } from '@sitecore/admin-ui'

const triggerRule = ref('滿額贈')
const contentText = ref('')
const subText = ref('')
const linkUrl = ref('')
const dateStart = ref('2025-07-01')
const dateEnd = ref('2025-07-31')
const enabled = ref(true)
</script>

<template>
  <div class="content">
    <PageHeader variant="compact" title="公告" />

    <div class="grid lg:grid-cols-[1fr_320px] gap-4 max-lg:block">
      <section class="panel block border border-border rounded-panel bg-surface overflow-hidden">
        <div class="px-4 py-3 border-b border-border">
          <h2 class="text-fs-xl font-semibold text-text-primary m-0">公告設定</h2>
        </div>
        <div class="p-4 space-y-4">
          <div>
            <label class="block text-fs-md text-text-secondary mb-1.5">觸發條件規則</label>
            <select v-model="triggerRule" class="w-full px-3 py-2 text-fs-md border border-border rounded-input bg-surface text-text-primary">
              <option>無</option>
              <option>滿額贈</option>
            </select>
          </div>
          <div>
            <label class="block text-fs-md text-text-secondary mb-1.5">內容文字</label>
            <input v-model="contentText" type="text" placeholder="標題" class="w-full px-3 py-2 text-fs-md border border-border rounded-input bg-surface text-text-primary" />
          </div>
          <div>
            <label class="block text-fs-md text-text-secondary mb-1.5">副文字</label>
            <input v-model="subText" type="text" placeholder="贈品" class="w-full px-3 py-2 text-fs-md border border-border rounded-input bg-surface text-text-primary" />
          </div>
          <div>
            <label class="block text-fs-md text-text-secondary mb-1.5">連結</label>
            <input v-model="linkUrl" type="text" placeholder="https://..." class="w-full px-3 py-2 text-fs-md border border-border rounded-input bg-surface text-text-primary" />
          </div>
          <div>
            <label class="block text-fs-md text-text-secondary mb-1.5">顯示期間</label>
            <div class="flex items-center gap-2">
              <input v-model="dateStart" type="date" class="px-3 py-2 text-fs-md border border-border rounded-input bg-surface text-text-primary" />
              <span class="text-text-muted text-fs-md">至</span>
              <input v-model="dateEnd" type="date" class="px-3 py-2 text-fs-md border border-border rounded-input bg-surface text-text-primary" />
            </div>
          </div>
          <div class="flex items-center justify-between">
            <label class="text-fs-md text-text-secondary">顯示開關</label>
            <button
              class="relative w-10 h-6 rounded-full transition-colors"
              :class="enabled ? 'bg-brand' : 'bg-surface-200'"
              role="switch"
              :aria-checked="enabled"
              @click="enabled = !enabled"
            >
              <span
                class="absolute top-0.5 w-5 h-5 rounded-full bg-white shadow transition-transform"
                :class="enabled ? 'left-[1.125rem]' : 'left-0.5'"
              />
            </button>
          </div>
        </div>
        <div class="flex items-center justify-between px-4 py-3 border-t border-border bg-surface-subtle">
          <span class="text-fs-base text-warning-fg font-medium">尚有未儲存變更</span>
          <div class="flex items-center gap-2">
            <Button variant="secondary">預覽</Button>
            <Button variant="primary">儲存</Button>
          </div>
        </div>
      </section>

      <section class="panel block border border-border rounded-panel bg-surface overflow-hidden">
        <div class="px-4 py-3 border-b border-border">
          <h2 class="text-fs-xl font-semibold text-text-primary m-0">即時預覽</h2>
        </div>
        <div class="p-4 space-y-3">
          <div class="bg-brand-600 text-white px-4 py-2 rounded text-fs-md font-medium">
            即將免運！滿額贈好禮
          </div>
          <p class="text-fs-base text-text-muted m-0">買家在所有商品頁面都會看到此公告。</p>
        </div>
      </section>
    </div>
  </div>
</template>
