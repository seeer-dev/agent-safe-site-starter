<script setup lang="ts">
withDefaults(defineProps<{
  title?: string
  description?: string
}>(), {
  title: '',
  description: '',
})
</script>

<template>
  <div class="empty-state">
    <div v-if="$slots['icon']" class="empty-state-icon">
      <slot name="icon" />
    </div>
    <h3 v-if="title" class="empty-state-title">{{ title }}</h3>
    <p v-if="description" class="empty-state-description">{{ description }}</p>
    <slot name="action" />
    <slot />
  </div>
</template>
