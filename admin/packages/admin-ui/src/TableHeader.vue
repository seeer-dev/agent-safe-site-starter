<script setup lang="ts">
defineProps<{ class?: string }>()
</script>

<template>
  <thead data-slot="table-header" class="admin-table-header" :class="$props.class">
    <slot />
  </thead>
</template>
