<script setup lang="ts">
withDefaults(defineProps<{
  modelValue?: string
  placeholder?: string
  disabled?: boolean
  id?: string
}>(), {
  modelValue: '',
  placeholder: '',
  disabled: false,
  id: '',
})

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
}>()

function onInput(ev: Event) {
  emit('update:modelValue', (ev.target as HTMLTextAreaElement).value)
}
</script>

<template>
  <textarea
    :id="id || undefined"
    :value="modelValue"
    :placeholder="placeholder"
    :disabled="disabled"
    class="inp"
    @input="onInput"
  />
</template>
