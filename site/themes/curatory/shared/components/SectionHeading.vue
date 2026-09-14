<script setup lang="ts">
import { cn } from '@/shared/lib/utils'

withDefaults(
  defineProps<{
    eyebrow: string
    title: string
    description?: string
    light?: boolean
  }>(),
  { description: '', light: false },
)
</script>

<template>
  <div class="w-full">
    <p :class="cn('text-[11px] font-medium uppercase tracking-[0.35em]', light ? 'text-gold' : 'text-primary')">
      {{ eyebrow }}
    </p>
    <div class="mt-3 flex flex-wrap items-end justify-between gap-x-6 gap-y-3">
      <h2 :class="cn('font-serif text-2xl font-semibold sm:text-3xl lg:text-4xl', light && 'text-cream')">
        {{ title }}
      </h2>
      <slot name="action" />
    </div>
    <hr :class="cn('mt-5 border-t', light ? 'border-cream/15' : 'border-border')" />
    <p v-if="description" :class="cn('mt-4 max-w-2xl text-sm leading-relaxed text-muted-foreground', light && 'text-cream/65')">
      {{ description }}
    </p>
  </div>
</template>
