<script setup lang="ts">
import { computed } from 'vue'
import { ChevronLeft, ChevronRight } from 'lucide-vue-next'
import { cn } from '@/shared/lib/utils'

const props = withDefaults(defineProps<{
  currentPage: number
  totalPages: number
  siblingCount?: number
}>(), {
  siblingCount: 1,
})

const emit = defineEmits<{
  pageChange: [page: number]
}>()

type PageItem = number | 'ellipsis'

const pages = computed<PageItem[]>(() => {
  if (props.totalPages <= 1) return []
  const result: PageItem[] = []
  const left = Math.max(1, props.currentPage - props.siblingCount)
  const right = Math.min(props.totalPages, props.currentPage + props.siblingCount)

  result.push(1)

  if (left > 2) {
    result.push('ellipsis')
  } else if (left === 2) {
    result.push(2)
  }

  for (let i = Math.max(2, left); i <= Math.min(props.totalPages - 1, right); i++) {
    if (!result.includes(i)) result.push(i)
  }

  if (right < props.totalPages - 1) {
    result.push('ellipsis')
  } else if (right === props.totalPages - 1) {
    if (!result.includes(props.totalPages - 1)) result.push(props.totalPages - 1)
  }

  if (props.totalPages > 1 && !result.includes(props.totalPages)) {
    result.push(props.totalPages)
  }

  return result
})

const canPrev = computed(() => props.currentPage > 1)
const canNext = computed(() => props.currentPage < props.totalPages)

const btnBase = 'grid h-8 min-w-8 place-items-center rounded-full text-xs font-medium transition-colors'
const btnActive = 'bg-cta text-cta-foreground'
const btnIdle = 'text-muted-foreground hover:bg-muted hover:text-foreground'
const btnDisabled = 'text-muted-foreground/40 cursor-not-allowed'
</script>

<template>
  <div v-if="totalPages > 1" class="flex items-center justify-center gap-1" role="navigation" aria-label="分頁">
    <button
      @click="canPrev && emit('pageChange', currentPage - 1)"
      :disabled="!canPrev"
      :class="cn(btnBase, 'w-8', canPrev ? btnIdle : btnDisabled)"
      aria-label="上一頁"
    >
      <ChevronLeft class="h-3.5 w-3.5" />
    </button>

    <template v-for="(p, i) in pages" :key="i">
      <span
        v-if="p === 'ellipsis'"
        class="grid h-8 w-6 place-items-center text-xs text-muted-foreground"
      >⋯</span>
      <button
        v-else
        @click="emit('pageChange', p)"
        :class="cn(btnBase, 'px-2', p === currentPage ? btnActive : btnIdle)"
        :aria-current="p === currentPage ? 'page' : undefined"
        :aria-label="`第 ${p} 頁`"
      >{{ p }}</button>
    </template>

    <button
      @click="canNext && emit('pageChange', currentPage + 1)"
      :disabled="!canNext"
      :class="cn(btnBase, 'w-8', canNext ? btnIdle : btnDisabled)"
      aria-label="下一頁"
    >
      <ChevronRight class="h-3.5 w-3.5" />
    </button>
  </div>
</template>
