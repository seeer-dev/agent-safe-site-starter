<script setup lang="ts">
defineProps<{ class?: string }>()
</script>

<template>
  <p
    data-slot="dialog-description"
    class="text-muted-foreground text-sm"
    :class="$props.class"
  >
    <slot />
  </p>
</template>
