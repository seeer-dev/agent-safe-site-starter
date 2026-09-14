<script setup lang="ts">
import { computed } from 'vue'
import { cn } from '@/shared/lib/utils'
import { formatNTD } from '@/shared/lib/format'

const props = withDefaults(
  defineProps<{ price: number; originalPrice?: number | null; size?: 'sm' | 'md' | 'lg' | 'xl' }>(),
  { originalPrice: null, size: 'md' },
)

const SIZE_CLASS = {
  sm: 'text-sm',
  md: 'text-base',
  lg: 'text-xl',
  xl: 'text-2xl sm:text-3xl',
} as const

const sale = computed(() => props.originalPrice != null && props.originalPrice > props.price)
const off = computed(() =>
  sale.value ? Math.round(((props.originalPrice! - props.price) / props.originalPrice!) * 100) : 0,
)
</script>

<template>
  <p :class="cn('flex items-baseline gap-2 font-medium tabular-nums', SIZE_CLASS[size])">
    <span v-if="sale" class="text-xs font-normal text-muted-foreground line-through sm:text-sm">
      {{ formatNTD(originalPrice!) }}
    </span>
    <span :class="cn(sale && 'text-primary')">{{ formatNTD(price) }}</span>
    <span
      v-if="sale"
      class="rounded-full bg-primary/10 px-1.5 py-0.5 text-[10px] font-normal tracking-wider text-primary"
    >
      -{{ off }}%
    </span>
  </p>
</template>
