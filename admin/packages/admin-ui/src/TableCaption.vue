<script setup lang="ts">
defineProps<{ class?: string }>()
</script>

<template>
  <caption data-slot="table-caption" class="admin-table-caption" :class="$props.class">
    <slot />
  </caption>
</template>
