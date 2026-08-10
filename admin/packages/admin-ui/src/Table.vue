<script setup lang="ts">
withDefaults(defineProps<{
  density?: 'comfortable' | 'compact'
  bordered?: boolean
  striped?: boolean
  class?: string
}>(), {
  density: 'comfortable',
  bordered: false,
  striped: false,
})
</script>

<template>
  <table
    data-slot="table"
    :data-density="density"
    :data-bordered="bordered ? 'true' : 'false'"
    :data-striped="striped ? 'true' : 'false'"
    class="admin-table"
    :class="$props.class"
  >
    <slot />
  </table>
</template>
