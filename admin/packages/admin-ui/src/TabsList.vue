<script setup lang="ts">
defineProps<{
  ariaLabel?: string
  class?: string
}>()
</script>

<template>
  <div
    class="bg-[var(--surface-100)] text-[var(--surface-600)] inline-flex h-9 w-fit items-center gap-0.5 rounded-lg p-1"
    data-slot="tabs-list"
    role="tablist"
    :aria-label="ariaLabel"
    :class="$props.class"
  >
    <slot />
  </div>
</template>
