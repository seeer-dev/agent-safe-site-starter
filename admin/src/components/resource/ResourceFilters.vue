<script setup lang="ts">
// 資源列表篩選工具列 — select 篩選用 dropdown menu（參考稿模式），
// text 篩選維持模糊包含輸入。optsSource 型篩選的選項由外層載入後經
// filterOpts 傳入。
import DropdownMenu from '@/components/ui/DropdownMenu.vue'
import Input from '@/components/ui/Input.vue'
import type { FilterDef, MenuItem } from '@/lib/types'

const props = defineProps<{
  filters: FilterDef[]
  modelValue: Record<string, string>
  /** filter key -> [value, label] 選項（optsSource 動態載入） */
  filterOpts?: Record<string, [string, string][]>
  pageSize: number
  filteredCount?: number
  totalCount?: number
}>()

const emit = defineEmits<{
  'update:modelValue': [v: Record<string, string>]
}>()

function set(k: string, v: string) {
  emit('update:modelValue', { ...props.modelValue, [k]: v })
}

function optsFor(f: FilterDef): [string, string][] {
  if (f.w !== 'select') return []
  const dyn = props.filterOpts?.[f.k]
  if (dyn) return [['', `全部${f.l}`], ...dyn]
  return f.opts ?? []
}

/** Dropdown items for a select filter — current value is check-marked. */
function menuItemsFor(f: FilterDef): MenuItem[] {
  const cur = props.modelValue[f.k] ?? ''
  return optsFor(f).map(([value, label]) => ({
    key: value === '' ? '__all__' : value,
    label,
    checked: value === cur,
  }))
}

/** Trigger label for a select filter — the current option's label. */
function filterLabel(f: FilterDef): string {
  const cur = props.modelValue[f.k] ?? ''
  const found = optsFor(f).find(([v]) => v === cur)
  return found ? found[1] : `全部${f.l}`
}

function onSelect(f: FilterDef, key: string) {
  set(f.k, key === '__all__' ? '' : key)
}

function clear() {
  const next: Record<string, string> = {}
  for (const k of Object.keys(props.modelValue)) next[k] = ''
  emit('update:modelValue', next)
}
</script>

<template>
  <div class="toolbar">
    <template v-if="filters.length">
      <template v-for="f in filters" :key="f.k">
        <DropdownMenu
          v-if="f.w === 'select'"
          :items="menuItemsFor(f)"
          :label="filterLabel(f)"
          align="start"
          trigger-class="flt-select"
          :aria-label="`篩選${f.l}`"
          @select="onSelect(f, $event)"
        />
        <div v-else class="flt-text">
          <Input
            :model-value="modelValue[f.k] ?? ''"
            :placeholder="`搜尋${f.l}…`"
            :aria-label="`搜尋${f.l}`"
            @update:model-value="set(f.k, $event)"
          />
        </div>
      </template>
      <button
        v-if="Object.values(modelValue).some((v) => v)"
        type="button"
        class="flt-clear"
        @click="clear"
      >清除篩選</button>
    </template>
    <span v-else class="muted">這個資源沒有定義篩選器</span>
    <div style="flex:1" />
    <span v-if="filteredCount !== undefined && filteredCount !== totalCount" class="muted">
      篩出 {{ filteredCount }} / {{ totalCount }} 筆
    </span>
    <span class="muted">每頁 {{ pageSize }} 筆</span>
  </div>
</template>

<style scoped>
.toolbar {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 15px;
  border-bottom: 1px solid var(--border);
  flex-wrap: wrap;
}
.flt-text {
  width: 180px;
}
.flt-clear {
  border: none;
  background: none;
  color: var(--brand-600);
  font: inherit;
  font-size: 12.5px;
  cursor: pointer;
  padding: 4px 6px;
  border-radius: 6px;
}
.flt-clear:hover {
  background: var(--brand-50);
}
.muted {
  color: var(--text-3);
  font-size: 12.5px;
}
</style>
