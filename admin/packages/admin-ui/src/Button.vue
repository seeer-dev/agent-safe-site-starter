<script setup lang="ts">
withDefaults(defineProps<{
  variant?: 'primary' | 'secondary' | 'ghost' | 'danger'
  size?: 'default' | 'sm'
  type?: 'button' | 'submit'
  disabled?: boolean
}>(), {
  variant: 'secondary',
  size: 'default',
  type: 'button',
  disabled: false,
})
</script>

<template>
  <button
    :type="type"
    class="btn"
    :class="[`btn-${variant}`, { 'btn-sm': size === 'sm' }]"
    :disabled="disabled"
  >
    <slot />
  </button>
</template>
