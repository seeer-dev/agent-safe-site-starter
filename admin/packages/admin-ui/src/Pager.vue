<script setup lang="ts">
import { ChevronRight } from 'lucide-vue-next'

const props = withDefaults(defineProps<{
  page: number
  pageCount: number
}>(), {
  page: 1,
  pageCount: 1,
})
const emit = defineEmits<{ select: [page: number] }>()

function goPrev() {
  if (props.page > 1) emit('select', props.page - 1)
}
function goNext() {
  if (props.page < props.pageCount) emit('select', props.page + 1)
}
function goTo(p: number) {
  emit('select', p)
}
</script>

<template>
  <div class="pager" role="navigation" aria-label="分頁導覽">
    <button
      class="pager-btn"
      :disabled="page <= 1"
      aria-label="上一頁"
      @click="goPrev"
    >
      <ChevronRight :size="14" style="transform: rotate(180deg)" aria-hidden="true" />
    </button>
    <button
      v-for="p in pageCount"
      :key="p"
      class="pager-btn"
      :aria-current="p === page ? 'page' : undefined"
      :aria-label="`第 ${p} 頁`"
      @click="goTo(p)"
    >{{ p }}</button>
    <button
      class="pager-btn"
      :disabled="page >= pageCount"
      aria-label="下一頁"
      @click="goNext"
    >
      <ChevronRight :size="14" aria-hidden="true" />
    </button>
  </div>
</template>
