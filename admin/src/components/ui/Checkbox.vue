<script setup lang="ts">
const props = withDefaults(defineProps<{
  checked?: boolean
}>(), {
  checked: false,
})

const emit = defineEmits<{
  (e: 'update:checked', value: boolean): void
  (e: 'click', ev: MouseEvent): void
}>()

function onClick(ev: MouseEvent) {
  emit('click', ev)
  emit('update:checked', !props.checked)
}
</script>

<template>
  <span
    role="checkbox"
    :aria-checked="checked"
    tabindex="0"
    :class="`ck ${checked ? 'on' : ''}`"
    @click="onClick"
  >
    <svg
      viewBox="0 0 16 16"
      fill="none"
      stroke="currentColor"
      stroke-width="2.5"
      stroke-linecap="round"
      stroke-linejoin="round"
    >
      <path d="M3.5 8.5l3 3 6-6.5" />
    </svg>
  </span>
</template>
