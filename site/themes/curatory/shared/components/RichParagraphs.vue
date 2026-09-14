<script setup lang="ts">
// 內文以 \n 切段落，支援 **粗體** 標記（對應 reference RichParagraphs）
import { computed } from 'vue'
import { cn } from '@/shared/lib/utils'

const props = withDefaults(defineProps<{ text: string }>(), { text: '' })

const paragraphs = computed(() =>
  props.text
    .split('\n')
    .map((p) => p.trim())
    .filter(Boolean)
    .map((p) =>
      p.split(/\*\*(.+?)\*\*/g).map((part, i) => ({ text: part, bold: i % 2 === 1 })),
    ),
)
</script>

<template>
  <div class="space-y-4">
    <p v-for="(para, i) in paragraphs" :key="i" class="leading-relaxed">
      <template v-for="(part, j) in para" :key="j">
        <strong v-if="part.bold" class="font-semibold text-foreground">{{ part.text }}</strong>
        <span v-else>{{ part.text }}</span>
      </template>
    </p>
  </div>
</template>
