<script setup lang="ts">
defineProps<{
  title: string
  description?: string
  icon?: string
}>()
</script>

<template>
  <div class="emptybox">
    <div v-if="icon" class="ic">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
        <path :d="icon" />
      </svg>
    </div>
    <b>{{ title }}</b>
    <p v-if="description">{{ description }}</p>
    <div v-if="$slots.action">
      <slot name="action" />
    </div>
  </div>
</template>
