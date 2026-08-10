<script setup lang="ts">
withDefaults(defineProps<{
  active?: boolean
  add?: boolean
  count?: string | number | null
  disabled?: boolean
}>(), {
  active: false,
  add: false,
  count: null,
  disabled: false,
})
</script>

<template>
  <button
    type="button"
    class="chip"
    :class="{ 'chip-active': active, 'chip-add': add }"
    :aria-pressed="active"
    :disabled="disabled"
  >
    <slot />
    <span v-if="count != null" class="chip-count">{{ count }}</span>
  </button>
</template>
