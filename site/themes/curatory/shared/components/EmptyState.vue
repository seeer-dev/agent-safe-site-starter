<script setup lang="ts">
import type { Component } from 'vue'

withDefaults(
  defineProps<{
    icon: Component
    title: string
    description?: string
    actionLabel?: string
  }>(),
  { description: '', actionLabel: '' },
)
const emit = defineEmits<{ action: [] }>()
</script>

<template>
  <div class="flex flex-col items-center justify-center py-20 text-center">
    <div class="flex size-16 items-center justify-center rounded-full border bg-muted/40">
      <component :is="icon" class="size-7 text-muted-foreground/60" :stroke-width="1.5" />
    </div>
    <h3 class="mt-5 font-serif text-xl font-semibold">{{ title }}</h3>
    <p v-if="description" class="mt-2 max-w-sm text-sm text-muted-foreground">{{ description }}</p>
    <button
      v-if="actionLabel"
      class="mt-6 inline-flex h-11 items-center rounded-full bg-primary px-6 text-sm font-medium text-primary-foreground transition-colors hover:brightness-95"
      @click="emit('action')"
    >
      {{ actionLabel }}
    </button>
  </div>
</template>
