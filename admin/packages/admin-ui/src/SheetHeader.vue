<script setup lang="ts">
defineProps<{ class?: string }>()
</script>

<template>
  <div
    data-slot="sheet-header"
    class="flex flex-col gap-2"
    :class="$props.class"
  >
    <slot />
  </div>
</template>
