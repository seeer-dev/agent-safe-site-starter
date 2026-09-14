<script setup lang="ts">
import { computed } from 'vue'
import { cn } from '@/shared/lib/utils'
import LogoMark from './LogoMark.vue'

const props = withDefaults(
  defineProps<{
    size?: number
    animate?: boolean
    light?: boolean
    storeName?: string
    storeNameEn?: string
    compact?: boolean
  }>(),
  { size: 38, animate: false, light: false, storeName: '質選所', storeNameEn: 'Curatory', compact: false },
)

const markSize = computed(() => (props.compact ? 30 : props.size))
</script>

<template>
  <span class="inline-flex items-center gap-2.5">
    <LogoMark :size="markSize" :animate="animate" />
    <span class="flex flex-col gap-1 leading-none">
      <span
        :class="cn('font-serif font-semibold tracking-[0.24em]', compact ? 'text-base' : markSize > 30 ? 'text-lg' : 'text-base')"
      >
        {{ storeName }}
      </span>
      <span
        :class="
          cn(
            'font-display text-[10px] font-medium uppercase tracking-[0.46em]',
            light ? 'text-cream/70' : 'text-muted-foreground',
          )
        "
      >
        {{ storeNameEn }}
      </span>
    </span>
  </span>
</template>
