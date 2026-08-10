<script setup lang="ts">
import { provide, ref, watch } from 'vue'

const props = withDefaults(defineProps<{
  defaultOpen?: boolean
  modelValue?: boolean | null
}>(), {
  defaultOpen: false,
  modelValue: null,
})

const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>()

const open = ref(props.modelValue === null ? props.defaultOpen : props.modelValue)

watch(() => props.modelValue, (v) => {
  if (v !== null) open.value = v as boolean
})

function setOpen(value: boolean) {
  open.value = value
  emit('update:modelValue', value)
}

provide('dialog-open', open)
provide('dialog-set-open', setOpen)
</script>

<template>
  <div data-slot="dialog"><slot /></div>
</template>
