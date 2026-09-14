<script setup lang="ts">
import { cn } from '@/shared/lib/utils'
withDefaults(defineProps<{ className?: string }>(), { className: '' })
</script>

<template>
  <div :class="cn('shimmer rounded-md', className)" aria-hidden="true" />
</template>
