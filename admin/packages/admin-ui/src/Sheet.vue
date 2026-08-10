<script setup lang="ts">
import { provide, ref, watch } from 'vue'

const props = withDefaults(defineProps<{
  defaultOpen?: boolean
  modelValue?: boolean | undefined
}>(), {
  defaultOpen: false,
})

const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>()

const open = ref(props.modelValue ?? props.defaultOpen)

watch(() => props.modelValue, (v) => {
  if (v !== undefined) open.value = v
})

function setOpen(value: boolean) {
  open.value = value
  emit('update:modelValue', value)
}

provide('sheet-open', open)
provide('sheet-set-open', setOpen)
</script>

<template>
  <div data-slot="sheet"><slot /></div>
</template>
