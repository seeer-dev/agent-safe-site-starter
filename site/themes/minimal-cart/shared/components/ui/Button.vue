<script setup lang="ts">
import { computed } from 'vue'
import { cn } from '@/shared/lib/utils'

const props = withDefaults(defineProps<{
  variant?: 'default' | 'secondary' | 'destructive' | 'outline' | 'ghost' | 'link' | 'cta'
  size?: 'default' | 'sm' | 'lg' | 'icon'
  as?: string
  disabled?: boolean
  type?: 'button' | 'submit' | 'reset'
}>(), {
  variant: 'default',
  size: 'default',
  as: 'button',
  disabled: false,
  type: 'button',
})

const variantClasses: Record<string, string> = {
  default: 'bg-primary text-primary-foreground hover:bg-primary/90',
  cta: 'bg-cta text-cta-foreground hover:bg-cta/90 shadow-sm',
  destructive: 'bg-destructive text-destructive-foreground hover:bg-destructive/90',
  outline: 'border border-input bg-background hover:bg-accent hover:text-accent-foreground',
  secondary: 'bg-secondary text-secondary-foreground hover:bg-secondary/80',
  ghost: 'hover:bg-accent hover:text-accent-foreground',
  link: 'text-primary underline-offset-4 hover:underline',
}

const sizeClasses: Record<string, string> = {
  default: 'h-10 px-4 py-2',
  sm: 'h-9 rounded-md px-3',
  lg: 'h-11 rounded-md px-8',
  icon: 'h-10 w-10',
}

const classes = computed(() =>
  cn(
    'inline-flex items-center justify-center gap-2 whitespace-nowrap rounded-md text-sm font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50',
    variantClasses[props.variant],
    sizeClasses[props.size],
  )
)
</script>

<template>
  <component :is="as" :class="classes" :type="as === 'button' ? type : undefined" :disabled="disabled">
    <slot />
  </component>
</template>
