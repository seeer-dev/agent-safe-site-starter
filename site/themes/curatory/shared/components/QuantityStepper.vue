<script setup lang="ts">
import { Minus, Plus } from 'lucide-vue-next'
import { cn } from '@/shared/lib/utils'
import { toast } from '@/shared/lib/toast'

const props = withDefaults(
  defineProps<{
    modelValue: number
    min?: number
    max?: number
    size?: 'sm' | 'md'
    disabled?: boolean
  }>(),
  { min: 1, max: 99, size: 'md', disabled: false },
)
const emit = defineEmits<{ 'update:modelValue': [value: number] }>()

function decrement() {
  if (props.disabled || props.modelValue <= props.min) return
  emit('update:modelValue', props.modelValue - 1)
}
function increment() {
  if (props.disabled || props.modelValue >= props.max) {
    toast.message('已達數量上限', { description: `此商品最多購買 ${props.max} 件` })
    return
  }
  emit('update:modelValue', props.modelValue + 1)
}
</script>

<template>
  <div
    :class="cn('inline-flex items-center rounded-full border bg-card', size === 'sm' ? 'gap-0' : 'gap-0.5', disabled && 'opacity-50')"
    role="group"
    aria-label="數量調整"
  >
    <button
      type="button"
      :disabled="disabled || modelValue <= min"
      aria-label="減少數量"
      :class="cn('flex items-center justify-center rounded-full text-muted-foreground transition-colors hover:text-foreground disabled:pointer-events-none disabled:opacity-30', size === 'sm' ? 'size-8' : 'size-10 sm:size-11')"
      @click="decrement"
    >
      <Minus :class="size === 'sm' ? 'size-3' : 'size-3.5'" />
    </button>
    <span :class="cn('text-center text-sm font-medium tabular-nums', size === 'sm' ? 'w-7' : 'w-9')" aria-live="polite">
      {{ modelValue }}
    </span>
    <button
      type="button"
      :disabled="disabled || modelValue >= max"
      aria-label="增加數量"
      :class="cn('flex items-center justify-center rounded-full text-muted-foreground transition-colors hover:text-foreground disabled:pointer-events-none disabled:opacity-30', size === 'sm' ? 'size-8' : 'size-10 sm:size-11')"
      @click="increment"
    >
      <Plus :class="size === 'sm' ? 'size-3' : 'size-3.5'" />
    </button>
  </div>
</template>
