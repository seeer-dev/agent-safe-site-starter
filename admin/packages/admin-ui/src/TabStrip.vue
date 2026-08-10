<script setup lang="ts">
withDefaults(defineProps<{
  ariaLabel?: string
  tabIndex?: number
}>(), {
  ariaLabel: '',
  tabIndex: 0,
})
</script>

<template>
  <nav
    class="tabs-bar max-w-full flex-nowrap overflow-x-auto overflow-y-hidden"
    role="navigation"
    data-page-local-overflow="tabs"
    :aria-label="ariaLabel"
    :tabindex="tabIndex"
  >
    <slot />
  </nav>
</template>
