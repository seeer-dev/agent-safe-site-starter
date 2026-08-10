<script setup lang="ts">
defineProps<{ class?: string }>()
</script>

<template>
  <p
    data-slot="sheet-description"
    class="text-muted-foreground text-sm"
    :class="$props.class"
  >
    <slot />
  </p>
</template>
