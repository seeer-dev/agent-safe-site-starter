<script setup lang="ts">
defineProps<{ class?: string }>()
</script>

<template>
  <tbody data-slot="table-body" class="admin-table-body" :class="$props.class">
    <slot />
  </tbody>
</template>
