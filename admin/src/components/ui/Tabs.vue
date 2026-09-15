<script setup lang="ts">
import { nextTick, ref } from 'vue'

export interface TabDef {
  key: string
  label: string
}

const props = defineProps<{
  tabs: TabDef[]
  modelValue: string
  ariaLabel?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [key: string]
}>()

const tablist = ref<HTMLElement | null>(null)

function activate(key: string) {
  if (key !== props.modelValue) emit('update:modelValue', key)
}

function focusTab(key: string) {
  nextTick(() => {
    tablist.value
      ?.querySelector<HTMLElement>(`[role="tab"][data-tab="${key}"]`)
      ?.focus()
  })
}

// Arrow keys both move focus AND activate (per the reference pattern).
function onKeydown(e: KeyboardEvent, idx: number) {
  const dir = e.key === 'ArrowRight' ? 1 : e.key === 'ArrowLeft' ? -1 : 0
  if (!dir) {
    if (e.key === 'Home') { e.preventDefault(); selectAt(0) }
    else if (e.key === 'End') { e.preventDefault(); selectAt(props.tabs.length - 1) }
    return
  }
  e.preventDefault()
  selectAt((idx + dir + props.tabs.length) % props.tabs.length)
}

function selectAt(idx: number) {
  const t = props.tabs[idx]
  if (!t) return
  activate(t.key)
  focusTab(t.key)
}
</script>

<template>
  <div class="tabs-wrap">
    <div
      ref="tablist"
      class="tabs"
      role="tablist"
      :aria-label="ariaLabel"
    >
      <button
        v-for="(t, i) in tabs"
        :key="t.key"
        type="button"
        role="tab"
        class="tab"
        :data-tab="t.key"
        :id="`tab-${t.key}`"
        :aria-selected="t.key === modelValue"
        :aria-controls="`tabpanel-${t.key}`"
        :tabindex="t.key === modelValue ? 0 : -1"
        @click="activate(t.key)"
        @keydown="onKeydown($event, i)"
      >{{ t.label }}</button>
    </div>
    <div
      v-for="t in tabs"
      :key="t.key"
      :id="`tabpanel-${t.key}`"
      role="tabpanel"
      :aria-labelledby="`tab-${t.key}`"
      :hidden="t.key !== modelValue"
    >
      <slot :name="`panel-${t.key}`" />
    </div>
  </div>
</template>
