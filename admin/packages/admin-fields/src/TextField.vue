<script setup lang="ts">
withDefaults(defineProps<{
  label: string
  modelValue: string
  type?: string
  disabled?: boolean
}>(), {
  type: 'text',
  disabled: false,
})

const emit = defineEmits<{
  (event: 'update:modelValue', value: string): void
}>()

function onInput(event: Event) {
  emit('update:modelValue', (event.target as HTMLInputElement).value)
}
</script>

<template>
  <label class="my-admin-3 grid gap-admin-2 text-admin-text">
    <span>{{ label }}</span>
    <input
      class="w-full rounded-admin-sm border border-admin-border-strong bg-admin-surface p-admin-2 font-[inherit] text-admin-text focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-admin-focus disabled:cursor-not-allowed disabled:opacity-50"
      :type="type"
      :value="modelValue"
      :disabled="disabled"
      @input="onInput"
    >
  </label>
</template>
