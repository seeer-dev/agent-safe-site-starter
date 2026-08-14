<script setup lang="ts">
withDefaults(defineProps<{
  tone?: 'neutral' | 'success' | 'warn' | 'danger' | 'info'
  label?: string
}>(), {
  tone: 'neutral',
  label: '',
})

const toneClass: Record<string, string> = {
  neutral: 'neutral',
  success: 'success',
  warn: 'warn',
  danger: 'danger',
  info: 'info',
}
</script>

<template>
  <span :class="`st ${toneClass[tone]}`">
    {{ label }}
  </span>
</template>
