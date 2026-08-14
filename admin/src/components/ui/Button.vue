<script setup lang="ts">
const props = withDefaults(defineProps<{
  variant?: 'pri' | 'sec' | 'danger' | 'ghost' | 'default'
  size?: 'sm' | 'default'
  disabled?: boolean
  type?: 'button' | 'submit' | 'reset'
}>(), {
  variant: 'default',
  size: 'default',
  disabled: false,
  type: 'button',
})

const emit = defineEmits<{
  (e: 'click', ev: MouseEvent): void
}>()

function onClick(ev: MouseEvent) {
  if (props.disabled) return
  emit('click', ev)
}

const variantClass: Record<string, string> = {
  default: '',
  pri: 'pri',
  sec: 'sec',
  danger: 'danger',
  ghost: 'ghost',
}
</script>

<template>
  <button
    :type="type"
    :disabled="disabled"
    :class="`btn ${variantClass[variant]} ${size === 'sm' ? 'sm' : ''} ${disabled ? 'dis' : ''}`"
    @click="onClick"
  >
    <slot />
  </button>
</template>
