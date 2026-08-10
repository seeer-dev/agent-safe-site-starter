<script setup lang="ts">
withDefaults(defineProps<{
  align?: 'start' | 'center' | 'end'
  class?: string
}>(), {})
</script>

<template>
  <th
    data-slot="table-head"
    :data-align="align ?? undefined"
    scope="col"
    class="admin-table-head"
    :class="$props.class"
  >
    <slot />
  </th>
</template>
