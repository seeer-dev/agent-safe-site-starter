<script setup lang="ts">
defineProps<{ class?: string }>()
</script>

<template>
  <div
    data-slot="dialog-footer"
    class="flex shrink-0 flex-col-reverse gap-2 sm:flex-row sm:justify-end"
    :class="$props.class"
  >
    <slot />
  </div>
</template>
