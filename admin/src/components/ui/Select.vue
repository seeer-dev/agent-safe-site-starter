<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{
  modelValue?: string
  options?: string[] | [string, string][]
  disabled?: boolean
  width?: string
  id?: string
}>(), {
  modelValue: '',
  options: () => [],
  disabled: false,
  width: '100%',
  id: '',
})

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
}>()

const normalized = computed(() => {
  const opts = props.options as string[] | [string, string][]
  return opts.map((o) => (Array.isArray(o) ? { value: o[0], label: o[1] } : { value: o, label: o }))
})

function onChange(ev: Event) {
  emit('update:modelValue', (ev.target as HTMLSelectElement).value)
}
</script>

<template>
  <select
    :id="id || undefined"
    :value="modelValue"
    :disabled="disabled"
    class="inp"
    :style="{ width }"
    @change="onChange"
  >
    <option v-for="opt in normalized" :key="opt.value" :value="opt.value">
      {{ opt.label }}
    </option>
  </select>
</template>
