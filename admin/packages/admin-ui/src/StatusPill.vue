<script setup lang="ts">
withDefaults(defineProps<{
  variant?: 'wait' | 'ship' | 'done' | 'active' | 'success' | 'hold' | 'off' | 'danger' | 'draft' | 'est' | 'ended' | 'muted' | 'info'
  size?: 'default' | 'compact'
}>(), {
  variant: 'muted',
  size: 'default',
})
</script>

<template>
  <span class="status-pill" :class="[`status-pill-${variant}`, { 'status-pill-compact': size === 'compact' }]">
    <slot />
  </span>
</template>
