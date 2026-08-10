<script setup lang="ts">
defineProps<{ class?: string }>()
</script>

<template>
  <div
    data-slot="dialog-header"
    class="flex shrink-0 flex-col gap-2 text-center sm:text-left"
    :class="$props.class"
  >
    <slot />
  </div>
</template>
