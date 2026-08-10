<script setup lang="ts">
import { provide, ref, watch } from 'vue'

const props = withDefaults(defineProps<{
  defaultValue?: string
  modelValue?: string | undefined
  class?: string
}>(), {
  defaultValue: '',
})

const emit = defineEmits<{ 'update:modelValue': [value: string] }>()

const selected = ref(props.modelValue ?? props.defaultValue)

watch(() => props.modelValue, (v) => {
  if (v !== undefined) selected.value = v
})

function select(value: string) {
  selected.value = value
  emit('update:modelValue', value)
}

provide('tabs-selected', selected)
provide('tabs-select', select)
</script>

<template>
  <div class="flex flex-col gap-2" data-slot="tabs" :class="$props.class">
    <slot />
  </div>
</template>
