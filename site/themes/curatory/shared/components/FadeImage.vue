<script setup lang="ts">
import { ref } from 'vue'
import { cn } from '@/shared/lib/utils'

withDefaults(defineProps<{ src: string; alt: string; eager?: boolean }>(), { eager: false })
const loaded = ref(false)
</script>

<template>
  <img
    :src="src"
    :alt="alt"
    :loading="eager ? 'eager' : 'lazy'"
    decoding="async"
    :class="cn('img-fade', loaded && 'img-loaded')"
    @load="loaded = true"
  />
</template>
