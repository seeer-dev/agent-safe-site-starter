<script setup lang="ts">
withDefaults(defineProps<{
  variant?: 'brand' | 'warning' | 'danger' | 'success'
}>(), {
  variant: 'brand',
})
</script>

<template>
  <div class="note" :class="`note-${variant}`">
    <slot />
  </div>
</template>
