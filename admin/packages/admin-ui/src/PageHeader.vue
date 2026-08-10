<script setup lang="ts">
withDefaults(defineProps<{
  title?: string
  subtitle?: string
  eyebrow?: string
  variant?: 'default' | 'compact'
}>(), {
  title: '',
  subtitle: '',
  eyebrow: '',
  variant: 'default',
})
</script>

<template>
  <header class="page-header" :data-variant="variant">
    <div>
      <div v-if="eyebrow" class="muted text-fs-md">{{ eyebrow }}</div>
      <h1 v-if="title" class="page-title" :class="{ 'page-title-compact': variant === 'compact' }">{{ title }}</h1>
      <slot v-else name="title" />
      <p v-if="subtitle" class="page-subtitle" :class="{ 'page-subtitle-compact': variant === 'compact' }">{{ subtitle }}</p>
      <slot v-else name="subtitle" />
    </div>
    <div v-if="$slots['action']" class="page-header-actions">
      <slot name="action" />
    </div>
  </header>
</template>
