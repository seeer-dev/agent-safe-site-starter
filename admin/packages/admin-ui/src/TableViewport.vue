<script setup lang="ts">
withDefaults(defineProps<{
  ariaLabel?: string
  class?: string
}>(), {
  ariaLabel: '',
})
</script>

<template>
  <div
    class="max-w-full overflow-x-auto overflow-y-hidden"
    data-page-local-overflow="table"
    :aria-label="ariaLabel"
    tabindex="0"
    :class="$props.class"
  >
    <slot />
  </div>
</template>
