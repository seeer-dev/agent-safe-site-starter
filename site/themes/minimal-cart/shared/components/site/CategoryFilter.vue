<script setup lang="ts">
import { ref, watch, nextTick, onMounted } from 'vue'
import { CATEGORIES } from '@/shared/lib/mock-data'
import type { Category } from '@/shared/lib/types'
import { cn } from '@/shared/lib/utils'

const props = defineProps<{
  active: Category
}>()

const emit = defineEmits<{
  change: [value: Category]
}>()

const containerRef = ref<HTMLElement | null>(null)
const pillStyle = ref({ transform: 'translateX(0)', width: '0px', opacity: 0 })

async function updatePill() {
  await nextTick()
  const container = containerRef.value
  const activeBtn = container?.querySelector('[data-active="true"]') as HTMLElement | null
  if (!container || !activeBtn) return
  const containerRect = container.getBoundingClientRect()
  const btnRect = activeBtn.getBoundingClientRect()
  // Subtract container padding (p-1 = 4px) so the pill aligns with the button.
  const pad = 4
  const x = btnRect.left - containerRect.left - pad
  const w = btnRect.width
  pillStyle.value = {
    transform: `translateX(${x}px)`,
    width: `${w}px`,
    opacity: 1,
  }
}

onMounted(updatePill)

watch(() => props.active, () => {
  updatePill()
})
</script>

<template>
  <div ref="containerRef" class="relative flex flex-wrap items-center gap-1 rounded-full border border-border/60 bg-muted/40 p-1">
    <!-- Animated pill background -->
    <span
      aria-hidden
      class="pointer-events-none absolute left-1 top-1 h-9 rounded-full bg-cta transition-all duration-300"
      :style="{
        transform: pillStyle.transform,
        width: pillStyle.width,
        opacity: pillStyle.opacity,
        transitionTimingFunction: 'cubic-bezier(0.32, 0.72, 0, 1)',
      }"
    />
    <button
      v-for="c in CATEGORIES"
      :key="c.value"
      @click="emit('change', c.value)"
      :data-active="c.value === active"
      :class="cn(
        'relative z-10 inline-flex h-9 items-center rounded-full px-4 text-sm font-medium transition-colors duration-200',
        c.value === active ? 'text-background' : 'text-foreground hover:bg-background/70',
      )"
    >
      {{ c.label }}
    </button>
  </div>
</template>
