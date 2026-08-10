<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import {
  Button,
  PageHeader,
  Panel,
  Checkbox,
} from '@sitecore/admin-ui'
import { ArrowLeft } from 'lucide-vue-next'

const router = useRouter()

const displayName = ref('')
const email = ref('')
const enableAccount = ref(true)
const sendInvite = ref(true)
const validationError = ref<string | null>(null)

const READONLY_NOTICE = '提交功能尚未開放，此表單僅為草稿預覽。'

function goBack() {
  router.push('/influencers')
}

function handleSubmit() {
  validationError.value = null
  const name = displayName.value.trim()
  const mail = email.value.trim()
  if (!name) {
    validationError.value = '請輸入顯示名稱'
    return
  }
  if (!mail) {
    validationError.value = '請輸入聯絡 Email'
    return
  }
  // No mutation hook exists — do not fake success.
  validationError.value = '提交功能尚未開放，無法建立網紅。'
}
</script>

<template>
  <div class="content space-y-4">
    <button
      type="button"
      class="back-link inline-flex items-center gap-1.5 text-text-secondary hover:text-text-primary text-fs-md mb-2.5"
      @click="goBack"
    >
      <ArrowLeft :size="15" />
      返回列表
    </button>

    <PageHeader variant="compact" title="新增網紅" />

    <div v-if="validationError" class="rounded-panel border border-danger-border bg-danger-bg text-danger-fg px-4 py-3 text-fs-md" role="alert">
      {{ validationError }}
    </div>

    <div class="grid grid-cols-1 lg:grid-cols-[1fr_280px] gap-4">
      <!-- Left: form panels -->
      <div class="space-y-4">
        <!-- 基本資料 -->
        <Panel>
          <div class="px-4 py-3 border-b border-border">
            <h2 class="text-fs-xl font-semibold text-text-primary m-0">基本資料</h2>
          </div>
          <div class="p-5 space-y-4">
            <label class="block">
              <span class="mb-1.5 block text-fs-md font-medium text-text-primary">
                顯示名稱 <span class="text-danger-fg">*</span>
              </span>
              <input
                v-model="displayName"
                type="text"
                class="w-full px-3 py-2 text-fs-md rounded-input border border-border bg-surface text-text-primary placeholder:text-text-muted focus:outline-none focus:border-brand-600"
                placeholder="請輸入網紅名稱"
              />
            </label>
            <label class="block">
              <span class="mb-1.5 block text-fs-md font-medium text-text-primary">
                聯絡 Email <span class="text-danger-fg">*</span>
              </span>
              <input
                v-model="email"
                type="email"
                class="w-full px-3 py-2 text-fs-md rounded-input border border-border bg-surface text-text-primary placeholder:text-text-muted focus:outline-none focus:border-brand-600"
                placeholder="name@example.com"
              />
            </label>
          </div>
        </Panel>

        <!-- 後台帳號 -->
        <Panel>
          <div class="px-4 py-3 border-b border-border">
            <h2 class="text-fs-xl font-semibold text-text-primary m-0">後台帳號</h2>
          </div>
          <div class="p-5 space-y-4">
            <div class="flex items-center justify-between">
              <div>
                <p class="text-fs-md font-medium text-text-primary">啟用後台帳號</p>
                <p class="text-fs-base text-text-muted">為此網紅建立可登入的後台帳號</p>
              </div>
              <button
                type="button"
                role="switch"
                :aria-checked="enableAccount"
                class="relative inline-flex h-5 w-9 items-center rounded-full transition-colors"
                :class="enableAccount ? 'bg-brand' : 'bg-border'"
                @click="enableAccount = !enableAccount"
              >
                <span
                  class="inline-block h-4 w-4 transform rounded-full bg-white transition-transform"
                  :class="enableAccount ? 'translate-x-4' : 'translate-x-0.5'"
                />
              </button>
            </div>
            <label v-if="enableAccount" class="flex items-center gap-2">
              <Checkbox v-model="sendInvite" />
              <span class="text-fs-md text-text-secondary">寄送邀請通知</span>
            </label>
            <label class="block">
              <span class="mb-1.5 block text-fs-md font-medium text-text-primary">初始密碼</span>
              <input
                type="text"
                class="w-full px-3 py-2 text-fs-md rounded-input border border-border bg-surface text-text-primary placeholder:text-text-muted focus:outline-none focus:border-brand-600"
                placeholder="留空將自動產生"
                disabled
                title="初始密碼由系統產生"
              />
            </label>
          </div>
        </Panel>

        <!-- 預設折扣碼 -->
        <Panel>
          <div class="px-4 py-3 border-b border-border">
            <h2 class="text-fs-xl font-semibold text-text-primary m-0">預設折扣碼</h2>
          </div>
          <div class="p-5 space-y-4">
            <div class="rounded-panel border border-warning-border bg-warning-bg text-warning-fg px-4 py-3 text-fs-md">
              折扣碼功能尚未支援，建立時無法設定。
            </div>
            <label class="block">
              <span class="mb-1.5 block text-fs-md font-medium text-text-primary">折扣碼</span>
              <input
                type="text"
                class="w-full px-3 py-2 text-fs-md rounded-input border border-border bg-surface text-text-primary placeholder:text-text-muted focus:outline-none focus:border-brand-600"
                placeholder="例如 SUMMER01"
                disabled
                title="折扣碼功能尚未開放"
              />
            </label>
          </div>
        </Panel>
      </div>

      <!-- Right: aside -->
      <aside class="space-y-4">
        <Panel>
          <div class="px-4 py-3 border-b border-border">
            <h2 class="text-fs-xl font-semibold text-text-primary m-0">建立須知</h2>
          </div>
          <div class="p-5">
            <ul class="space-y-2.5 text-fs-md text-text-secondary m-0 list-none p-0">
              <li class="flex gap-2">
                <span class="text-brand-700 shrink-0">•</span>
                <span>網紅建立後可參與檔期並獲得分潤</span>
              </li>
              <li class="flex gap-2">
                <span class="text-brand-700 shrink-0">•</span>
                <span>啟用帳號後網紅可登入後台查看成績</span>
              </li>
              <li class="flex gap-2">
                <span class="text-brand-700 shrink-0">•</span>
                <span>折扣碼需在檔期中指派</span>
              </li>
              <li class="flex gap-2">
                <span class="text-brand-700 shrink-0">•</span>
                <span>分潤比例依檔期設定</span>
              </li>
              <li class="flex gap-2">
                <span class="text-brand-700 shrink-0">•</span>
                <span>建立後可隨時停用或重新啟用</span>
              </li>
            </ul>
          </div>
        </Panel>
      </aside>
    </div>

    <!-- Footer buttons -->
    <div class="flex justify-end gap-2 pt-4 border-t border-border" data-footer-actions>
      <Button variant="secondary" @click="goBack">取消</Button>
      <Button variant="primary" disabled :title="READONLY_NOTICE" @click="handleSubmit">建立網紅</Button>
    </div>
  </div>
</template>
