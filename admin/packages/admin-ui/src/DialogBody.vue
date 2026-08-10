<script setup lang="ts">
defineProps<{ class?: string }>()
</script>

<template>
  <div
    data-slot="dialog-body"
    tabindex="0"
    class="min-h-0 flex-1 overflow-x-hidden overflow-y-auto"
    :class="$props.class"
  >
    <slot />
  </div>
</template>
