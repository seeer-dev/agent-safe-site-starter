<script setup lang="ts">
withDefaults(defineProps<{
  align?: 'start' | 'center' | 'end'
  colspan?: number
  class?: string
}>(), {})
</script>

<template>
  <td
    data-slot="table-cell"
    :data-align="align ?? undefined"
    :colspan="colspan"
    class="admin-table-cell"
    :class="$props.class"
  >
    <slot />
  </td>
</template>
