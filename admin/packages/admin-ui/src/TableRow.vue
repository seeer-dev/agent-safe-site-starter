<script setup lang="ts">
withDefaults(defineProps<{
  selected?: boolean
  interactive?: boolean
  class?: string
}>(), {
  selected: false,
  interactive: false,
})
</script>

<template>
  <tr
    data-slot="table-row"
    :data-state="selected ? 'selected' : undefined"
    :data-interactive="interactive ? 'true' : 'false'"
    :aria-selected="selected ? true : undefined"
    class="admin-table-row"
    :class="$props.class"
  >
    <slot />
  </tr>
</template>
