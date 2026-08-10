<script setup lang="ts">
defineProps<{ class?: string }>()
</script>

<template>
  <h2
    data-slot="sheet-title"
    class="text-lg leading-none font-semibold"
    :class="$props.class"
  >
    <slot />
  </h2>
</template>
