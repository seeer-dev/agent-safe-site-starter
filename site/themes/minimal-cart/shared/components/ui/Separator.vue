<script setup lang="ts">
import { cn } from '@/shared/lib/utils'

const props = withDefaults(defineProps<{
  orientation?: 'horizontal' | 'vertical'
  class?: string
}>(), {
  orientation: 'horizontal',
})
</script>

<template>
  <div
    :class="cn(
      'shrink-0 bg-border',
      props.orientation === 'horizontal' ? 'h-[1px] w-full' : 'h-full w-[1px]',
      props.class,
    )"
    role="separator"
  />
</template>
