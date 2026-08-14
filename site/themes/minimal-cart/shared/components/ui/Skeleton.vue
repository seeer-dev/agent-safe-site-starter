<script setup lang="ts">
import { cn } from '@/shared/lib/utils'

withDefaults(defineProps<{
  class?: string
}>(), {})
</script>

<template>
  <div :class="cn('skeleton rounded-md', $props.class)" />
</template>
