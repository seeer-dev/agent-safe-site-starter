<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import Panel from '@/components/ui/Panel.vue'
import Input from '@/components/ui/Input.vue'
import Checkbox from '@/components/ui/Checkbox.vue'
import Button from '@/components/ui/Button.vue'
import { api } from '@/lib/api-client'

interface SettingsRow {
  draft: Record<string, unknown>
  published: Record<string, unknown>
  version: number
  draft_updated_unix: number
  published_unix: number
}

const DEFAULTS = {
  storeName: '',
  storeNameEn: '',
  tagline: '',
  promoBanner: '',
  promoBannerEnabled: false,
  freeShippingThreshold: 0,
  lowStockThreshold: 5,
  notificationMaster: true,
  contactEmail: '',
  contactPhone: '',
}

const row = ref<SettingsRow | null>(null)
const form = ref<Record<string, any>>({ ...DEFAULTS })
const loading = ref(false)
const saving = ref(false)
const publishing = ref(false)
const error = ref<string | null>(null)
const notice = ref<string | null>(null)

const dirty = computed(() => {
  if (!row.value) return false
  return JSON.stringify(form.value) !== JSON.stringify({ ...DEFAULTS, ...(row.value.draft ?? {}) })
})
const unpublishedChanges = computed(() => {
  if (!row.value) return false
  return JSON.stringify(row.value.draft ?? {}) !== JSON.stringify(row.value.published ?? {})
})

async function load() {
  loading.value = true
  error.value = null
  try {
    row.value = await api.get<SettingsRow>('/admin/store-settings')
    form.value = { ...DEFAULTS, ...(row.value.draft ?? {}) }
  } catch (e: any) {
    error.value = e?.message ?? '載入失敗'
  } finally {
    loading.value = false
  }
}

async function save() {
  if (!row.value || saving.value) return
  saving.value = true
  error.value = null
  notice.value = null
  try {
    row.value = await api.put<SettingsRow>('/admin/store-settings', {
      draft: form.value,
      expected_version: row.value.version,
    })
    form.value = { ...DEFAULTS, ...(row.value.draft ?? {}) }
    notice.value = '草稿已儲存（尚未發布）'
  } catch (e: any) {
    error.value = e?.message ?? '儲存失敗'
    if (e?.status === 409) await load()
  } finally {
    saving.value = false
  }
}

async function publish() {
  if (!row.value || publishing.value) return
  publishing.value = true
  error.value = null
  notice.value = null
  try {
    if (dirty.value) await save()
    row.value = await api.post<SettingsRow>('/admin/store-settings/publish', {
      expected_draft_version: row.value!.version,
    })
    form.value = { ...DEFAULTS, ...(row.value.draft ?? {}) }
    notice.value = '已發布到前台'
  } catch (e: any) {
    error.value = e?.message ?? '發布失敗'
    if (e?.status === 409) await load()
  } finally {
    publishing.value = false
  }
}

function fmtUnix(u: number): string {
  if (!u) return '—'
  return new Date(u * 1000).toLocaleString('zh-TW')
}

onMounted(load)
</script>

<template>
  <div class="page">
    <header class="phd-row">
      <div>
        <h2>商店設定</h2>
        <p class="sub">前台顯示的商店資訊與門檻設定。先存草稿，再發布到前台。</p>
      </div>
      <div class="actions">
        <Button variant="sec" :disabled="loading || saving || !dirty" @click="save">
          {{ saving ? '儲存中…' : '儲存草稿' }}
        </Button>
        <Button variant="pri" :disabled="loading || publishing || (!dirty && !unpublishedChanges)" @click="publish">
          {{ publishing ? '發布中…' : '發布' }}
        </Button>
      </div>
    </header>

    <p v-if="error" class="err">{{ error }}</p>
    <p v-if="notice" class="ok">{{ notice }}</p>

    <div v-if="loading" class="sub">載入中…</div>
    <template v-else>
      <Panel title="商店資訊">
        <div class="grid2">
          <label class="fld"><span>商店名稱</span><Input v-model="form.storeName" /></label>
          <label class="fld"><span>英文名稱</span><Input v-model="form.storeNameEn" /></label>
          <label class="fld wide"><span>標語</span><Input v-model="form.tagline" /></label>
          <label class="fld"><span>聯絡 Email</span><Input v-model="form.contactEmail" /></label>
          <label class="fld"><span>聯絡電話</span><Input v-model="form.contactPhone" /></label>
        </div>
      </Panel>

      <Panel title="公告列">
        <div class="grid2">
          <label class="fld wide"><span>公告文字</span><Input v-model="form.promoBanner" /></label>
          <label class="fld row"><Checkbox v-model:checked="form.promoBannerEnabled" /><span>啟用公告列</span></label>
        </div>
      </Panel>

      <Panel title="門檻與通知">
        <div class="grid2">
          <label class="fld"><span>免運門檻（NT$）</span><Input :model-value="String(form.freeShippingThreshold)" type="number" @update:model-value="form.freeShippingThreshold = Number($event) || 0" /></label>
          <label class="fld"><span>低庫存警示（件）</span><Input :model-value="String(form.lowStockThreshold)" type="number" @update:model-value="form.lowStockThreshold = Number($event) || 0" /></label>
          <label class="fld row"><Checkbox v-model:checked="form.notificationMaster" /><span>啟用訂單通知信</span></label>
        </div>
      </Panel>

      <Panel v-if="row" title="版本狀態">
        <div class="meta">
          <span>草稿版本 v{{ row.version }}（{{ fmtUnix(row.draft_updated_unix) }}）</span>
          <span>前台發布於 {{ fmtUnix(row.published_unix) }}</span>
          <span v-if="unpublishedChanges" class="warn">有未發布的草稿變更</span>
          <span v-else class="ok">前台已是最新</span>
        </div>
      </Panel>
    </template>
  </div>
</template>

<style scoped>
.page { display: flex; flex-direction: column; gap: 16px; }
.phd-row { display: flex; align-items: flex-start; justify-content: space-between; gap: 16px; }
.sub { color: var(--muted); font-size: 13px; margin: 4px 0 0; }
.actions { display: flex; gap: 8px; }
.err { color: var(--danger, #c0392b); }
.ok { color: var(--ok, #1a7f37); }
.warn { color: var(--warn, #b45309); }
.grid2 { display: grid; grid-template-columns: 1fr 1fr; gap: 12px 20px; padding: 4px 0 8px; }
.fld { display: flex; flex-direction: column; gap: 4px; font-size: 13px; }
.fld > span { color: var(--muted); }
.fld.wide { grid-column: 1 / -1; }
.fld.row { flex-direction: row; align-items: center; gap: 8px; }
.meta { display: flex; gap: 20px; flex-wrap: wrap; font-size: 13px; color: var(--muted); }
@media (max-width: 720px) { .grid2 { grid-template-columns: 1fr; } }
</style>
