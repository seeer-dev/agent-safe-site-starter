<script setup lang="ts">
import { inject, computed, type Ref } from 'vue'

const props = defineProps<{
  value: string
  class?: string
}>()

const selected = inject<Ref<string>>('tabs-selected')!
const select = inject<(v: string) => void>('tabs-select')!

const isActive = computed(() => selected.value === props.value)
</script>

<template>
  <button
    type="button"
    role="tab"
    :aria-selected="isActive"
    :data-state="isActive ? 'active' : 'inactive'"
    data-slot="tabs-trigger"
    class="data-[state=active]:bg-[var(--surface-0)] data-[state=active]:text-[var(--surface-900)] data-[state=active]:shadow-sm text-[var(--surface-600)] inline-flex h-7 items-center justify-center gap-1.5 rounded-md border border-transparent px-3 py-1 text-[13px] font-medium whitespace-nowrap transition-all hover:text-[var(--surface-800)] disabled:pointer-events-none disabled:opacity-50"
    :class="$props.class"
    @click="select(value)"
  >
    <slot />
  </button>
</template>
